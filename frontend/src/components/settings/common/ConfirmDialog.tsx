import { Loader2 } from 'lucide-react'
import type { ReactNode } from 'react'

/** 确认对话框组件接收的属性 */
type ConfirmDialogProps = {
  /** 展开 */
  open: boolean
  /** 标题 */
  title: string
  /** 描述 */
  description?: ReactNode
  /** 确认标签 */
  confirmLabel: string
  /** 取消标签 */
  cancelLabel: string
  /** 确认语气 */
  confirmTone?: 'default' | 'danger'
  /** 处理中 */
  busy?: boolean
  /** 处理确认事件，无返回值 */
  onConfirm: () => void
  /** 处理取消事件，无返回值 */
  onCancel: () => void
}

/** 确认对话框，接收组件输入属性，返回组件内容 */
export function ConfirmDialog(props: ConfirmDialogProps) {
  if (!props.open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        aria-label={props.cancelLabel}
        className="absolute inset-0 bg-black/35"
        onClick={props.onCancel}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        className="relative z-10 w-full max-w-sm rounded-3xl border border-border bg-background p-5 shadow-2xl"
      >
        <div className="space-y-2">
          <h3 id="confirm-dialog-title" className="text-base font-semibold text-foreground">
            {props.title}
          </h3>
          {props.description && (
            <div className="text-sm leading-6 text-muted-foreground">
              {props.description}
            </div>
          )}
        </div>

        <div className="mt-5 flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={props.onCancel}
            className="rounded-xl border border-border px-4 py-2 text-sm text-foreground transition-colors hover:bg-accent"
          >
            {props.cancelLabel}
          </button>
          <button
            type="button"
            disabled={props.busy}
            onClick={props.onConfirm}
            className={
              props.confirmTone === 'danger'
                ? 'flex items-center gap-1.5 rounded-xl bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60'
                : 'flex items-center gap-1.5 rounded-xl bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60'
            }
          >
            {props.busy && <Loader2 size={14} className="animate-spin" />}
            {props.confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
