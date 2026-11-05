package memory_dto

// 封装获取记忆所需的请求参数
type GetMemoryInput struct {
	// 唯一标识
	ID uint `json:"id"`
}

// 封装获取记忆返回给调用方的数据
type GetMemoryOutput struct {
	// 记忆
	Memory MemoryItem `json:"memory"`
}
