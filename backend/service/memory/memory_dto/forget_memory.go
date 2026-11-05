package memory_dto

// 封装遗忘记忆所需的请求参数
type ForgetMemoryInput struct {
	// 唯一标识
	ID uint `json:"id"`
}

// 封装遗忘记忆返回给调用方的数据
type ForgetMemoryOutput struct {
}
