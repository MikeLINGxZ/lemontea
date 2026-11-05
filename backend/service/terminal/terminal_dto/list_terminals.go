package terminal_dto

import pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"

// ListTerminalsInput 指定要查询终端列表的聊天会话
type ListTerminalsInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
}

// ListTerminalsOutput 返回指定聊天会话的终端元数据
type ListTerminalsOutput struct {
	// 条目列表
	Items []pkgterminal.Info `json:"items"`
}
