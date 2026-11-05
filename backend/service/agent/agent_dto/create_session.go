package agent_dto

// 封装创建并保存一个新会话所需的请求参数
type CreateSessionInput struct {
	// 标题
	Title string `json:"title"`
	// 标签列表
	Tags []string `json:"tags,omitempty"`
}

// 封装创建并保存一个新会话返回给调用方的数据
type CreateSessionOutput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 标题
	Title string `json:"title"`
	// 标签列表
	Tags []string `json:"tags"`
}
