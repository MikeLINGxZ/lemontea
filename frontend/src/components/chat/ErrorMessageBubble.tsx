import { useState } from 'react'
import { AlertCircle, ChevronDown, ChevronUp } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ResponseMeta } from './ResponseMeta'
import { isObjectRecord } from '@/lib/guards'

/** 错误消息气泡组件接收的属性 */
interface Props {
  /** 内容 */
  content: string
  /** 模型名称 */
  modelName?: string
  /** 输入 Token 数 */
  tokensIn?: number
  /** 输出 Token 数 */
  tokensOut?: number
}

/** 渲染错误消息气泡界面，接收内容、模型名称、输入 Token 数、输出 Token 数，返回组件内容 */
export function ErrorMessageBubble({ content, modelName, tokensIn, tokensOut }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [expanded, setExpanded] = useState(false) // 展开及其更新方法

  let msg = content // 可展示的错误消息
  let detail = '' // 详情
  try {
    const parsed: unknown = JSON.parse(content) // 保存待校验的 JSON 解析结果
    if (isObjectRecord(parsed)) {
      msg = typeof parsed.msg === 'string' ? parsed.msg : content
      detail = typeof parsed.detail === 'string' ? parsed.detail : ''
    }
  } catch {
    // 解析失败时展示原始错误文本
  }

  /** 已本地化流式输出错误列表 */
  const localizedStreamErrors = new Set([
    '大模型响应出错',
    'Model response error',
  ])

  if (localizedStreamErrors.has(msg)) {
    msg = t('chat.modelResponseError')
  }

  return (
    <div className="flex max-w-[70%] flex-col gap-1">
      <div className="flex w-full items-center justify-start gap-1 text-left text-sm text-destructive">
        <span className="select-text">{msg}</span>
        {detail && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex-shrink-0 flex items-center gap-0.5 text-destructive/70 hover:text-destructive transition-colors"
            title={expanded ? t('chat.hideErrorDetail') : t('chat.errorDetail')}
          >
            <AlertCircle size={14} />
            {expanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
          </button>
        )}
      </div>
      {expanded && detail && (
        <div className="w-full rounded-lg border border-border bg-muted/50 px-3 py-2 text-left font-mono text-xs text-muted-foreground whitespace-pre-wrap break-all select-text">
          {detail}
        </div>
      )}
      <ResponseMeta modelName={modelName} tokensIn={tokensIn} tokensOut={tokensOut} />
    </div>
  )
}
