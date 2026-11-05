import { useState } from 'react'
import { Check, ChevronDown, Wrench, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

/** 工具调用区块组件接收的属性 */
type Props = {
  /** 工具名称 */
  toolName: string
  /** 用途 */
  purpose: string
  /** 参数 */
  args: string
  /** 本次调用的结果 */
  result: string
  /** 用户备注 */
  userComment?: string
  /** 状态 */
  status: 'executing' | 'completed' | 'failed' | 'user_commented' | 'user_rejected'
}

/** 渲染工具调用区块界面，接收组件属性，返回组件内容 */
export function ToolCallBlock({ toolName, purpose, args, result, userComment, status }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [expanded, setExpanded] = useState(false) // 展开及其更新方法

  /** 状态图标 */
  const statusIcon = {
    executing: <Wrench size={12} className="animate-spin" />,
    completed: <Check size={12} className="text-green-500" />,
    failed: <X size={12} className="text-red-500" />,
    user_commented: <Check size={12} className="text-amber-500" />,
    user_rejected: <X size={12} className="text-red-500" />,
  }[status]

  return (
    <div className="my-0">
      <button
        onClick={() => setExpanded((value) => !value)}
        className="flex w-full min-w-0 items-center gap-1.5 rounded-md py-0.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
      >
        <Wrench size={14} className="shrink-0" />
        <span className="shrink-0 text-foreground">{toolName || 'tool'}</span>
        <ChevronDown
          size={14}
          className={expanded ? 'shrink-0 rotate-180 transition-transform' : 'shrink-0 rotate-0 transition-transform'}
        />
        <span className="min-w-0 flex-1 truncate text-left font-normal">{purpose}</span>
        <span className="shrink-0">{statusIcon}</span>
        <span className={cn(
          'shrink-0 font-normal text-muted-foreground',
          status === 'completed' && 'text-green-600',
          status === 'failed' && 'text-red-600',
          status === 'user_commented' && 'text-amber-600',
          status === 'user_rejected' && 'text-red-600'
        )}>
          {t(`toolCall.${status}`)}
        </span>
      </button>
      {expanded && (
        <div className="ml-1.5 mt-1 space-y-2 border-l border-border pl-3">
          {userComment && (
            <div>
              <div className="mb-1 text-[10px] font-semibold uppercase text-muted-foreground">
                {t('toolCall.userComment')}
              </div>
              <div className="select-text whitespace-pre-wrap rounded bg-muted/70 p-2 text-xs text-foreground">
                {userComment}
              </div>
            </div>
          )}
          {args && (
            <div>
              <div className="mb-1 text-[10px] font-semibold uppercase text-muted-foreground">Args</div>
              <pre className="select-text overflow-x-auto whitespace-pre-wrap rounded bg-muted/70 p-2 text-xs">{args}</pre>
            </div>
          )}
          {result && (
            <div>
              <div className="mb-1 text-[10px] font-semibold uppercase text-muted-foreground">Result</div>
              <pre className="select-text max-h-48 overflow-x-auto overflow-y-auto whitespace-pre-wrap rounded bg-muted/70 p-2 text-xs">{result}</pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
