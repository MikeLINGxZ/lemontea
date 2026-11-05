import { useRef, useCallback } from 'react'
import { cn } from '@/lib/utils'

/** 编辑扩展 JSON 并同步高亮层、错误行与滚动位置，接收组件输入属性，返回组件内容 */
export function PluginToolConfigEditor(props: {
  /** 值 */
  value: string
  /** 错误行 */
  errorLine: number | null
  /** 处理本次变更事件，接收值，无返回值 */
  onChange: (value: string) => void
}) {
  const textareaRef = useRef<HTMLTextAreaElement>(null) // 多行输入框引用，供后续读取当前值
  const highlightRef = useRef<HTMLPreElement>(null) // 高亮引用，供后续读取当前值

  /** 同步滚动，无返回值 */
  const syncScroll = useCallback(() => {
    if (highlightRef.current && textareaRef.current) {
      highlightRef.current.scrollTop = textareaRef.current.scrollTop
      highlightRef.current.scrollLeft = textareaRef.current.scrollLeft
    }
  }, [])

  const lines = props.value.split('\n') // 行列表

  return (
    <div className="relative min-h-[18rem] w-full rounded-2xl border border-border bg-background focus-within:border-primary/50">
      <pre
        ref={highlightRef}
        className="pointer-events-none absolute inset-0 overflow-hidden whitespace-pre px-4 py-3 font-mono text-sm leading-relaxed"
        aria-hidden="true"
      >
        <code>
          {lines.map((line, i) => (
            <div key={i} className={cn(
              props.errorLine !== null && i + 1 === props.errorLine && 'bg-red-500/20 dark:bg-red-500/30'
            )}>
              {line || ' '}
            </div>
          ))}
        </code>
      </pre>
      <textarea
        ref={textareaRef}
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
        onScroll={syncScroll}
        wrap="off"
        className="relative w-full resize-none whitespace-pre overflow-auto bg-transparent px-4 py-3 font-mono text-sm leading-relaxed text-transparent caret-foreground outline-hidden"
        style={{ minHeight: '18rem' }}
        spellCheck={false}
      />
    </div>
  )
}
