import { type ReactNode, useContext } from 'react'
import { SettingsMenuContext } from '@/components/settings/SettingsShell'
import { cn } from '@/lib/utils'
import type { GeneralSettingsTab } from '@/types/settings'

/** 渲染常规设置二级菜单，选择后在窄屏收起侧栏，接收组件输入属性，返回组件内容 */
export function SettingsSubmenuList(props: {
  /** 标题 */
  title: string
  /** 条目列表 */
  items: { /* 键 */ key: GeneralSettingsTab; /* 标签 */ label: string; /* 图标 */ icon?: ReactNode }[]
  /** 值 */
  value: GeneralSettingsTab
  /** 处理本次变更事件，接收值，无返回值 */
  onChange: (value: GeneralSettingsTab) => void
  /** 操作 */
  action?: ReactNode
}) {
  const { isH5, onCollapseSubmenu } = useContext(SettingsMenuContext) // 取出是否采用窄屏布局、on折叠二级菜单

  return (
    <div className="flex h-full flex-col">
      <div className="mb-4 flex items-center justify-between gap-3 px-2">
        <h2 className="text-lg font-semibold text-foreground">{props.title}</h2>
        {props.action && <div className="flex items-center gap-1">{props.action}</div>}
      </div>

      <div className="space-y-1">
        {props.items.map((item) => (
          <button
            key={item.key}
            type="button"
            onClick={() => { props.onChange(item.key); onCollapseSubmenu() }}
            className={cn(
              'flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors',
              !isH5 && item.key === props.value
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-accent hover:text-foreground'
            )}
          >
            {item.icon && <span className="shrink-0">{item.icon}</span>}
            <span className="text-sm font-medium">{item.label}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
