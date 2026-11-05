package settings_dto

// 封装应用文件设置所需的请求参数
type ApplyFileSettingsInput struct {
	// 目标目录
	TargetDir string `json:"target_dir"`
}

// 封装应用文件设置返回给调用方的数据
type ApplyFileSettingsOutput struct {
}
