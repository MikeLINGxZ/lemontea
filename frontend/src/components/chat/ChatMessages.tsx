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

/** 空消息列表 */
const EMPTY_MESSAGES: DisplayMessage[] = []

/** 快捷no拖动样式 */
const quickNoDragStyle = {
  '--wails-draggable': 'no-drag',
  WebkitAppRegion: 'no-drag',
} as CSSProperties & Record<'--wails-draggable', string>

/** 消息分组组件接收的属性 */
interface MessageGroupProps {
  /** 语言 */
  language: string
  /** 消息列表 */
  messages: DisplayMessage[]
}

/** 按稳定标识渲染已完成的历史消息，接收语言、消息列表，返回组件内容 */
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

/** 渲染当前会话正在生成的内容和思考文本，接收语言、消息、会话编号、显示时间戳、工具消息列表，返回组件内容 */
const StreamingMessageRow = memo(function StreamingMessageRow({
  language,
  message,
  sessionId,
  showTimestamp,
  toolMessages,
}: {
  /** 语言 */
  language: string
  /** 消息 */
  message: Message
  /** 会话编号 */
  sessionId: number
  /** 显示时间戳 */
  showTimestamp: boolean
  /** 工具消息列表 */
  toolMessages: DisplayMessage[]
}) {
  /** 从状态仓库读取流式输出状态、待处理确认、响应终点确认 */
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

/** 聊天消息列表组件接收的属性 */
type ChatMessagesProps = {
  /** 会话编号 */
  sessionId?: number
  /** 展示变体 */
  variant?: 'default' | 'quick'
}

/** 组装历史消息、流式消息和工具输出，并在用户停留底部时自动滚动，接收会话编号、展示变体，返回组件内容 */
export function ChatMessages({ sessionId, variant = 'default' }: ChatMessagesProps) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const language = useAppStore(s => s.language) // 从状态仓库读取语言
  const isQuick = variant === 'quick' // 是否快捷
  /** 从状态仓库读取当前会话编号、当前消息列表、是否流式输出、流式输出内容、流式输出思考内容、初始化事件监听器 */
  const {
    activeSessionId,
    currentMessages,
    isStreaming,
    streamingContent,
    streamingThinking,
    initEventListeners,
  } = useChatStore(useShallow((state) => {
    const resolvedSessionId = sessionId ?? state.currentConversationId // 已处理会话编号
    /** 会话 */
    const conversation = resolvedSessionId
      ? state.conversations.find(item => item.id === resolvedSessionId)
      : undefined
    /** 流式输出状态 */
    const streamingState = resolvedSessionId
      ? state.streamingMessages[resolvedSessionId]
      : undefined
    /** 会话状态 */
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

  /** 最新用户消息 */
  const latestUserMessage = useMemo(
    () => [...currentMessages].reverse().find(
      message => message.role === 'user' && message.contentType === 'text'
    ),
    [currentMessages]
  )

  const lastMessage = currentMessages[currentMessages.length - 1] // 最后一个消息
  /** 流式输出消息 */
  const streamingMessage =
    isStreaming && lastMessage?.role === 'assistant' ? lastMessage : undefined
  /** 是否需显示流式输出时间戳 */
  const shouldShowStreamingTimestamp = Boolean(
    streamingMessage && shouldShowTimestamp(currentMessages, currentMessages.length - 1)
  )

  /** 将末尾的流式占位移出历史消息，避免同一条回复被渲染两次 */
  const historicalMessages = useMemo(
    () => groupMessagesForDisplay(streamingMessage ? currentMessages.slice(0, -1) : currentMessages),
    [currentMessages, streamingMessage]
  )

  /** 按最后一条用户消息划分轮次，记录本轮正在输出的工具调用 */
  const currentTurnToolIds = useMemo(() => {
    if (!isStreaming) return new Set<number>()
    const slice = currentMessages.slice(0, -1) // 截取后的消息列表
    let lastUserIdx = -1 // 最后一个用户索引
    for (/* 循环索引 */ let i = slice.length - 1; i >= 0; i--) {
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

  /** 流式输出工具消息列表 */
  const streamToolMessages = useMemo(
    () => isStreaming ? historicalMessages.filter(m => m.contentType === 'tool_call' && currentTurnToolIds.has(m.id)) : EMPTY_MESSAGES,
    [historicalMessages, isStreaming, currentTurnToolIds]
  )

  /** 其余历史项消息列表 */
  const otherHistoricalMessages = useMemo(
    () => isStreaming ? historicalMessages.filter(m => m.contentType !== 'tool_call' || !currentTurnToolIds.has(m.id)) : historicalMessages,
    [historicalMessages, isStreaming, currentTurnToolIds]
  )

  /** 使用消息内容变化驱动滚动，并在收到下一轮用户输入时恢复跟随 */
  const { containerRef, isAtBottom, scrollToBottom } = useAutoScroll([
    currentMessages.length,
    isStreaming,
    streamingContent,
    streamingThinking,
  ], latestUserMessage?.id)

  const isNewChat = currentMessages.length === 0 // 是否新聊天

  /** 会话切换后等待布局完成，再将新会话滚动到底部 */
  useEffect(() => {
    const frame = requestAnimationFrame(() => scrollToBottom(false)) // 帧

    return () => cancelAnimationFrame(frame)
  }, [activeSessionId, scrollToBottom])

  /** 进入聊天时订阅后端事件，卸载时解除订阅 */
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
