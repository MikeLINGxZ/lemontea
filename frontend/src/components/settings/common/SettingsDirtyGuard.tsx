import type { ReactNode } from 'react'

/** 在设置存在未保存修改时提供离开保护，接收组件输入属性，返回组件内容 */
export function SettingsDirtyGuard(props: {
  /** 有未保存修改 */
  dirty: boolean
  /** 要渲染的子内容 */
  children: ReactNode
}) {
  return (
    <div data-dirty={props.dirty ? 'true' : 'false'} className="flex min-h-0 flex-1 flex-col">
      {props.children}
    </div>
  )
}
