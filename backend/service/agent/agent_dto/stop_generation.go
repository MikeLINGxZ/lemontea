package agent_dto

// 封装取消指定会话仍在运行的模型请求所需的请求参数
type StopGenerationInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
}

// 封装取消指定会话仍在运行的模型请求返回给调用方的数据
type StopGenerationOutput struct {
}
