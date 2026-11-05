package notification_dto

import "time"

// ListNotificationsInput 指定是否包含已处理的通知
type ListNotificationsInput struct {
	// 包含已处理
	IncludeResolved bool `json:"include_resolved"`
}

// NotificationItem 表示返回给前端的通知
type NotificationItem struct {
	// 唯一标识
	ID uint `json:"id"`
	// 会话编号
	SessionID uint `json:"session_id"`
	// 类别
	Kind string `json:"kind"`
	// 标题
	Title string `json:"title"`
	// 消息
	Message string `json:"message"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 已处理时间
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// ListNotificationsOutput 包含查询得到的通知列表
type ListNotificationsOutput struct {
	// 条目列表
	Items []NotificationItem `json:"items"`
}
