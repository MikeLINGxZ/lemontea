import { cn } from '@/lib/utils'

/** 渲染设置操作栏界面，接收组件输入属性，返回组件内容 */
export function SettingsActionBar(props: {
  /** 主操作标签，未提供时隐藏主操作 */
  primaryLabel?: string
  /** 主操作禁用状态 */
  primaryDisabled?: boolean
  /** 处理主操作点击事件，无返回值 */
  onPrimaryClick: () => void
  /** 次操作标签 */
  secondaryLabel?: string
  /** 次操作禁用状态 */
  secondaryDisabled?: boolean
  /** 处理次操作点击事件，无返回值 */
  onSecondaryClick?: () => void
  /** 危险操作标签 */
  dangerLabel?: string
  /** 处理危险操作点击事件，无返回值 */
  onDangerClick?: () => void
  /** 重新加载标签 */
  reloadLabel?: string
  /** 处理重新加载点击事件，无返回值 */
  onReloadClick?: () => void
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="flex items-center gap-3">
        {props.reloadLabel && props.onReloadClick && (
          <button
            type="button"
            onClick={props.onReloadClick}
            className="rounded-xl border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent"
          >
            {props.reloadLabel}
          </button>
        )}
        {props.dangerLabel && props.onDangerClick && (
          <button
            type="button"
            onClick={props.onDangerClick}
            className="rounded-xl border border-red-300 px-4 py-2 text-sm text-red-500 transition-colors hover:bg-red-500/10"
          >
            {props.dangerLabel}
          </button>
        )}
      </div>

      <div className="flex items-center gap-3">
        {props.secondaryLabel && props.onSecondaryClick && (
          <button
            type="button"
            disabled={props.secondaryDisabled}
            onClick={props.onSecondaryClick}
            className={cn(
              'rounded-xl border border-border px-4 py-2 text-sm transition-colors',
              props.secondaryDisabled
                ? 'cursor-not-allowed text-muted-foreground'
                : 'text-foreground hover:bg-accent'
            )}
          >
            {props.secondaryLabel}
          </button>
        )}
        {props.primaryLabel && (
          <button
            type="button"
            disabled={props.primaryDisabled}
            onClick={props.onPrimaryClick}
            className={cn(
              'rounded-xl px-4 py-2 text-sm font-medium transition-colors',
              props.primaryDisabled
                ? 'cursor-not-allowed bg-muted text-muted-foreground'
                : 'bg-primary text-primary-foreground hover:opacity-90'
            )}
          >
            {props.primaryLabel}
          </button>
        )}
      </div>
    </div>
  )
}
