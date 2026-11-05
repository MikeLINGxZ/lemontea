import { useState } from 'react'
import { ShieldAlert, Check, X, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

/** 工具确认卡片组件接收的属性 */
type Props = {
  /** 工具名称 */
  toolName: string
  /** 用途 */
  purpose: string
  /** 参数 */
  args: string
  /** 处理确认事件，接收消息，无返回值 */
  onConfirm: (message: string) => void
  /** 处理拒绝事件，接收消息，无返回值 */
  onReject: (message: string) => void
  /** 处理提交备注事件，接收消息，无返回值 */
  onSubmitComment: (message: string) => void
  /** 已处理 */
  resolved: boolean
  /** 已同意 */
  approved: boolean
}

/** 渲染工具确认卡片界面，接收组件属性，返回组件内容 */
export function ToolConfirmCard({
  toolName,
  purpose,
  args,
  onConfirm,
  onReject,
  onSubmitComment,
  resolved,
  approved,
}: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [feedback, setFeedback] = useState('') // 反馈及其更新方法
  const [commentOpen, setCommentOpen] = useState(false) // 备注展开及其更新方法
  void resolved
  void approved

  return (
    <div className="my-1 overflow-hidden rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950">
      <div className="flex items-center gap-2 px-3 py-2">
        <ShieldAlert size={14} className="shrink-0 text-amber-600" />
        <span className="text-xs font-medium text-foreground">{toolName}</span>
        <span className="text-xs text-muted-foreground">{t('toolCall.waitingConfirm')}</span>
      </div>
      <div className="px-3 pb-2 text-xs text-muted-foreground">
        <div className="mb-1 font-medium text-foreground">{t('toolCall.purpose')}: {purpose}</div>
        {args && (
          <pre className="mb-2 max-h-32 overflow-x-auto overflow-y-auto whitespace-pre-wrap rounded bg-muted p-2">{args}</pre>
        )}
      </div>
      <div className="flex items-center justify-between gap-3 border-t border-amber-200 px-3 py-2 dark:border-amber-800">
        <div className="flex items-center gap-2">
          <button
            onClick={() => onConfirm('')}
            className="flex shrink-0 items-center gap-1 rounded bg-green-100 px-2 py-1 text-xs text-green-700 transition-colors hover:bg-green-200 dark:bg-green-900 dark:text-green-300 dark:hover:bg-green-800"
          >
            <Check size={12} />
            {t('toolCall.confirm')}
          </button>
          <button
            onClick={() => onReject('')}
            className="flex shrink-0 items-center gap-1 rounded bg-red-100 px-2 py-1 text-xs text-red-700 transition-colors hover:bg-red-200 dark:bg-red-900 dark:text-red-300 dark:hover:bg-red-800"
          >
            <X size={12} />
            {t('toolCall.reject')}
          </button>
        </div>
        <div className="flex min-w-0 flex-1 items-center justify-end gap-2">
          <button
            onClick={() => setCommentOpen((value) => !value)}
            className="flex shrink-0 items-center gap-1 rounded bg-amber-100 px-2 py-1 text-xs text-amber-800 transition-colors hover:bg-amber-200 dark:bg-amber-900 dark:text-amber-200 dark:hover:bg-amber-800"
          >
            <Sparkles size={12} />
            {t('toolCall.tellAi')}
          </button>
          {commentOpen && (
            <>
              <input
                value={feedback}
                onChange={(event) => setFeedback(event.target.value)}
                placeholder={t('toolCall.inputPlaceholder')}
                className="min-w-0 max-w-sm flex-1 rounded border border-input bg-transparent px-2 py-1 text-xs outline-hidden focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
              />
              <button
                onClick={() => onSubmitComment(feedback)}
                disabled={!feedback.trim()}
                className="flex shrink-0 items-center gap-1 rounded bg-amber-100 px-2 py-1 text-xs text-amber-800 transition-colors hover:bg-amber-200 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-amber-900 dark:text-amber-200 dark:hover:bg-amber-800"
              >
                <Check size={12} />
                {t('toolCall.submitComment')}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
