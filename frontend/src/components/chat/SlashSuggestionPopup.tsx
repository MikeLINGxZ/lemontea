import { forwardRef, useImperativeHandle, useState, useEffect } from 'react'
import type { SuggestionKeyDownProps } from '@tiptap/suggestion'

/** 定义斜杠指令建议条目的数据结构 */
export type SlashSuggestionItem = {
  /** 名称 */
  name: string
  /** 描述 */
  description: string
}

/** 斜杠指令建议弹层组件接收的属性 */
type Props = {
  /** 条目列表 */
  items: SlashSuggestionItem[]
  /** 命令处理，接收当前条目，无返回值 */
  command: (item: SlashSuggestionItem) => void
}

/** 定义斜杠指令建议句柄的数据结构 */
export type SlashSuggestionHandle = {
  /** 处理键向下事件，接收组件输入属性，返回是否满足条件 */
  onKeyDown: (props: SuggestionKeyDownProps) => boolean
}

/** 展示斜杠指令候选项，支持方向键选中和回车提交，接收条目列表、命令、引用，返回组件内容 */
export const SlashSuggestionPopup = forwardRef<SlashSuggestionHandle, Props>(({ items, command }, ref) => {
  const [index, setIndex] = useState(0) // 索引及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    setIndex((i) => Math.min(i, Math.max(items.length - 1, 0)))
  }, [items])

  useImperativeHandle(ref, () => ({
    /** 处理键向下事件，接收触发本次回调的事件，返回对应的处理结果 */
    onKeyDown: ({ event }: { /* 触发本次回调的事件 */ event: KeyboardEvent }) => {
      if (event.key === 'ArrowDown') {
        setIndex((i) => (i + 1) % items.length)
        return true
      }
      if (event.key === 'ArrowUp') {
        setIndex((i) => (i + items.length - 1) % items.length)
        return true
      }
      if (event.key === 'Enter') {
        const target = items[index] // 要处理的目标
        if (target) command(target)
        return true
      }
      return false
    },
  }), [items, index, command])

  if (items.length === 0) return null

  return (
    <div className="max-h-60 w-72 overflow-y-auto rounded-md border bg-popover p-1 shadow-lg">
      {items.map((it, i) => (
        <button
          key={it.name}
          type="button"
          onClick={() => command(it)}
          className={`flex w-full flex-col items-start gap-0.5 rounded-md px-2 py-1 text-left text-xs ${
            i === index ? 'bg-accent text-accent-foreground' : 'hover:bg-muted'
          }`}
        >
          <span className="font-medium">/{it.name}</span>
          <span className="line-clamp-2 opacity-70">{it.description}</span>
        </button>
      ))}
    </div>
  )
})
