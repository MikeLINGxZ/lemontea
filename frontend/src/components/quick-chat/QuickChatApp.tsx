import { useEffect, useRef, useState } from 'react'
import type { CSSProperties, KeyboardEvent } from 'react'
import { Events, Window as RuntimeWindow } from '@wailsio/runtime'
import { ArrowUp, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Agent as AgentBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/agent'
import { Plugin as PluginBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin'
import { Provider as ProviderBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { cn } from '@/lib/utils'
import { ChatContent } from '@/components/chat/ChatArea'
import { useAppStore } from '@/store/appStore'
import { useChatStore } from '@/store/chatStore'
import type { Conversation } from '@/types'

/** 快捷聊天模式的可选值 */
type QuickChatMode = 'new' | 'continue'

/** 快捷聊天重置事件 */
const QUICK_CHAT_RESET_EVENT = 'quick_chat:reset'
/** 快捷聊天选择器尺寸 */
const QUICK_CHAT_PICKER_SIZE = { width: 720, height: 54 }
/** 快捷聊天会话尺寸 */
const QUICK_CHAT_CONVERSATION_SIZE = { width: 720, height: 520 }

/** 可拖动样式 */
const draggableStyle = {
  '--wails-draggable': 'drag',
  WebkitAppRegion: 'drag',
} as CSSProperties & Record<'--wails-draggable', string>

/** no拖动样式 */
const noDragStyle = {
  '--wails-draggable': 'no-drag',
  WebkitAppRegion: 'no-drag',
} as CSSProperties & Record<'--wails-draggable', string>

/** 最新可继续会话处理，接收会话列表，返回处理结果，无法取得时为空 */
function latestContinuableConversation(conversations: Conversation[]): Conversation | null {
  return conversations.find((conversation) => conversation.kind === 'user') ?? null
}

/** 加载快捷聊天运行配置，返回对应的处理结果 */
async function loadQuickChatRunConfig() {
  const providersResult = await ProviderBinding.ProviderAndModelList({}) // 保存服务商列表结果
  const providerModels = providersResult?.provider_models ?? [] // 模型服务商模型列表
  /** 默认模型服务商 */
  const defaultProvider = providerModels.find(pm => pm.provider.is_default && pm.provider.enabled)
    ?? providerModels.find(pm => pm.provider.enabled)
  /** 默认模型 */
  const defaultModel = defaultProvider?.models.find(model => model.is_default && model.enable)
    ?? defaultProvider?.models.find(model => model.enable)

  if (!defaultProvider || !defaultModel) return null

  const toolsResult = await PluginBinding.ListAvailableTools({}) // 保存工具列表结果
  const enabledToolIds = useAppStore.getState().enabledToolIds // 启用状态工具编号列表
  /** 启用状态用户工具列表 */
  const enabledUserTools = (toolsResult?.tools ?? [])
    .filter(tool => tool.category !== 'builtin' && enabledToolIds.includes(tool.id))
    .map(tool => tool.id)

  return {
    baseUrl: defaultProvider.provider.base_url,
    apiKey: defaultProvider.provider.api_key,
    modelName: defaultModel.model,
    providerType: defaultProvider.provider.provider_type,
    enabledUserTools,
  }
}

/** 管理快捷聊天的会话选择、窗口尺寸和消息发送，返回组件内容 */
export function QuickChatApp() {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const loadSessions = useChatStore((state) => state.loadSessions) // 按游标加载会话列表，重置时从第一页开始
  const conversations = useChatStore((state) => state.conversations) // 从状态仓库读取会话列表
  const sendMessage = useChatStore((state) => state.sendMessage) // 将用户消息和模型配置提交到后端，并记录本轮生成状态
  const [selectedMode, setSelectedMode] = useState<QuickChatMode>('continue') // 选中项模式及其更新方法
  const [message, setMessage] = useState('') // 消息及其更新方法
  const [confirming, setConfirming] = useState(false) // 正在确认及其更新方法
  const [activeSessionId, setActiveSessionId] = useState<number | null>(null) // 当前会话编号及其更新方法
  const inputRef = useRef<HTMLInputElement>(null) // 输入引用，供后续读取当前值
  const userSelectedRef = useRef(false) // 用户选中项引用，供后续读取当前值
  const latestConversation = latestContinuableConversation(conversations) // 最新会话
  const canContinue = latestConversation !== null // 能否继续
  /** 当前会话标题 */
  const activeConversationTitle = activeSessionId
    ? conversations.find((conversation) => conversation.id === activeSessionId)?.title
    : undefined
  const quickChatTitle = activeConversationTitle?.trim() || t('quickChat.title') // 快捷聊天标题

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    void loadSessions(true)
  }, [loadSessions])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    void RuntimeWindow.SetSize(QUICK_CHAT_PICKER_SIZE.width, QUICK_CHAT_PICKER_SIZE.height).catch(() => {})
    void RuntimeWindow.Center().catch(() => {})
  }, [])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 处理窗口键向下，接收触发本次回调的事件，无返回值 */
    const handleWindowKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key !== 'Escape') return
      event.preventDefault()
      void Window.CloseQuickChat({})
    }

    window.addEventListener('keydown', handleWindowKeyDown)
    return () => window.removeEventListener('keydown', handleWindowKeyDown)
  }, [])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 保存事件解绑方法 */
    const offReset = Events.On(QUICK_CHAT_RESET_EVENT, () => {
      userSelectedRef.current = false
      setActiveSessionId(null)
      setMessage('')
      setConfirming(false)
      setSelectedMode(canContinue ? 'continue' : 'new')
      void loadSessions(true)
      window.setTimeout(() => inputRef.current?.focus(), 0)
    })

    return () => {
      if (typeof offReset === 'function') {
        offReset()
      }
    }
  }, [canContinue, loadSessions])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (userSelectedRef.current) return
    setSelectedMode(canContinue ? 'continue' : 'new')
  }, [canContinue])

  /** 选择模式，接收模式，无返回值 */
  const selectMode = (mode: QuickChatMode) => {
    if (mode === 'continue' && !canContinue) return
    userSelectedRef.current = true
    setSelectedMode(mode)
    inputRef.current?.focus()
  }

  /** 解析会话编号，返回对应的处理结果 */
  const resolveSessionId = async () => {
    if (activeSessionId) {
      return activeSessionId
    }

    if (selectedMode === 'continue' && latestConversation) {
      return latestConversation.id
    }

    const result = await AgentBinding.CreateSession({ title: '', tags: [] }) // 保存本次调用的结果
    return result?.session_id ?? 0
  }

  /** 确认选区，完成后结束异步任务 */
  const confirmSelection = async () => {
    if (confirming) return
    const content = message.trim() // 内容
    if (!content) return

    setConfirming(true)
    try {
      const sessionId = await resolveSessionId() // 保存会话编号
      if (sessionId <= 0) return

      const runConfig = await loadQuickChatRunConfig() // 保存运行配置
      if (!runConfig) return

      setActiveSessionId(sessionId)
      setMessage('')
      await RuntimeWindow.SetSize(QUICK_CHAT_CONVERSATION_SIZE.width, QUICK_CHAT_CONVERSATION_SIZE.height).catch(() => {})
      await RuntimeWindow.Center().catch(() => {})
      await sendMessage({
        sessionId,
        content,
        baseUrl: runConfig.baseUrl,
        apiKey: runConfig.apiKey,
        modelName: runConfig.modelName,
        providerType: runConfig.providerType,
        enabledUserTools: runConfig.enabledUserTools,
        attachments: [],
      })
    } finally {
      setConfirming(false)
      inputRef.current?.focus()
    }
  }

  /** 处理键向下，接收触发本次回调的事件，无返回值 */
  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (activeSessionId) {
      return
    }

    if (event.key === 'Tab') {
      event.preventDefault()
      selectMode(selectedMode === 'continue' ? 'new' : (canContinue ? 'continue' : 'new'))
      return
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      void confirmSelection()
      return
    }
  }

  return (
    <div
      className="flex h-screen w-screen flex-col overflow-hidden bg-transparent text-foreground"
      onKeyDown={handleKeyDown}
    >
      {activeSessionId ? (
        <div
          data-testid="quick-chat-conversation"
          style={draggableStyle}
          className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border/80 bg-background/95 shadow-xl shadow-black/15 backdrop-blur-xl"
        >
          <div
            data-testid="quick-chat-titlebar"
            className="flex h-12 shrink-0 items-center justify-center border-b border-border/70 px-4"
          >
            <h1 className="max-w-full truncate text-sm font-semibold text-foreground">
              {quickChatTitle}
            </h1>
          </div>
          <ChatContent sessionId={activeSessionId} variant="quick" />
        </div>
      ) : (
        <section
          data-testid="quick-chat-shell"
          style={draggableStyle}
          aria-label={t('quickChat.modeLabel')}
          className="grid h-full w-full shrink-0 grid-cols-[auto_auto_auto_1fr_auto] items-center rounded-lg border border-border/80 bg-background/95 px-3 shadow-xl shadow-black/15 backdrop-blur-xl"
        >
          <button
            type="button"
            aria-pressed={selectedMode === 'new'}
            style={noDragStyle}
            onClick={() => selectMode('new')}
            className={cn(
              'h-7 rounded-md px-2.5 text-sm font-semibold outline-hidden transition-colors focus-visible:ring-2 focus-visible:ring-ring',
              selectedMode === 'new' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {t('quickChat.newCompact')}
          </button>
          <span className="px-1 text-border" aria-hidden="true">|</span>
          <button
            type="button"
            aria-pressed={selectedMode === 'continue'}
            disabled={!canContinue}
            style={noDragStyle}
            onClick={() => selectMode('continue')}
            className={cn(
              'h-7 rounded-md px-2.5 text-sm font-semibold outline-hidden transition-colors focus-visible:ring-2 focus-visible:ring-ring',
              selectedMode === 'continue' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground',
              !canContinue && 'cursor-not-allowed opacity-45 hover:text-muted-foreground',
            )}
          >
            {t('quickChat.continue')}
          </button>

          <input
            ref={inputRef}
            type="text"
            aria-label={t('quickChat.inputLabel')}
            value={message}
            disabled={confirming}
            style={noDragStyle}
            onChange={(event) => setMessage(event.target.value)}
            placeholder={t(selectedMode === 'continue' && canContinue
              ? 'quickChat.continuePlaceholder'
              : 'quickChat.newPlaceholder')}
            className="ml-3 min-w-0 bg-transparent text-base text-foreground outline-hidden placeholder:text-muted-foreground"
          />

          <button
            type="button"
            disabled={confirming}
            style={noDragStyle}
            onClick={() => { void confirmSelection() }}
            className="ml-3 inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-wait disabled:opacity-70"
            aria-label={confirming ? t('quickChat.opening') : t('quickChat.open')}
          >
            {confirming ? <Loader2 size={16} className="animate-spin" /> : <ArrowUp size={16} />}
          </button>
        </section>
      )}
    </div>
  )
}
