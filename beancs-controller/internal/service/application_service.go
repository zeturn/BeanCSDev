package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type ApplicationService struct {
	db           *gorm.DB
	projects     *ProjectService
	dependencies *DependencyService
}

func NewApplicationService(db *gorm.DB, projects *ProjectService, dependencies *DependencyService) *ApplicationService {
	return &ApplicationService{db: db, projects: projects, dependencies: dependencies}
}

func (s *ApplicationService) CreateMonorepo(ctx context.Context, userID, tenantID, tenantCode string, req dto.CreateMonorepoApplicationRequest) (*dto.ApplicationResponse, error) {
	normalizeMonorepoApplicationRequest(&req)
	if err := s.ensureApplicationNameAvailable(ctx, req.Name); err != nil {
		return nil, err
	}
	app := &model.Application{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Type:         model.ApplicationTypeMonorepo,
		GitHubRepo:   req.GitHubRepo,
		GitHubBranch: req.GitHubBranch,
		Namespace:    req.Namespace,
		OwnerID:      userID,
		TenantID:     tenantID,
		TenantCode:   tenantCode,
		Status:       model.ApplicationStatusCreating,
	}
	if err := s.db.WithContext(ctx).Create(app).Error; err != nil {
		return nil, err
	}
	var created []model.Project
	var createdDeps []model.ManagedDependency
	depsByName := map[string]model.ManagedDependency{}
	var failures []string
	for _, dependency := range req.Dependencies {
		dep, err := s.dependencies.Create(ctx, userID, app.ID, dependency)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", dependency.Name, err.Error()))
			continue
		}
		createdDeps = append(createdDeps, *dep)
		depsByName[dep.Name] = *dep
		depsByName[dependency.Name] = *dep
	}
	for _, component := range req.Components {
		component.Env = s.componentEnvWithDependencies(component, depsByName)
		projectReq := monorepoProjectRequest(req, component)
		project, err := s.projects.CreateProject(ctx, userID, tenantID, tenantCode, projectReq)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", component.ProjectName, err.Error()))
			continue
		}
		updates := map[string]any{
			"application_id":        app.ID,
			"component_name":        component.Name,
			"component_path":        component.ComponentPath,
			"build_context":         projectReq.BuildContext,
			"build_args":            projectReq.BuildArgs,
			"health_check":          projectReq.HealthCheck,
			"volumes":               projectReq.Volumes,
			"watch_paths":           model.StringList(projectReq.WatchPaths),
			"depends_on":            model.StringList(component.DependsOn),
			"env_from_dependencies": projectReq.EnvFromDependencies,
		}
		if err := s.db.WithContext(ctx).Model(project).Updates(updates).Error; err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", component.ProjectName, err.Error()))
			continue
		}
		_ = s.db.WithContext(ctx).First(project, project.ID).Error
		projectID := project.ID
		appComponent := model.ApplicationComponent{
			ApplicationID: app.ID,
			Name:          component.Name,
			Kind:          component.Kind,
			ProjectID:     &projectID,
			DependsOn:     component.DependsOn,
			Status:        project.Status,
		}
		if appComponent.Kind == "" {
			appComponent.Kind = model.ApplicationComponentKindService
		}
		_ = s.db.WithContext(ctx).Create(&appComponent).Error
		created = append(created, *project)
	}
	status := model.ApplicationStatusActive
	if len(failures) > 0 {
		status = model.ApplicationStatusPartialFailed
	}
	if len(created) == 0 && len(failures) > 0 {
		_ = s.db.WithContext(ctx).Model(app).Update("status", status).Error
		return &dto.ApplicationResponse{Application: *app, Projects: created, Dependencies: s.dependencies.MaskList(createdDeps)}, fmt.Errorf("all components failed: %s", strings.Join(failures, "; "))
	}
	if err := s.db.WithContext(ctx).Model(app).Update("status", status).Error; err != nil {
		return nil, err
	}
	app.Status = status
	if len(failures) > 0 {
		return &dto.ApplicationResponse{Application: *app, Projects: created, Dependencies: s.dependencies.MaskList(createdDeps)}, fmt.Errorf("some components failed: %s", strings.Join(failures, "; "))
	}
	return &dto.ApplicationResponse{Application: *app, Projects: created, Dependencies: s.dependencies.MaskList(createdDeps)}, nil
}

func (s *ApplicationService) List(ctx context.Context, userID string) ([]dto.ApplicationResponse, error) {
	var apps []model.Application
	if err := s.db.WithContext(ctx).Where("owner_id = ?", userID).Order("created_at desc").Find(&apps).Error; err != nil {
		return nil, err
	}
	out := make([]dto.ApplicationResponse, 0, len(apps))
	for _, app := range apps {
		var projects []model.Project
		_ = s.db.WithContext(ctx).Where("application_id = ?", app.ID).Order("name asc").Find(&projects).Error
		var deps []model.ManagedDependency
		ids := s.dependencies.dependencyIDsForApplication(ctx, app.ID)
		_ = s.db.WithContext(ctx).Where("id IN ?", ids).Order("name asc").Find(&deps).Error
		var components []model.ApplicationComponent
		_ = s.db.WithContext(ctx).Where("application_id = ?", app.ID).Order("kind asc, name asc").Find(&components).Error
		out = append(out, dto.ApplicationResponse{Application: app, Projects: projects, Dependencies: s.dependencies.MaskList(deps), Components: components})
	}
	return out, nil
}

func (s *ApplicationService) Get(ctx context.Context, userID string, id uint) (*dto.ApplicationResponse, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, userID).First(&app).Error; err != nil {
		return nil, err
	}
	var projects []model.Project
	if err := s.db.WithContext(ctx).Where("application_id = ?", app.ID).Order("name asc").Find(&projects).Error; err != nil {
		return nil, err
	}
	var deps []model.ManagedDependency
	ids := s.dependencies.dependencyIDsForApplication(ctx, app.ID)
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Order("name asc").Find(&deps).Error; err != nil {
		return nil, err
	}
	var components []model.ApplicationComponent
	if err := s.db.WithContext(ctx).Where("application_id = ?", app.ID).Order("kind asc, name asc").Find(&components).Error; err != nil {
		return nil, err
	}
	return &dto.ApplicationResponse{Application: app, Projects: projects, Dependencies: s.dependencies.MaskList(deps), Components: components}, nil
}

func (s *ApplicationService) Delete(ctx context.Context, userID string, id uint) error {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, userID).First(&app).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(&app).Update("status", model.ApplicationStatusDeleting).Error; err != nil {
		return err
	}
	var projects []model.Project
	if err := s.db.WithContext(ctx).Where("application_id = ?", app.ID).Order("name asc").Find(&projects).Error; err != nil {
		return err
	}
	var failures []string
	for i := range projects {
		project := projects[i]
		_ = s.db.WithContext(ctx).Model(&project).Update("auto_deploy", false).Error
		if err := s.projects.DeleteProject(ctx, &project); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", project.Name, err.Error()))
		}
	}
	if len(failures) > 0 {
		_ = s.db.WithContext(ctx).Model(&app).Update("status", model.ApplicationStatusPartialDeleted).Error
		return fmt.Errorf("application partially deleted: %s", strings.Join(failures, "; "))
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.ApplicationComponent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.ManagedDependency{}).Error; err != nil {
			return err
		}
		return tx.Delete(&app).Error
	})
}

func normalizeMonorepoApplicationRequest(req *dto.CreateMonorepoApplicationRequest) {
	req.Name = harborName(req.Name)
	req.GitHubRepo = strings.TrimSpace(req.GitHubRepo)
	if req.GitHubBranch == "" {
		req.GitHubBranch = "main"
	}
	if req.ResourcePreset == "" {
		req.ResourcePreset = "small"
	}
	for i := range req.Components {
		component := &req.Components[i]
		component.Name = harborName(component.Name)
		component.ProjectName = harborName(component.ProjectName)
		component.DockerfilePath = strings.TrimSpace(component.DockerfilePath)
		component.BuildContext = strings.TrimSpace(component.BuildContext)
		if component.BuildContext == "" {
			component.BuildContext = "."
		}
		component.ComponentPath = strings.Trim(strings.TrimSpace(component.ComponentPath), "/")
		if component.ComponentPath == "" {
			component.ComponentPath = strings.TrimSuffix(component.DockerfilePath, "/Dockerfile")
		}
		if component.ResourcePreset == "" {
			component.ResourcePreset = req.ResourcePreset
		}
		if component.Replicas == 0 {
			component.Replicas = 1
		}
		if component.Env == nil {
			component.Env = map[string]string{}
		}
		component.DependsOn = normalizeStringList(component.DependsOn)
		if len(component.Ports) == 0 {
			component.ExposureMode = model.ExposureInternalOnly
		}
	}
}

func monorepoProjectRequest(app dto.CreateMonorepoApplicationRequest, component dto.MonorepoComponentRequest) dto.CreateProjectRequest {
	exposureMode := component.ExposureMode
	if exposureMode == "" {
		exposureMode = aggregateExposureMode(component.Ports)
	}
	namespace := component.Namespace
	if namespace == "" {
		namespace = app.Namespace
	}
	return dto.CreateProjectRequest{
		Name:                   component.ProjectName,
		DisplayName:            coalesce(component.DisplayName, component.ProjectName),
		Description:            coalesce(component.Description, app.Description),
		TeamID:                 app.TeamID,
		BuildSource:            model.BuildSourceGitHub,
		GitHubCredentialID:     app.GitHubCredentialID,
		GitHubRepo:             app.GitHubRepo,
		GitHubBranch:           app.GitHubBranch,
		DockerfilePath:         component.DockerfilePath,
		BuildContext:           component.BuildContext,
		BuildArgs:              component.BuildArgs,
		AutoDeploy:             app.AutoDeploy,
		Namespace:              namespace,
		BasaltPassInstanceID:   app.BasaltPassInstanceID,
		CloudflareCredentialID: app.CloudflareCredentialID,
		CloudflareZoneID:       app.CloudflareZoneID,
		ExposureMode:           exposureMode,
		Subdomain:              component.Subdomain,
		ResourcePreset:         component.ResourcePreset,
		Ports:                  component.Ports,
		Replicas:               component.Replicas,
		Env:                    component.Env,
		DependsOn:              component.DependsOn,
		EnvFromDependencies:    envFromDependencyJSON(component.EnvFromDependencies),
		HealthCheck:            component.HealthCheck,
		Volumes:                component.Volumes,
		WatchPaths:             component.WatchPaths,
	}
}

func (s *ApplicationService) componentEnvWithDependencies(component dto.MonorepoComponentRequest, depsByName map[string]model.ManagedDependency) map[string]string {
	env := map[string]string{}
	for k, v := range component.Env {
		env[k] = v
	}
	for _, ref := range component.EnvFromDependencies {
		dep, ok := depsByName[ref.Dependency]
		if !ok {
			continue
		}
		depEnv, err := s.dependencies.EnvForDependencyCredential(context.Background(), dep, ref.CredentialID, ref.Credential, ref.Preset, ref.Mappings)
		if err != nil {
			continue
		}
		for k, v := range depEnv {
			env[k] = v
		}
	}
	return env
}

func envFromDependencyJSON(items []dto.EnvFromDependencyRequest) model.JSONMap {
	out := make([]any, 0, len(items))
	for _, item := range items {
		entry := map[string]any{"dependency": item.Dependency}
		if item.DependencyID != 0 {
			entry["dependency_id"] = item.DependencyID
		}
		if item.Credential != "" {
			entry["credential"] = item.Credential
		}
		if item.CredentialID != 0 {
			entry["credential_id"] = item.CredentialID
		}
		if item.Preset != "" {
			entry["preset"] = item.Preset
		}
		if len(item.Mappings) > 0 {
			entry["mappings"] = item.Mappings
		}
		out = append(out, entry)
	}
	return model.JSONMap{"items": out}
}

func normalizeStringList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func (s *ApplicationService) ensureApplicationNameAvailable(ctx context.Context, name string) error {
	var existing model.Application
	err := s.db.WithContext(ctx).Select("id").Where("name = ?", name).First(&existing).Error
	if err == nil {
		return fmt.Errorf("application name %q already exists; choose a different application name", name)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}
