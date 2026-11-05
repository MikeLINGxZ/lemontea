package notification_dto

// DismissNotificationInput 指定要删除的通知
type DismissNotificationInput struct {
	// 唯一标识
	ID uint `json:"id"`
}

// DismissNotificationOutput 表示通知删除成功
type DismissNotificationOutput struct{}
