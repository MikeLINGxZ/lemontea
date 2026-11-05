import { isObjectRecord } from '@/lib/guards'
import { memo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import 'highlight.js/styles/github-dark-dimmed.min.css'
import { ThinkingBlock } from './ThinkingBlock'
import { ToolCallBlock } from './ToolCallBlock'
import { ToolCallsGroup } from './ToolCallsGroup'
import { ToolConfirmCard } from './ToolConfirmCard'
import { AttachmentChips } from './AttachmentChips'
import { ErrorMessageBubble } from './ErrorMessageBubble'
import { ResponseMeta } from './ResponseMeta'
import { InteractiveTerminalBlock } from '@/components/chat/InteractiveTerminalBlock'
import { File as FileBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file'
import type { DisplayMessage } from '@/types'
import { cn } from '@/lib/utils'
import {
  getInteractiveTerminalState,
  isInteractiveTerminalControl,
  type InteractiveTerminalState,
} from '@/lib/terminalOutput'

/** 消息条目组件接收的属性 */
interface Props {
  /** 消息 */
  message: DisplayMessage
  /** 是否流式输出 */
  isStreaming?: boolean
  /** 流式输出内容 */
  streamingContent?: string
  /** 流式输出思考内容 */
  streamingThinking?: string
  /** 工具消息列表 */
  toolMessages?: DisplayMessage[]
  /** 待处理确认 */
  pendingConfirm?: {
    /** 请求编号 */
    requestId: string
    /** 工具名称 */
    toolName: string
    /** 参数 */
    args: string
    /** 用途 */
    purpose: string
  } | null
  /** 处理确认事件，接收消息，无返回值 */
  onConfirm?: (message: string) => void
  /** 处理拒绝事件，接收消息，无返回值 */
  onReject?: (message: string) => void
  /** 处理提交备注事件，接收消息，无返回值 */
  onSubmitComment?: (message: string) => void
}

/** 按消息角色和内容类型渲染正文、工具结果、终端或异常，接收组件属性，返回组件内容 */
export const MessageItem = memo(function MessageItem({
  message,
  isStreaming,
  streamingContent,
  streamingThinking,
  toolMessages,
  pendingConfirm,
  onConfirm,
  onReject,
  onSubmitComment,
}: Props) {
  /** 取出消息发送方的角色、消息内容类型、内容、模型名称、输入 Token 数、输出 Token 数、附加数据 */
  const { role, contentType, content, modelName, tokensIn, tokensOut, extra } = message
  /** 打开附件，接收文件路径，无返回值 */
  const openAttachment = (path: string) => {
    void FileBinding.OpenFile({ path })
  }

  if (role === 'user' && contentType === 'text') {
    const attachments = message.attachments ?? [] // 附件列表

    return (
      <div className="flex justify-end">
        <div className="max-w-[70%] select-text rounded-2xl rounded-br-sm bg-primary text-primary-foreground px-4 py-2.5 text-sm dark:bg-muted dark:text-foreground">
          {content && <div className="whitespace-pre-wrap break-words">{content}</div>}
          {attachments.length > 0 && (
            <AttachmentChips
              items={attachments}
              variant="message"
              onOpen={(attachment) => openAttachment(attachment.path)}
            />
          )}
        </div>
      </div>
    )
  }

  if (role === 'user' && contentType === 'confirm_response') {
    return null
  }

  if (message.isToolGroup && message.groupedTools) {
    return <ToolCallsGroup tools={message.groupedTools} />
  }

  if (contentType === 'tool_call') {
    let toolName = '' // 工具名称
    let args = content // 参数
    /** 状态 */
    let status: 'completed' | 'failed' | 'executing' | 'user_commented' | 'user_rejected' = 'completed'
    try {
      const parsed: unknown = JSON.parse(content) // 保存待校验的 JSON 解析结果
      toolName = isObjectRecord(parsed) && typeof parsed.name === 'string' ? parsed.name : ''
      args = JSON.stringify(isObjectRecord(parsed) ? parsed.args ?? parsed : parsed, null, 2)
    } catch {
      // 无法解析工具调用时沿用原始文本
    }

    if (message.toolConfirmAction === 'comment') {
      status = 'user_commented'
    } else if (message.toolConfirmAction === 'reject') {
      status = 'user_rejected'
    }

    /** 终端状态 */
    const terminalState = message.toolResult ? getInteractiveTerminalState(message.toolResult) : null
    if (terminalState) {
      if (!terminalState.visible) return null
      return (
        <InteractiveTerminalBlock
          output={terminalState.output}
          active={terminalState.active || Boolean(isStreaming)}
        />
      )
    }

    return (
      <ToolCallBlock
        toolName={toolName}
        purpose={extra}
        args={args}
        result={message.toolResult ?? ''}
        userComment={message.toolConfirmComment}
        status={status}
      />
    )
  }

  if (contentType === 'thinking') {
    return <ThinkingBlock content={content} defaultExpanded={false} active={false} />
  }

  if (contentType === 'error') {
    return (
      <ErrorMessageBubble
        content={content}
        modelName={modelName}
        tokensIn={tokensIn}
        tokensOut={tokensOut}
      />
    )
  }

  /** 显示内容 */
  const displayContent = isStreaming && streamingContent !== undefined ? streamingContent : content
  /** 显示思考内容 */
  const displayThinking = isStreaming && streamingThinking !== undefined ? streamingThinking : ''
  const shouldRenderMarkdown = Boolean(displayContent) // 是否需渲染 Markdown
  const flattenedToolMessages = flattenToolMessages(toolMessages ?? []) // 展平工具消息列表
  /** 终端状态 */
  const terminalState = isStreaming ? latestInteractiveTerminalState(flattenedToolMessages) : null
  /** 终端控制编号列表 */
  const terminalControlIds = new Set(
    flattenedToolMessages
      .filter((toolMsg) => isInteractiveTerminalControl(toolMsg.toolResult ?? ''))
      .map((toolMsg) => toolMsg.id)
  )
  /** 非交互式工具消息列表 */
  const nonInteractiveToolMessages = flattenedToolMessages.filter((toolMsg) => !terminalControlIds.has(toolMsg.id))

  return (
    <div className="flex flex-col gap-0.5">
      {displayThinking && (
        <ThinkingBlock
          content={displayThinking}
          defaultExpanded={Boolean(isStreaming && !displayContent)}
          active={Boolean(isStreaming)}
        />
      )}

      {terminalState?.visible && (
        <InteractiveTerminalBlock
          output={terminalState.output}
          active={terminalState.active || Boolean(isStreaming)}
        />
      )}

      {nonInteractiveToolMessages.length > 0 && (
        <div className="flex flex-col gap-0.5">
          {nonInteractiveToolMessages.length === 1 ? (
            <MessageItem message={nonInteractiveToolMessages[0]} />
          ) : (
            <ToolCallsGroup tools={nonInteractiveToolMessages} />
          )}
        </div>
      )}

      {displayContent && (
        <div className="llm-markdown select-text text-sm text-foreground leading-relaxed">
          {shouldRenderMarkdown ? (
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeHighlight]}
              components={{
                /** 当前条目处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                p: ({ className, children }) => (
                  <p className={cn('llm-markdown-paragraph', className)}>{children}</p>
                ),
                /** h1处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                h1: ({ className, children }) => (
                  <h1 className={cn('llm-markdown-heading llm-markdown-h1', className)}>{children}</h1>
                ),
                /** h2处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                h2: ({ className, children }) => (
                  <h2 className={cn('llm-markdown-heading llm-markdown-h2', className)}>{children}</h2>
                ),
                /** h3处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                h3: ({ className, children }) => (
                  <h3 className={cn('llm-markdown-heading llm-markdown-h3', className)}>{children}</h3>
                ),
                /** h4处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                h4: ({ className, children }) => (
                  <h4 className={cn('llm-markdown-heading llm-markdown-h4', className)}>{children}</h4>
                ),
                /** ul处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                ul: ({ className, children }) => (
                  <ul className={cn('llm-markdown-list llm-markdown-list-disc', className)}>{children}</ul>
                ),
                /** ol处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                ol: ({ className, children }) => (
                  <ol className={cn('llm-markdown-list llm-markdown-list-decimal', className)}>{children}</ol>
                ),
                /** li处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                li: ({ className, children }) => (
                  <li className={cn('llm-markdown-list-item', className)}>{children}</li>
                ),
                /** 引用块处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                blockquote: ({ className, children }) => (
                  <blockquote className={cn('llm-markdown-blockquote', className)}>{children}</blockquote>
                ),
                /** hr处理，接收类名名称，返回表达式结果 */
                hr: ({ className }) => <hr className={cn('llm-markdown-divider', className)} />,
                /** 比较的第一项处理，接收类名名称、要渲染的子内容、组件输入属性，返回表达式结果 */
                a: ({ className, children, ...props }) => (
                  <a
                    className={cn('llm-markdown-link', className)}
                    target="_blank"
                    rel="noreferrer"
                    {...props}
                  >
                    {children}
                  </a>
                ),
                /** 表格处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                table: ({ className, children }) => (
                  <div className="llm-markdown-table">
                    <table className={cn(className)}>{children}</table>
                  </div>
                ),
                /** 表头处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                thead: ({ className, children }) => (
                  <thead className={cn('llm-markdown-thead', className)}>{children}</thead>
                ),
                /** th处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                th: ({ className, children }) => (
                  <th className={cn('llm-markdown-th', className)}>{children}</th>
                ),
                /** td处理，接收类名名称、要渲染的子内容，返回表达式结果 */
                td: ({ className, children }) => (
                  <td className={cn('llm-markdown-td', className)}>{children}</td>
                ),
                /** 预格式化文本处理，接收要渲染的子内容，返回表达式结果 */
                pre: ({ children }) => (
                  <pre className="llm-markdown-pre">{children}</pre>
                ),
                /** 代码处理，接收要渲染的子内容、类名名称，返回对应的处理结果 */
                code: ({ children, className }) => {
                  const isBlock = className?.includes('language-') // 是否区块
                  return isBlock ? (
                    <code className={className}>{children}</code>
                  ) : (
                    <code className="llm-inline-code">{children}</code>
                  )
                },
              }}
            >
              {displayContent}
            </ReactMarkdown>
          ) : (
            <div className="whitespace-pre-wrap break-words">
              {displayContent}
            </div>
          )}
        </div>
      )}

      {pendingConfirm && onConfirm && onReject && onSubmitComment && (
        <ToolConfirmCard
          toolName={pendingConfirm.toolName}
          purpose={pendingConfirm.purpose}
          args={pendingConfirm.args}
          onConfirm={onConfirm}
          onReject={onReject}
          onSubmitComment={onSubmitComment}
          resolved={false}
          approved={false}
        />
      )}

      {!isStreaming && (
        <ResponseMeta modelName={modelName} tokensIn={tokensIn} tokensOut={tokensOut} />
      )}
    </div>
  )
})

MessageItem.displayName = 'MessageItem'

/** 展开工具消息列表处理，接收工具消息列表，返回对应的处理结果 */
function flattenToolMessages(toolMessages: DisplayMessage[]): DisplayMessage[] {
  return toolMessages.flatMap((message) => message.isToolGroup && message.groupedTools ? message.groupedTools : [message])
}

/** 最新交互式终端状态处理，接收工具消息列表，返回处理结果，无法取得时为空 */
function latestInteractiveTerminalState(toolMessages: DisplayMessage[]): InteractiveTerminalState | null {
  let latest: InteractiveTerminalState | null = null // 最新
  for (/* 工具消息 */ const toolMessage of toolMessages) {
    const state = getInteractiveTerminalState(toolMessage.toolResult ?? '') // 当前状态
    if (state) latest = state
  }
  return latest
}
