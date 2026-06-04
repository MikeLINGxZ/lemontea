package view_model

// SettingsBootstrap contains the initial settings payload required by the frontend settings window.
type SettingsBootstrap struct {
	Locale            string `json:"locale"`
	Language          string `json:"language"`
	FontSize          string `json:"font_size"`
	DataDir           string `json:"data_dir"`
	LogLevel          string `json:"log_level"`
	QuickChatShortcut string `json:"quick_chat_shortcut"`
	DefaultProviderID uint   `json:"default_provider_id"`
	Version           string `json:"version"`
}
