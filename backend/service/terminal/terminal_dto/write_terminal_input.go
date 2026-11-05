package terminal_dto

// WriteTerminalInputInput 携带要发送给活动终端的用户输入
type WriteTerminalInputInput struct {
	// 终端编号
	TerminalID string `json:"terminal_id"`
	// 数据
	Data string `json:"data"`
}

// WriteTerminalInputOutput 表示终端输入已接收
type WriteTerminalInputOutput struct{}
