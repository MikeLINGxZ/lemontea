package notification_dto

// NotifyInput 包含创建通知所需的数据
type NotifyInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 类别
	Kind string `json:"kind"`
	// 标题
	Title string `json:"title"`
	// 消息
	Message string `json:"message"`
}

// NotifyOutput 返回新建通知的标识
type NotifyOutput struct {
	// 通知编号
	NotificationID uint `json:"notification_id"`
}
