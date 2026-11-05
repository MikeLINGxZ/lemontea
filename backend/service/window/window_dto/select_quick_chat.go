package window_dto

// SelectQuickChatInput 指定在快捷聊天窗口中选中的会话
type SelectQuickChatInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
}

// SelectQuickChatOutput 是发出会话选择事件后的空响应
type SelectQuickChatOutput struct{}
