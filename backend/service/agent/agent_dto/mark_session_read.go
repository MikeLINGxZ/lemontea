package agent_dto

// 封装清除指定会话的未读状态所需的请求参数
type MarkSessionReadInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
}

// 封装清除指定会话的未读状态返回给调用方的数据
type MarkSessionReadOutput struct {
}
