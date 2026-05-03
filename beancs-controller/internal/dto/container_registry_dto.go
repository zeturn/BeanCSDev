package dto

// ContainerRegistryPreset 前端可选的预置镜像源类型
type ContainerRegistryPreset struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	ExampleHost string `json:"example_host"`
	Hint        string `json:"hint"`
}

type CreateContainerRegistryRequest struct {
	Kind        string `json:"kind" validate:"required"`
	Name        string `json:"name"`
	Host        string `json:"host" validate:"required"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	InsecureTLS bool   `json:"insecure_tls"`
}

type UpdateContainerRegistryRequest struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	InsecureTLS *bool  `json:"insecure_tls"`
}

type ContainerRegistryResponse struct {
	ID          uint   `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	APIBase     string `json:"api_base"`
	InsecureTLS bool   `json:"insecure_tls"`
	HasAuth     bool   `json:"has_auth"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateContainerImageRequest struct {
	RegistryID uint   `json:"registry_id" validate:"required"`
	Repository string `json:"repository" validate:"required"`
}

type ContainerImageResponse struct {
	ID          uint                       `json:"id"`
	RegistryID  uint                       `json:"registry_id"`
	Repository  string                     `json:"repository"`
	Tags        []string                   `json:"tags"`
	RefreshedAt string                     `json:"refreshed_at"`
	Registry    *ContainerRegistryResponse `json:"registry,omitempty"`
}

type ListTagsResponse struct {
	Repository  string   `json:"repository"`
	Tags        []string `json:"tags"`
	RefreshedAt string   `json:"refreshed_at,omitempty"`
	Cached      bool     `json:"cached,omitempty"`
}
