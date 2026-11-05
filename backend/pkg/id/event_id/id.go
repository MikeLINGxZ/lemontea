package event_id

// LocaleChanged 在应用的区域设置发生变化时触发
const LocaleChanged = "locale_changed"

// LanguageChanged 在应用语言发生变化时触发
const LanguageChanged = "language_changed"

// FontSizeChanged 在应用字号发生变化时触发
const FontSizeChanged = "font_size_changed"

// DefaultProviderChanged 在默认供应商发生变化时触发
const DefaultProviderChanged = "default_provider_changed"

// AppAlert 由后端触发，用于请求前端显示提示
const AppAlert = "app:alert"

// 智能体会话已创建
const AgentSessionSpawned = "agent:session:spawned"

// 智能体流式输出分片
const AgentStreamChunk = "agent:stream:chunk"

// 智能体流式输出工具调用
const AgentStreamToolCall = "agent:stream:tool_call"

// 智能体流式输出确认请求
const AgentStreamConfirmRequest = "agent:stream:confirm_request"

// 智能体流式输出工具结果
const AgentStreamToolResult = "agent:stream:tool_result"

// 智能体流式输出完成
const AgentStreamDone = "agent:stream:done"

// 智能体流式输出错误
const AgentStreamError = "agent:stream:error"

// 智能体会话状态
const AgentSessionStatus = "agent:session:status"

// QuickChatSelected 请求主窗口激活在快捷聊天弹窗中选中的会话
const QuickChatSelected = "quick_chat:selected"

// QuickChatReset 请求快捷聊天窗口直接回到输入界面，无需重新加载
const QuickChatReset = "quick_chat:reset"
