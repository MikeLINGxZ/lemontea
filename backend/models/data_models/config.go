package data_models

// ExtensionTool stores discovered tool metadata for an imported extension.
type ExtensionTool struct {
	ToolID          string `json:"tool_id"`
	ServerID        string `json:"server_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Enabled         bool   `json:"enabled"`
	RequiresConfirm bool   `json:"requires_confirm"`
}

// ExtensionItem stores one imported MCP tool or plugin entry.
type ExtensionItem struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Author         string          `json:"author"`
	Version        string          `json:"version"`
	Kind           string          `json:"kind"`
	Enabled        bool            `json:"enabled"`
	RuntimeStatus  string          `json:"runtime_status"`
	RuntimeMessage string          `json:"runtime_message"`
	RootDir        string          `json:"root_dir"`
	SourceDir      string          `json:"source_dir"`
	ConfigFilePath string          `json:"config_file_path"`
	Tools          []ExtensionTool `json:"tools"`
}

// Config data structure of config file (config.json)
type Config struct {
	Locale            string          `json:"locale"`
	Language          string          `json:"language"`
	FontSize          string          `json:"font_size"`
	DataDir           string          `json:"data_dir"`
	LogLevel          string          `json:"log_level"`
	QuickChatShortcut string          `json:"quick_chat_shortcut"`
	DefaultProviderID uint            `json:"default_provider_id"`
	Extensions        []ExtensionItem `json:"extensions"`
	// DisabledSkills lists skill names the user has explicitly disabled.
	DisabledSkills []string     `json:"disabled_skills,omitempty"`
	Memory         MemoryConfig `json:"memory"`
}

// MemoryConfig stores long-term memory feature settings.
type MemoryConfig struct {
	Enabled bool `json:"enabled"`
}
