package settings_dto

// 封装保存显示设置所需的请求参数
type SaveDisplaySettingsInput struct {
	// 字体尺寸
	FontSize string `json:"font_size"`
}

// 封装保存显示设置返回给调用方的数据
type SaveDisplaySettingsOutput struct {
}
