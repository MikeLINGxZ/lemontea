import { ChevronDown, Check } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { cn } from '@/lib/utils'

/** 定义设置选择选项的数据结构 */
type SettingsSelectOption = {
  /** 值 */
  value: string
  /** 标签 */
  label: string
  /** 图标 */
  icon?: string
}

/** 渲染设置下拉选项，并在选择、点击外部或按退出键时收起，接收组件输入属性，返回组件内容 */
export function SettingsSelect(props: {
  /** 无障碍标签 */
  ariaLabel: string
  /** 值 */
  value: string
  /** 可用选项列表 */
  options: SettingsSelectOption[]
  /** 处理本次变更事件，接收值，无返回值 */
  onChange: (value: string) => void
}) {
  const [open, setOpen] = useState(false) // 展开及其更新方法
  const containerRef = useRef<HTMLDivElement | null>(null) // 容器引用，供后续读取当前值
  const listboxId = useId() // 为控件及其关联内容生成稳定的无障碍标识
  /** 当前选中的下拉选项 */
  const selectedOption = props.options.find((option) => option.value === props.value) ?? props.options[0]

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!open) {
      return undefined
    }

    /** 处理指针向下，接收触发本次回调的事件，无返回值 */
    const handlePointerDown = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false)
      }
    }

    /** 处理退出键，接收触发本次回调的事件，无返回值 */
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
      }
    }

    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [open])

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        role="combobox"
        aria-label={props.ariaLabel}
        aria-expanded={open}
        aria-controls={listboxId}
        onClick={() => setOpen((current) => !current)}
        className={cn(
          'flex h-11 w-full items-center justify-between rounded-xl border border-border/80 bg-card/80 px-3.5 text-left',
          'text-sm text-foreground transition-colors hover:bg-accent/60 focus:outline-hidden focus:ring-2 focus:ring-primary/20'
        )}
      >
        <span className="inline-flex items-center gap-2 truncate">
          {selectedOption?.icon && <span className="shrink-0" aria-hidden="true">{selectedOption.icon}</span>}
          <span className="truncate">{selectedOption?.label}</span>
        </span>
        <ChevronDown size={16} className={cn('shrink-0 text-muted-foreground transition-transform', open && 'rotate-180')} />
      </button>

      {open && (
        <div
          id={listboxId}
          role="listbox"
          aria-label={props.ariaLabel}
          className="absolute left-0 top-[calc(100%+0.5rem)] z-30 w-full rounded-2xl border border-border/80 bg-popover p-1.5 shadow-xl max-h-72 overflow-y-auto overscroll-contain"
        >
          {props.options.map((option) => {
            const selected = option.value === props.value // 选中项
            return (
              <button
                key={option.value}
                type="button"
                role="option"
                aria-selected={selected}
                onClick={() => {
                  props.onChange(option.value)
                  setOpen(false)
                }}
                className={cn(
                  'flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-left text-sm transition-colors',
                  selected
                    ? 'bg-primary/10 font-medium text-primary'
                    : 'text-foreground hover:bg-accent'
                )}
              >
                <span className="inline-flex items-center gap-2 min-w-0 flex-1 truncate">
                  {option.icon && <span className="shrink-0" aria-hidden="true">{option.icon}</span>}
                  <span className="truncate">{option.label}</span>
                </span>
                {selected && <Check size={14} className="shrink-0" />}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
