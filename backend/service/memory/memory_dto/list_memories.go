package memory_dto

// 封装列出记忆列表所需的请求参数
type ListMemoriesInput struct {
	// 搜索词
	Query string `json:"query"`
	// 类型
	Type string `json:"type"`
	// 目标
	Target string `json:"target"`
	// 包含已遗忘
	IncludeForgotten bool `json:"include_forgotten"`
	// 偏移
	Offset int `json:"offset"`
	// 上限
	Limit int `json:"limit"`
}

// 封装列出记忆列表返回给调用方的数据
type ListMemoriesOutput struct {
	// 条目列表
	Items []MemoryItem `json:"items"`
	// 总量
	Total int64 `json:"total"`
}
