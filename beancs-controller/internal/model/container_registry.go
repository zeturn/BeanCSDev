package model

import "time"

// ContainerRegistry 用户注册的容器镜像源（OCI Distribution v2 兼容）
type ContainerRegistry struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"size:128;not null;index" json:"user_id"`
	Kind        string    `gorm:"size:64;not null" json:"kind"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	APIBase     string    `gorm:"size:512;not null" json:"api_base"`
	UsernameEnc []byte    `gorm:"type:bytea" json:"-"`
	PasswordEnc []byte    `gorm:"type:bytea" json:"-"`
	InsecureTLS bool      `gorm:"not null;default:false" json:"insecure_tls"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ContainerImage 跟踪某个仓库的标签缓存（用于列表展示与刷新）
type ContainerImage struct {
	ID          uint               `gorm:"primaryKey" json:"id"`
	UserID      string             `gorm:"size:128;not null;index" json:"user_id"`
	RegistryID  uint               `gorm:"not null;index" json:"registry_id"`
	Repository  string             `gorm:"size:512;not null" json:"repository"`
	TagsJSON    string             `gorm:"type:text" json:"-"`
	RefreshedAt time.Time          `json:"refreshed_at"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Registry    *ContainerRegistry `gorm:"foreignKey:RegistryID" json:"-"`
}
