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

type QuickChatMode = 'new' | 'continue'

const QUICK_CHAT_RESET_EVENT = 'quick_chat:reset'
const QUICK_CHAT_PICKER_SIZE = { width: 720, height: 54 }
const QUICK_CHAT_CONVERSATION_SIZE = { width: 720, height: 520 }

const draggableStyle = {
  '--wails-draggable': 'drag',
  WebkitAppRegion: 'drag',
} as CSSProperties & Record<'--wails-draggable', string>

const noDragStyle = {
  '--wails-draggable': 'no-drag',
  WebkitAppRegion: 'no-drag',
} as CSSProperties & Record<'--wails-draggable', string>

function latestContinuableConversation(conversations: Conversation[]): Conversation | null {
  return conversations.find((conversation) => conversation.kind === 'user') ?? null
}

async function loadQuickChatRunConfig() {
  const providersResult = await ProviderBinding.ProviderAndModelList({})
  const providerModels = providersResult?.provider_models ?? []
  const defaultProvider = providerModels.find(pm => pm.provider.is_default && pm.provider.enabled)
    ?? providerModels.find(pm => pm.provider.enabled)
  const defaultModel = defaultProvider?.models.find(model => model.is_default && model.enable)
    ?? defaultProvider?.models.find(model => model.enable)

  if (!defaultProvider || !defaultModel) return null

  const toolsResult = await PluginBinding.ListAvailableTools({})
  const enabledToolIds = useAppStore.getState().enabledToolIds
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

export function QuickChatApp() {
  const { t } = useTranslation()
  const loadSessions = useChatStore((state) => state.loadSessions)
  const conversations = useChatStore((state) => state.conversations)
  const sendMessage = useChatStore((state) => state.sendMessage)
  const [selectedMode, setSelectedMode] = useState<QuickChatMode>('continue')
  const [message, setMessage] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [activeSessionId, setActiveSessionId] = useState<number | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const userSelectedRef = useRef(false)
  const latestConversation = latestContinuableConversation(conversations)
  const canContinue = latestConversation !== null
  const activeConversationTitle = activeSessionId
    ? conversations.find((conversation) => conversation.id === activeSessionId)?.title
    : undefined
  const quickChatTitle = activeConversationTitle?.trim() || t('quickChat.title')

  useEffect(() => {
    void loadSessions(true)
  }, [loadSessions])

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    void RuntimeWindow.SetSize(QUICK_CHAT_PICKER_SIZE.width, QUICK_CHAT_PICKER_SIZE.height).catch(() => {})
    void RuntimeWindow.Center().catch(() => {})
  }, [])

  useEffect(() => {
    const handleWindowKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key !== 'Escape') return
      event.preventDefault()
      void Window.CloseQuickChat({})
    }

    window.addEventListener('keydown', handleWindowKeyDown)
    return () => window.removeEventListener('keydown', handleWindowKeyDown)
  }, [])

  useEffect(() => {
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

  useEffect(() => {
    if (userSelectedRef.current) return
    setSelectedMode(canContinue ? 'continue' : 'new')
  }, [canContinue])

  const selectMode = (mode: QuickChatMode) => {
    if (mode === 'continue' && !canContinue) return
    userSelectedRef.current = true
    setSelectedMode(mode)
    inputRef.current?.focus()
  }

  const resolveSessionId = async () => {
    if (activeSessionId) {
      return activeSessionId
    }

    if (selectedMode === 'continue' && latestConversation) {
      return latestConversation.id
    }

    const result = await AgentBinding.CreateSession({ title: '', tags: [] })
    return result?.session_id ?? 0
  }

  const confirmSelection = async () => {
    if (confirming) return
    const content = message.trim()
    if (!content) return

    setConfirming(true)
    try {
      const sessionId = await resolveSessionId()
      if (sessionId <= 0) return

      const runConfig = await loadQuickChatRunConfig()
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
              'h-7 rounded-md px-2.5 text-sm font-semibold outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring',
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
              'h-7 rounded-md px-2.5 text-sm font-semibold outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring',
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
            className="ml-3 min-w-0 bg-transparent text-base text-foreground outline-none placeholder:text-muted-foreground"
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
