package settings_dto

// 封装保存地区设置所需的请求参数
type SaveLocaleSettingsInput struct {
	// 地区
	Locale string `json:"locale"`
	// 语言
	Language string `json:"language"`
}

// 封装保存地区设置返回给调用方的数据
type SaveLocaleSettingsOutput struct {
}
