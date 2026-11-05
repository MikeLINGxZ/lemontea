package data_models

// 描述消息使用的数据结构
type Message struct {
	// ORM模型
	OrmModel
	// 会话编号
	SessionID uint `gorm:"index" json:"session_id"`
	// 父级编号
	ParentID *uint `gorm:"index" json:"parent_id"`
	// 角色
	Role string `gorm:"type:varchar(50)" json:"role"`
	// 内容类型
	ContentType string `gorm:"type:varchar(50)" json:"content_type"`
	// 内容
	Content string `gorm:"type:text" json:"content"`
	// 模型名称
	ModelName string `gorm:"type:varchar(255)" json:"model_name"`
	// 智能体名称
	AgentName string `gorm:"type:varchar(255)" json:"agent_name"`
	// Token输入
	TokensIn int `json:"tokens_in"`
	// Token输出
	TokensOut int `json:"tokens_out"`
	// 附加数据
	Extra string `gorm:"type:text" json:"extra"`
	// 附件列表
	Attachments string `gorm:"type:text" json:"attachments"`
}
