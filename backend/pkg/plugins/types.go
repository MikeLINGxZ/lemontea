package plugins

import "encoding/json"

const (
	TypeAgent   = "agent_plugin"
	TypeGeneral = "general_plugin"

	CurrentPluginAPIVersion = 1

	StatusDisabled = "disabled"
	StatusEnabled  = "enabled"
	StatusError    = "error"

	SourceTypePlugin = "plugin"
)

type Manifest struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Version          string          `json:"version"`
	PluginAPIVersion int             `json:"plugin_api_version"`
	Description      string          `json:"description"`
	Type             string          `json:"type"`
	Main             string          `json:"main"`
	MinHostVersion   string          `json:"minHostVersion"`
	Author           string          `json:"author"`
	Permissions      []string        `json:"permissions"`
	Views            []PluginView    `json:"views"`
	SettingsView     *PluginView     `json:"settingsView"`
	Capabilities     *Capabilities   `json:"capabilities"`
	Raw              json.RawMessage `json:"-"`
}

type Capabilities struct {
	UseTools  []PluginTool  `json:"useTools"`
	ViewTools []PluginTool  `json:"viewTools"`
	Agents    []PluginAgent `json:"agents"`
	Views     []PluginView  `json:"views"`
	Hooks     []string      `json:"hooks"`
}

type PluginTool struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	InputSchema         map[string]interface{} `json:"inputSchema"`
	ViewID              string                 `json:"viewId"`
	RequireConfirmation bool                   `json:"requireConfirmation"`
}

type PluginAgent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PluginView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Entry string `json:"entry"`
}

type PluginRecord struct {
	Manifest       Manifest     `json:"manifest"`
	InstallPath    string       `json:"installPath"`
	DataPath       string       `json:"dataPath"`
	Enabled        bool         `json:"enabled"`
	Status         string       `json:"status"`
	LastError      string       `json:"lastError"`
	Runtime        Capabilities `json:"runtime"`
	InstalledAt    string       `json:"installedAt"`
	LastStartedAt  string       `json:"lastStartedAt"`
	LastStoppedAt  string       `json:"lastStoppedAt"`
	RestartCount   int          `json:"restartCount"`
	RuntimeHealthy bool         `json:"runtimeHealthy"`
}

type Summary struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Version          string        `json:"version"`
	PluginAPIVersion int           `json:"plugin_api_version"`
	Description      string        `json:"description"`
	Type             string        `json:"type"`
	Author           string        `json:"author"`
	Enabled          bool          `json:"enabled"`
	Status           string        `json:"status"`
	LastError        string        `json:"last_error"`
	HasSettings      bool          `json:"has_settings"`
	Permissions      []string      `json:"permissions"`
	UseTools         []PluginTool  `json:"use_tools"`
	ViewTools        []PluginTool  `json:"view_tools"`
	Agents           []PluginAgent `json:"agents"`
	Views            []PluginView  `json:"views"`
	Hooks            []string      `json:"hooks"`
}

type RuntimeStatus struct {
	Available       bool   `json:"available"`
	RuntimePath     string `json:"runtime_path"`
	NodePath        string `json:"node_path"`
	Version         string `json:"version"`
	DownloadURL     string `json:"download_url"`
	Error           string `json:"error"`
	Downloading     bool   `json:"downloading"`
	Progress        int    `json:"progress"`
	DownloadedBytes int64  `json:"downloaded_bytes"`
	TotalBytes      int64  `json:"total_bytes"`
	Phase           string `json:"phase"`
}

type HookMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BeforeLLMSendPayload struct {
	ChatUUID string        `json:"chat_uuid"`
	Messages []HookMessage `json:"messages"`
}

type BeforeLLMSendResult struct {
	Messages []HookMessage `json:"messages"`
}

type AfterLLMSendPayload struct {
	ChatUUID      string `json:"chat_uuid"`
	MessageUUID   string `json:"message_uuid"`
	FinishReason  string `json:"finish_reason"`
	FinishError   string `json:"finish_error"`
	AssistantText string `json:"assistant_text"`
}
