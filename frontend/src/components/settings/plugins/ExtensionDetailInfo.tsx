import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { PluginToolConfigEditor } from '@/components/settings/plugins/PluginToolConfigEditor'
import type { ExtensionItem } from '@/types/settings'

/** 运行环境状态类名处理，接收状态，返回对应的处理结果 */
const runtimeStatusClass = (status: string) => {
  switch (status) {
    case 'ready':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400'
    case 'error':
      return 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-400'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

/** 运行环境状态圆点处理，接收状态，返回对应的处理结果 */
const runtimeStatusDot = (status: string) => {
  switch (status) {
    case 'ready':
      return 'bg-emerald-500'
    case 'error':
      return 'bg-red-500'
    default:
      return 'bg-muted-foreground'
  }
}

/** 渲染运行状态徽标，接收扩展运行状态，返回对应颜色和文案的徽标 */
export function RuntimeStatusBadge({ status }: { /** 扩展运行状态 */ status: ExtensionItem['runtime_status'] }): React.JSX.Element {
  const { t } = useTranslation() // 获取运行状态文案的翻译方法

  return (
    <span className={cn(
      'inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium',
      runtimeStatusClass(status)
    )}>
      <span className={cn('inline-block h-1.5 w-1.5 rounded-full', runtimeStatusDot(status))} />
      {status === 'error'
        ? t('settingsPage.plugins.runtimeStatusError')
        : status === 'ready'
          ? t('settingsPage.plugins.runtimeStatusReady')
          : t('settingsPage.plugins.runtimeStatusIdle')}
    </span>
  )
}

/** 渲染扩展详情信息界面，接收组件输入属性，返回组件内容 */
export function ExtensionDetailInfo(props: {
  /** 扩展 */
  extension: ExtensionItem
  /** 草稿 */
  draft: string
  /** JSON有效 */
  jsonValid: boolean
  /** 错误行 */
  errorLine: number | null
  /** 处理草稿变更事件，接收值，无返回值 */
  onDraftChange: (value: string) => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const ext = props.extension // 扩展名

  if (ext.kind === 'cli') {
    return (
      <details key={ext.id} className="mt-6 pb-4 pt-4">
        <summary className="cursor-pointer text-sm font-semibold text-foreground transition-colors hover:text-foreground/80">
          {t('settingsPage.plugins.developerConfig')}
        </summary>
        <div className="mt-3 space-y-3">
          <div className="space-y-3 text-sm text-muted-foreground">
            <div>
              <span className="font-medium text-foreground">{t('settingsPage.plugins.installDirLabel')}:</span>{' '}
              <span className="break-all">{ext.root_dir}</span>
            </div>
            <div>
              <span className="font-medium text-foreground">{t('settingsPage.plugins.manifestPathLabel')}:</span>{' '}
              <span className="break-all">{ext.config_file_path}</span>
            </div>
          </div>
          {ext.config_file_path && (
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <h3 className="text-sm font-semibold text-foreground">{t('settingsPage.plugins.configuration')}</h3>
                {!props.jsonValid && (
                  <span className="text-xs font-medium text-red-600 dark:text-red-400">
                    {t('settingsPage.plugins.configJsonError', { line: props.errorLine })}
                  </span>
                )}
              </div>
              <PluginToolConfigEditor value={props.draft} errorLine={props.errorLine} onChange={props.onDraftChange} />
            </div>
          )}
        </div>
      </details>
    )
  }

  if (ext.kind === 'mcp') {
    return (
      <div className="space-y-6 py-8">
        <div className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
          <div>
            <span className="font-medium text-foreground">{t('settingsPage.plugins.toolsLabel')}:</span>{' '}
            {ext.tools.length}
          </div>
          <div>
            <span className="font-medium text-foreground">{t('settingsPage.plugins.runtimeStatusLabel')}:</span>{' '}
            <RuntimeStatusBadge status={ext.runtime_status} />
          </div>
          {ext.config_file_path && (
            <div className="md:col-span-2">
              <span className="font-medium text-foreground">{t('settingsPage.plugins.configLabel')}:</span>{' '}
              {ext.config_file_path}
            </div>
          )}
          {ext.runtime_message && (
            <div className="md:col-span-2 rounded-xl border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              <span className="font-medium">{t('settingsPage.plugins.runtimeMessageLabel')}:</span>{' '}
              {ext.runtime_message}
            </div>
          )}
        </div>

        {ext.config_file_path && (
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <h3 className="text-sm font-semibold text-foreground">{t('settingsPage.plugins.configuration')}</h3>
              {!props.jsonValid && (
                <span className="text-xs font-medium text-red-600 dark:text-red-400">
                  {t('settingsPage.plugins.configJsonError', { line: props.errorLine })}
                </span>
              )}
            </div>
            <PluginToolConfigEditor value={props.draft} errorLine={props.errorLine} onChange={props.onDraftChange} />
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-6 py-8">
      <div className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
        <div>
          <span className="font-medium text-foreground">{t('settingsPage.plugins.typeLabel')}:</span>{' '}
          {t('settingsPage.plugins.kindPlugin')}
        </div>
        <div>
          <span className="font-medium text-foreground">{t('settingsPage.plugins.versionLabel')}:</span>{' '}
          {ext.version || 'default'}
        </div>
        <div>
          <span className="font-medium text-foreground">{t('settingsPage.plugins.authorLabel')}:</span>{' '}
          {ext.author || '-'}
        </div>
      </div>
    </div>
  )
}
