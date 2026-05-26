package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type DependencyService struct {
	db       *gorm.DB
	registry *DependencyDefinitionRegistry
}

func NewDependencyService(db *gorm.DB, registry *DependencyDefinitionRegistry) *DependencyService {
	return &DependencyService{db: db, registry: registry}
}

func (s *DependencyService) Registry() *DependencyDefinitionRegistry {
	return s.registry
}

func (s *DependencyService) Create(ctx context.Context, userID string, applicationID uint, req dto.CreateManagedDependencyRequest) (*model.ManagedDependency, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", applicationID, userID).First(&app).Error; err != nil {
		return nil, err
	}
	def, ok := s.registry.Get(req.Type)
	if !ok {
		return nil, fmt.Errorf("unknown dependency type %q", req.Type)
	}
	if req.DeployMethod == "" {
		req.DeployMethod = def.Spec.DefaultDeployMethod
	}
	if !containsString(def.Spec.SupportedDeployMethods, req.DeployMethod) && req.DeployMethod != def.Spec.DefaultDeployMethod {
		return nil, fmt.Errorf("deploy_method %q is not supported by %s", req.DeployMethod, req.Type)
	}
	config := applyDependencyConfigDefaults(def, req.Config)
	secretData := dependencySecretData(def, config)
	outputs := resolveDependencyOutputs(def, req.Name, config, secretData)
	secretName := fmt.Sprintf("%s-%s-credentials", app.Name, req.Name)
	dep := &model.ManagedDependency{
		ApplicationID:     app.ID,
		Name:              req.Name,
		Type:              req.Type,
		Version:           req.Version,
		DeployMethod:      req.DeployMethod,
		Namespace:         coalesce(app.Namespace, coalesce(app.Name, req.Name)),
		ServiceName:       req.Name,
		SecretName:        secretName,
		DefinitionName:    def.Metadata.Name,
		DefinitionVersion: "v1",
		Config:            config,
		Outputs:           outputs,
		Status:            model.DependencyStatusReady,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(dep).Error; err != nil {
			return err
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
	return dep, nil
}

func (s *DependencyService) List(ctx context.Context, userID string, applicationID uint) ([]model.ManagedDependency, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", applicationID, userID).First(&app).Error; err != nil {
		return nil, err
	}
	var deps []model.ManagedDependency
	err := s.db.WithContext(ctx).Where("application_id = ?", app.ID).Order("name asc").Find(&deps).Error
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
	var dep model.ManagedDependency
	if err := s.db.WithContext(ctx).Where("application_id = ? AND name = ?", *project.ApplicationID, req.Dependency).First(&dep).Error; err != nil {
		return nil, err
	}
	project.DependsOn = appendUnique(project.DependsOn, dep.Name)
	entry := map[string]any{"dependency": dep.Name}
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
	def, ok := s.registry.Get(dep.DefinitionName)
	if !ok {
		return nil, fmt.Errorf("dependency definition %q not found", dep.DefinitionName)
	}
	outputs := flattenDependencyOutputs(dep.Outputs)
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
	maskedOutputs := model.JSONMap{}
	for key, raw := range dep.Outputs {
		if m, ok := raw.(map[string]any); ok {
			copy := map[string]any{}
			for k, v := range m {
				copy[k] = v
			}
			if secret, _ := copy["secret"].(bool); secret {
				copy["value"] = "********"
			}
			maskedOutputs[key] = copy
			continue
		}
		maskedOutputs[key] = raw
	}
	dep.Config = maskedConfig
	dep.Outputs = maskedOutputs
	return dep
}

func (s *DependencyService) MaskList(deps []model.ManagedDependency) []model.ManagedDependency {
	out := make([]model.ManagedDependency, 0, len(deps))
	for _, dep := range deps {
		out = append(out, s.Mask(dep))
	}
	return out
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
