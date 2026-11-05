import { useState } from 'react'
import { ChevronDown, Loader2, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PluginToolListItem } from '@/components/settings/plugins/PluginToolListItem'
import { useCliInstallStore } from '@/store/cliInstallStore'
import type { ExtensionItem } from '@/types/settings'

/** 渲染插件工具列表界面，接收组件输入属性，返回组件内容 */
export function PluginToolList(props: {
  /** 条目列表 */
  items: ExtensionItem[]
  /** 当前选中项编号 */
  selectedId: string | null
  /** 处理选择事件，接收当前条目的唯一编号，无返回值 */
  onSelect: (id: string) => void
  /** 处理创建事件，接收类别标识，无返回值 */
  onCreate: (kind: 'mcp' | 'plugin' | 'cli') => void
  /** 处理切换启用事件，接收当前条目，无返回值 */
  onToggleEnable: (item: ExtensionItem) => void
  /** 处理重新加载事件，接收当前条目的唯一编号，无返回值 */
  onReload: (id: string) => void
  /** 处理删除事件，接收当前条目的唯一编号，无返回值 */
  onDelete: (id: string) => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [menuOpen, setMenuOpen] = useState(false) // 菜单展开及其更新方法
  const pendingInstalls = useCliInstallStore((s) => s.items) // 从状态仓库读取待处理安装记录

  return (
    <div className="flex h-full flex-col">
      <div className="mb-4 flex items-center justify-between gap-3 px-2">
        <h2 className="text-lg font-semibold text-foreground">{t('settingsPage.plugins.title')}</h2>
        <div className="relative flex items-center gap-1">
          <button
            type="button"
            onClick={() => setMenuOpen((current) => !current)}
            className="inline-flex h-8 items-center gap-1.5 rounded-lg px-2 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <Plus size={14} />
            <span>{t('settingsPage.plugins.add')}</span>
            <ChevronDown size={12} />
          </button>
          {menuOpen && (
            <div className="absolute right-0 top-10 z-20 min-w-40 rounded-xl border border-border bg-popover py-1 shadow-lg">
              <button
                type="button"
                className="flex w-full items-center px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent"
                onClick={() => { props.onCreate('mcp'); setMenuOpen(false) }}
              >
                {t('settingsPage.plugins.addMcp')}
              </button>
              {/*  type="button"*/}
              {/*  className="flex w-full items-center px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent"*/}
              {/*  onClick={() => { props.onCreate('plugin'); setMenuOpen(false) }}*/}
              {/*  {t('settingsPage.plugins.addPlugin')}*/}
              <button
                type="button"
                className="flex w-full items-center px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent"
                onClick={() => { props.onCreate('cli'); setMenuOpen(false) }}
              >
                {t('settingsPage.plugins.addCli')}
              </button>
            </div>
          )}
        </div>
      </div>

      {menuOpen && (
        <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
      )}

      <div className="space-y-2">
        {pendingInstalls.map((install) => (
          <div key={`installing:${install.npm_package}`} className="rounded-2xl bg-accent/50 px-3 py-3">
            <div className="flex items-center gap-2">
              <Loader2 size={14} className="animate-spin text-muted-foreground" />
              <span className="text-sm font-medium">{install.name || install.npm_package}</span>
              <span className="rounded-full bg-purple-100 px-2 py-0.5 text-[11px] text-purple-700 dark:bg-purple-500/20 dark:text-purple-400">
                {t('settingsPage.plugins.kindCli')}
              </span>
            </div>
            <div className="mt-1 text-xs text-muted-foreground">{install.detail || install.phase}</div>
            {install.action_url && install.phase === 'waiting_auth' && (
              <a
                href={install.action_url}
                target="_blank"
                rel="noreferrer"
                className="mt-2 inline-flex text-xs text-primary underline underline-offset-2"
              >
                {install.action_label || install.action_url}
              </a>
            )}
          </div>
        ))}
        {props.items.map((item) => (
          <PluginToolListItem
            key={item.id}
            item={item}
            selected={item.id === props.selectedId}
            onSelect={props.onSelect}
            onToggleEnable={props.onToggleEnable}
            onReload={props.onReload}
            onDelete={props.onDelete}
          />
        ))}
      </div>
    </div>
  )
}
