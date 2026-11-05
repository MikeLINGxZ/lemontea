package agent_dto

// 封装根据请求更新会话的收藏状态所需的请求参数
type ToggleStarSessionInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 已收藏
	Starred bool `json:"starred"`
}

// 封装根据请求更新会话的收藏状态返回给调用方的数据
type ToggleStarSessionOutput struct {
}
