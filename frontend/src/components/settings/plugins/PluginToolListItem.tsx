import { useState } from 'react'
import { MoreHorizontal, RefreshCw, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { ConfirmDialog } from '@/components/settings/common/ConfirmDialog'
import { useCliInstallStore } from '@/store/cliInstallStore'
import type { ExtensionItem } from '@/types/settings'

/** 渲染插件工具列表条目界面，接收组件输入属性，返回组件内容 */
export function PluginToolListItem(props: {
  /** 当前条目 */
  item: ExtensionItem
  /** 选中项 */
  selected: boolean
  /** 处理选择事件，接收当前条目的唯一编号，无返回值 */
  onSelect: (id: string) => void
  /** 处理切换启用事件，接收当前条目，无返回值 */
  onToggleEnable: (item: ExtensionItem) => void
  /** 处理重新加载事件，接收当前条目的唯一编号，无返回值 */
  onReload: (id: string) => void
  /** 处理删除事件，接收当前条目的唯一编号，无返回值 */
  onDelete: (id: string) => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [menuOpen, setMenuOpen] = useState(false) // 菜单展开及其更新方法
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false) // 确认删除展开及其更新方法
  /** 从状态仓库读取安装进度 */
  const installProgress = useCliInstallStore((s) =>
    s.items.find((i) => i.name === props.item.name || i.extension_id === props.item.id),
  )
  const progressWidth = installProgress ? phaseWidth(installProgress.phase) : '0%' // 进度宽度

  return (
    <>
      <div className={cn(
        'group relative flex w-full items-start gap-3 rounded-2xl px-3 py-3 transition-colors',
        props.selected ? 'bg-primary/10 text-primary' : 'bg-background hover:bg-accent',
      )}>
        <div
          role="button"
          tabIndex={0}
          onClick={() => props.onSelect(props.item.id)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') props.onSelect(props.item.id)
          }}
          className="flex min-w-0 flex-1 cursor-pointer items-start"
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="truncate text-sm font-medium">{props.item.name}</span>
                  <span className={cn(
                    'rounded-full px-2 py-0.5 text-[11px]',
                    props.item.kind === 'mcp'
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400'
                      : props.item.kind === 'cli'
                      ? 'bg-purple-100 text-purple-700 dark:bg-purple-500/20 dark:text-purple-400'
                      : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400'
                  )}>
                    {props.item.kind === 'mcp'
                      ? t('settingsPage.plugins.kindMcp')
                      : props.item.kind === 'cli'
                      ? t('settingsPage.plugins.kindCli')
                      : t('settingsPage.plugins.kindPlugin')}
                  </span>
                </div>
                <div className="mt-1 truncate text-xs text-muted-foreground">{props.item.description || props.item.author || props.item.version}</div>
                {installProgress && installProgress.phase !== 'done' && installProgress.phase !== 'failed' && (
                  <div className="mt-2 flex items-center gap-2">
                    <div className="h-1 flex-1 overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-primary transition-all duration-300"
                        style={{ width: progressWidth }}
                      />
                    </div>
                    <span className="text-[11px] text-muted-foreground">
                      {installProgress.detail || installProgress.phase}
                    </span>
                  </div>
                )}
                {installProgress?.action_url && installProgress.phase === 'waiting_auth' && (
                  <a
                    href={installProgress.action_url}
                    target="_blank"
                    rel="noreferrer"
                    className="mt-2 inline-flex text-xs text-primary underline underline-offset-2"
                  >
                    {installProgress.action_label || installProgress.action_url}
                  </a>
                )}
                {installProgress?.phase === 'failed' && (
                  <div className="mt-2 rounded-md bg-red-100 px-2 py-1 text-xs text-red-700 dark:bg-red-500/10 dark:text-red-400">
                    {installProgress.detail || t('settingsPage.plugins.cliInstallFailed')}
                  </div>
                )}
              </div>
              <div className="flex shrink-0 items-center gap-1">
                <button
                  type="button"
                  role="switch"
                  aria-checked={props.item.enabled}
                  onClick={(event) => {
                    event.stopPropagation()
                    props.onToggleEnable(props.item)
                  }}
                  className={cn(
                    'relative inline-flex h-[18px] w-[31px] shrink-0 items-center rounded-full transition-colors duration-200',
                    props.item.enabled ? 'bg-primary' : 'bg-muted',
                  )}
                >
                  <span className={cn(
                    'inline-block h-[14px] w-[14px] rounded-full bg-white shadow-xs transition-transform duration-200',
                    props.item.enabled ? 'translate-x-[15px]' : 'translate-x-[2px]',
                  )} />
                </button>
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation()
                    setMenuOpen(current => !current)
                  }}
                  className="flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                >
                  <MoreHorizontal size={12} />
                </button>
              </div>
            </div>
          </div>
        </div>

        {menuOpen && (
          <>
            <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
            <div className="absolute right-0 top-full z-20 mt-1 min-w-36 rounded-lg border border-border bg-popover py-1 shadow-md">
              <button
                type="button"
                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-accent"
                onClick={() => {
                  props.onReload(props.item.id)
                  setMenuOpen(false)
                }}
              >
                <RefreshCw size={13} />
                {t('settingsPage.plugins.reload')}
              </button>
              <button
                type="button"
                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-destructive hover:bg-accent"
                onClick={() => {
                  setConfirmDeleteOpen(true)
                  setMenuOpen(false)
                }}
              >
                <Trash2 size={13} />
                {t('settingsPage.plugins.delete')}
              </button>
            </div>
          </>
        )}
      </div>

      <ConfirmDialog
        open={confirmDeleteOpen}
        title={t('settingsPage.plugins.confirmDeleteTitle')}
        description={props.item.name}
        confirmLabel={t('settingsPage.plugins.delete')}
        cancelLabel={t('settingsPage.providers.cancel')}
        confirmTone="danger"
        onConfirm={() => {
          props.onDelete(props.item.id)
          setConfirmDeleteOpen(false)
        }}
        onCancel={() => setConfirmDeleteOpen(false)}
      />
    </>
  )
}

/** 阶段宽度处理，接收阶段，返回处理后的文本 */
function phaseWidth(phase: string): string {
  switch (phase) {
    case 'pending':
      return '12%'
    case 'downloading':
      return '30%'
    case 'installed':
      return '55%'
    case 'generating':
      return '72%'
    case 'initializing':
      return '82%'
    case 'waiting_auth':
      return '88%'
    case 'verifying':
      return '94%'
    case 'done':
      return '100%'
    default:
      return '30%'
  }
}
