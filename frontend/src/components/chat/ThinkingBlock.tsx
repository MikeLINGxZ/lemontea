import { useState } from 'react'
import { Brain, ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'

/** 思考内容区块组件接收的属性 */
interface Props {
  /** 内容 */
  content: string
  /** 默认展开 */
  defaultExpanded?: boolean
  /** 当前 */
  active?: boolean
}

/** 渲染思考内容区块界面，接收内容、默认展开、当前，返回组件内容 */
export function ThinkingBlock({ content, defaultExpanded = false, active = false }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [expanded, setExpanded] = useState(defaultExpanded) // 展开及其更新方法

  return (
    <div className="mb-0">
      <button
        onClick={() => setExpanded(v => !v)}
        className="flex w-full min-w-0 items-center gap-1.5 rounded-md py-0.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
      >
        <Brain size={14} className="shrink-0" />
        <span>{t(active ? 'chat.thinking' : 'chat.thought')}</span>
        <ChevronDown
          size={14}
          className={expanded ? 'shrink-0 rotate-180 transition-transform' : 'shrink-0 rotate-0 transition-transform'}
        />
        <span className="min-w-0 flex-1" />
      </button>
      {expanded && (
        <div className="ml-1.5 mt-1 select-text whitespace-pre-wrap border-l border-border pl-3 text-sm leading-relaxed text-foreground">
          {content}
        </div>
      )}
    </div>
  )
}
