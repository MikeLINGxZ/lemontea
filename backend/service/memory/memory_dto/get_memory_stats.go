package memory_dto

// 封装获取记忆统计所需的请求参数
type GetMemoryStatsInput struct {
}

// 封装获取记忆统计返回给调用方的数据
type GetMemoryStatsOutput struct {
	// 统计
	Stats MemoryStatsItem `json:"stats"`
}
