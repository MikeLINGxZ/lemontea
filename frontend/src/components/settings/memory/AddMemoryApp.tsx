import { useState } from 'react'
import { Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Memory as MemoryBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory'
import { CreateMemoryInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory/memory_dto/models'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { AlertViewport } from '@/components/alert/AlertViewport'
import { AppSettingsSyncProvider } from '@/components/providers/AppSettingsSyncProvider'
import { AlertEventProvider } from '@/components/providers/AlertEventProvider'
import { FontSizeProvider } from '@/components/providers/FontSizeProvider'
import { ThemeProvider } from '@/components/providers/ThemeProvider'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsSelect } from '@/components/settings/common/SettingsSelect'
import { cn } from '@/lib/utils'

/** 记忆的编辑草稿 */
type MemoryDraft = {
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

/** 空草稿 */
const EMPTY_DRAFT: MemoryDraft = {
  summary: '',
  content: '',
  type: 'information',
  target: 'user',
  importance: 70,
  confidence: 85,
  pinned: false,
}

/** 新增记忆应用，返回组件内容 */
export function AddMemoryApp() {
  return (
    <AppSettingsSyncProvider>
      <AlertEventProvider>
        <ThemeProvider>
          <FontSizeProvider>
            <AddMemoryWindow />
            <AlertViewport />
          </FontSizeProvider>
        </ThemeProvider>
      </AlertEventProvider>
    </AppSettingsSyncProvider>
  )
}

/** 新增记忆窗口，返回组件内容 */
function AddMemoryWindow() {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [draft, setDraft] = useState<MemoryDraft>(EMPTY_DRAFT) // 草稿及其更新方法
  const [saving, setSaving] = useState(false) // 保存中及其更新方法

  /** 处理保存，完成后结束异步任务 */
  const handleSave = async () => {
    if (saving || (!draft.summary.trim() && !draft.content.trim())) return
    setSaving(true)
    try {
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
      await Window.CloseAddMemory({})
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background text-foreground">
      <div className="shrink-0 px-6 pb-3 pt-12">
        <h1 className="text-lg font-semibold">{t('settingsPage.memory.new')}</h1>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 pb-4">
        <div className="space-y-5">
          <label className="block">
            <span className="mb-1.5 block text-sm font-medium text-foreground">{t('settingsPage.memory.summary')}</span>
            <input
              value={draft.summary}
              onChange={(event) => setDraft({ ...draft, summary: event.target.value })}
              className="h-11 w-full appearance-none rounded-xl border border-border/80 bg-card/80 px-3.5 text-sm text-foreground outline-hidden transition-colors placeholder:text-muted-foreground focus:border-primary"
            />
          </label>

          <label className="block">
            <span className="mb-1.5 block text-sm font-medium text-foreground">{t('settingsPage.memory.content')}</span>
            <textarea
              value={draft.content}
              onChange={(event) => setDraft({ ...draft, content: event.target.value })}
              rows={8}
              className="w-full resize-none appearance-none rounded-xl border border-border/80 bg-card/80 px-3.5 py-3 text-sm leading-6 text-foreground outline-hidden transition-colors placeholder:text-muted-foreground focus:border-primary"
            />
          </label>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-foreground">{t('settingsPage.memory.target')}</span>
              <SettingsSelect
                ariaLabel={t('settingsPage.memory.target')}
                value={draft.target}
                onChange={(value) => setDraft({ ...draft, target: value })}
                options={[
                  { value: 'user', label: t('settingsPage.memory.targetUser') },
                  { value: 'memory', label: t('settingsPage.memory.targetMemory') },
                ]}
              />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-foreground">{t('settingsPage.memory.type')}</span>
              <SettingsSelect
                ariaLabel={t('settingsPage.memory.type')}
                value={draft.type}
                onChange={(value) => setDraft({ ...draft, type: value })}
                options={[
                  { value: 'fact', label: t('settingsPage.memory.typeFact') },
                  { value: 'information', label: t('settingsPage.memory.typeInformation') },
                  { value: 'event', label: t('settingsPage.memory.typeEvent') },
                ]}
              />
            </label>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-foreground">{t('settingsPage.memory.importance')}</span>
              <input
                type="text"
                inputMode="numeric"
                value={String(draft.importance)}
                onChange={(event) => setDraft({ ...draft, importance: Number(event.target.value) })}
                className="h-11 w-full appearance-none rounded-xl border border-border/80 bg-card/80 px-3.5 text-sm text-foreground outline-hidden transition-colors placeholder:text-muted-foreground focus:border-primary"
              />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-foreground">{t('settingsPage.memory.confidence')}</span>
              <input
                type="text"
                inputMode="numeric"
                value={String(draft.confidence)}
                onChange={(event) => setDraft({ ...draft, confidence: Number(event.target.value) })}
                className="h-11 w-full appearance-none rounded-xl border border-border/80 bg-card/80 px-3.5 text-sm text-foreground outline-hidden transition-colors placeholder:text-muted-foreground focus:border-primary"
              />
            </label>
          </div>

          <button
            type="button"
            aria-pressed={draft.pinned}
            onClick={() => setDraft({ ...draft, pinned: !draft.pinned })}
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

      <div className="shrink-0 px-6 pb-6 pt-5">
        <SettingsActionBar
          primaryLabel={saving ? '...' : t('settingsPage.memory.save')}
          primaryDisabled={saving || (!draft.summary.trim() && !draft.content.trim())}
          onPrimaryClick={() => { void handleSave() }}
          secondaryLabel={t('settingsPage.providers.cancel')}
          onSecondaryClick={() => { void Window.CloseAddMemory({}) }}
        />
      </div>
    </div>
  )
}
