import { Plus } from 'lucide-react'
import { useContext } from 'react'
import { useTranslation } from 'react-i18next'
import { SettingsMenuContext } from '@/components/settings/SettingsShell'
import { ProviderListItem } from '@/components/settings/providers/ProviderListItem'
import type { ProviderItem } from '@/types/settings'

/** 渲染模型服务商列表界面，接收组件输入属性，返回组件内容 */
export function ProviderList(props: {
  /** 条目列表 */
  items: ProviderItem[]
  /** 当前选中项编号 */
  selectedId: number | null
  /** 处理选择事件，接收当前条目的唯一编号，无返回值 */
  onSelect: (id: number) => void
  /** 处理创建事件，无返回值 */
  onCreate: () => void
  /** 处理切换启用事件，接收当前条目，无返回值 */
  onToggleEnable: (item: ProviderItem) => void
  /** 处理集合默认事件，接收当前条目的唯一编号，无返回值 */
  onSetDefault: (id: number) => void
  /** 处理删除事件，接收当前条目的唯一编号，无返回值 */
  onDelete: (id: number) => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const { isH5, onCollapseSubmenu } = useContext(SettingsMenuContext) // 取出是否采用窄屏布局、on折叠二级菜单

  return (
    <div className="flex h-full flex-col">
      <div className="mb-4 flex items-center justify-between gap-3 px-2">
        <h2 className="text-lg font-semibold text-foreground">{t('settingsPage.primary.providers')}</h2>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={props.onCreate}
            className="inline-flex h-8 items-center gap-1.5 rounded-lg px-2 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            aria-label={t('settingsPage.providers.add')}
          >
            <Plus size={14} />
            <span>{t('settingsPage.providers.add')}</span>
          </button>
        </div>
      </div>

      <div className="space-y-2">
        {props.items.map((item) => (
          <ProviderListItem
            key={item.id}
            item={item}
            selected={!isH5 && item.id === props.selectedId}
            onSelect={(id) => { props.onSelect(id); onCollapseSubmenu() }}
            onToggleEnable={props.onToggleEnable}
            onSetDefault={props.onSetDefault}
            onDelete={props.onDelete}
          />
        ))}
      </div>
    </div>
  )
}
