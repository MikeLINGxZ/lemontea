import { memo, useEffect, useMemo } from 'react'
import type { CSSProperties } from 'react'
import { ChevronDown, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn, shouldShowTimestamp, formatTimestamp, groupMessagesForDisplay } from '@/lib/utils'
import { useAutoScroll } from '@/hooks/useAutoScroll'
import { useAppStore } from '@/store/appStore'
import { useChatStore } from '@/store/chatStore'
import { useShallow } from 'zustand/react/shallow'
import { MessageItem } from './MessageItem'
import { TerminalSessionPanels } from './TerminalSessionPanels'
import { WelcomeScreen } from './WelcomeScreen'
import type { Message, DisplayMessage } from '@/types'

const EMPTY_MESSAGES: DisplayMessage[] = []

const quickNoDragStyle = {
  '--wails-draggable': 'no-drag',
  WebkitAppRegion: 'no-drag',
} as CSSProperties & Record<'--wails-draggable', string>

interface MessageGroupProps {
  language: string
  messages: DisplayMessage[]
}

const HistoricalMessages = memo(function HistoricalMessages({
  language,
  messages,
}: MessageGroupProps) {
  return (
    <>
      {messages.map((message, index) => (
        <div key={message.id}>
          {shouldShowTimestamp(messages, index) && (
            <div className="text-center py-2">
              <span className="text-xs text-muted-foreground px-2">
                {formatTimestamp(message.createdAt, language)}
              </span>
            </div>
          )}
          <MessageItem message={message} />
        </div>
      ))}
    </>
  )
})

HistoricalMessages.displayName = 'HistoricalMessages'

const StreamingMessageRow = memo(function StreamingMessageRow({
  language,
  message,
  sessionId,
  showTimestamp,
  toolMessages,
}: {
  language: string
  message: Message
  sessionId: number
  showTimestamp: boolean
  toolMessages: DisplayMessage[]
}) {
  const { streamingState, pendingConfirm, respondToConfirm } = useChatStore(useShallow((state) => ({
    streamingState: state.streamingMessages[sessionId],
    pendingConfirm: state.pendingConfirms[sessionId] ?? null,
    respondToConfirm: state.respondToConfirm,
  })))

  return (
    <div>
      {showTimestamp && (
        <div className="text-center py-2">
          <span className="text-xs text-muted-foreground px-2">
            {formatTimestamp(message.createdAt, language)}
          </span>
        </div>
      )}
      <MessageItem
        message={message}
        isStreaming={true}
        streamingContent={streamingState?.content}
        streamingThinking={streamingState?.thinking}
        toolMessages={toolMessages}
        pendingConfirm={pendingConfirm}
        onConfirm={(content) => {
          void respondToConfirm(sessionId, true, content, 'approve')
        }}
        onReject={(content) => {
          void respondToConfirm(sessionId, false, content, 'reject')
        }}
        onSubmitComment={(content) => {
          void respondToConfirm(sessionId, false, content, 'comment')
        }}
      />
    </div>
  )
})

StreamingMessageRow.displayName = 'StreamingMessageRow'

type ChatMessagesProps = {
  sessionId?: number
  variant?: 'default' | 'quick'
}

export function ChatMessages({ sessionId, variant = 'default' }: ChatMessagesProps) {
  const { t } = useTranslation()
  const language = useAppStore(s => s.language)
  const isQuick = variant === 'quick'
  const {
    activeSessionId,
    currentMessages,
    isStreaming,
    streamingContent,
    streamingThinking,
    initEventListeners,
  } = useChatStore(useShallow((state) => {
    const resolvedSessionId = sessionId ?? state.currentConversationId
    const conversation = resolvedSessionId
      ? state.conversations.find(item => item.id === resolvedSessionId)
      : undefined
    const streamingState = resolvedSessionId
      ? state.streamingMessages[resolvedSessionId]
      : undefined
    const sessionStatus = resolvedSessionId
      ? (state.sessionStatuses[resolvedSessionId] ?? conversation?.status)
      : undefined

    return {
      activeSessionId: resolvedSessionId,
      currentMessages: resolvedSessionId
        ? (state.messages[resolvedSessionId] ?? EMPTY_MESSAGES)
        : EMPTY_MESSAGES,
      isStreaming: sessionStatus === 'loading' || sessionStatus === 'waiting-unread',
      streamingContent: streamingState?.content,
      streamingThinking: streamingState?.thinking,
      initEventListeners: state.initEventListeners,
    }
  }))

  const latestUserMessage = useMemo(
    () => [...currentMessages].reverse().find(
      message => message.role === 'user' && message.contentType === 'text'
    ),
    [currentMessages]
  )

  const lastMessage = currentMessages[currentMessages.length - 1]
  const streamingMessage =
    isStreaming && lastMessage?.role === 'assistant' ? lastMessage : undefined
  const shouldShowStreamingTimestamp = Boolean(
    streamingMessage && shouldShowTimestamp(currentMessages, currentMessages.length - 1)
  )

  const historicalMessages = useMemo(
    () => groupMessagesForDisplay(streamingMessage ? currentMessages.slice(0, -1) : currentMessages),
    [currentMessages, streamingMessage]
  )

  const currentTurnToolIds = useMemo(() => {
    if (!isStreaming) return new Set<number>()
    const slice = currentMessages.slice(0, -1)
    let lastUserIdx = -1
    for (let i = slice.length - 1; i >= 0; i--) {
      if (slice[i].role === 'user' && slice[i].contentType === 'text') {
        lastUserIdx = i
        break
      }
    }
    if (lastUserIdx === -1) return new Set<number>()
    return new Set(
      slice.slice(lastUserIdx + 1)
        .filter(m => m.contentType === 'tool_call')
        .map(m => m.id)
    )
  }, [isStreaming, currentMessages])

  const streamToolMessages = useMemo(
    () => isStreaming ? historicalMessages.filter(m => m.contentType === 'tool_call' && currentTurnToolIds.has(m.id)) : EMPTY_MESSAGES,
    [historicalMessages, isStreaming, currentTurnToolIds]
  )

  const otherHistoricalMessages = useMemo(
    () => isStreaming ? historicalMessages.filter(m => m.contentType !== 'tool_call' || !currentTurnToolIds.has(m.id)) : historicalMessages,
    [historicalMessages, isStreaming, currentTurnToolIds]
  )

  const { containerRef, isAtBottom, scrollToBottom } = useAutoScroll([
    currentMessages.length,
    isStreaming,
    streamingContent,
    streamingThinking,
  ], latestUserMessage?.id)

  const isNewChat = currentMessages.length === 0

  useEffect(() => {
    const frame = requestAnimationFrame(() => scrollToBottom(false))

    return () => cancelAnimationFrame(frame)
  }, [activeSessionId, scrollToBottom])

  useEffect(() => initEventListeners(), [initEventListeners])

  return (
    <div
      data-testid={isQuick ? 'quick-chat-messages' : undefined}
      style={isQuick ? quickNoDragStyle : undefined}
      className="relative flex flex-col flex-1 min-h-0"
    >
      <div
        ref={containerRef}
        className={cn('chat-scroll-area flex-1 overflow-y-auto', isQuick ? 'py-3' : 'py-4')}
      >
        {isNewChat ? (
          isQuick ? null : (
            <div className="mx-auto flex min-h-full w-full max-w-5xl items-center justify-center px-4">
              <WelcomeScreen />
            </div>
          )
        ) : (
          <div className={cn(
            'mx-auto flex w-full flex-col',
            isQuick ? 'max-w-none gap-2 px-3' : 'max-w-5xl gap-2.5 px-4',
          )}>
            <HistoricalMessages language={language} messages={otherHistoricalMessages} />

            {streamingMessage && activeSessionId && (
              <StreamingMessageRow
                language={language}
                message={streamingMessage}
                sessionId={activeSessionId}
                showTimestamp={shouldShowStreamingTimestamp}
                toolMessages={streamToolMessages}
              />
            )}

            <TerminalSessionPanels sessionId={activeSessionId ?? null} />

            {isStreaming && (
              <div className="flex items-center gap-2 px-4 text-muted-foreground">
                <Loader2 size={14} className="animate-spin" />
              </div>
            )}
          </div>
        )}
      </div>

      {!isAtBottom && (
        <button
          onClick={() => scrollToBottom(true)}
          className={cn(
            'absolute left-1/2 -translate-x-1/2 z-10',
            isQuick ? 'bottom-2' : 'bottom-4',
            'flex items-center gap-1.5 px-3 py-1.5 rounded-full',
            'bg-background border border-border shadow-md',
            'text-xs text-foreground hover:bg-accent transition-colors'
          )}
        >
          <ChevronDown size={14} />
          {t('chat.scrollToBottom')}
        </button>
      )}
    </div>
  )
}
