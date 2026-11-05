import type { ReactNode } from 'react'

/** 渲染设置字段 行界面，接收组件输入属性，返回组件内容 */
export function SettingsFieldRow(props: {
  /** 标签 */
  label: string
  /** 描述 */
  description?: string
  /** 要渲染的子内容 */
  children: ReactNode
}) {
  return (
    <div className="space-y-3 rounded-2xl bg-card/30 p-1">
      <div className="space-y-1">
        <p className="text-sm font-medium text-foreground">{props.label}</p>
        {props.description && <p className="text-sm text-muted-foreground">{props.description}</p>}
      </div>
      <div className="min-w-0">{props.children}</div>
    </div>
  )
}
