package data_models

// Session 保存聊天会话，包括普通聊天和自动任务会话
type Session struct {
	// ORM模型
	OrmModel
	// 标题
	Title string `gorm:"type:varchar(255)" json:"title"`
	// 已收藏
	Starred bool `gorm:"type:bool;default:0;index" json:"starred"`
	// 当前状态
	Status string `gorm:"type:varchar(50);default:'idle'" json:"status"`
	// 类别
	Kind string `gorm:"type:varchar(20);default:'user';index" json:"kind"`
	// 标签列表
	Tags string `gorm:"type:text" json:"tags"`
}
