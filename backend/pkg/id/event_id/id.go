package event_id

// LocaleChanged event is triggered when the application locale is changed.
const LocaleChanged = "locale_changed"

// LanguageChanged event is triggered when the application language is changed.
const LanguageChanged = "language_changed"

// FontSizeChanged event is triggered when the application font size is changed.
const FontSizeChanged = "font_size_changed"

// DefaultProviderChanged event is triggered when the default provider is changed.
const DefaultProviderChanged = "default_provider_changed"

// AppAlert event is triggered when the backend wants the frontend to display an alert.
const AppAlert = "app:alert"

const AgentSessionSpawned = "agent:session:spawned"
const AgentStreamChunk = "agent:stream:chunk"
const AgentStreamToolCall = "agent:stream:tool_call"
const AgentStreamConfirmRequest = "agent:stream:confirm_request"
const AgentStreamToolResult = "agent:stream:tool_result"
const AgentStreamDone = "agent:stream:done"
const AgentStreamError = "agent:stream:error"
const AgentSessionStatus = "agent:session:status"

// QuickChatSelected event asks the home window to activate a session selected from the quick chat popup.
const QuickChatSelected = "quick_chat:selected"

// QuickChatReset event asks the quick chat window to return to its input shell without reloading.
const QuickChatReset = "quick_chat:reset"
