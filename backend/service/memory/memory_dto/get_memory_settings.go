package memory_dto

// 封装获取记忆设置所需的请求参数
type GetMemorySettingsInput struct {
}

// 封装获取记忆设置返回给调用方的数据
type GetMemorySettingsOutput struct {
	// 启用状态
	Enabled bool `json:"enabled"`
}
