import { create } from 'zustand'
import { Events } from '@wailsio/runtime'
import { isObjectRecord } from '@/lib/guards'
import { inferAttachmentMeta } from '@/lib/attachments'
import { Agent as AgentBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/agent'
import type { Type as ProviderType } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider/models'
import type {
  Attachment,
  ConfirmRequestEvent,
  Conversation,
  ConversationStatus,
  Message,
  SessionSpawnedEvent,
  SessionStatusEvent,
  StreamChunkEvent,
  StreamDoneEvent,
  StreamErrorEvent,
  ToolCallEvent,
  ToolResultEvent,
} from '../types'

/** 流式输出消息的状态数据 */
type StreamingMessageState = { /* 内容 */ content: string; /* 思考内容 */ thinking: string }
/** 定义待处理流式输出补丁的数据结构 */
type PendingStreamingPatch = {
  /** 内容增量 */
  contentDelta: string
  /** 思考内容增量 */
  thinkingDelta: string
  /** 内容快照 */
  contentSnapshot?: string
  /** 思考内容快照 */
  thinkingSnapshot?: string
}
/** 定义待处理标题生成的数据结构 */
type PendingTitleGeneration = {
  /** 服务的基础地址 */
  baseUrl: string
  /** 服务认证使用的 API 密钥 */
  apiKey: string
  /** 模型名称 */
  modelName: string
  /** 模型服务商类型 */
  providerType: ProviderType
}
/** 定义标题生成配置的数据结构 */
type TitleGenerationConfig = PendingTitleGeneration

/** 聊天状态仓库保存的数据和操作 */
interface ChatStore {
  /** 会话列表 */
  conversations: Conversation[]
  /** 消息列表 */
  messages: Record<number, Message[]>
  /** 会话状态表 */
  sessionStatuses: Record<number, ConversationStatus | undefined>
  /** 当前会话编号 */
  currentConversationId: number | null
  /** 流式输出消息列表 */
  streamingMessages: Record<number, StreamingMessageState>
  /** 待处理标题生成任务 */
  pendingTitleGenerations: Record<number, PendingTitleGeneration | undefined>
  /** 会话运行配置集合 */
  sessionRunConfigs: Record<number, PendingTitleGeneration | undefined>
  /** 按会话编号保存的待确认工具调用 */
  pendingConfirms: Record<number, {
    /** 请求编号 */
    requestId: string
    /** 工具名称 */
    toolName: string
    /** 参数 */
    args: string
    /** 用途 */
    purpose: string
  } | null>
  /** 会话列表加载中 */
  sessionsLoading: boolean
  /** 是否有更多会话列表 */
  hasMoreSessions: boolean
  /** 会话列表游标 */
  sessionsCursor: number
  /** 按游标加载会话列表，重置时从第一页开始，接收重置，完成后结束异步任务 */
  loadSessions: (reset?: boolean) => Promise<void>
  /** 读取指定会话的历史消息并恢复附件与流式占位，接收会话编号，完成后结束异步任务 */
  loadMessages: (sessionId: number) => Promise<void>
  /** 创建会话，接收参数，返回操作结果的编号 */
  createConversation: (params?: { /* 标题 */ title?: string; /* 标签列表 */ tags?: string[] }) => Promise<number>
  /** 删除会话，接收当前条目的唯一编号，完成后结束异步任务 */
  deleteConversation: (id: number) => Promise<void>
  /** 重命名会话，接收当前条目的唯一编号、标题，完成后结束异步任务 */
  renameConversation: (id: number, title: string) => Promise<void>
  /** 切换收藏，接收当前条目的唯一编号、已收藏，完成后结束异步任务 */
  toggleStar: (id: number, starred: boolean) => Promise<void>
  /** 标记为已读处理，接收当前条目的唯一编号，完成后结束异步任务 */
  markAsRead: (id: number) => Promise<void>
  /** 设置当前会话，接收当前条目的唯一编号，无返回值 */
  setCurrentConversation: (id: number | null) => void
  /** 设置会话状态，接收当前条目的唯一编号、状态，无返回值 */
  setConversationStatus: (id: number, status: ConversationStatus) => void
  /** 将用户消息和模型配置提交到后端，并记录本轮生成状态，接收参数，完成后结束异步任务 */
  sendMessage: (params: {
    /** 会话编号 */
    sessionId: number
    /** 内容 */
    content: string
    /** 服务的基础地址 */
    baseUrl: string
    /** 服务认证使用的 API 密钥 */
    apiKey: string
    /** 模型名称 */
    modelName: string
    /** 模型服务商类型 */
    providerType: ProviderType
    /** 启用状态用户工具列表 */
    enabledUserTools: string[]
    /** 附件列表 */
    attachments: Attachment[]
    /** 系统提示词 */
    systemPrompt?: string
    /** 技能名称 */
    skillName?: string
  }) => Promise<void>
  /** 停止生成，接收会话编号，完成后结束异步任务 */
  stopGeneration: (sessionId: number) => Promise<void>
  /** 向后端提交工具确认操作和用户说明，接收会话编号、已同意、消息、操作，完成后结束异步任务 */
  respondToConfirm: (
    sessionId: number,
    approved: boolean,
    message: string,
    action?: 'approve' | 'reject' | 'comment'
  ) => Promise<void>
  /** 生成标题，接收会话编号、服务的基础地址、服务认证使用的 API 密钥、模型名称、模型服务商类型，完成后结束异步任务 */
  generateTitle: (
    sessionId: number,
    baseUrl: string,
    apiKey: string,
    modelName: string,
    providerType: ProviderType
  ) => Promise<void>
  /** 生成标题草稿，接收会话编号、回退配置，返回处理结果，无法取得时为空 */
  generateTitleDraft: (sessionId: number, fallbackConfig?: TitleGenerationConfig) => Promise<string | null>
  /** 订阅会话、流式输出、工具及确认事件，返回清理回调 */
  initEventListeners: () => () => void
}

/** 空流式输出消息 */
const EMPTY_STREAMING_MESSAGE: StreamingMessageState = { content: '', thinking: '' }
/** 空流式输出补丁 */
const EMPTY_STREAMING_PATCH: PendingStreamingPatch = { contentDelta: '', thinkingDelta: '' }

/** 在下一渲染帧调度回调，环境不支持时改用定时器，接收回调，返回计算后的数值 */
function scheduleNextFrame(callback: FrameRequestCallback): number {
  if (typeof globalThis.requestAnimationFrame === 'function') {
    return globalThis.requestAnimationFrame(callback)
  }

  return globalThis.setTimeout(() => callback(Date.now()), 16) as number
}

/** 取消尚未执行的动画帧或备用定时器，接收异步任务句柄，无返回值 */
function cancelScheduledFrame(handle: number) {
  if (typeof globalThis.cancelAnimationFrame === 'function') {
    globalThis.cancelAnimationFrame(handle)
    return
  }

  globalThis.clearTimeout(handle)
}

/** 从聊天异常中提取消息和详情，解析失败时保留原始异常文本，接收捕获的异常，返回对应的处理结果 */
function parseIError(err: unknown): { /* 可展示的错误消息 */ msg: string; /* 详情 */ detail: string } {
  try {
    const message = err instanceof Error ? err.message : String(err) // 消息
    const parsed: unknown = JSON.parse(message) // 保存待校验的 JSON 解析结果
    if (!isObjectRecord(parsed)) return { msg: message, detail: message }
    return {
      msg: typeof parsed.msg === 'string' ? parsed.msg : message,
      detail: typeof parsed.detail === 'string' ? parsed.detail : message,
    }
  } catch {
    return { msg: String(err), detail: String(err) }
  }
}

/** 判断是否为流式输出状态，接收状态，返回是否满足条件 */
function isStreamingStatus(status: ConversationStatus | undefined): boolean {
  return status === 'loading' || status === 'waiting-unread'
}

/** 保存会话、消息、流式缓冲及工具确认状态，并提供 Wails 事件处理方法 */
export const useChatStore = create<ChatStore>()((set, get) => {
  const pendingStreamingBuffers = new Map<number, PendingStreamingPatch>() // 按会话编号保存的待合并流式文本
  const scheduledStreamingFlushes = new Map<number, number>() // 按会话编号保存的流式刷新任务
  const latestStreamingChunkSeq = new Map<string, number>() // 按会话与内容类型记录的最新分片序号

  /** 清除流式输出缓冲区，接收会话编号，无返回值 */
  const clearStreamingBuffer = (sessionId: number) => {
    const handle = scheduledStreamingFlushes.get(sessionId) // 异步任务句柄
    if (handle !== undefined) {
      cancelScheduledFrame(handle)
      scheduledStreamingFlushes.delete(sessionId)
    }
    pendingStreamingBuffers.delete(sessionId)
  }

  /** 清除流式输出序号表，接收会话编号，无返回值 */
  const clearStreamingSeqs = (sessionId: number) => {
    latestStreamingChunkSeq.delete(`${sessionId}:content`)
    latestStreamingChunkSeq.delete(`${sessionId}:thinking`)
  }

  /** 将待处理的快照或增量合并到流式消息并清空缓冲，接收会话编号，无返回值 */
  const flushStreamingBuffer = (sessionId: number) => {
    const pending = pendingStreamingBuffers.get(sessionId) // 当前帧尚未写入仓库的快照和增量
    clearStreamingBuffer(sessionId)

    if (!pending || (!pending.contentDelta && !pending.thinkingDelta &&
      pending.contentSnapshot === undefined && pending.thinkingSnapshot === undefined)) return

    // 快照替换当前文本，增量则追加到已有内容
    set((state) => {
      const previous = state.streamingMessages[sessionId] ?? EMPTY_STREAMING_MESSAGE // 上次刷新后已经显示的内容

      return {
        streamingMessages: {
          ...state.streamingMessages,
          [sessionId]: {
            content: pending.contentSnapshot ?? previous.content + pending.contentDelta,
            thinking: pending.thinkingSnapshot ?? previous.thinking + pending.thinkingDelta,
          },
        },
      }
    })
  }

  /** 为会话安排下一帧刷新，同一会话每帧最多提交一次 */
  const scheduleStreamingFlush = (sessionId: number) => {
    if (scheduledStreamingFlushes.has(sessionId)) return

    /** 异步任务句柄 */
    const handle = scheduleNextFrame(() => {
      flushStreamingBuffer(sessionId)
    })

    scheduledStreamingFlushes.set(sessionId, handle)
  }

  /** 按会话和内容类型缓存新分片，丢弃重复或倒序的分片，接收数据，无返回值 */
  const appendStreamingDelta = (data: StreamChunkEvent) => {
    const key = data.contentType === 'thinking' ? 'thinking' : 'content' // 将正文和思考分到独立的缓冲槽
    const seqKey = `${data.sessionId}:${key}` // 以会话和内容类型为维度记录最后接收的序号
    if (typeof data.seq === 'number') {
      const previousSeq = latestStreamingChunkSeq.get(seqKey) ?? 0 // 先前序号
      if (data.seq <= previousSeq) return
      latestStreamingChunkSeq.set(seqKey, data.seq)
    }

    /** 当前会话正在累积的快照和增量 */
    const previous = pendingStreamingBuffers.get(data.sessionId) ?? EMPTY_STREAMING_PATCH
    if (typeof data.content === 'string') {
      pendingStreamingBuffers.set(data.sessionId, {
        ...previous,
        [`${key}Snapshot`]: data.content,
        [`${key}Delta`]: '',
      })
    } else {
      const deltaKey = `${key}Delta` as 'contentDelta' | 'thinkingDelta' // 增量键
      pendingStreamingBuffers.set(data.sessionId, {
        ...previous,
        [deltaKey]: previous[deltaKey] + data.delta,
      })
    }

    scheduleStreamingFlush(data.sessionId)
  }

  /** 为正在生成的会话准备助手占位消息，接收会话编号，无返回值 */
  const ensureStreamingPlaceholder = (sessionId: number) => {
    /** 根据当前仓库状态计算并提交更新 */
    set((state) => {
      const conversation = state.conversations.find((item) => item.id === sessionId) // 会话
      const status = state.sessionStatuses[sessionId] ?? conversation?.status // 状态
      if (!isStreamingStatus(status)) {
        return state
      }

      const sessionMessages = state.messages[sessionId] // 会话消息列表
      if (!sessionMessages) {
        return state
      }

      const lastMessage = sessionMessages[sessionMessages.length - 1] // 最后一个消息
      if (lastMessage?.role === 'assistant' && lastMessage?.contentType !== 'tool_call') {
        return state
      }

      return {
        messages: {
          ...state.messages,
          [sessionId]: [
            ...sessionMessages,
            {
              id: -(sessionId * 1000) - sessionMessages.length - 1,
              sessionId,
              parentId: lastMessage?.role === 'user' ? lastMessage.id : null,
              role: 'assistant',
              contentType: 'text',
              content: '',
              modelName: '',
              agentName: '',
              tokensIn: 0,
              tokensOut: 0,
              extra: '',
              createdAt: new Date().toISOString(),
            },
          ],
        },
      }
    })
  }

  return ({
  conversations: [],
  messages: {},
  sessionStatuses: {},
  currentConversationId: null,
  streamingMessages: {},
  pendingTitleGenerations: {},
  sessionRunConfigs: {},
  pendingConfirms: {},
  sessionsLoading: false,
  hasMoreSessions: true,
  sessionsCursor: 0,

  /** 按游标加载会话列表，重置时从第一页开始，接收重置，完成后结束异步任务 */
  loadSessions: async (reset = false) => {
    if (get().sessionsLoading) return
    set({ sessionsLoading: true })

    try {
      const cursor = reset ? 0 : get().sessionsCursor // 游标
      /** 保存本次调用的结果 */
      const result = await AgentBinding.ListSessions({
        cursor,
        limit: 20,
        starred_only: false,
        include_hidden: true,
      })
      if (!result) return

      /** 本次加载的会话列表 */
      const nextConversations: Conversation[] = (result.sessions ?? []).map((session) => ({
        id: session.id,
        title: session.title,
        kind: ((session.tags ?? []).includes('task') || session.kind === 'task' ? 'task' : 'user') as Conversation['kind'],
        tags: session.tags ?? (session.kind === 'task' ? ['task'] : []),
        createdAt: session.created,
        updatedAt: session.updated,
        starred: session.starred,
        status: session.status as ConversationStatus,
      }))

      set((state) => ({
        conversations: reset ? nextConversations : [...state.conversations, ...nextConversations],
        sessionStatuses: {
          ...state.sessionStatuses,
          ...Object.fromEntries(nextConversations.map((conversation) => [conversation.id, conversation.status])),
        },
        hasMoreSessions: result.has_more,
        sessionsCursor: result.next_cursor,
      }))
    } finally {
      set({ sessionsLoading: false })
    }
  },

  /** 读取指定会话的历史消息并恢复附件与流式占位，接收会话编号，完成后结束异步任务 */
  loadMessages: async (sessionId) => {
    /** 保存本次调用的结果 */
    const result = await AgentBinding.LoadSessionMessages({
      session_id: sessionId,
      offset: 0,
      limit: 100,
    })
    if (!result) return

    /** 消息列表 */
    const messages: Message[] = (result.messages ?? []).map((message) => {
      let attachments: Attachment[] | undefined // 附件列表
      const raw = message.attachments // 尚未解析的原始值
      if (raw && raw.trim() !== '') {
        try {
          const parsed: unknown = JSON.parse(raw) // 保存待校验的 JSON 解析结果
          if (Array.isArray(parsed)) {
            /** 只读取具有有效路径的附件，旧记录缺少元数据时从路径补齐 */
            const validAttachments = parsed.flatMap((item: unknown): Attachment[] => {
              if (!isObjectRecord(item) || typeof item.path !== 'string' || !item.path) return []

              const inferred = inferAttachmentMeta(item.path) // 路径对应的默认附件信息
              const mime = typeof item.mime === 'string' && item.mime ? item.mime : inferred.mime // 保存的 MIME 类型优先
              const kind = item.kind === 'image' || item.kind === 'file' ? item.kind : mime.startsWith('image/') ? 'image' : 'file' // 展示类型
              return [{ path: item.path, name: typeof item.name === 'string' && item.name ? item.name : inferred.name, mime, kind }]
            })
            if (validAttachments.length > 0) attachments = validAttachments
          }
        } catch {
          // 附件数据无法解析时继续显示消息正文
        }
      }
      return {
        id: message.id,
        sessionId: message.session_id,
        parentId: message.parent_id ?? null,
        role: message.role as Message['role'],
        contentType: message.content_type as Message['contentType'],
        content: message.content,
        modelName: message.model_name,
        agentName: message.agent_name,
        tokensIn: message.tokens_in,
        tokensOut: message.tokens_out,
        extra: message.extra,
        attachments,
        createdAt: message.created,
      }
    })

    // 任务仍在运行时补入助手占位消息，让打开会话后可以继续接收流式内容
    const sessionStatus =
      get().sessionStatuses[sessionId] ??
      get().conversations.find((c) => c.id === sessionId)?.status
    const lastMsg = messages[messages.length - 1] // 会话的最后一条消息
    if (isStreamingStatus(sessionStatus) && (!lastMsg || lastMsg.role !== 'assistant' ||
      lastMsg.contentType === 'tool_call')) {
      messages.push({
        id: -(sessionId * 1000),
        sessionId,
        parentId: null,
        role: 'assistant',
        contentType: 'text',
        content: '',
        modelName: '',
        agentName: '',
        tokensIn: 0,
        tokensOut: 0,
        extra: '',
        createdAt: new Date().toISOString(),
      })
    }

    set((state) => ({
      messages: { ...state.messages, [sessionId]: messages },
    }))
  },

  /** 创建会话，接收参数，返回对应的处理结果 */
  createConversation: async (params) => {
    /** 保存本次调用的结果 */
    const result = await AgentBinding.CreateSession({ title: params?.title ?? '', tags: params?.tags ?? [] })
    if (!result) return 0

    const tags = result.tags ?? params?.tags ?? [] // 标签列表
    /** 会话 */
    const conversation: Conversation = {
      id: result.session_id,
      title: result.title,
      kind: tags.includes('task') ? 'task' : 'user',
      tags,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      starred: false,
      status: 'idle',
    }

    set((state) => ({
      conversations: [conversation, ...state.conversations],
      messages: { ...state.messages, [result.session_id]: [] },
      pendingTitleGenerations: {
        ...state.pendingTitleGenerations,
        [result.session_id]: undefined,
      },
      sessionRunConfigs: {
        ...state.sessionRunConfigs,
        [result.session_id]: undefined,
      },
      sessionStatuses: {
        ...state.sessionStatuses,
        [result.session_id]: 'idle',
      },
    }))
    return result.session_id
  },

  /** 删除会话，接收当前条目的唯一编号，完成后结束异步任务 */
  deleteConversation: async (id) => {
    await AgentBinding.DeleteSession({ session_id: id })
    clearStreamingBuffer(id)
    clearStreamingSeqs(id)
    /** 根据当前仓库状态计算并提交更新 */
    set((state) => {
      const nextMessages = { ...state.messages } // 下一项消息列表
      const nextPendingTitleGenerations = { ...state.pendingTitleGenerations } // 下一项待处理标题生成任务
      const nextSessionRunConfigs = { ...state.sessionRunConfigs } // 下一项会话运行配置集合
      const nextPendingConfirms = { ...state.pendingConfirms } // 下一项待处理确认项
      const nextStreamingMessages = { ...state.streamingMessages } // 下一项流式输出消息列表
      const nextSessionStatuses = { ...state.sessionStatuses } // 下一项会话状态表

      delete nextMessages[id]
      delete nextPendingTitleGenerations[id]
      delete nextSessionRunConfigs[id]
      delete nextPendingConfirms[id]
      delete nextStreamingMessages[id]
      delete nextSessionStatuses[id]

      return {
        conversations: state.conversations.filter((conversation) => conversation.id !== id),
        currentConversationId: state.currentConversationId === id ? null : state.currentConversationId,
        messages: nextMessages,
        pendingTitleGenerations: nextPendingTitleGenerations,
        sessionRunConfigs: nextSessionRunConfigs,
        pendingConfirms: nextPendingConfirms,
        streamingMessages: nextStreamingMessages,
        sessionStatuses: nextSessionStatuses,
      }
    })
  },

  /** 重命名会话，接收当前条目的唯一编号、标题，完成后结束异步任务 */
  renameConversation: async (id, title) => {
    await AgentBinding.RenameSession({ session_id: id, title })
    set((state) => ({
      conversations: state.conversations.map((conversation) =>
        conversation.id === id
          ? { ...conversation, title, updatedAt: new Date().toISOString() }
          : conversation
      ),
    }))
  },

  /** 切换收藏，接收当前条目的唯一编号、已收藏，完成后结束异步任务 */
  toggleStar: async (id, starred) => {
    await AgentBinding.ToggleStarSession({ session_id: id, starred })
    set((state) => ({
      conversations: state.conversations.map((conversation) =>
        conversation.id === id ? { ...conversation, starred } : conversation
      ),
    }))
  },

  /** 标记为已读处理，接收当前条目的唯一编号，完成后结束异步任务 */
  markAsRead: async (id) => {
    const conversation = get().conversations.find((item) => item.id === id) // 会话
    if (!conversation || conversation.status === 'idle' || conversation.status === 'loading') return

    set((state) => ({
      conversations: state.conversations.map((item) =>
        item.id === id ? { ...item, status: 'idle' } : item
      ),
    }))

    await AgentBinding.MarkSessionRead({ session_id: id })
  },

  /** 设置当前会话，接收当前条目的唯一编号，无返回值 */
  setCurrentConversation: (id) => {
    if (id) void get().markAsRead(id)
    if (id && get().messages[id] === undefined) {
      void get().loadMessages(id)
    }
    set({ currentConversationId: id })
  },

  /** 设置会话状态，接收当前条目的唯一编号、状态，返回表达式结果 */
  setConversationStatus: (id, status) =>
    set((state) => ({
      conversations: state.conversations.map((conversation) =>
        conversation.id === id ? { ...conversation, status } : conversation
      ),
    })),

  /** 将用户消息和模型配置提交到后端，并记录本轮生成状态，接收参数，完成后结束异步任务 */
  sendMessage: async (params) => {
    const now = new Date().toISOString() // 当前时间
    const tempUserId = -Date.now() // 临时用户编号
    const tempAssistantId = tempUserId - 1 // 临时助手编号
    const isFirstMessage = (get().messages[params.sessionId]?.length ?? 0) === 0 // 是否第一项消息
    clearStreamingBuffer(params.sessionId)
    clearStreamingSeqs(params.sessionId)

    set((state) => ({
      conversations: state.conversations.map((conversation) =>
        conversation.id === params.sessionId
          ? {
              ...conversation,
              status: 'loading',
              updatedAt: now,
            }
          : conversation
      ),
      messages: {
        ...state.messages,
        [params.sessionId]: [
          ...(state.messages[params.sessionId] ?? []),
          {
            id: tempUserId,
            sessionId: params.sessionId,
            parentId: null,
            role: 'user',
            contentType: 'text',
            content: params.content,
            modelName: '',
            agentName: '',
            tokensIn: 0,
            tokensOut: 0,
            extra: '',
            attachments: params.attachments.length > 0 ? params.attachments : undefined,
            createdAt: now,
          },
          {
            id: tempAssistantId,
            sessionId: params.sessionId,
            parentId: tempUserId,
            role: 'assistant',
            contentType: 'text',
            content: '',
            modelName: params.modelName,
            agentName: '',
            tokensIn: 0,
            tokensOut: 0,
            extra: '',
            createdAt: now,
          },
        ],
      },
      pendingConfirms: {
        ...state.pendingConfirms,
        [params.sessionId]: null,
      },
      sessionStatuses: {
        ...state.sessionStatuses,
        [params.sessionId]: 'loading',
      },
      sessionRunConfigs: {
        ...state.sessionRunConfigs,
        [params.sessionId]: {
          baseUrl: params.baseUrl,
          apiKey: params.apiKey,
          modelName: params.modelName,
          providerType: params.providerType,
        },
      },
      streamingMessages: {
        ...state.streamingMessages,
        [params.sessionId]: EMPTY_STREAMING_MESSAGE,
      },
      pendingTitleGenerations: {
        ...state.pendingTitleGenerations,
        [params.sessionId]: isFirstMessage
          ? {
              baseUrl: params.baseUrl,
              apiKey: params.apiKey,
              modelName: params.modelName,
              providerType: params.providerType,
            }
          : state.pendingTitleGenerations[params.sessionId],
      },
    }))

    try {
      await AgentBinding.SendMessage({
        session_id: params.sessionId,
        content: params.content,
        system_prompt: params.systemPrompt ?? '',
        skill_name: params.skillName ?? '',
        base_url: params.baseUrl,
        api_key: params.apiKey,
        model_name: params.modelName,
        provider_type: params.providerType,
        enabled_user_tools: params.enabledUserTools,
        attachments: params.attachments.map(att => ({
          path: att.path,
          name: att.name,
          mime: att.mime,
        })),
      })
    } catch (err: unknown) {
      const parsed = parseIError(err) // 从文本解析出的数据
      /** 根据当前仓库状态计算并提交更新 */
      set((state) => {
        const nextStreaming = { ...state.streamingMessages } // 下一项流式输出
        delete nextStreaming[params.sessionId]
        return {
          messages: {
            ...state.messages,
            [params.sessionId]: (state.messages[params.sessionId] ?? []).map((m) =>
              m.id === tempAssistantId
                ? { ...m, contentType: 'error' as const, content: JSON.stringify(parsed) }
                : m
            ),
          },
          conversations: state.conversations.map((c) =>
            c.id === params.sessionId ? { ...c, status: 'idle' } : c
          ),
          streamingMessages: nextStreaming,
          sessionStatuses: {
            ...state.sessionStatuses,
            [params.sessionId]: 'idle',
          },
          pendingTitleGenerations: {
            ...state.pendingTitleGenerations,
            [params.sessionId]: undefined,
          },
        }
      })
    }
  },

  /** 停止生成，接收会话编号，完成后结束异步任务 */
  stopGeneration: async (sessionId) => {
    await AgentBinding.StopGeneration({ session_id: sessionId })
  },

  /** 向后端提交工具确认操作和用户说明，接收会话编号、已同意、消息、操作，完成后结束异步任务 */
  respondToConfirm: async (sessionId, approved, message, action = approved ? 'approve' : 'reject') => {
    await AgentBinding.RespondToConfirm({
      session_id: sessionId,
      approved,
      message,
      action,
    })
    set((state) => ({
      pendingConfirms: { ...state.pendingConfirms, [sessionId]: null },
    }))
  },

  /** 生成标题，接收会话编号、服务的基础地址、服务认证使用的 API 密钥、模型名称、模型服务商类型，完成后结束异步任务 */
  generateTitle: async (sessionId, baseUrl, apiKey, modelName, providerType) => {
    /** 保存本次调用的结果 */
    const result = await AgentBinding.GenerateTitle({
      session_id: sessionId,
      base_url: baseUrl,
      api_key: apiKey,
      model_name: modelName,
      provider_type: providerType,
    })
    if (!result?.title) return

    set((state) => ({
      conversations: state.conversations.map((conversation) =>
        conversation.id === sessionId ? { ...conversation, title: result.title } : conversation
      ),
    }))
  },

  /** 生成标题草稿，接收会话编号、回退配置，返回对应的处理结果 */
  generateTitleDraft: async (sessionId, fallbackConfig) => {
    const runConfig = get().sessionRunConfigs[sessionId] ?? fallbackConfig // 运行配置
    if (!runConfig) return null

    /** 保存本次调用的结果 */
    const result = await AgentBinding.GenerateTitle({
      session_id: sessionId,
      base_url: runConfig.baseUrl,
      api_key: runConfig.apiKey,
      model_name: runConfig.modelName,
      provider_type: runConfig.providerType,
    })

    return result?.title?.trim() || null
  },

  /** 订阅会话、流式输出、工具及确认事件，返回一次性清理回调 */
  initEventListeners: () => {
    if ((globalThis as { /* 应用聊天监听器已初始化 */ __lemonteaChatListenersInitialized?: boolean }).__lemonteaChatListenersInitialized) {
      return () => {}
    }
    ;(globalThis as { /* 应用聊天监听器已初始化 */ __lemonteaChatListenersInitialized?: boolean }).__lemonteaChatListenersInitialized = true

    const offs: Array<() => void> = [] // 解绑函数列表

    // 流式正文和思考分片合并到当前会话的帧缓冲区
    offs.push(Events.On('agent:stream:chunk', (event: { /* 流式分片 */ data: StreamChunkEvent | StreamChunkEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue

        ensureStreamingPlaceholder(data.sessionId)
        appendStreamingDelta(data)
      }
    }))

    // 确认请求先进入队列，等待用户选择后再恢复工具调用
    offs.push(Events.On('agent:stream:confirm_request', (event: { /* 确认请求 */ data: ConfirmRequestEvent | ConfirmRequestEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue

        set((state) => ({
          pendingConfirms: {
            ...state.pendingConfirms,
            [data.sessionId]: {
              requestId: data.requestId,
              toolName: data.toolName,
              args: data.args,
              purpose: data.purpose,
            },
          },
        }))
      }
    }))

    offs.push(Events.On('agent:stream:tool_call', (event: { /* 事件携带的数据 */ data: ToolCallEvent | ToolCallEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue

        let parsedArgs: unknown // 解析结果参数
        try {
          parsedArgs = JSON.parse(data.args)
        } catch {
          parsedArgs = {}
        }
        const payload = JSON.stringify({ name: data.toolName, args: parsedArgs }) // 载荷
        const tempId = -Date.now() - (Math.random() * 1000 | 0) // 临时编号

        /** 根据当前仓库状态计算并提交更新 */
        set((state) => {
          const sessionMessages = state.messages[data.sessionId] ?? [] // 会话消息列表
          const insertIndex = sessionMessages.length > 0 ? sessionMessages.length - 1 : 0 // 插入索引

          /** 工具调用消息 */
          const toolCallMessage: Message = {
            id: tempId,
            sessionId: data.sessionId,
            parentId: null,
            role: 'assistant',
            contentType: 'tool_call',
            content: payload,
            modelName: '',
            agentName: '',
            tokensIn: 0,
            tokensOut: 0,
            extra: data.purpose,
            createdAt: new Date().toISOString(),
          }

          const newMessages = [...sessionMessages] // 新消息列表
          newMessages.splice(insertIndex, 0, toolCallMessage)

          return {
            messages: {
              ...state.messages,
              [data.sessionId]: newMessages,
            },
          }
        })
      }
    }))

    offs.push(Events.On('agent:stream:tool_result', (event: { /* 事件携带的数据 */ data: ToolResultEvent | ToolResultEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue

        /** 根据当前仓库状态计算并提交更新 */
        set((state) => {
          const sessionMessages = state.messages[data.sessionId] ?? [] // 会话消息列表
          const insertIndex = sessionMessages.length > 0 ? sessionMessages.length - 1 : 0 // 插入索引
          /** 最新工具结果索引 */
          const latestToolResultIndex = findLatestStreamToolResultIndex(sessionMessages, data.toolName)

          if (latestToolResultIndex >= 0) {
            const nextMessages = [...sessionMessages] // 下一项消息列表
            nextMessages[latestToolResultIndex] = {
              ...nextMessages[latestToolResultIndex],
              content: data.result,
              createdAt: new Date().toISOString(),
            }
            return {
              messages: {
                ...state.messages,
                [data.sessionId]: nextMessages,
              },
            }
          }

          const tempId = -Date.now() - (Math.random() * 1000 | 0) // 临时编号

          /** 工具结果消息 */
          const toolResultMessage: Message = {
            id: tempId,
            sessionId: data.sessionId,
            parentId: null,
            role: 'tool',
            contentType: 'tool_result',
            content: data.result,
            modelName: '',
            agentName: data.toolName,
            tokensIn: 0,
            tokensOut: 0,
            extra: '',
            createdAt: new Date().toISOString(),
          }

          const newMessages = [...sessionMessages] // 新消息列表
          newMessages.splice(insertIndex, 0, toolResultMessage)

          return {
            messages: {
              ...state.messages,
              [data.sessionId]: newMessages,
            },
          }
        })
      }
    }))

    offs.push(Events.On('agent:stream:done', (event: { /* 事件携带的数据 */ data: StreamDoneEvent | StreamDoneEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue

        flushStreamingBuffer(data.sessionId)
        clearStreamingSeqs(data.sessionId)

        /** 根据当前仓库状态计算并提交更新 */
        set((state) => {
          const nextStreaming = { ...state.streamingMessages } // 下一项流式输出
          delete nextStreaming[data.sessionId]
          return {
            streamingMessages: nextStreaming,
            sessionStatuses: {
              ...state.sessionStatuses,
              [data.sessionId]: 'idle',
            },
          }
        })

        void get().loadMessages(data.sessionId)
      }
    }))

    offs.push(Events.On('agent:stream:error', (event: { /* 事件携带的数据 */ data: StreamErrorEvent | StreamErrorEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue

        flushStreamingBuffer(data.sessionId)
        clearStreamingSeqs(data.sessionId)

        /** 根据当前仓库状态计算并提交更新 */
        set((state) => {
          const nextStreaming = { ...state.streamingMessages } // 下一项流式输出
          delete nextStreaming[data.sessionId]
          return {
            streamingMessages: nextStreaming,
            sessionStatuses: {
              ...state.sessionStatuses,
              [data.sessionId]: 'error-unread',
            },
          }
        })

        void get().loadMessages(data.sessionId)
      }
    }))

    // 先显示新会话和占位消息，等后端列表刷新后用完整记录补齐
    offs.push(Events.On('agent:session:spawned', (event: { /* 新会话摘要 */ data: SessionSpawnedEvent | SessionSpawnedEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue
        if (get().conversations.some((c) => c.id === data.sessionId)) continue

        const now = new Date().toISOString() // 当前时间
        const tempUserId = -Date.now() // 临时用户编号
        const tempAssistantId = tempUserId - 1 // 临时助手编号
        const tags = data.tags ?? (data.kind === 'task' ? ['task'] : []) // 标签列表
        /** 新会话 */
        const newConversation: Conversation = {
          id: data.sessionId,
          title: data.title,
          kind: tags.includes('task') ? 'task' : 'user',
          tags,
          createdAt: now,
          updatedAt: now,
          starred: false,
          status: 'loading',
        }

        set((state) => ({
          conversations: [newConversation, ...state.conversations],
          messages: data.userMessage
            ? {
                ...state.messages,
                [data.sessionId]: [
                  {
                    id: tempUserId,
                    sessionId: data.sessionId,
                    parentId: null,
                    role: 'user',
                    contentType: 'text',
                    content: data.userMessage,
                    modelName: '',
                    agentName: '',
                    tokensIn: 0,
                    tokensOut: 0,
                    extra: '',
                    createdAt: now,
                  },
                  {
                    id: tempAssistantId,
                    sessionId: data.sessionId,
                    parentId: tempUserId,
                    role: 'assistant',
                    contentType: 'text',
                    content: '',
                    modelName: '',
                    agentName: '',
                    tokensIn: 0,
                    tokensOut: 0,
                    extra: '',
                    createdAt: now,
                  },
                ],
              }
            : state.messages,
          sessionStatuses: {
            ...state.sessionStatuses,
            [data.sessionId]: 'loading',
          },
        }))

        void get().loadSessions(true)
      }
    }))

    offs.push(Events.On('agent:session:status', (event: { /* 事件携带的数据 */ data: SessionStatusEvent | SessionStatusEvent[] }) => {
      for (/* 数据 */ const data of Array.isArray(event.data) ? event.data : [event.data]) {
        if (!data) continue

        // 重新加载列表以纳入其他窗口创建的会话
        const known = get().conversations.some((c) => c.id === data.sessionId)
        if (!known) {
          void get().loadSessions(true)
          continue
        }

        /** 是否需完成处理当前会话 */
        const shouldFinalizeCurrentConversation =
          data.status === 'done-unread' || data.status === 'error-unread'

        const pendingTitleGeneration = get().pendingTitleGenerations[data.sessionId] // 待处理标题生成
        if (shouldFinalizeCurrentConversation && pendingTitleGeneration) {
          set((state) => ({
            pendingTitleGenerations: {
              ...state.pendingTitleGenerations,
              [data.sessionId]: undefined,
            },
          }))
          void get().generateTitle(
            data.sessionId,
            pendingTitleGeneration.baseUrl,
            pendingTitleGeneration.apiKey,
            pendingTitleGeneration.modelName,
            pendingTitleGeneration.providerType
          )
        }

        if (data.sessionId === get().currentConversationId && shouldFinalizeCurrentConversation) {
          set((state) => ({
            sessionStatuses: {
              ...state.sessionStatuses,
              [data.sessionId]: 'idle',
            },
            conversations: state.conversations.map((conversation) =>
              conversation.id === data.sessionId ? { ...conversation, status: 'idle' } : conversation
            ),
          }))
          void AgentBinding.MarkSessionRead({ session_id: data.sessionId })
          continue
        }

        set((state) => ({
          sessionStatuses: {
            ...state.sessionStatuses,
            [data.sessionId]: data.status,
          },
          conversations: state.conversations.map((conversation) =>
            conversation.id === data.sessionId ? { ...conversation, status: data.status } : conversation
          ),
        }))

        ensureStreamingPlaceholder(data.sessionId)
      }
    }))

    return () => {
      offs.forEach((off) => off())
      ;(globalThis as { /* 应用聊天监听器已初始化 */ __lemonteaChatListenersInitialized?: boolean }).__lemonteaChatListenersInitialized = false
    }
  },
  })
})

/** 查找最新流式输出工具结果索引，接收消息列表、工具名称，返回计算后的数值 */
function findLatestStreamToolResultIndex(messages: Message[], toolName: string): number {
  for (/* 循环索引 */ let i = messages.length - 1; i >= 0; i--) {
    const message = messages[i] // 消息
    if (message.contentType === 'tool_result' && message.agentName === toolName) {
      return i
    }
    if (message.contentType === 'tool_call') {
      break
    }
  }
  return -1
}
