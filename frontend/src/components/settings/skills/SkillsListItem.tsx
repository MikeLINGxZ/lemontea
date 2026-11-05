import { cn } from '@/lib/utils'
import { Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SkillItem, SkillSource } from '@/types/skills'

/** 技能来源到翻译键后缀的映射 */
const sourceKeySuffix: Record<SkillSource, string> = {
  builtin: 'Builtin',
  user: 'User',
  ai: 'AI',
}

/** 技能列表列表条目组件接收的属性 */
type Props = {
  /** 当前条目 */
  item: SkillItem
  /** 选中项 */
  selected: boolean
  /** 处理选择事件，接收名称，无返回值 */
  onSelect: (name: string) => void
  /** 处理切换事件，接收当前条目，无返回值 */
  onToggle: (item: SkillItem) => void
  /** 处理删除事件，接收当前条目，无返回值 */
  onDelete: (item: SkillItem) => void
}

/** 渲染技能列表列表条目界面，接收当前条目、选中项、on选择、on切换、on删除，返回组件内容 */
export function SkillsListItem({ item, selected, onSelect, onToggle, onDelete }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const badge = t(`settingsPage.skills.source${sourceKeySuffix[item.source]}`) // 徽标
  const canDelete = item.source !== 'builtin' // 能否删除

  return (
    <div
      className={cn(
        'group relative flex w-full items-start gap-3 rounded-2xl px-3 py-3 transition-colors',
        selected ? 'bg-primary/10 text-primary' : 'bg-background hover:bg-accent'
      )}
    >
      <div
        role="button"
        tabIndex={0}
        onClick={() => onSelect(item.name)}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            onSelect(item.name)
          }
        }}
        className="flex min-w-0 flex-1 cursor-pointer items-start"
      >
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate text-sm font-medium text-foreground" title={item.name}>
              {item.name}
            </span>
            <span className="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
              {badge}
            </span>
          </div>
          <p className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
            {item.description}
          </p>
        </div>
      </div>

      <div className="flex shrink-0 flex-col items-center gap-2">
        <button
          type="button"
          role="switch"
          aria-checked={!item.disabled}
          aria-label={t('settingsPage.skills.enabled')}
          onClick={(event) => {
            event.stopPropagation()
            onToggle(item)
          }}
          className={cn(
            'relative mt-0.5 inline-flex h-[18px] w-[31px] items-center rounded-full transition-colors duration-200',
            item.disabled ? 'bg-switch-off' : 'bg-switch-on'
          )}
        >
          <span
            className={cn(
              'inline-block h-[14px] w-[14px] rounded-full bg-white shadow-xs transition-transform duration-200',
              item.disabled ? 'translate-x-[2px]' : 'translate-x-[15px]'
            )}
          />
        </button>
        {canDelete ? (
          <button
            type="button"
            aria-label={t('settingsPage.skills.delete')}
            title={t('settingsPage.skills.delete')}
            onClick={(event) => {
              event.stopPropagation()
              onDelete(item)
            }}
            className="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
          >
            <Trash2 size={14} />
          </button>
        ) : null}
      </div>
    </div>
  )
}
