import { useTranslation } from 'react-i18next'
import type { ExtensionToolItem } from '@/types/settings'

/** CLI工具列表区块组件接收的属性 */
type CliToolsSectionProps = {
  /** 工具列表 */
  tools: ExtensionToolItem[]
  /** 处理中 */
  busy: boolean
  /** JSON有效 */
  jsonValid: boolean
  /** 处理切换启用状态事件，接收工具名称、是否启用，无返回值 */
  onToggleEnabled: (toolName: string, enabled: boolean) => void
  /** 处理切换需要确认事件，接收工具名称、需要确认，无返回值 */
  onToggleRequiresConfirm: (toolName: string, requiresConfirm: boolean) => void
}

/** 渲染CLI工具列表区块界面，接收组件输入属性，返回组件内容 */
export function CliToolsSection(props: CliToolsSectionProps) {
  const { t } = useTranslation() // 获取当前语言的翻译方法

  return (
    <section className="mt-6 space-y-3 border-t border-border pt-4">
      <div className="space-y-1">
        <h3 className="text-sm font-semibold text-foreground">{t('settingsPage.plugins.cliToolsTitle')}</h3>
        <p className="text-sm text-muted-foreground">{t('settingsPage.plugins.cliToolsDescription')}</p>
      </div>

      {props.tools.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border bg-muted/30 px-4 py-5 text-sm text-muted-foreground">
          {t('settingsPage.plugins.cliToolsEmpty')}
        </div>
      ) : (
        <div className="space-y-3">
          {props.tools.map((tool) => (
            <div key={tool.tool_id} className="rounded-2xl border border-border bg-card/70 px-4 py-4">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                <div className="min-w-0 flex-1 space-y-1 pr-0 lg:pr-6">
                  <div className="text-sm font-medium text-foreground">{tool.name}</div>
                  <div className="max-w-2xl text-sm leading-6 text-muted-foreground">
                    {tool.description || t('settingsPage.plugins.emptyDescription')}
                  </div>
                </div>
                <div className="flex flex-col gap-1.5 lg:w-auto lg:flex-none lg:items-end">
                  <label className="flex items-center gap-2 px-1 py-1 text-sm text-foreground">
                    <span className="whitespace-nowrap text-sm font-medium">
                      {t('settingsPage.plugins.cliToolEnabled')}
                    </span>
                    <input
                      type="checkbox"
                      checked={tool.enabled}
                      disabled={props.busy || !props.jsonValid}
                      onChange={(event) => props.onToggleEnabled(tool.name, event.target.checked)}
                      className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
                    />
                  </label>
                  <label className="flex items-center gap-2 px-1 py-1 text-sm text-foreground">
                    <span className="whitespace-nowrap text-sm font-medium">
                      {t('settingsPage.plugins.cliToolRequiresConfirm')}
                    </span>
                    <input
                      type="checkbox"
                      checked={tool.requires_confirm}
                      disabled={props.busy || !props.jsonValid}
                      onChange={(event) => props.onToggleRequiresConfirm(tool.name, event.target.checked)}
                      className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
                    />
                  </label>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  )
}
