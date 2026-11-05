import { useState } from 'react'
import { CircleX, AlertTriangle, CheckCircle2, Info } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { AlertAction, AlertItem } from '@/alert/types'
import { useAlertStore } from '@/alert/store'

/** 提醒卡片组件接收的属性 */
type AlertCardProps = {
  /** 提醒 */
  alert: AlertItem
}

/** 定义变体颜色组的数据结构 */
type VariantColors = {
  /** 卡片 */
  card: string
  /** 图标 */
  icon: string
  /** 标题 */
  title: string
  /** 消息 */
  message: string
  /** 弱化 */
  muted: string
  /** 数量 */
  count: string
  /** 分隔线 */
  divider: string
  /** 关闭 */
  close: string
}

/** 变体颜色组处理，接收类别标识，返回对应的处理结果 */
function variantColors(kind: AlertItem['kind']): VariantColors {
  switch (kind) {
    case 'error':
      return {
        card: 'bg-red-50 border-red-200 dark:bg-[#2a1215] dark:border-red-800/60',
        icon: 'text-red-500 dark:text-red-400',
        title: 'text-red-800/65 dark:text-red-300/65',
        message: 'text-red-900 dark:text-red-50',
        muted: 'text-red-700/55 dark:text-red-300/55',
        count: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
        divider: 'border-red-200 dark:border-red-800/55',
        close: 'text-red-700/40 hover:text-red-700/75 dark:text-red-300/40 dark:hover:text-red-300/75',
      }
    case 'warning':
      return {
        card: 'bg-amber-50 border-amber-200 dark:bg-[#2b1d0e] dark:border-amber-800/60',
        icon: 'text-amber-500 dark:text-amber-400',
        title: 'text-amber-800/65 dark:text-amber-300/65',
        message: 'text-amber-900 dark:text-amber-50',
        muted: 'text-amber-700/55 dark:text-amber-300/55',
        count: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300',
        divider: 'border-amber-200 dark:border-amber-800/55',
        close: 'text-amber-700/40 hover:text-amber-700/75 dark:text-amber-300/40 dark:hover:text-amber-300/75',
      }
    case 'success':
      return {
        card: 'bg-emerald-50 border-emerald-200 dark:bg-[#0d2219] dark:border-emerald-800/60',
        icon: 'text-emerald-500 dark:text-emerald-400',
        title: 'text-emerald-800/65 dark:text-emerald-300/65',
        message: 'text-emerald-900 dark:text-emerald-50',
        muted: 'text-emerald-700/55 dark:text-emerald-300/55',
        count: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-300',
        divider: 'border-emerald-200 dark:border-emerald-800/55',
        close: 'text-emerald-700/40 hover:text-emerald-700/75 dark:text-emerald-300/40 dark:hover:text-emerald-300/75',
      }
    default:
      return {
        card: 'bg-blue-50 border-blue-200 dark:bg-[#0d1d2b] dark:border-blue-800/60',
        icon: 'text-blue-500 dark:text-blue-400',
        title: 'text-blue-800/65 dark:text-blue-300/65',
        message: 'text-blue-900 dark:text-blue-50',
        muted: 'text-blue-700/55 dark:text-blue-300/55',
        count: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
        divider: 'border-blue-200 dark:border-blue-800/55',
        close: 'text-blue-700/40 hover:text-blue-700/75 dark:text-blue-300/40 dark:hover:text-blue-300/75',
      }
  }
}

/** 类别图标处理，接收类别标识、图标类名，返回对应的处理结果 */
function kindIcon(kind: AlertItem['kind'], iconClass: string) {
  const props = { size: 16, strokeWidth: 2, className: `shrink-0 ${iconClass}` } as const // 组件输入属性
  if (kind === 'error') return <CircleX {...props} />
  if (kind === 'warning') return <AlertTriangle {...props} />
  if (kind === 'success') return <CheckCircle2 {...props} />
  return <Info {...props} />
}

/** 操作类名名称处理，接收操作，返回处理后的文本 */
function actionClassName(action: AlertAction): string {
  if (action.style === 'danger') return 'bg-red-500 text-white hover:bg-red-600'
  if (action.style === 'secondary') return 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
  return 'bg-primary text-primary-foreground hover:bg-primary/90'
}

/** 渲染单条提醒的正文、详情及操作，并管理展开和关闭动画，接收提醒，返回组件内容 */
export function AlertCard({ alert }: AlertCardProps) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const closeAlert = useAlertStore((state) => state.closeAlert) // 从状态仓库读取关闭提醒
  const [detailsExpanded, setDetailsExpanded] = useState<boolean>(false) // 详情展开及其更新方法
  const visibleActions = alert.actions.slice(0, 2) // 可见操作列表
  const colors = variantColors(alert.kind) // 颜色组

  /** 复制详情，完成后结束异步任务 */
  const copyDetails = async (): Promise<void> => {
    if (!alert.detail || !navigator.clipboard) return
    /** 载荷 */
    const payload = [
      alert.title,
      alert.message,
      alert.code ? `Code: ${alert.code}` : '',
      alert.detail,
    ].filter((item) => item.length > 0).join('\n')
    await navigator.clipboard.writeText(payload)
  }

  return (
    <article
      role={alert.kind === 'error' ? 'alert' : 'status'}
      aria-live={alert.ariaPriority}
      className={`alert-card rounded-lg border px-4 py-3 shadow-[0_6px_16px_0_rgba(0,0,0,0.08),0_3px_6px_-4px_rgba(0,0,0,0.12)] dark:shadow-[0_6px_16px_0_rgba(0,0,0,0.5)] ${colors.card}`}
    >
      <div className="flex items-start gap-2.5">
        <span className="mt-0.5">{kindIcon(alert.kind, colors.icon)}</span>
        <div className="min-w-0 flex-1">
          {alert.title ? (
            <p className={`mb-0.5 text-[11px] font-semibold uppercase tracking-[0.16em] ${colors.title}`}>
              {alert.title}
            </p>
          ) : null}
          <div className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
            {alert.showCount && alert.count > 1 ? (
              <span className={`shrink-0 rounded-full px-2 py-0.5 text-[11px] font-semibold ${colors.count}`}>
                {`x${alert.count}`}
              </span>
            ) : null}
            <p className={`text-[14px] font-medium leading-5 ${colors.message}`}>
              {alert.message}
            </p>
            {alert.detail ? (
              <button
                type="button"
                className={`shrink-0 text-xs underline-offset-2 transition hover:underline ${colors.muted}`}
                aria-expanded={detailsExpanded}
                onClick={() => setDetailsExpanded((v) => !v)}
              >
                {detailsExpanded ? t('alert.hideDetails') : t('alert.showDetails')}
              </button>
            ) : null}
          </div>
          {alert.code ? (
            <p className={`mt-1 text-[11px] font-medium tracking-[0.08em] ${colors.muted}`}>{alert.code}</p>
          ) : null}

          {detailsExpanded && alert.detail ? (
            <div className={`mt-2 border-t pt-2 ${colors.divider}`}>
              <pre className={`max-h-32 overflow-x-auto whitespace-pre-wrap break-all text-left font-mono text-xs leading-5 ${colors.muted}`}>
                {alert.detail}
              </pre>
              <button
                type="button"
                className={`mt-2 text-xs underline-offset-2 transition hover:underline ${colors.muted}`}
                onClick={() => { void copyDetails() }}
              >
                {t('alert.copyDetails')}
              </button>
            </div>
          ) : null}

          {visibleActions.length > 0 ? (
            <div className="mt-3 flex flex-wrap justify-end gap-2">
              {visibleActions.map((action) => (
                <button
                  key={action.id}
                  type="button"
                  className={`rounded-full px-3 py-1.5 text-xs font-semibold transition ${actionClassName(action)}`}
                  onClick={() => {
                    action.onClick?.()
                    if (action.href) window.open(action.href, '_blank', 'noopener,noreferrer')
                    if (action.closeOnClick) closeAlert(alert.id, 'action')
                  }}
                >
                  {action.label}
                </button>
              ))}
            </div>
          ) : null}
        </div>

        {alert.dismissible ? (
          <button
            type="button"
            className={`mt-0.5 shrink-0 text-lg leading-none transition ${colors.close}`}
            aria-label={t('alert.dismiss')}
            onClick={() => closeAlert(alert.id, 'user')}
          >
            ×
          </button>
        ) : null}
      </div>
    </article>
  )
}
