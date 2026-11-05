package agent_dto

// 封装按查询条件和游标返回会话列表所需的请求参数
type ListSessionsInput struct {
	// 游标
	Cursor uint `json:"cursor"`
	// 上限
	Limit int `json:"limit"`
	// 已收藏仅
	StarredOnly bool `json:"starred_only"`
	// 包含隐藏
	IncludeHidden bool `json:"include_hidden"`
}

// 描述会话条目使用的数据结构
type SessionItem struct {
	// 唯一标识
	ID uint `json:"id"`
	// 标题
	Title string `json:"title"`
	// 类别
	Kind string `json:"kind"`
	// 标签列表
	Tags []string `json:"tags"`
	// 已收藏
	Starred bool `json:"starred"`
	// 当前状态
	Status string `json:"status"`
	// 创建
	Created string `json:"created"`
	// 更新
	Updated string `json:"updated"`
}

// 封装按查询条件和游标返回会话列表返回给调用方的数据
type ListSessionsOutput struct {
	// 会话列表
	Sessions []SessionItem `json:"sessions"`
	// 下一项游标
	NextCursor uint `json:"next_cursor"`
	// 是否有更多
	HasMore bool `json:"has_more"`
}
