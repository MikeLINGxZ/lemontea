package settings_dto

// 封装校验快捷键设置所需的请求参数
type ValidateShortcutSettingsInput struct {
	// 快捷聊天快捷键
	QuickChatShortcut string `json:"quick_chat_shortcut"`
}

// 封装校验快捷键设置返回给调用方的数据
type ValidateShortcutSettingsOutput struct {
	// 快捷聊天快捷键
	QuickChatShortcut string `json:"quick_chat_shortcut"`
	// 当前状态
	Status string `json:"status"`
	// 消息
	Message string `json:"message"`
}

// 封装保存快捷键设置所需的请求参数
type SaveShortcutSettingsInput struct {
	// 快捷聊天快捷键
	QuickChatShortcut string `json:"quick_chat_shortcut"`
}

// 封装保存快捷键设置返回给调用方的数据
type SaveShortcutSettingsOutput struct {
	// 快捷聊天快捷键
	QuickChatShortcut string `json:"quick_chat_shortcut"`
	// 当前状态
	Status string `json:"status"`
	// 消息
	Message string `json:"message"`
}
