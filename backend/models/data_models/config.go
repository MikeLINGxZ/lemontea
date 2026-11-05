package data_models

// ExtensionTool 保存导入扩展时发现的工具元数据
type ExtensionTool struct {
	// 工具编号
	ToolID string `json:"tool_id"`
	// 服务编号
	ServerID string `json:"server_id"`
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 启用状态
	Enabled bool `json:"enabled"`
	// 需要确认
	RequiresConfirm bool `json:"requires_confirm"`
}

// ExtensionItem 保存一项已导入的 MCP 工具或插件
type ExtensionItem struct {
	// 唯一标识
	ID string `json:"id"`
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 作者
	Author string `json:"author"`
	// 版本
	Version string `json:"version"`
	// 类别
	Kind string `json:"kind"`
	// 启用状态
	Enabled bool `json:"enabled"`
	// 运行环境状态
	RuntimeStatus string `json:"runtime_status"`
	// 运行环境消息
	RuntimeMessage string `json:"runtime_message"`
	// 根目录目录
	RootDir string `json:"root_dir"`
	// 来源目录
	SourceDir string `json:"source_dir"`
	// 配置文件路径
	ConfigFilePath string `json:"config_file_path"`
	// 工具列表
	Tools []ExtensionTool `json:"tools"`
}

// Config 对应 config.json 的配置结构
type Config struct {
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
	// 扩展列表
	Extensions []ExtensionItem `json:"extensions"`
	// DisabledSkills 列出用户主动禁用的技能名称
	DisabledSkills []string `json:"disabled_skills,omitempty"`
	// 记忆
	Memory MemoryConfig `json:"memory"`
}

// MemoryConfig 保存长期记忆功能的设置
type MemoryConfig struct {
	// 启用状态
	Enabled bool `json:"enabled"`
}
