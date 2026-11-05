package window_dto

// 封装打开设置所需的请求参数
type OpenSettingsInput struct {
	// 标签页
	Tab string `json:"tab"`
}

// 封装打开设置返回给调用方的数据
type OpenSettingsOutput struct {
}
