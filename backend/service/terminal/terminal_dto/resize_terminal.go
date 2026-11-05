package terminal_dto

// ResizeTerminalInput 携带前端终端的当前尺寸
type ResizeTerminalInput struct {
	// 终端编号
	TerminalID string `json:"terminal_id"`
	// 行数
	Rows uint16 `json:"rows"`
	// 列数
	Cols uint16 `json:"cols"`
}

// ResizeTerminalOutput 表示尺寸调整请求已处理
type ResizeTerminalOutput struct{}
