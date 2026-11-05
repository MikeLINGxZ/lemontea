/** 渲染设置面板页头界面，接收组件输入属性，返回组件内容 */
export function SettingsPanelHeader(props: {
  /** 标题 */
  title: string
  /** 描述 */
  description?: string
  /** 侧边栏 */
  aside?: React.ReactNode
  /** 徽标 */
  badge?: React.ReactNode
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <h2 className="flex items-center gap-2 text-2xl font-semibold text-foreground">
          {props.title}
          {props.badge}
        </h2>
        {props.description && <p className="mt-2 max-w-2xl text-sm text-muted-foreground">{props.description}</p>}
      </div>
      {props.aside}
    </div>
  )
}
