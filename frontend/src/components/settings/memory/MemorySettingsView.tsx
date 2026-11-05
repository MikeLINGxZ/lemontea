import { useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { Check, ChevronDown, Plus, RotateCcw, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Memory as MemoryBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { CreateMemoryInput, ForgetMemoryInput, ListMemoriesInput, RestoreMemoryInput, UpdateMemoryInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory/memory_dto/models'
import type { MemoryItem } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory/memory_dto/models'
import { SettingsMenuContext } from '@/components/settings/SettingsShell'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { SettingsSelect } from '@/components/settings/common/SettingsSelect'
import { cn } from '@/lib/utils'

/** 记忆的编辑草稿 */
type MemoryDraft = {
  /** 当前条目的唯一编号 */
  id: number | null
  /** 摘要 */
  summary: string
  /** 内容 */
  content: string
  /** 类型 */
  type: string
  /** 要处理的目标 */
  target: string
  /** 重要度 */
  importance: number
  /** 置信度 */
  confidence: number
  /** 已置顶 */
  pinned: boolean
}

/** 定义记忆控制器的数据结构 */
type MemoryController = {
  /** 条目列表 */
  items: MemoryItem[]
  /** 选中项 */
  selected: MemoryItem | null
  /** 当前选中项编号 */
  selectedId: number | null
  /** 草稿 */
  draft: MemoryDraft
  /** 草稿有未保存修改 */
  draftDirty: boolean
  /** 搜索词 */
  query: string
  /** 要处理的目标 */
  target: string
  /** 类型 */
  type: string
  /** 包含已遗忘 */
  includeForgotten: boolean
  /** 加载中 */
  loading: boolean
  /** 设置选中项编号，接收当前条目的唯一编号，无返回值 */
  setSelectedId: (id: number | null) => void
  /** 设置草稿，接收草稿，无返回值 */
  setDraft: (draft: MemoryDraft) => void
  /** 重置草稿，无返回值 */
  resetDraft: () => void
  /** 设置搜索词，接收值，无返回值 */
  setQuery: (value: string) => void
  /** 设置要处理的目标，接收值，无返回值 */
  setTarget: (value: string) => void
  /** 设置类型，接收值，无返回值 */
  setType: (value: string) => void
  /** 设置包含已遗忘，接收值，无返回值 */
  setIncludeForgotten: (value: boolean) => void
  /** 加载记忆列表，完成后结束异步任务 */
  loadMemories: () => Promise<void>
  /** 将当前草稿保存到对应的后端服务，完成后结束异步任务 */
  saveDraft: () => Promise<void>
  /** 遗忘选中项，完成后结束异步任务 */
  forgetSelected: () => Promise<void>
  /** 恢复选中项，完成后结束异步任务 */
  restoreSelected: () => Promise<void>
  /** 启动新，无返回值 */
  startNew: () => void
}

/** 空草稿 */
const EMPTY_DRAFT: MemoryDraft = {
  id: null,
  summary: '',
  content: '',
  type: 'information',
  target: 'user',
  importance: 70,
  confidence: 85,
  pinned: false,
}

/** 记忆草稿 相同处理，接收左侧、右侧，返回是否满足条件 */
function memoryDraftsEqual(left: MemoryDraft, right: MemoryDraft): boolean {
  return left.id === right.id &&
    left.summary === right.summary &&
    left.content === right.content &&
    left.type === right.type &&
    left.target === right.target &&
    left.importance === right.importance &&
    left.confidence === right.confidence &&
    left.pinned === right.pinned
}

/** 统一管理记忆列表、筛选条件和编辑草稿，返回页面操作与状态，接收当前，返回对应的处理结果 */
export function useMemoryController(active: boolean): MemoryController {
  const [items, setItems] = useState<MemoryItem[]>([]) // 条目列表及其更新方法
  const [selectedId, setSelectedId] = useState<number | null>(null) // 当前选中项编号及其更新方法
  const [query, setQuery] = useState('') // 搜索词及其更新方法
  const [target, setTarget] = useState('') // 要处理的目标及其更新方法
  const [type, setType] = useState('') // 类型及其更新方法
  const [includeForgotten, setIncludeForgotten] = useState(false) // 包含已遗忘及其更新方法
  const [loading, setLoading] = useState(false) // 加载中及其更新方法
  const [draft, setDraft] = useState<MemoryDraft>(EMPTY_DRAFT) // 草稿及其更新方法
  const [initialDraft, setInitialDraft] = useState<MemoryDraft>(EMPTY_DRAFT) // 初始草稿及其更新方法

  /** 选中项 */
  const selected = useMemo(
    () => items.find((item) => item.id === selectedId) ?? null,
    [items, selectedId],
  )

  /** 草稿有未保存修改 */
  const draftDirty = useMemo(
    () => !memoryDraftsEqual(draft, initialDraft),
    [draft, initialDraft],
  )

  /** 加载记忆列表，完成后结束异步任务 */
  const loadMemories = useCallback(async () => {
    setLoading(true)
    try {
      /** 保存列表结果 */
      const listResult = await MemoryBinding.ListMemories(new ListMemoriesInput({
        query,
        target,
        type,
        include_forgotten: includeForgotten,
        offset: 0,
        limit: 100,
      }))
      const nextItems = listResult?.items ?? [] // 下一项条目列表
      setItems(nextItems)
      setSelectedId((current) => {
        if (current !== null && nextItems.some((item) => item.id === current)) return current
        return nextItems[0]?.id ?? null
      })
    } finally {
      setLoading(false)
    }
  }, [includeForgotten, query, target, type])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (active) {
      void loadMemories()
    }
  }, [active, loadMemories])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!active) return
    /** 处理焦点，无返回值 */
    const handleFocus = () => { void loadMemories() }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [active, loadMemories])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!selected) {
      setDraft(EMPTY_DRAFT)
      setInitialDraft(EMPTY_DRAFT)
      return
    }
    /** 下一项草稿 */
    const nextDraft: MemoryDraft = {
      id: selected.id,
      summary: selected.summary,
      content: selected.content,
      type: selected.type,
      target: selected.target,
      importance: selected.importance,
      confidence: selected.confidence,
      pinned: selected.pinned,
    }
    setDraft(nextDraft)
    setInitialDraft(nextDraft)
  }, [selected])

  /** 启动新，无返回值 */
  const startNew = useCallback(() => {
    setSelectedId(null)
    setDraft(EMPTY_DRAFT)
    setInitialDraft(EMPTY_DRAFT)
  }, [])

  /** 重置草稿，无返回值 */
  const resetDraft = useCallback(() => {
    setDraft(initialDraft)
  }, [initialDraft])

  /** 将当前草稿保存到对应的后端服务，完成后结束异步任务 */
  const saveDraft = useCallback(async () => {
    if (draft.id === null) {
      await MemoryBinding.CreateMemory(new CreateMemoryInput({
        summary: draft.summary,
        content: draft.content,
        type: draft.type,
        target: draft.target,
        source: 'manual',
        importance: draft.importance,
        confidence: draft.confidence,
        pinned: draft.pinned,
      }))
    } else {
      await MemoryBinding.UpdateMemory(new UpdateMemoryInput({
        id: draft.id,
        summary: draft.summary,
        content: draft.content,
        type: draft.type,
        target: draft.target,
        source: selected?.source ?? 'manual',
        importance: draft.importance,
        confidence: draft.confidence,
        pinned: draft.pinned,
      }))
    }
    await loadMemories()
  }, [draft, loadMemories, selected?.source])

  /** 遗忘选中项，完成后结束异步任务 */
  const forgetSelected = useCallback(async () => {
    if (!selected) return
    await MemoryBinding.ForgetMemory(new ForgetMemoryInput({ id: selected.id }))
    await loadMemories()
  }, [loadMemories, selected])

  /** 恢复选中项，完成后结束异步任务 */
  const restoreSelected = useCallback(async () => {
    if (!selected) return
    await MemoryBinding.RestoreMemory(new RestoreMemoryInput({ id: selected.id }))
    await loadMemories()
  }, [loadMemories, selected])

  return {
    items,
    selected,
    selectedId,
    draft,
    draftDirty,
    query,
    target,
    type,
    includeForgotten,
    loading,
    setSelectedId,
    setDraft,
    resetDraft,
    setQuery,
    setTarget,
    setType,
    setIncludeForgotten,
    loadMemories,
    saveDraft,
    forgetSelected,
    restoreSelected,
    startNew,
  }
}

/** 在左侧列表搜索和筛选记忆，并切换当前编辑项，接收组件输入属性，返回组件内容 */
export function MemoryList(props: { /* 控制器 */ controller: MemoryController }) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const { isH5, onCollapseSubmenu } = useContext(SettingsMenuContext) // 取出是否采用窄屏布局、on折叠二级菜单
  const controller = props.controller // 控制器

  return (
    <div className="flex h-full flex-col gap-3 p-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">{t('settingsPage.memory.title')}</h3>
        <button
          type="button"
          aria-label={t('settingsPage.memory.new')}
          onClick={() => {
            void Window.OpenAddMemory({})
            onCollapseSubmenu()
          }}
          className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <Plus size={16} />
        </button>
      </div>

      <label className="flex h-8 items-center gap-2 rounded-lg border border-border/80 bg-card/80 px-2.5 text-xs text-foreground transition-colors focus-within:border-primary">
        <Search size={14} className="text-muted-foreground" />
        <input
          value={controller.query}
          onChange={(event) => controller.setQuery(event.target.value)}
          placeholder={t('settingsPage.memory.search')}
          className="min-w-0 flex-1 appearance-none bg-transparent text-xs outline-hidden placeholder:text-muted-foreground"
        />
      </label>

      <div className="grid grid-cols-2 gap-2">
        <CompactSelect
          ariaLabel={t('settingsPage.memory.target')}
          value={controller.target}
          onChange={controller.setTarget}
          options={[
            { value: '', label: t('settingsPage.memory.allTargets') },
            { value: 'user', label: t('settingsPage.memory.targetUser') },
            { value: 'memory', label: t('settingsPage.memory.targetMemory') },
          ]}
        />
        <CompactSelect
          ariaLabel={t('settingsPage.memory.type')}
          value={controller.type}
          onChange={controller.setType}
          options={[
            { value: '', label: t('settingsPage.memory.allTypes') },
            { value: 'fact', label: t('settingsPage.memory.typeFact') },
            { value: 'information', label: t('settingsPage.memory.typeInformation') },
            { value: 'event', label: t('settingsPage.memory.typeEvent') },
          ]}
        />
      </div>

      <button
        type="button"
        role="switch"
        aria-checked={controller.includeForgotten}
        onClick={() => controller.setIncludeForgotten(!controller.includeForgotten)}
        className="flex items-center justify-between rounded-xl px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <span>{t('settingsPage.memory.showForgotten')}</span>
        <span className={cn(
          'relative inline-flex h-[18px] w-[31px] shrink-0 items-center rounded-full transition-colors duration-200',
          controller.includeForgotten ? 'bg-primary' : 'bg-muted',
        )}>
          <span className={cn(
            'inline-block h-[14px] w-[14px] rounded-full bg-white shadow-xs transition-transform duration-200',
            controller.includeForgotten ? 'translate-x-[15px]' : 'translate-x-[2px]',
          )} />
        </span>
      </button>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {controller.items.length === 0 && !controller.loading ? (
          <p className="px-2 py-6 text-center text-sm text-muted-foreground">{t('settingsPage.memory.empty')}</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {controller.items.map((item) => (
              <li key={item.id}>
                <button
                  type="button"
                  onClick={() => {
                    controller.setSelectedId(item.id)
                    onCollapseSubmenu()
                  }}
                  className={cn(
                    'group flex w-full items-start rounded-xl px-3 py-2.5 text-left transition-colors',
                    !isH5 && item.id === controller.selectedId
                      ? 'bg-primary/10 text-primary'
                      : 'bg-background hover:bg-accent',
                  )}
                >
                  <span className="min-w-0 flex-1">
                    <span className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-sm font-medium" title={item.summary}>
                        {item.summary || t('settingsPage.memory.untitled')}
                      </span>
                      {item.is_forgotten ? (
                        <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                          {t('settingsPage.memory.forgotten')}
                        </span>
                      ) : null}
                    </span>
                    <span className="mt-1 block line-clamp-2 text-xs leading-5 text-muted-foreground">
                      {item.content}
                    </span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

/** 编辑选中的记忆并提供保存、遗忘和恢复操作，接收组件输入属性，返回组件内容 */
export function MemorySettingsView(props: { /* 控制器 */ controller: MemoryController }) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const { isH5 } = useContext(SettingsMenuContext) // 是否采用窄屏布局
  const { controller } = props // 控制器
  const { draft, selected } = controller // 取出草稿、选中项

  return (
    <SettingsContentLayout
      noContentScroll
      header={
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/50 py-5">
          <SettingsPanelHeader
            title={draft.id === null ? t('settingsPage.memory.new') : (selected?.summary || t('settingsPage.memory.untitled'))}
          />
          <button
            type="button"
            onClick={() => { void controller.loadMemories() }}
            className="inline-flex h-9 items-center gap-2 rounded-lg px-3 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <RotateCcw size={15} />
            {t('settingsPage.memory.refresh')}
          </button>
        </div>
      }
      footprint={
        <SettingsActionBar
          primaryLabel={t('settingsPage.memory.save')}
          primaryDisabled={!controller.draftDirty || (!draft.summary.trim() && !draft.content.trim())}
          onPrimaryClick={() => { void controller.saveDraft() }}
          secondaryLabel={controller.draftDirty ? t('settingsPage.providers.cancel') : (selected?.is_forgotten ? t('settingsPage.memory.restore') : undefined)}
          onSecondaryClick={controller.draftDirty ? controller.resetDraft : (selected?.is_forgotten ? () => { void controller.restoreSelected() } : undefined)}
          dangerLabel={selected && !selected.is_forgotten ? t('settingsPage.memory.forget') : undefined}
          onDangerClick={selected && !selected.is_forgotten ? () => { void controller.forgetSelected() } : undefined}
        />
      }
    >
      <div className={cn(
        'flex min-h-0 flex-1 flex-col overflow-y-auto pb-4 pt-6',
        isH5 ? '-mr-5 pr-5' : '-mr-10 pr-10',
      )}>
        <div className="max-w-3xl space-y-5">
          <Field label={t('settingsPage.memory.summary')}>
            <TextInput value={draft.summary} onChange={(value) => controller.setDraft({ ...draft, summary: value })} />
          </Field>

          <Field label={t('settingsPage.memory.content')}>
            <textarea
              value={draft.content}
              onChange={(event) => controller.setDraft({ ...draft, content: event.target.value })}
              rows={8}
              className="w-full resize-none appearance-none rounded-xl border border-border/80 bg-card/80 px-3.5 py-3 text-sm leading-6 text-foreground outline-hidden transition-colors placeholder:text-muted-foreground focus:border-primary"
            />
          </Field>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label={t('settingsPage.memory.target')}>
              <SettingsSelect
                ariaLabel={t('settingsPage.memory.target')}
                value={draft.target}
                onChange={(value) => controller.setDraft({ ...draft, target: value })}
                options={[
                  { value: 'user', label: t('settingsPage.memory.targetUser') },
                  { value: 'memory', label: t('settingsPage.memory.targetMemory') },
                ]}
              />
            </Field>
            <Field label={t('settingsPage.memory.type')}>
              <SettingsSelect
                ariaLabel={t('settingsPage.memory.type')}
                value={draft.type}
                onChange={(value) => controller.setDraft({ ...draft, type: value })}
                options={[
                  { value: 'fact', label: t('settingsPage.memory.typeFact') },
                  { value: 'information', label: t('settingsPage.memory.typeInformation') },
                  { value: 'event', label: t('settingsPage.memory.typeEvent') },
                ]}
              />
            </Field>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label={t('settingsPage.memory.importance')}>
              <TextInput
                type="number"
                value={String(draft.importance)}
                onChange={(value) => controller.setDraft({ ...draft, importance: Number(value) })}
              />
            </Field>
            <Field label={t('settingsPage.memory.confidence')}>
              <TextInput
                type="number"
                value={String(draft.confidence)}
                onChange={(value) => controller.setDraft({ ...draft, confidence: Number(value) })}
              />
            </Field>
          </div>

          <button
            type="button"
            aria-pressed={draft.pinned}
            onClick={() => controller.setDraft({ ...draft, pinned: !draft.pinned })}
            className="inline-flex items-center gap-2 rounded-xl px-2 py-1.5 text-sm text-foreground transition-colors hover:bg-accent"
          >
            <span className={cn(
              'flex h-5 w-5 items-center justify-center rounded-md border transition-colors',
              draft.pinned ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-card',
            )}>
              {draft.pinned ? <Check size={13} /> : null}
            </span>
            {t('settingsPage.memory.pinned')}
          </button>
        </div>
      </div>
    </SettingsContentLayout>
  )
}

/** 渲染紧凑选择界面，接收组件输入属性，返回组件内容 */
function CompactSelect(props: {
  /** 无障碍标签 */
  ariaLabel: string
  /** 值 */
  value: string
  /** 处理本次变更事件，接收值，无返回值 */
  onChange: (value: string) => void
  /** 可用选项列表 */
  options: Array<{ /* 值 */ value: string; /* 标签 */ label: string }>
}) {
  const [open, setOpen] = useState(false) // 展开及其更新方法
  /** 选中项 */
  const selected = props.options.find((option) => option.value === props.value) ?? props.options[0]

  return (
    <div className="relative">
      <button
        type="button"
        aria-label={props.ariaLabel}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        className="flex h-8 w-full items-center justify-between gap-1.5 rounded-lg border border-border/80 bg-card/80 px-2.5 text-left text-xs text-foreground transition-colors hover:bg-accent focus:border-primary focus:outline-hidden"
      >
        <span className="min-w-0 flex-1 truncate">{selected?.label}</span>
        <ChevronDown size={13} className={cn('shrink-0 text-muted-foreground transition-transform', open ? 'rotate-180' : '')} />
      </button>
      {open ? (
        <div className="absolute left-0 right-0 top-full z-20 mt-1 overflow-hidden rounded-lg border border-border bg-popover py-1 shadow-lg">
          {props.options.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => {
                props.onChange(option.value)
                setOpen(false)
              }}
              className={cn(
                'flex h-7 w-full items-center justify-between gap-2 px-2.5 text-left text-xs transition-colors hover:bg-accent',
                option.value === props.value ? 'text-primary' : 'text-foreground',
              )}
            >
              <span className="min-w-0 flex-1 truncate">{option.label}</span>
              {option.value === props.value ? <Check size={12} /> : null}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  )
}

/** 渲染字段界面，接收组件输入属性，返回组件内容 */
function Field(props: { /* 标签 */ label: string; /* 要渲染的子内容 */ children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium text-foreground">{props.label}</span>
      {props.children}
    </label>
  )
}

/** 渲染文本输入界面，接收组件输入属性，返回组件内容 */
function TextInput(props: {
  /** 值 */
  value: string
  /** 处理本次变更事件，接收值，无返回值 */
  onChange: (value: string) => void
  /** 类型 */
  type?: 'text' | 'number'
}) {
  return (
    <input
      type="text"
      inputMode={props.type === 'number' ? 'numeric' : undefined}
      value={props.value}
      onChange={(event) => props.onChange(event.target.value)}
      className="h-11 w-full appearance-none rounded-xl border border-border/80 bg-card/80 px-3.5 text-sm text-foreground outline-hidden transition-colors placeholder:text-muted-foreground focus:border-primary"
    />
  )
}
