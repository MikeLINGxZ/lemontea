package agent_dto

// 封装删除会话所需的请求参数
type DeleteSessionInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
}

// 封装删除会话返回给调用方的数据
type DeleteSessionOutput struct {
}
