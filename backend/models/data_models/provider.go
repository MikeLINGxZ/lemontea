package data_models

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"

// 描述提供商使用的数据结构
type Provider struct {
	// ORM模型
	OrmModel
	ProviderName string        `gorm:"type:varchar(255)" json:"provider_name"`  // 提供方名称
	ProviderType provider.Type `gorm:"type:varchar(255)" json:"provider_type"`  // 提供方类型
	BaseUrl      string        `gorm:"type:varchar(255)" json:"base_url"`       // 基础 URL
	ApiKey       string        `gorm:"type:varchar(255)" json:"api_key"`        // API 密钥
	Enable       bool          `gorm:"index;type:bool;default:1" json:"enable"` // 启用
}

// 描述提供商默认模型使用的数据结构
type ProviderDefaultModel struct {
	// ORM模型
	OrmModel
	// 提供商编号
	ProviderID uint `gorm:"index:idx_provider_model" json:"provider_id"`
	// 模型编号
	ModelId uint `gorm:"index:idx_provider_model" json:"model_id"`
}
