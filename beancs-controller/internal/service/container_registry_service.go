package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cryptoutil "github.com/zeturn/beancs-controller/internal/crypto"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"github.com/zeturn/beancs-controller/internal/registry"
	"gorm.io/gorm"
)

type ContainerRegistryService struct {
	db     *gorm.DB
	cipher cryptoutil.Cipher
	lister *registry.TagsLister
}

func NewContainerRegistryService(db *gorm.DB, cipher cryptoutil.Cipher) *ContainerRegistryService {
	return &ContainerRegistryService{db: db, cipher: cipher, lister: &registry.TagsLister{}}
}

func (s *ContainerRegistryService) Presets() []dto.ContainerRegistryPreset {
	return []dto.ContainerRegistryPreset{
		{Kind: "ghcr", Label: "GitHub Container Registry", ExampleHost: registry.DefaultExampleHost("ghcr"), Hint: "公共镜像可直接拉 tag；私有仓库需填写 GitHub PAT（用户名 + Token）"},
		{Kind: "dockerhub", Label: "Docker Hub", ExampleHost: registry.DefaultExampleHost("dockerhub"), Hint: "官方镜像仓库路径常为 library/nginx"},
		{Kind: "gitlab", Label: "GitLab Container Registry", ExampleHost: registry.DefaultExampleHost("gitlab"), Hint: "registry.gitlab.com/group/project"},
		{Kind: "quay", Label: "Quay.io", ExampleHost: registry.DefaultExampleHost("quay"), Hint: "组织/镜像 路径与 Web 上显示一致"},
		{Kind: "harbor", Label: "Harbor", ExampleHost: registry.DefaultExampleHost("harbor"), Hint: "自建 Harbor 地址，可勾选跳过 TLS 校验（仅内网）"},
		{Kind: "docker_registry", Label: "Docker Registry (Distribution)", ExampleHost: registry.DefaultExampleHost("docker_registry"), Hint: "官方 registry:2 等 OCI 兼容服务"},
		{Kind: "ecr", Label: "AWS ECR", ExampleHost: registry.DefaultExampleHost("ecr"), Hint: "需仓库级鉴权；请使用 ECR 登录后的用户名/密码或 IAM 方式（部分场景需额外集成）"},
		{Kind: "gar", Label: "Google Artifact Registry", ExampleHost: registry.DefaultExampleHost("gar"), Hint: "如 us-docker.pkg.dev/project/repo"},
		{Kind: "acr", Label: "Azure Container Registry", ExampleHost: registry.DefaultExampleHost("acr"), Hint: "myregistry.azurecr.io"},
		{Kind: "aliyun", Label: "阿里云 ACR", ExampleHost: registry.DefaultExampleHost("aliyun"), Hint: "企业版/个人版 registry 地址以控制台为准"},
		{Kind: "custom", Label: "自定义 OCI 兼容", ExampleHost: "https://registry.example.com", Hint: "任意支持 Docker Registry HTTP API v2 的地址"},
	}
}

func (s *ContainerRegistryService) ListRegistries(ctx context.Context, userID string) ([]dto.ContainerRegistryResponse, error) {
	var rows []model.ContainerRegistry
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]dto.ContainerRegistryResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toRegistryResponse(&r))
	}
	return out, nil
}

func (s *ContainerRegistryService) Create(ctx context.Context, userID string, req dto.CreateContainerRegistryRequest) (*dto.ContainerRegistryResponse, error) {
	if !registry.ValidateKind(req.Kind) {
		return nil, fmt.Errorf("invalid registry kind")
	}
	base, err := registry.ResolveAPIBase(req.Kind, req.Host)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("%s / %s", req.Kind, base)
	}
	var userEnc, passEnc []byte
	if strings.TrimSpace(req.Username) != "" {
		ue, err := s.cipher.EncryptString(strings.TrimSpace(req.Username))
		if err != nil {
			return nil, err
		}
		userEnc = ue
	}
	if req.Password != "" {
		pe, err := s.cipher.EncryptString(req.Password)
		if err != nil {
			return nil, err
		}
		passEnc = pe
	}
	row := model.ContainerRegistry{
		UserID:      userID,
		Kind:        strings.ToLower(strings.TrimSpace(req.Kind)),
		Name:        name,
		APIBase:     base,
		UsernameEnc: userEnc,
		PasswordEnc: passEnc,
		InsecureTLS: req.InsecureTLS,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	resp := s.toRegistryResponse(&row)
	return &resp, nil
}

func (s *ContainerRegistryService) Update(ctx context.Context, userID string, id uint, req dto.UpdateContainerRegistryRequest) (*dto.ContainerRegistryResponse, error) {
	var row model.ContainerRegistry
	if err := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).First(&row).Error; err != nil {
		return nil, fmt.Errorf("registry not found")
	}
	if req.Name != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if req.Host != "" {
		base, err := registry.ResolveAPIBase(row.Kind, req.Host)
		if err != nil {
			return nil, err
		}
		row.APIBase = base
	}
	if req.InsecureTLS != nil {
		row.InsecureTLS = *req.InsecureTLS
	}
	if req.Username != "" || req.Password != "" {
		if strings.TrimSpace(req.Username) != "" {
			ue, err := s.cipher.EncryptString(strings.TrimSpace(req.Username))
			if err != nil {
				return nil, err
			}
			row.UsernameEnc = ue
		}
		if req.Password != "" {
			pe, err := s.cipher.EncryptString(req.Password)
			if err != nil {
				return nil, err
			}
			row.PasswordEnc = pe
		}
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	resp := s.toRegistryResponse(&row)
	return &resp, nil
}

func (s *ContainerRegistryService) Delete(ctx context.Context, userID string, id uint) error {
	res := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.ContainerRegistry{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("registry not found")
	}
	_ = s.db.WithContext(ctx).Where("user_id = ? AND registry_id = ?", userID, id).Delete(&model.ContainerImage{})
	return nil
}

func (s *ContainerRegistryService) ListImages(ctx context.Context, userID string) ([]dto.ContainerImageResponse, error) {
	var rows []model.ContainerImage
	if err := s.db.WithContext(ctx).Preload("Registry").Where("user_id = ?", userID).Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]dto.ContainerImageResponse, 0, len(rows))
	for _, im := range rows {
		out = append(out, s.toImageResponse(&im))
	}
	return out, nil
}

func (s *ContainerRegistryService) CreateImage(ctx context.Context, userID string, req dto.CreateContainerImageRequest) (*dto.ContainerImageResponse, error) {
	repo := normalizeRepo(req.Repository)
	if repo == "" {
		return nil, fmt.Errorf("repository is required")
	}
	var reg model.ContainerRegistry
	if err := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, req.RegistryID).First(&reg).Error; err != nil {
		return nil, fmt.Errorf("registry not found")
	}
	row := model.ContainerImage{
		UserID:     userID,
		RegistryID: reg.ID,
		Repository: repo,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	if err := s.refreshImageLocked(ctx, &row, &reg); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Preload("Registry").First(&row, row.ID).Error; err != nil {
		return nil, err
	}
	resp := s.toImageResponse(&row)
	return &resp, nil
}

func (s *ContainerRegistryService) DeleteImage(ctx context.Context, userID string, id uint) error {
	res := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.ContainerImage{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("image not found")
	}
	return nil
}

func (s *ContainerRegistryService) RefreshImage(ctx context.Context, userID string, id uint) (*dto.ContainerImageResponse, error) {
	var row model.ContainerImage
	if err := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).First(&row).Error; err != nil {
		return nil, fmt.Errorf("image not found")
	}
	var reg model.ContainerRegistry
	if err := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, row.RegistryID).First(&reg).Error; err != nil {
		return nil, fmt.Errorf("registry not found")
	}
	if err := s.refreshImageLocked(ctx, &row, &reg); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Preload("Registry").First(&row, row.ID).Error; err != nil {
		return nil, err
	}
	resp := s.toImageResponse(&row)
	return &resp, nil
}

func (s *ContainerRegistryService) refreshImageLocked(ctx context.Context, row *model.ContainerImage, reg *model.ContainerRegistry) error {
	tags, err := s.fetchTags(ctx, reg, row.Repository)
	if err != nil {
		return err
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	row.TagsJSON = string(b)
	row.RefreshedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(row).Updates(map[string]interface{}{
		"tags_json":    row.TagsJSON,
		"refreshed_at": row.RefreshedAt,
	}).Error
}

func (s *ContainerRegistryService) fetchTags(ctx context.Context, reg *model.ContainerRegistry, repository string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	var u, p *string
	if len(reg.UsernameEnc) > 0 {
		plain, err := s.cipher.DecryptString(reg.UsernameEnc)
		if err != nil {
			return nil, err
		}
		u = &plain
	}
	if len(reg.PasswordEnc) > 0 {
		plain, err := s.cipher.DecryptString(reg.PasswordEnc)
		if err != nil {
			return nil, err
		}
		p = &plain
	}
	return s.lister.ListTags(ctx, reg.APIBase, repository, u, p, reg.InsecureTLS)
}

func (s *ContainerRegistryService) ListTagsLive(ctx context.Context, userID string, registryID uint, repository string) (*dto.ListTagsResponse, error) {
	repo := normalizeRepo(repository)
	if repo == "" {
		return nil, fmt.Errorf("repository is required")
	}
	var reg model.ContainerRegistry
	if err := s.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, registryID).First(&reg).Error; err != nil {
		return nil, fmt.Errorf("registry not found")
	}
	tags, err := s.fetchTags(ctx, &reg, repo)
	if err != nil {
		return nil, err
	}
	return &dto.ListTagsResponse{
		Repository:  repo,
		Tags:        tags,
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
		Cached:      false,
	}, nil
}

func (s *ContainerRegistryService) toRegistryResponse(r *model.ContainerRegistry) dto.ContainerRegistryResponse {
	return dto.ContainerRegistryResponse{
		ID:          r.ID,
		Kind:        r.Kind,
		Name:        r.Name,
		APIBase:     r.APIBase,
		InsecureTLS: r.InsecureTLS,
		HasAuth:     len(r.UsernameEnc) > 0 || len(r.PasswordEnc) > 0,
		CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *ContainerRegistryService) toImageResponse(im *model.ContainerImage) dto.ContainerImageResponse {
	var tags []string
	if im.TagsJSON != "" {
		_ = json.Unmarshal([]byte(im.TagsJSON), &tags)
	}
	resp := dto.ContainerImageResponse{
		ID:          im.ID,
		RegistryID:  im.RegistryID,
		Repository:  im.Repository,
		Tags:        tags,
		RefreshedAt: im.RefreshedAt.UTC().Format(time.RFC3339),
	}
	if im.Registry != nil {
		r := s.toRegistryResponse(im.Registry)
		resp.Registry = &r
	}
	return resp
}

func normalizeRepo(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	return s
}
