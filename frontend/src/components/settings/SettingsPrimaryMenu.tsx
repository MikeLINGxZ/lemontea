import { Blocks, Brain, Info, Package2, SlidersHorizontal, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import type { SettingsPrimaryTab } from '@/types/settings'

/** 渲染设置主操作菜单界面，接收组件输入属性，返回组件内容 */
export function SettingsPrimaryMenu(props: {
  /** 当前标签页 */
  activeTab: SettingsPrimaryTab
  /** 处理本次变更事件，接收标签页，无返回值 */
  onChange: (tab: SettingsPrimaryTab) => void
  /** 隐藏页头 */
  hideHeader?: boolean
  /** 是否采用窄屏布局 */
  isH5?: boolean
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  /** 菜单条目列表 */
  const menuItems: { /* 键 */ key: SettingsPrimaryTab; /* 标签 */ label: string; /* 图标 */ icon: typeof SlidersHorizontal }[] = [
    { key: 'general', label: t('settingsPage.primary.general'), icon: SlidersHorizontal },
    { key: 'providers', label: t('settingsPage.primary.providers'), icon: Package2 },
    { key: 'plugins', label: t('settingsPage.primary.plugins'), icon: Blocks },
    { key: 'skills', label: t('settingsPage.primary.skills'), icon: Sparkles },
    { key: 'memory', label: t('settingsPage.primary.memory'), icon: Brain },
    { key: 'about', label: t('settingsPage.primary.about'), icon: Info },
  ]

  return (
    <aside className="flex h-full w-64 shrink-0 flex-col border-r border-border/50 bg-card/50 px-3 pb-3 pt-[var(--settings-top-padding)] max-md:w-full">
      {!props.hideHeader && (
        <div className="px-2.5 pb-4 pt-2">
          <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Lemontea</p>
          <h1 className="mt-2 text-2xl font-semibold text-foreground">{t('settings.settings')}</h1>
        </div>
      )}

      <nav className="space-y-1">
        {menuItems.map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            type="button"
            onClick={() => props.onChange(key)}
            className={cn(
              'flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left text-sm transition-colors',
              !props.isH5 && props.activeTab === key
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-accent hover:text-foreground'
            )}
          >
            <Icon size={17} />
            <span className="font-medium">{label}</span>
          </button>
        ))}
      </nav>
    </aside>
  )
}
