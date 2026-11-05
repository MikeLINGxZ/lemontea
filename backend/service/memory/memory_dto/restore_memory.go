package memory_dto

// 封装恢复记忆所需的请求参数
type RestoreMemoryInput struct {
	// 唯一标识
	ID uint `json:"id"`
}

// 封装恢复记忆返回给调用方的数据
type RestoreMemoryOutput struct {
}
