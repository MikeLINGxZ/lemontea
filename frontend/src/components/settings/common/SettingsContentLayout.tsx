import type { ReactNode } from 'react'
import { useContext } from 'react'
import { SettingsMenuContext } from '@/components/settings/SettingsShell'
import { cn } from '@/lib/utils'

/** 排列设置页头、滚动内容和底部操作区，接收组件输入属性，返回组件内容 */
export function SettingsContentLayout(props: {
  /** 页头 */
  header: ReactNode
  /** 底部操作区 */
  footprint?: ReactNode
  /** 要渲染的子内容 */
  children?: ReactNode
  /** no内容滚动 */
  noContentScroll?: boolean
}) {
  const { isH5 } = useContext(SettingsMenuContext) // 是否采用窄屏布局
  const horizontalPadding = isH5 ? 'px-5' : 'px-10' // 横向内边距

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className={cn('shrink-0', horizontalPadding)}>{props.header}</div>
      {props.children && (
        <div className={cn(
          props.noContentScroll ? 'flex min-h-0 flex-1 flex-col overflow-hidden' : 'min-h-0 flex-1 overflow-y-auto',
          horizontalPadding,
        )}>
          {props.children}
        </div>
      )}
      {!props.children && <div className="min-h-0 flex-1" />}
      {props.footprint && (
        <div className={cn('shrink-0 pb-6 pt-5', horizontalPadding)}>{props.footprint}</div>
      )}
    </div>
  )
}
