package settings_dto

type ValidateShortcutSettingsInput struct {
	QuickChatShortcut string `json:"quick_chat_shortcut"`
}

type ValidateShortcutSettingsOutput struct {
	QuickChatShortcut string `json:"quick_chat_shortcut"`
	Status            string `json:"status"`
	Message           string `json:"message"`
}

type SaveShortcutSettingsInput struct {
	QuickChatShortcut string `json:"quick_chat_shortcut"`
}

type SaveShortcutSettingsOutput struct {
	QuickChatShortcut string `json:"quick_chat_shortcut"`
	Status            string `json:"status"`
	Message           string `json:"message"`
}
