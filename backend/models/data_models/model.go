package data_models

// 描述模型使用的数据结构
type Model struct {
	// ORM模型
	OrmModel
	ProviderId uint `gorm:"index" json:"provider_id"` // 提供方 ID
	// 模型
	Model string `gorm:"index" json:"model"`
	// 所属按
	OwnedBy string `gorm:"type:varchar(255)" json:"owned_by"`
	// 对象
	Object string `gorm:"type:varchar(255)" json:"object"`
	// 启用
	Enable bool `gorm:"index;type:bool;default:1" json:"enable"`
	// 别名
	Alias *string `gorm:"index;type:varchar(255)" json:"alias"`
	// 是否自定义
	IsCustom bool `gorm:"index;type:bool;default:0" json:"is_custom"`
}
