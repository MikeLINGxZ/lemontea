package view_model

// SettingsBootstrap 包含前端设置窗口初始化所需的数据
type SettingsBootstrap struct {
	// 地区
	Locale string `json:"locale"`
	// 语言
	Language string `json:"language"`
	// 字体尺寸
	FontSize string `json:"font_size"`
	// 数据目录
	DataDir string `json:"data_dir"`
	// 日志级别
	LogLevel string `json:"log_level"`
	// 快捷聊天快捷键
	QuickChatShortcut string `json:"quick_chat_shortcut"`
	// 默认提供商编号
	DefaultProviderID uint `json:"default_provider_id"`
	// 版本
	Version string `json:"version"`
}
