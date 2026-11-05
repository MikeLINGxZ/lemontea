package agent_dto

// 封装把用户对工具调用的确认结果交给正在等待的会话所需的请求参数
type RespondToConfirmInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 已同意
	Approved bool `json:"approved"`
	// 消息
	Message string `json:"message"`
	// 操作
	Action string `json:"action"`
}

// 封装把用户对工具调用的确认结果交给正在等待的会话返回给调用方的数据
type RespondToConfirmOutput struct {
}
