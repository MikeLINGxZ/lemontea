package window_dto

// SelectQuickChatInput identifies the session selected from the quick chat window.
type SelectQuickChatInput struct {
	SessionID uint `json:"session_id"`
}

// SelectQuickChatOutput carries no fields after the selection event is emitted.
type SelectQuickChatOutput struct{}
