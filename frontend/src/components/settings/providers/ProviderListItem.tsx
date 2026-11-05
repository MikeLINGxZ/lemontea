import { useState } from 'react'
import { MoreHorizontal, Star } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ConfirmDialog } from '@/components/settings/common/ConfirmDialog'
import { cn } from '@/lib/utils'
import type { ProviderItem } from '@/types/settings'

/** 渲染服务商摘要并提供启用和默认设置操作，接收组件输入属性，返回组件内容 */
export function ProviderListItem(props: {
  /** 当前条目 */
  item: ProviderItem
  /** 选中项 */
  selected: boolean
  /** 处理选择事件，接收当前条目的唯一编号，无返回值 */
  onSelect: (id: number) => void
  /** 处理切换启用事件，接收当前条目，无返回值 */
  onToggleEnable: (item: ProviderItem) => void
  /** 处理集合默认事件，接收当前条目的唯一编号，无返回值 */
  onSetDefault: (id: number) => void
  /** 处理删除事件，接收当前条目的唯一编号，无返回值 */
  onDelete: (id: number) => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [menuOpen, setMenuOpen] = useState(false) // 菜单展开及其更新方法
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false) // 确认删除展开及其更新方法
  const { item } = props // 当前条目

  /** 关闭菜单，无返回值 */
  const closeMenu = () => { setMenuOpen(false) }

  return (
    <>
      <div
        className={cn(
          'group relative flex w-full items-start gap-3 rounded-2xl px-3 py-3 transition-colors',
          props.selected ? 'bg-primary/10 text-primary' : 'bg-background hover:bg-accent'
        )}
      >
        <div
          role="button"
          tabIndex={0}
          onClick={() => props.onSelect(item.id)}
          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') props.onSelect(item.id) }}
          className="flex min-w-0 flex-1 cursor-pointer items-start gap-3"
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl">
            {item.icon ? (
              <img
                src={item.icon}
                alt={item.provider_name}
                className="h-9 w-9 rounded-xl object-contain"
                onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
              />
            ) : (
              <span className="text-xs font-semibold uppercase text-primary">
                {item.provider_name.slice(0, 2)}
              </span>
            )}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className="truncate text-sm font-medium"
                    title={item.provider_name}
                  >
                    {item.provider_name}
                  </span>
                  {item.is_default && (
                    <span className="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
                      {t('settingsPage.providers.defaultTag')}
                    </span>
                  )}
                </div>
              </div>

              <div className="flex shrink-0 items-center gap-1">
                <button
                  type="button"
                  role="switch"
                  aria-checked={item.enabled}
                  aria-label={t('settingsPage.addProvider.form.enable')}
                  onClick={(e) => { e.stopPropagation(); props.onToggleEnable(item) }}
                  className={cn(
                    'relative inline-flex h-[18px] w-[31px] shrink-0 items-center rounded-full transition-colors duration-200',
                    item.enabled ? 'bg-switch-on' : 'bg-switch-off'
                  )}
                >
                  <span className={cn(
                    'inline-block h-[14px] w-[14px] rounded-full bg-white shadow-xs transition-transform duration-200',
                    item.enabled ? 'translate-x-[15px]' : 'translate-x-[2px]'
                  )} />
                </button>

                <button
                  type="button"
                  aria-label={t('settingsPage.providers.more')}
                  onClick={(e) => { e.stopPropagation(); setMenuOpen(v => !v) }}
                  className={cn(
                    'flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground',
                    menuOpen && 'bg-accent text-foreground'
                  )}
                >
                  <MoreHorizontal size={12} />
                </button>
              </div>
            </div>

            {item.base_url && (
              <div className="mt-1">
                <span
                  className="block truncate text-xs leading-5 text-muted-foreground"
                  title={item.base_url}
                >
                  {item.base_url}
                </span>
              </div>
            )}
          </div>
        </div>

        {menuOpen && (
          <>
            <div className="fixed inset-0 z-10" onClick={closeMenu} />
            <div className="absolute right-0 top-full z-20 mt-1 min-w-36 rounded-lg border border-border bg-popover py-1 shadow-md">
              <button
                type="button"
                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-accent"
                onClick={(e) => { e.stopPropagation(); props.onSetDefault(item.id); closeMenu() }}
              >
                <Star size={13} />
                {t('settingsPage.providers.setDefault')}
              </button>
              <button
                type="button"
                className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-destructive hover:bg-accent"
                onClick={(e) => {
                  e.stopPropagation()
                  setConfirmDeleteOpen(true)
                  closeMenu()
                }}
              >
                {t('settingsPage.providers.delete')}
              </button>
            </div>
          </>
        )}
      </div>

      <ConfirmDialog
        open={confirmDeleteOpen}
        title={t('settingsPage.providers.confirmDelete')}
        description={item.provider_name}
        confirmLabel={t('settingsPage.providers.delete')}
        cancelLabel={t('settingsPage.providers.cancel')}
        confirmTone="danger"
        onConfirm={() => {
          props.onDelete(item.id)
          setConfirmDeleteOpen(false)
        }}
        onCancel={() => setConfirmDeleteOpen(false)}
      />
    </>
  )
}
