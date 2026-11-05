package data_models

import "time"

// Notification 保存后台任务发给用户的通知
type Notification struct {
	// ORM模型
	OrmModel
	// 会话编号
	SessionID uint `gorm:"index" json:"session_id"`
	// 类别
	Kind string `gorm:"type:varchar(40);index" json:"kind"`
	// 标题
	Title string `gorm:"type:varchar(255)" json:"title"`
	// 消息
	Message string `gorm:"type:text" json:"message"`
	// 已处理时间
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}
