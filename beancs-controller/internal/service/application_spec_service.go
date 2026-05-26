package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	appspec "github.com/zeturn/beancs-controller/internal/application/spec"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

var ErrApplicationSpecNotFound = errors.New("application spec config not found")

type RepoConfigFile struct {
	Path    string `json:"path"`
	Content []byte `json:"-"`
	Hash    string `json:"hash"`
}

type ApplicationSpecService struct {
	db           *gorm.DB
	credentials  *CredentialService
	dependencies *DependencyService
	applications *ApplicationService
}

func NewApplicationSpecService(db *gorm.DB, credentials *CredentialService, dependencies *DependencyService, applications *ApplicationService) *ApplicationSpecService {
	return &ApplicationSpecService{db: db, credentials: credentials, dependencies: dependencies, applications: applications}
}

func (s *ApplicationSpecService) ValidateFromRepo(ctx context.Context, userID string, req dto.ApplicationSpecRepoRequest) (*dto.ApplicationSpecResponse, error) {
	file, doc, validation, _, err := s.loadValidatePlan(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &dto.ApplicationSpecResponse{ConfigPath: file.Path, Found: true, Document: doc, Validation: validation}, nil
}

func (s *ApplicationSpecService) PlanFromRepo(ctx context.Context, userID string, req dto.ApplicationSpecRepoRequest) (*dto.ApplicationSpecResponse, error) {
	file, doc, validation, plan, err := s.loadValidatePlan(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &dto.ApplicationSpecResponse{ConfigPath: file.Path, Found: true, Document: doc, Validation: validation, Plan: &plan}, nil
}

func (s *ApplicationSpecService) ApplyFromRepo(ctx context.Context, userID, tenantID, tenantCode string, req dto.ApplicationSpecRepoRequest) (*dto.ApplicationSpecResponse, *dto.ApplicationResponse, error) {
	file, doc, validation, plan, err := s.loadValidatePlan(ctx, userID, req)
	if err != nil {
		return nil, nil, err
	}
	resp := &dto.ApplicationSpecResponse{ConfigPath: file.Path, Found: true, Document: doc, Validation: validation, Plan: &plan}
	if !validation.Valid {
		return resp, nil, fmt.Errorf("application spec is invalid")
	}
	if req.DryRun {
		return resp, nil, nil
	}
	monorepo := s.specToMonorepoRequest(ctx, userID, doc, req)
	app, err := s.applications.CreateMonorepo(ctx, userID, tenantID, tenantCode, monorepo)
	if app != nil {
		updates := map[string]any{"spec_path": file.Path, "spec_hash": file.Hash, "spec_raw": specRawMap(doc)}
		_ = s.db.WithContext(ctx).Model(&model.Application{}).Where("id = ?", app.ID).Updates(updates).Error
	}
	return resp, app, err
}

func (s *ApplicationSpecService) loadValidatePlan(ctx context.Context, userID string, req dto.ApplicationSpecRepoRequest) (*RepoConfigFile, *appspec.ApplicationSpecDocument, appspec.ValidationResult, appspec.ApplicationPlan, error) {
	req.GitHubRepo = strings.TrimSpace(req.GitHubRepo)
	if req.GitHubBranch == "" {
		req.GitHubBranch = "main"
	}
	token, owner, repo, err := s.githubAccess(ctx, userID, req)
	if err != nil {
		return nil, nil, appspec.ValidationResult{}, appspec.ApplicationPlan{}, err
	}
	file, err := s.FindConfig(ctx, token, owner, repo, req.GitHubBranch, req.ConfigPath)
	if err != nil {
		return nil, nil, appspec.ValidationResult{}, appspec.ApplicationPlan{}, err
	}
	doc, err := appspec.Parse(file.Content)
	if err != nil {
		return file, nil, appspec.ValidationResult{}, appspec.ApplicationPlan{}, err
	}
	files, err := githubRepositoryTreeFiles(ctx, token, owner, repo, req.GitHubBranch)
	if err != nil {
		return file, nil, appspec.ValidationResult{}, appspec.ApplicationPlan{}, err
	}
	repoFiles := map[string]bool{}
	for _, f := range files {
		repoFiles[f] = true
	}
	options := appspec.ValidateOptions{RepoFiles: repoFiles, Dependencies: s.dependencyDefinitions()}
	validation := appspec.Validate(doc, options)
	plan := appspec.Plan(doc, validation, options)
	return file, doc, validation, plan, nil
}

func (s *ApplicationSpecService) FindConfig(ctx context.Context, token, owner, repo, branch, explicitPath string) (*RepoConfigFile, error) {
	candidates := []string{}
	if strings.TrimSpace(explicitPath) != "" {
		candidates = append(candidates, strings.TrimSpace(explicitPath))
	}
	candidates = append(candidates, ".beancs/app.yaml", ".beancs/application.yaml", "beancs.yaml")
	for _, candidate := range candidates {
		body, ok, err := githubTextFile(ctx, token, owner, repo, candidate, branch)
		if err != nil {
			return nil, err
		}
		if ok {
			sum := sha256.Sum256(body)
			return &RepoConfigFile{Path: candidate, Content: body, Hash: hex.EncodeToString(sum[:])}, nil
		}
	}
	return nil, ErrApplicationSpecNotFound
}

func (s *ApplicationSpecService) githubAccess(ctx context.Context, userID string, req dto.ApplicationSpecRepoRequest) (string, string, string, error) {
	if err := s.credentials.RequireAccess(userID, model.CredentialTypeGitHub, req.GitHubCredentialID, false); err != nil {
		return "", "", "", err
	}
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, req.GitHubCredentialID).Error; err != nil {
		return "", "", "", err
	}
	token, err := s.credentials.GitHubToken(ctx, cred)
	if err != nil {
		return "", "", "", err
	}
	owner, repo, ok := splitRepo(req.GitHubRepo)
	if !ok {
		return "", "", "", fmt.Errorf("github_repo must be in owner/repo format")
	}
	return token, owner, repo, nil
}

func (s *ApplicationSpecService) dependencyDefinitions() map[string]appspec.DependencyDefinitionView {
	out := map[string]appspec.DependencyDefinitionView{}
	if s.dependencies == nil || s.dependencies.Registry() == nil {
		return out
	}
	for _, def := range s.dependencies.Registry().List() {
		outputs := map[string]bool{}
		for name := range def.Spec.Outputs {
			outputs[name] = true
		}
		presets := map[string][]string{}
		for presetName, preset := range def.Spec.EnvPresets {
			for envName := range preset.Env {
				presets[presetName] = append(presets[presetName], envName)
			}
		}
		out[def.Metadata.Name] = appspec.DependencyDefinitionView{Type: def.Spec.Type, Outputs: outputs, EnvPresets: presets}
		out[def.Spec.Type] = appspec.DependencyDefinitionView{Type: def.Spec.Type, Outputs: outputs, EnvPresets: presets}
	}
	return out
}

func (s *ApplicationSpecService) specToMonorepoRequest(ctx context.Context, userID string, doc *appspec.ApplicationSpecDocument, applyReq dto.ApplicationSpecRepoRequest) dto.CreateMonorepoApplicationRequest {
	autoDeploy := doc.Spec.AutoDeploy != nil && doc.Spec.AutoDeploy.Enabled && doc.Spec.AutoDeploy.Mode != "disabled"
	req := dto.CreateMonorepoApplicationRequest{
		Name:                   doc.Metadata.Name,
		DisplayName:            doc.Metadata.DisplayName,
		Description:            doc.Metadata.Description,
		GitHubCredentialID:     applyReq.GitHubCredentialID,
		GitHubRepo:             doc.Spec.Repo.Name,
		GitHubBranch:           doc.Spec.Repo.Branch,
		AutoDeploy:             &autoDeploy,
		BasaltPassInstanceID:   applyReq.BasaltPassInstanceID,
		CloudflareCredentialID: applyReq.CloudflareCredentialID,
		CloudflareZoneID:       applyReq.CloudflareZoneID,
	}
	if doc.Spec.Namespace != nil && doc.Spec.Namespace.Strategy == "shared" {
		req.Namespace = doc.Spec.Namespace.Name
	}
	publicDomain := s.cloudflareDomainForSpec(ctx, userID, applyReq)
	for _, dep := range doc.Spec.Dependencies {
		req.Dependencies = append(req.Dependencies, dto.CreateManagedDependencyRequest{
			Name:         dep.Name,
			Type:         dep.Type,
			DeployMethod: dep.DeployMethod,
			Version:      dep.Version,
			Config:       model.JSONMap(dep.Config),
		})
	}
	componentSecrets := map[string]map[string]string{}
	for _, component := range doc.Spec.Components {
		componentReq := specComponentToRequest(component)
		applyComponentDomains(&componentReq, req.Namespace, publicDomain, applyReq.ComponentDomains)
		secrets := resolveComponentSecrets(component, componentSecrets)
		if len(secrets) > 0 {
			if componentReq.Env == nil {
				componentReq.Env = map[string]string{}
			}
			for key, value := range secrets {
				componentReq.Env[key] = value
			}
			componentSecrets[component.Name] = secrets
		}
		req.Components = append(req.Components, componentReq)
	}
	return req
}

func (s *ApplicationSpecService) cloudflareDomainForSpec(ctx context.Context, userID string, applyReq dto.ApplicationSpecRepoRequest) string {
	if s == nil || s.credentials == nil || applyReq.CloudflareCredentialID == nil {
		return ""
	}
	if err := s.credentials.RequireAccess(userID, model.CredentialTypeCloudflare, *applyReq.CloudflareCredentialID, false); err != nil {
		return ""
	}
	cred, err := s.credentials.CloudflareCredentialForDomain(ctx, *applyReq.CloudflareCredentialID, applyReq.CloudflareZoneID, "")
	if err != nil {
		return ""
	}
	return strings.Trim(strings.ToLower(cred.Domain), ".")
}

func applyComponentDomains(component *dto.MonorepoComponentRequest, namespace, publicDomain string, overrides map[string]string) {
	override := componentDomainOverride(component.Name, component.ProjectName, overrides)
	for i := range component.Ports {
		port := &component.Ports[i]
		if override != "" && shouldApplyDomainOverride(port.Exposure, override) {
			port.Domain = override
		}
		if port.Exposure == model.ExposurePublic && port.Domain == "" && publicDomain != "" {
			port.Domain = component.ProjectName + "." + publicDomain
		}
		if port.Exposure == model.ExposurePrivate && port.Domain == "" {
			port.Domain = component.ProjectName + "." + coalesce(namespace, "proj-"+component.ProjectName) + ".ts.net"
		}
		if port.Protocol != "" && port.Protocol != "http" {
			port.Protocol = ""
		}
	}
}

func shouldApplyDomainOverride(exposure, override string) bool {
	if override == "" {
		return false
	}
	if exposure == model.ExposurePublic {
		return !strings.HasSuffix(override, ".ts.net")
	}
	if exposure == model.ExposurePrivate {
		return strings.HasSuffix(override, ".ts.net")
	}
	return false
}

func componentDomainOverride(componentName, projectName string, overrides map[string]string) string {
	for _, key := range []string{projectName, componentName} {
		if value := strings.Trim(strings.ToLower(overrides[key]), "."); value != "" {
			return value
		}
	}
	return ""
}

func specComponentToRequest(component appspec.ComponentSpec) dto.MonorepoComponentRequest {
	req := dto.MonorepoComponentRequest{
		Name:        component.Name,
		Kind:        component.Kind,
		ProjectName: component.ProjectName,
		Replicas:    1,
		Env:         stringifyEnv(component.Env),
		DependsOn:   component.DependsOn,
		WatchPaths:  component.WatchPaths,
		HealthCheck: model.JSONMap(toJSONMap(component.HealthCheck)),
		Volumes:     model.JSONMap{"items": component.Volumes},
	}
	if component.Replicas != nil {
		req.Replicas = *component.Replicas
	}
	if component.Build != nil {
		req.BuildContext = component.Build.Context
		req.DockerfilePath = component.Build.Dockerfile
		req.BuildArgs = model.JSONMap{}
		for k, v := range component.Build.Args {
			req.BuildArgs[k] = v
		}
	}
	for _, port := range component.Ports {
		exposure := port.Exposure
		if exposure == "internal" {
			exposure = model.ExposureInternalOnly
		}
		req.Ports = append(req.Ports, model.ProjectPort{Name: port.Name, Port: port.Port, Protocol: port.Protocol, Exposure: exposure})
	}
	if len(req.Ports) == 0 {
		req.ExposureMode = model.ExposureInternalOnly
	}
	for _, ref := range component.EnvFromDependencies {
		req.EnvFromDependencies = append(req.EnvFromDependencies, dto.EnvFromDependencyRequest{Dependency: ref.Dependency, Preset: ref.Preset, Mappings: envMappingsToAny(ref.Mappings)})
	}
	return req
}

func resolveComponentSecrets(component appspec.ComponentSpec, existing map[string]map[string]string) map[string]string {
	out := map[string]string{}
	for _, secret := range component.Secrets {
		name := strings.TrimSpace(secret.Name)
		if name == "" {
			continue
		}
		if secret.Generate != nil {
			length := secret.Generate.Length
			if length <= 0 {
				length = 32
			}
			out[name] = randomToken(length)
			continue
		}
		if source := strings.TrimSpace(secret.FromComponent); source != "" {
			if value := existing[source][name]; value != "" {
				out[name] = value
			}
		}
	}
	return out
}

func githubTextFile(ctx context.Context, token, owner, repo, filePath, ref string) ([]byte, bool, error) {
	body, ok, err := githubContentRead(ctx, token, owner, repo, filePath, ref)
	if err != nil || !ok {
		return nil, ok, err
	}
	var payload struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("GitHub returned invalid JSON")
	}
	if payload.Type != "file" {
		return nil, false, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(payload.Content, "\n", ""))
	if err != nil {
		return nil, false, err
	}
	return decoded, true, nil
}

func stringifyEnv(in map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		switch value := v.(type) {
		case string:
			out[k] = value
		default:
			raw, _ := json.Marshal(value)
			out[k] = string(raw)
		}
	}
	return out
}

func envMappingsToAny(in map[string]appspec.EnvMapping) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = map[string]any{"output": v.Output, "secret": v.Secret}
	}
	return out
}

func specRawMap(doc *appspec.ApplicationSpecDocument) model.JSONMap {
	raw, _ := json.Marshal(doc)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return model.JSONMap(out)
}

func toJSONMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	raw, _ := json.Marshal(value)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

func ApplicationSpecHTTPStatus(err error) int {
	if errors.Is(err, ErrApplicationSpecNotFound) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}
