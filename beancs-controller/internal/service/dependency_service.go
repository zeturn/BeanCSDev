package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type DependencyService struct {
	db          *gorm.DB
	registry    *DependencyDefinitionRegistry
	credentials *CredentialService
	gitops      *GitOpsService
	k8s         *k8s.Manager
}

func NewDependencyService(db *gorm.DB, registry *DependencyDefinitionRegistry) *DependencyService {
	return &DependencyService{db: db, registry: registry}
}

func (s *DependencyService) Registry() *DependencyDefinitionRegistry {
	return s.registry
}

func (s *DependencyService) SetDeployers(credentials *CredentialService, gitops *GitOpsService, k8sManager *k8s.Manager) {
	s.credentials = credentials
	s.gitops = gitops
	s.k8s = k8sManager
}

func (s *DependencyService) CreateStandalone(ctx context.Context, userID, tenantID, tenantCode string, req dto.CreateManagedDependencyRequest) (*model.ManagedDependency, error) {
	appName := harborName(coalesce(req.ApplicationName, "dep-"+req.Name))
	if err := ensureDependencyApplicationNameAvailable(ctx, s.db, appName); err != nil {
		return nil, err
	}
	app := &model.Application{
		Name:        appName,
		DisplayName: coalesce(req.DisplayName, req.Name),
		Type:        model.ApplicationTypeSingle,
		Namespace:   coalesce(req.Namespace, appName),
		OwnerID:     userID,
		TenantID:    tenantID,
		TenantCode:  tenantCode,
		Status:      model.ApplicationStatusCreating,
	}
	if err := s.db.WithContext(ctx).Create(app).Error; err != nil {
		return nil, err
	}
	dep, err := s.Create(ctx, userID, app.ID, req)
	if err != nil {
		_ = s.db.WithContext(ctx).Model(app).Update("status", model.ApplicationStatusPartialFailed).Error
		return nil, err
	}
	if req.GitHubCredentialID != 0 && dep.DeployMethod != model.DependencyDeployMethodExternal {
		if err := s.deployStandalone(ctx, userID, *app, *dep, req.GitHubCredentialID); err != nil {
			_ = s.db.WithContext(ctx).Model(app).Update("status", model.ApplicationStatusPartialFailed).Error
			return dep, err
		}
	}
	_ = s.db.WithContext(ctx).Model(app).Update("status", model.ApplicationStatusActive).Error
	return dep, nil
}

func (s *DependencyService) deployStandalone(ctx context.Context, userID string, app model.Application, dep model.ManagedDependency, credentialID uint) error {
	if s.credentials == nil || s.gitops == nil {
		return nil
	}
	if err := s.credentials.RequireAccess(userID, model.CredentialTypeGitHub, credentialID, false); err != nil {
		return err
	}
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, credentialID).Error; err != nil {
		return err
	}
	token, err := s.credentials.GitHubToken(ctx, cred)
	if err != nil {
		return err
	}
	if s.k8s != nil {
		if err := s.k8s.CreateNamespace(ctx, dep.Namespace, app.Name); err != nil {
			return err
		}
		if err := s.k8s.UpsertSecret(ctx, dep.Namespace, dep.SecretName, app.Name, dependencySecretRuntimeData(dep)); err != nil {
			return err
		}
	}
	if err := s.gitops.CommitDependencyManifests(ctx, token, cred, app, dep); err != nil {
		return err
	}
	if s.k8s != nil && cred.GitOpsRepo != "" {
		return s.k8s.ApplyArgoCDApplication(ctx, dependencyRootArgoApplicationName(app.Name, dep.Name), gitOpsRepoURL(cred), fmt.Sprintf("apps/%s/dependencies/%s", app.Name, dep.Name), dep.Namespace)
	}
	return nil
}

func (s *DependencyService) Create(ctx context.Context, userID string, applicationID uint, req dto.CreateManagedDependencyRequest) (*model.ManagedDependency, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", applicationID, userID).First(&app).Error; err != nil {
		return nil, err
	}
	if req.ExistingDependencyID != 0 {
		return s.attachExisting(ctx, userID, app, req)
	}
	def, ok := s.registry.Get(req.Type)
	if !ok {
		return nil, fmt.Errorf("unknown dependency type %q", req.Type)
	}
	req.Type = def.Spec.Type
	if req.DeployMethod == "" {
		req.DeployMethod = def.Spec.DefaultDeployMethod
	}
	if !containsString(def.Spec.SupportedDeployMethods, req.DeployMethod) && req.DeployMethod != def.Spec.DefaultDeployMethod {
		return nil, fmt.Errorf("deploy_method %q is not supported by %s", req.DeployMethod, req.Type)
	}
	config := applyDependencyConfigDefaults(def, req.Config)
	secretData := dependencySecretData(def, config)
	serviceName := req.Name
	external := req.External || req.DeployMethod == model.DependencyDeployMethodExternal
	controlled := !external
	if req.Controlled != nil {
		controlled = *req.Controlled
	}
	if !external {
		controlled = true
	}
	if external {
		serviceName = strings.TrimSpace(fmt.Sprint(config["host"]))
		if serviceName == "" {
			return nil, fmt.Errorf("external dependency %q requires config.host", req.Name)
		}
		if controlled {
			if err := validateExternalAdminConfig(req.Type, config); err != nil {
				return nil, err
			}
		}
	}
	outputs := resolveDependencyOutputs(def, dependencyRuntimeHost(serviceName, coalesce(app.Namespace, coalesce(app.Name, req.Name)), external), config, secretData)
	outputs = applyExternalDependencyOutputs(outputs, config, external)
	secretName := fmt.Sprintf("%s-%s-credentials", app.Name, req.Name)
	dep := &model.ManagedDependency{
		ApplicationID:     app.ID,
		Name:              req.Name,
		Type:              req.Type,
		Version:           req.Version,
		DeployMethod:      req.DeployMethod,
		Namespace:         coalesce(app.Namespace, coalesce(app.Name, req.Name)),
		ServiceName:       serviceName,
		SecretName:        secretName,
		DefinitionName:    def.Metadata.Name,
		DefinitionVersion: "v1",
		Config:            config,
		Outputs:           outputs,
		Status:            model.DependencyStatusReady,
		Shared:            req.Shared,
		External:          external,
		Controlled:        controlled,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(dep).Error; err != nil {
			return err
		}
		defaultCredential := s.defaultDependencyCredential(dep, req.Credential)
		if defaultCredential != nil {
			if err := tx.Create(defaultCredential).Error; err != nil {
				return err
			}
		}
		component := model.ApplicationComponent{
			ApplicationID: app.ID,
			Name:          dep.Name,
			Kind:          model.ApplicationComponentKindDependency,
			DependencyID:  &dep.ID,
			Status:        model.DependencyStatusReady,
		}
		return tx.Create(&component).Error
	})
	if err != nil {
		return nil, err
	}
	if external && controlled && req.Credential != nil {
		if cred, err := s.findCredential(ctx, dep.ID, 0, req.Credential.Name); err == nil {
			if err := s.reconcileRuntimeCredential(ctx, *dep, cred); err != nil {
				_ = s.db.WithContext(ctx).Model(&cred).Updates(map[string]any{"status": model.DependencyStatusFailed, "description": strings.TrimSpace(coalesce(cred.Description, "") + " runtime reconcile failed: " + err.Error())}).Error
				return nil, err
			}
		}
	}
	return dep, nil
}

func (s *DependencyService) attachExisting(ctx context.Context, userID string, app model.Application, req dto.CreateManagedDependencyRequest) (*model.ManagedDependency, error) {
	var dep model.ManagedDependency
	if err := s.db.WithContext(ctx).
		Joins("JOIN applications ON applications.id = managed_dependencies.application_id").
		Where("managed_dependencies.id = ? AND applications.owner_id = ? AND (managed_dependencies.shared = TRUE OR managed_dependencies.external = TRUE)", req.ExistingDependencyID, userID).
		First(&dep).Error; err != nil {
		return nil, err
	}
	componentName := coalesce(req.Name, dep.Name)
	err := s.db.WithContext(ctx).Create(&model.ApplicationComponent{
		ApplicationID: app.ID,
		Name:          componentName,
		Kind:          model.ApplicationComponentKindDependency,
		DependencyID:  &dep.ID,
		Status:        model.DependencyStatusReady,
	}).Error
	if err != nil {
		return nil, err
	}
	return &dep, nil
}

func (s *DependencyService) List(ctx context.Context, userID string, applicationID uint) ([]model.ManagedDependency, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", applicationID, userID).First(&app).Error; err != nil {
		return nil, err
	}
	var deps []model.ManagedDependency
	ids := s.dependencyIDsForApplication(ctx, app.ID)
	err := s.db.WithContext(ctx).Where("id IN ?", ids).Order("name asc").Find(&deps).Error
	return deps, err
}

func (s *DependencyService) ListReusable(ctx context.Context, userID string) ([]model.ManagedDependency, error) {
	var deps []model.ManagedDependency
	err := s.db.WithContext(ctx).
		Joins("JOIN applications ON applications.id = managed_dependencies.application_id").
		Where("applications.owner_id = ? AND (managed_dependencies.shared = TRUE OR managed_dependencies.external = TRUE)", userID).
		Order("managed_dependencies.name asc").
		Find(&deps).Error
	return deps, err
}

func (s *DependencyService) LinkProject(ctx context.Context, userID string, projectID uint, req dto.LinkProjectDependencyRequest) (*model.Project, error) {
	var project model.Project
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", projectID, userID).First(&project).Error; err != nil {
		return nil, err
	}
	if project.ApplicationID == nil {
		return nil, fmt.Errorf("project is not part of an application")
	}
	dep, err := s.findDependencyForApplication(ctx, *project.ApplicationID, req.DependencyID, req.Dependency)
	if err != nil {
		return nil, err
	}
	project.DependsOn = appendUnique(project.DependsOn, dep.Name)
	entry := map[string]any{"dependency": dep.Name}
	if req.DependencyID != 0 {
		entry["dependency_id"] = req.DependencyID
	}
	if req.Credential != "" {
		entry["credential"] = req.Credential
	}
	if req.CredentialID != 0 {
		entry["credential_id"] = req.CredentialID
	}
	if req.Preset != "" {
		entry["preset"] = req.Preset
	}
	if len(req.Mappings) > 0 {
		entry["mappings"] = req.Mappings
	}
	project.EnvFromDependencies = appendEnvFromDependency(project.EnvFromDependencies, entry)
	if err := s.db.WithContext(ctx).Model(&project).Updates(map[string]any{
		"depends_on":            project.DependsOn,
		"env_from_dependencies": project.EnvFromDependencies,
	}).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *DependencyService) EnvForDependency(dep model.ManagedDependency, preset string, mappings map[string]any) (map[string]string, error) {
	return s.EnvForDependencyCredential(context.Background(), dep, 0, "", preset, mappings)
}

func (s *DependencyService) EnvForDependencyCredential(ctx context.Context, dep model.ManagedDependency, credentialID uint, credentialName, preset string, mappings map[string]any) (map[string]string, error) {
	def, ok := s.registry.Get(dep.DefinitionName)
	if !ok {
		return nil, fmt.Errorf("dependency definition %q not found", dep.DefinitionName)
	}
	outputSource := dep.Outputs
	if credentialID != 0 || credentialName != "" {
		cred, err := s.findCredential(ctx, dep.ID, credentialID, credentialName)
		if err != nil {
			return nil, err
		}
		outputSource = cred.Outputs
	}
	outputs := flattenDependencyOutputs(outputSource)
	env := map[string]string{}
	if preset != "" {
		p, ok := def.Spec.EnvPresets[preset]
		if !ok {
			return nil, fmt.Errorf("env preset %q not found for %s", preset, dep.Type)
		}
		keys := make([]string, 0, len(p.Env))
		for key := range p.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			val := p.Env[key]
			if val.Output != "" {
				env[key] = outputs[val.Output]
			} else if val.Value != "" {
				env[key] = renderTemplate(val.Value, outputs)
			}
		}
	}
	for key, raw := range mappings {
		switch v := raw.(type) {
		case string:
			env[key] = renderTemplate(v, outputs)
		case map[string]any:
			if output, ok := v["secretOutput"].(string); ok {
				env[key] = outputs[output]
			}
			if output, ok := v["output"].(string); ok {
				env[key] = outputs[output]
			}
		}
	}
	return env, nil
}

func (s *DependencyService) CreateCredential(ctx context.Context, userID string, dependencyID uint, req dto.CreateDependencyCredentialRequest) (*model.DependencyCredential, error) {
	dep, err := s.dependencyForOwner(ctx, userID, dependencyID)
	if err != nil {
		return nil, err
	}
	cred := s.buildDependencyCredential(dep, req)
	if err := s.db.WithContext(ctx).Create(cred).Error; err != nil {
		return nil, err
	}
	if err := s.reconcileRuntimeCredential(ctx, dep, *cred); err != nil {
		_ = s.db.WithContext(ctx).Model(cred).Updates(map[string]any{"status": model.DependencyStatusFailed, "description": strings.TrimSpace(coalesce(cred.Description, "") + " runtime reconcile failed: " + err.Error())}).Error
		return nil, err
	}
	return cred, nil
}

func (s *DependencyService) ListCredentials(ctx context.Context, userID string, dependencyID uint) ([]model.DependencyCredential, error) {
	if _, err := s.dependencyForOwner(ctx, userID, dependencyID); err != nil {
		return nil, err
	}
	var creds []model.DependencyCredential
	err := s.db.WithContext(ctx).Where("dependency_id = ?", dependencyID).Order("name asc").Find(&creds).Error
	return s.MaskCredentials(creds), err
}

func (s *DependencyService) Mask(dep model.ManagedDependency) model.ManagedDependency {
	def, ok := s.registry.Get(dep.DefinitionName)
	if !ok {
		def, _ = s.registry.Get(dep.Type)
	}
	maskedConfig := model.JSONMap{}
	for key, value := range dep.Config {
		maskedConfig[key] = value
	}
	props, _ := def.Spec.ConfigSchema["properties"].(map[string]any)
	for key, raw := range props {
		prop, _ := raw.(map[string]any)
		if secret, _ := prop["secret"].(bool); secret {
			if _, exists := maskedConfig[key]; exists {
				maskedConfig[key] = "********"
			}
		}
	}
	dep.Config = maskedConfig
	dep.Outputs = maskOutputSecrets(dep.Outputs)
	return dep
}

func (s *DependencyService) MaskList(deps []model.ManagedDependency) []model.ManagedDependency {
	out := make([]model.ManagedDependency, 0, len(deps))
	for _, dep := range deps {
		out = append(out, s.Mask(dep))
	}
	return out
}

func (s *DependencyService) MaskCredential(cred model.DependencyCredential) model.DependencyCredential {
	cred.Config = maskJSONSecrets(cred.Config)
	cred.Outputs = maskOutputSecrets(cred.Outputs)
	return cred
}

func (s *DependencyService) MaskCredentials(creds []model.DependencyCredential) []model.DependencyCredential {
	out := make([]model.DependencyCredential, 0, len(creds))
	for _, cred := range creds {
		out = append(out, s.MaskCredential(cred))
	}
	return out
}

func (s *DependencyService) dependencyIDsForApplication(ctx context.Context, applicationID uint) []uint {
	seen := map[uint]bool{}
	var deps []model.ManagedDependency
	_ = s.db.WithContext(ctx).Select("id").Where("application_id = ?", applicationID).Find(&deps).Error
	for _, dep := range deps {
		seen[dep.ID] = true
	}
	var components []model.ApplicationComponent
	_ = s.db.WithContext(ctx).Select("dependency_id").Where("application_id = ? AND dependency_id IS NOT NULL", applicationID).Find(&components).Error
	for _, component := range components {
		if component.DependencyID != nil {
			seen[*component.DependencyID] = true
		}
	}
	out := make([]uint, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	if len(out) == 0 {
		out = append(out, 0)
	}
	return out
}

func (s *DependencyService) findDependencyForApplication(ctx context.Context, applicationID uint, dependencyID uint, name string) (model.ManagedDependency, error) {
	var dep model.ManagedDependency
	ids := s.dependencyIDsForApplication(ctx, applicationID)
	q := s.db.WithContext(ctx).Where("id IN ?", ids)
	if dependencyID != 0 {
		q = q.Where("id = ?", dependencyID)
	} else {
		q = q.Where("name = ?", name)
	}
	err := q.First(&dep).Error
	return dep, err
}

func (s *DependencyService) dependencyForOwner(ctx context.Context, userID string, dependencyID uint) (model.ManagedDependency, error) {
	var dep model.ManagedDependency
	err := s.db.WithContext(ctx).
		Joins("JOIN applications ON applications.id = managed_dependencies.application_id").
		Where("managed_dependencies.id = ? AND applications.owner_id = ?", dependencyID, userID).
		First(&dep).Error
	return dep, err
}

func (s *DependencyService) findCredential(ctx context.Context, dependencyID uint, credentialID uint, name string) (model.DependencyCredential, error) {
	var cred model.DependencyCredential
	q := s.db.WithContext(ctx).Where("dependency_id = ?", dependencyID)
	if credentialID != 0 {
		q = q.Where("id = ?", credentialID)
	} else {
		q = q.Where("name = ?", name)
	}
	err := q.First(&cred).Error
	return cred, err
}

func ensureDependencyApplicationNameAvailable(ctx context.Context, db *gorm.DB, name string) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.Application{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("application name %q already exists; choose a different application name", name)
	}
	return nil
}

func applyDependencyConfigDefaults(def DependencyDefinition, input model.JSONMap) model.JSONMap {
	out := model.JSONMap{}
	for k, v := range input {
		out[k] = v
	}
	props, _ := def.Spec.ConfigSchema["properties"].(map[string]any)
	for key, raw := range props {
		prop, _ := raw.(map[string]any)
		if _, exists := out[key]; exists {
			continue
		}
		if gen, ok := prop["generate"].(map[string]any); ok {
			length := 32
			if rawLength, ok := gen["length"].(int); ok {
				length = rawLength
			}
			out[key] = randomToken(length)
			continue
		}
		if def, ok := prop["default"]; ok {
			out[key] = def
		}
	}
	return out
}

func (s *DependencyService) defaultDependencyCredential(dep *model.ManagedDependency, req *dto.CreateDependencyCredentialRequest) *model.DependencyCredential {
	if req != nil {
		return s.buildDependencyCredential(*dep, *req)
	}
	if dep.External {
		return nil
	}
	return &model.DependencyCredential{
		DependencyID: dep.ID,
		Name:         "default",
		Config:       dep.Config,
		Outputs:      dep.Outputs,
		Status:       model.DependencyStatusReady,
	}
}

func (s *DependencyService) buildDependencyCredential(dep model.ManagedDependency, req dto.CreateDependencyCredentialRequest) *model.DependencyCredential {
	config := model.JSONMap{}
	for k, v := range dep.Config {
		config[k] = v
	}
	for k, v := range req.Config {
		config[k] = v
	}
	definitionName := coalesce(dep.DefinitionName, dep.Type)
	def, _ := s.registry.Get(definitionName)
	secretData := dependencySecretData(def, config)
	outputs := resolveDependencyOutputs(def, dependencyRuntimeHost(dep.ServiceName, dep.Namespace, dep.External), config, secretData)
	outputs = applyExternalDependencyOutputs(outputs, config, dep.External)
	return &model.DependencyCredential{
		DependencyID: dep.ID,
		Name:         req.Name,
		Description:  req.Description,
		Config:       config,
		Outputs:      outputs,
		Status:       model.DependencyStatusReady,
	}
}

func (s *DependencyService) reconcileRuntimeCredential(ctx context.Context, dep model.ManagedDependency, cred model.DependencyCredential) error {
	if s.k8s == nil {
		return nil
	}
	switch dep.Type {
	case "mysql":
		outputs := flattenDependencyOutputs(cred.Outputs)
		if dep.External {
			if !dep.Controlled {
				return nil
			}
			return s.k8s.ReconcileExternalMySQLCredential(ctx, externalDatabaseRuntime(dep, cred, outputs))
		}
		return s.k8s.ReconcileMySQLCredential(ctx, k8s.MySQLCredentialRuntime{
			Namespace:      dep.Namespace,
			ServiceName:    dep.ServiceName,
			SecretName:     dep.SecretName,
			Database:       coalesce(outputs["database"], fmt.Sprint(cred.Config["database"])),
			Username:       coalesce(outputs["username"], fmt.Sprint(cred.Config["username"])),
			Password:       coalesce(outputs["password"], fmt.Sprint(cred.Config["password"])),
			Port:           coalesce(coalesce(outputs["port"], fmt.Sprint(cred.Config["port"])), "3306"),
			DependencyName: dep.Name,
			CredentialName: cred.Name,
		})
	case "postgresql":
		if dep.External && dep.Controlled {
			return s.k8s.ReconcileExternalPostgreSQLCredential(ctx, externalDatabaseRuntime(dep, cred, flattenDependencyOutputs(cred.Outputs)))
		}
		return nil
	case "rabbitmq":
		if dep.External && dep.Controlled {
			return s.k8s.ReconcileExternalRabbitMQCredential(ctx, externalDatabaseRuntime(dep, cred, flattenDependencyOutputs(cred.Outputs)))
		}
		return nil
	default:
		return nil
	}
}

func validateExternalAdminConfig(depType string, config model.JSONMap) error {
	switch depType {
	case "mysql", "postgresql", "rabbitmq":
	default:
		return fmt.Errorf("controlled external %s dependencies are not supported yet", depType)
	}
	if strings.TrimSpace(fmt.Sprint(config["admin_username"])) == "" || strings.TrimSpace(fmt.Sprint(config["admin_password"])) == "" {
		return fmt.Errorf("controlled external %s dependency requires admin_username and admin_password", depType)
	}
	if depType == "rabbitmq" && strings.TrimSpace(fmt.Sprint(config["management_port"])) == "" {
		config["management_port"] = "15672"
	}
	return nil
}

func externalDatabaseRuntime(dep model.ManagedDependency, cred model.DependencyCredential, outputs map[string]string) k8s.ExternalCredentialRuntime {
	port := coalesce(outputs["port"], fmt.Sprint(cred.Config["port"]))
	if strings.TrimSpace(port) == "" || port == "<nil>" {
		port = fmt.Sprint(dep.Config["port"])
	}
	return k8s.ExternalCredentialRuntime{
		Namespace:       "",
		Host:            coalesce(outputs["host"], fmt.Sprint(dep.Config["host"])),
		Port:            port,
		Database:        coalesce(outputs["database"], fmt.Sprint(cred.Config["database"])),
		Username:        coalesce(outputs["username"], fmt.Sprint(cred.Config["username"])),
		Password:        coalesce(outputs["password"], fmt.Sprint(cred.Config["password"])),
		AdminUsername:   fmt.Sprint(dep.Config["admin_username"]),
		AdminPassword:   fmt.Sprint(dep.Config["admin_password"]),
		ManagementPort:  fmt.Sprint(dep.Config["management_port"]),
		DependencyName:  dep.Name,
		CredentialName:  cred.Name,
		DependencyType:  dep.Type,
		ControlledLabel: "external",
	}
}

func dependencySecretData(def DependencyDefinition, config model.JSONMap) map[string]string {
	out := map[string]string{}
	referencedSecretKeys := map[string]bool{}
	for _, output := range def.Spec.Outputs {
		if key := output.ValueFrom["secretKey"]; key != "" {
			referencedSecretKeys[key] = true
		}
	}
	props, _ := def.Spec.ConfigSchema["properties"].(map[string]any)
	for key, raw := range props {
		prop, _ := raw.(map[string]any)
		secret, _ := prop["secret"].(bool)
		if !secret && !referencedSecretKeys[key] {
			continue
		}
		if val, ok := config[key]; ok {
			out[key] = fmt.Sprint(val)
		}
	}
	return out
}

func resolveDependencyOutputs(def DependencyDefinition, serviceName string, config model.JSONMap, secretData map[string]string) model.JSONMap {
	plain := map[string]string{}
	secret := map[string]bool{}
	for name, output := range def.Spec.Outputs {
		switch {
		case output.Value != "":
			plain[name] = output.Value
		case output.ValueFrom["serviceHost"] != "":
			plain[name] = serviceName
		case output.ValueFrom["config"] != "":
			plain[name] = fmt.Sprint(config[output.ValueFrom["config"]])
		case output.ValueFrom["secretKey"] != "":
			key := output.ValueFrom["secretKey"]
			plain[name] = secretData[key]
			secret[name] = true
		}
	}
	for name, output := range def.Spec.Outputs {
		if output.Template != "" {
			plain[name] = renderTemplate(output.Template, plain)
		}
		if output.SecretTemplate != "" {
			plain[name] = renderTemplate(output.SecretTemplate, plain)
			secret[name] = true
		}
	}
	out := model.JSONMap{}
	for key, value := range plain {
		out[key] = map[string]any{"value": value, "secret": secret[key]}
	}
	return out
}

func applyExternalDependencyOutputs(outputs model.JSONMap, config model.JSONMap, external bool) model.JSONMap {
	if !external {
		return outputs
	}
	if host := strings.TrimSpace(fmt.Sprint(config["host"])); host != "" {
		outputs["host"] = map[string]any{"value": host, "secret": false}
	}
	if port := strings.TrimSpace(fmt.Sprint(config["port"])); port != "" && port != "<nil>" {
		outputs["port"] = map[string]any{"value": port, "secret": false}
	}
	return outputs
}

func dependencyRuntimeHost(serviceName, namespace string, external bool) string {
	serviceName = strings.TrimSpace(serviceName)
	if external || serviceName == "" {
		return serviceName
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" || strings.Contains(serviceName, ".") {
		return serviceName
	}
	return serviceName + "." + namespace + ".svc.cluster.local"
}

func flattenDependencyOutputs(outputs model.JSONMap) map[string]string {
	out := map[string]string{}
	for key, raw := range outputs {
		if m, ok := raw.(map[string]any); ok {
			out[key] = fmt.Sprint(m["value"])
			continue
		}
		out[key] = fmt.Sprint(raw)
	}
	return out
}

func renderTemplate(tpl string, values map[string]string) string {
	out := tpl
	for key, value := range values {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}

func randomToken(length int) string {
	if length <= 0 {
		length = 32
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("x", length)
	}
	return base64.RawURLEncoding.EncodeToString(buf)[:length]
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}

func appendEnvFromDependency(existing model.JSONMap, entry map[string]any) model.JSONMap {
	if existing == nil {
		existing = model.JSONMap{}
	}
	list, _ := existing["items"].([]any)
	list = append(list, entry)
	existing["items"] = list
	return existing
}

func maskJSONSecrets(in model.JSONMap) model.JSONMap {
	out := model.JSONMap{}
	for key, value := range in {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") {
			out[key] = "********"
			continue
		}
		out[key] = value
	}
	return out
}

func maskOutputSecrets(in model.JSONMap) model.JSONMap {
	out := model.JSONMap{}
	for key, raw := range in {
		if m, ok := raw.(map[string]any); ok {
			copy := map[string]any{}
			for k, v := range m {
				copy[k] = v
			}
			if secret, _ := copy["secret"].(bool); secret {
				copy["value"] = "********"
			}
			out[key] = copy
			continue
		}
		out[key] = raw
	}
	return out
}
