import { ArrowLeft } from 'lucide-react'
import { type ReactNode, useContext } from 'react'
import { SettingsMenuContext } from '@/components/settings/SettingsShell'
import { cn, isMac } from '@/lib/utils'

/** 渲染窄屏导航 栏界面，接收组件输入属性，返回组件内容 */
function H5NavBar(props: { /* 处理点击事件，无返回值 */ onClick: () => void }) {
  return (
    <div className={cn(
      'flex shrink-0 items-end border-b border-border/50',
      isMac ? 'window-titlebar h-[50px] justify-end pr-5 pb-3' : 'py-2.5 pl-5 pr-5',
    )}>
      <button
        type="button"
        onClick={props.onClick}
        className="flex items-center gap-1.5 rounded-lg text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft size={16} />
        <span className="font-medium">返回</span>
      </button>
    </div>
  )
}

/** 渲染设置区块布局界面，接收组件输入属性，返回组件内容 */
export function SettingsSectionLayout(props: {
  /** 侧栏 */
  sidebar?: ReactNode
  /** 侧栏类名名称 */
  sidebarClassName?: string
  /** 要渲染的子内容 */
  children: ReactNode
}) {
  /** 取出是否采用窄屏布局、二级菜单折叠、on展开菜单、on展开二级菜单 */
  const { isH5, submenuCollapsed, onOpenMenu, onOpenSubmenu } = useContext(SettingsMenuContext)

  if (isH5 && props.sidebar) {
    if (!submenuCollapsed) {
      return (
        <section className="flex min-h-0 min-w-0 flex-1 flex-col">
          {onOpenMenu && <H5NavBar onClick={onOpenMenu} />}
          <div className="flex min-h-0 min-w-0 flex-1 flex-col px-3 pb-3 pt-6">
            {props.sidebar}
          </div>
        </section>
      )
    }

    return (
      <section className="flex min-h-0 min-w-0 flex-1 flex-col">
        {onOpenSubmenu && <H5NavBar onClick={onOpenSubmenu} />}
        <div className="flex min-h-0 min-w-0 flex-1 flex-col pt-6">
          {props.children}
        </div>
      </section>
    )
  }

  if (!props.sidebar) {
    if (isH5) {
      return (
        <section className="flex min-h-0 min-w-0 flex-1 flex-col">
          {onOpenMenu && <H5NavBar onClick={onOpenMenu} />}
          <div className="flex min-h-0 min-w-0 flex-1 flex-col pt-6">
            {props.children}
          </div>
        </section>
      )
    }
    return <section className="flex min-h-0 min-w-0 flex-1 flex-col pt-[var(--settings-top-padding)]">{props.children}</section>
  }

  return (
    <section className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
      <div className={cn('shrink-0 border-r border-border/50 bg-background/40 px-3 pb-3 pt-[var(--settings-top-padding)]', props.sidebarClassName ?? 'w-56')}>{props.sidebar}</div>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col pt-[var(--settings-top-padding)]">{props.children}</div>
    </section>
  )
}
