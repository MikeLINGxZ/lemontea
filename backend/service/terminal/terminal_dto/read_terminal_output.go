package terminal_dto

import pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"

// ReadTerminalOutputInput 指定要读取的终端和起始字节游标
type ReadTerminalOutputInput struct {
	// 终端编号
	TerminalID string `json:"terminal_id"`
	// 游标
	Cursor int64 `json:"cursor"`
}

// ReadTerminalOutputOutput 返回指定游标之后的终端输出片段
type ReadTerminalOutputOutput struct {
	// 分片列表
	Chunks []pkgterminal.TerminalOutputChunk `json:"chunks"`
}
