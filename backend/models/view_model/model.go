package view_model

// 描述模型使用的数据结构
type Model struct {
	// 唯一标识
	ID uint `json:"id"`
	// 提供商编号
	ProviderId uint `gorm:"index" json:"provider_id"`
	// 模型
	Model string `json:"model"`
	// 所属按
	OwnedBy string `json:"owned_by"`
	// 对象
	Object string `json:"object"`
	// 启用
	Enable bool `json:"enable"`
	// 别名
	Alias *string `json:"alias"`
	// 是否自定义
	IsCustom  bool `json:"is_custom"`
	IsDefault bool `json:"is_default"` // 是否是默认模型
}
