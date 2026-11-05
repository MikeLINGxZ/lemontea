package notification_dto

// ResolveNotificationInput 指定要处理的通知
type ResolveNotificationInput struct {
	// 唯一标识
	ID uint `json:"id"`
}

// ResolveNotificationOutput 表示通知的接受或拒绝操作已完成
type ResolveNotificationOutput struct{}
