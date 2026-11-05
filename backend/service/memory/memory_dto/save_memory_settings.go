package memory_dto

// 封装保存记忆设置所需的请求参数
type SaveMemorySettingsInput struct {
	// 启用状态
	Enabled bool `json:"enabled"`
}

// 封装保存记忆设置返回给调用方的数据
type SaveMemorySettingsOutput struct {
}
