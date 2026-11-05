/** 主题的可选值 */
export type Theme = 'auto' | 'light' | 'dark'
/** 字体尺寸的可选值 */
export type FontSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'
/** 语言的可选值 */
export type Language = 'zh-CN' | 'en'
/** 会话标签页的可选值 */
export type ConversationTab = 'chats' | 'favorites'
/** 会话状态的可选值 */
export type ConversationStatus =
  | 'idle'
  | 'loading'
  | 'done-unread'
  | 'error-unread'
  | 'waiting-unread'

/** 消息内容类型的可选值 */
export type MessageContentType =
  | 'text'
  | 'tool_call'
  | 'tool_result'
  | 'thinking'
  | 'confirm_request'
  | 'confirm_response'
  | 'error'

/** 消息角色的可选值 */
export type MessageRole = 'user' | 'assistant' | 'tool' | 'system'

/** 附件类别的可选值 */
export type AttachmentKind = 'image' | 'file'

/** 定义附件的数据结构 */
export type Attachment = {
  /** 文件路径 */
  path: string
  /** 名称 */
  name: string
  /** 文件的 MIME 类型 */
  mime: string
  /** 类别标识 */
  kind: AttachmentKind
}

/** 定义会话的数据结构 */
export type Conversation = {
  /** 当前条目的唯一编号 */
  id: number
  /** 标题 */
  title: string
  /** 类别标识 */
  kind: 'user' | 'task'
  /** 标签列表 */
  tags?: string[]
  /** 创建时间 */
  createdAt: string
  /** 更新时间 */
  updatedAt: string
  /** 已收藏 */
  starred: boolean
  /** 状态 */
  status: ConversationStatus
}

/** 定义消息的数据结构 */
export type Message = {
  /** 当前条目的唯一编号 */
  id: number
  /** 会话编号 */
  sessionId: number
  /** 父消息编号 */
  parentId: number | null
  /** 消息发送方的角色 */
  role: MessageRole
  /** 消息内容类型 */
  contentType: MessageContentType
  /** 内容 */
  content: string
  /** 模型名称 */
  modelName: string
  /** 智能体名称 */
  agentName: string
  /** 输入 Token 数 */
  tokensIn: number
  /** 输出 Token 数 */
  tokensOut: number
  /** 附加数据 */
  extra: string
  /** 附件列表 */
  attachments?: Attachment[]
  /** 创建时间 */
  createdAt: string
}

/** 定义显示消息的数据结构 */
export type DisplayMessage = Message & {
  /** 工具结果 */
  toolResult?: string
  /** 工具确认操作 */
  toolConfirmAction?: 'approve' | 'reject' | 'comment'
  /** 工具确认备注 */
  toolConfirmComment?: string
  /** 是否工具分组 */
  isToolGroup?: boolean
  /** 已分组工具列表 */
  groupedTools?: DisplayMessage[]
}

/** 流式输出分片事件的数据 */
export type StreamChunkEvent = {
  /** 会话编号 */
  sessionId: number
  /** 序号 */
  seq?: number
  /** 增量 */
  delta: string
  /** 内容 */
  content?: string
  /** 消息内容类型 */
  contentType: 'text' | 'thinking'
}

/** 工具调用事件的数据 */
export type ToolCallEvent = {
  /** 会话编号 */
  sessionId: number
  /** 工具名称 */
  toolName: string
  /** 参数 */
  args: string
  /** 用途 */
  purpose: string
}

/** 工具结果事件的数据 */
export type ToolResultEvent = {
  /** 会话编号 */
  sessionId: number
  /** 工具名称 */
  toolName: string
  /** 本次调用的结果 */
  result: string
}

/** 确认请求事件的数据 */
export type ConfirmRequestEvent = {
  /** 会话编号 */
  sessionId: number
  /** 请求编号 */
  requestId: string
  /** 工具名称 */
  toolName: string
  /** 参数 */
  args: string
  /** 用途 */
  purpose: string
}

/** 流式输出完成事件的数据 */
export type StreamDoneEvent = {
  /** 会话编号 */
  sessionId: number
  /** 用量 */
  usage: { /* 输入 */ input: number; /* 输出 */ output: number }
}

/** 流式输出错误事件的数据 */
export type StreamErrorEvent = {
  /** 会话编号 */
  sessionId: number
  /** 错误 */
  error: string
}

/** 会话状态事件的数据 */
export type SessionStatusEvent = {
  /** 会话编号 */
  sessionId: number
  /** 状态 */
  status: ConversationStatus
}

/** 后端创建任务或会话时推送的会话摘要 */
export type SessionSpawnedEvent = {
  /** 新会话编号 */
  sessionId: number
  /** 初始会话标题 */
  title: string
  /** 后端提供的会话类别 */
  kind: string
  /** 后端为新会话设置的标签 */
  tags?: string[]
  /** 用于构造首轮消息的用户输入 */
  userMessage?: string
}

/** 定义工具的数据结构 */
export type Tool = {
  /** 当前条目的唯一编号 */
  id: string
  /** 名称 */
  name: string
  /** 描述 */
  description: string
  /** 是否启用 */
  enabled: boolean
}

/** 定义模型模型服务商的数据结构 */
export type ModelProvider = {
  /** 当前条目的唯一编号 */
  id: string
  /** 名称 */
  name: string
  /** 模型列表 */
  models: Model[]
}

/** 定义模型的数据结构 */
export type Model = {
  /** 当前条目的唯一编号 */
  id: string
  /** 名称 */
  name: string
  /** 模型服务商编号 */
  providerId: string
}
