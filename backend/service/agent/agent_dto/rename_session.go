package agent_dto

// 封装重命名会话所需的请求参数
type RenameSessionInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 标题
	Title string `json:"title"`
}

// 封装重命名会话返回给调用方的数据
type RenameSessionOutput struct {
}
