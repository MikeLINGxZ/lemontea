import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsDirtyGuard } from '@/components/settings/common/SettingsDirtyGuard'
import { SettingsFieldRow } from '@/components/settings/common/SettingsFieldRow'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { cn } from '@/lib/utils'
import type { FontSize } from '@/types'

/** 渲染显示设置视图界面，接收组件输入属性，返回组件内容 */
export function DisplaySettingsView(props: {
  /** 值 */
  value: FontSize
  /** 草稿 */
  draft: FontSize
  /** 处理草稿变更事件，接收更新后的值，无返回值 */
  onDraftChange: (next: FontSize) => void
  /** 处理应用事件，无返回值 */
  onApply: () => void
  /** 处理重置事件，无返回值 */
  onReset: () => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [previewFontSize, setPreviewFontSize] = useState<FontSize>(props.draft) // 预览字体尺寸及其更新方法
  /** 可用选项列表 */
  const options: { /* 值 */ value: FontSize; /* 标签 */ label: string; /* 预览 */ preview: string }[] = [
    { value: 'xs', label: t('settingsPage.fontSizes.xs'), preview: t('settingsPage.fontPreview.xs') },
    { value: 'sm', label: t('settingsPage.fontSizes.sm'), preview: t('settingsPage.fontPreview.sm') },
    { value: 'md', label: t('settingsPage.fontSizes.md'), preview: t('settingsPage.fontPreview.md') },
    { value: 'lg', label: t('settingsPage.fontSizes.lg'), preview: t('settingsPage.fontPreview.lg') },
    { value: 'xl', label: t('settingsPage.fontSizes.xl'), preview: t('settingsPage.fontPreview.xl') },
  ]

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    setPreviewFontSize(props.draft)
  }, [props.draft])

  const dirty = props.value !== previewFontSize // 有未保存修改

  return (
    <SettingsDirtyGuard dirty={dirty}>
      <SettingsContentLayout
        header={
          <SettingsPanelHeader
            title={t('settingsPage.general.display.title')}
            description={t('settingsPage.general.display.description')}
          />
        }
        footprint={
          <SettingsActionBar
            primaryLabel={t('settingsPage.actions.apply')}
            secondaryLabel={t('settingsPage.actions.reset')}
            primaryDisabled={!dirty}
            onPrimaryClick={props.onApply}
            onSecondaryClick={props.onReset}
          />
        }
      >
        <div className="space-y-4 pt-6">
          <SettingsFieldRow
            label={t('settingsPage.general.display.textSizeLabel')}
            description={t('settingsPage.general.display.textSizeDescription')}
          >
            <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-5">
              {options.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  aria-label={option.label}
                  onClick={() => {
                    setPreviewFontSize(option.value)
                    props.onDraftChange(option.value)
                  }}
                  className={cn(
                    'flex min-h-20 flex-col justify-between rounded-xl border px-2.5 py-2.5 text-left transition-colors',
                    previewFontSize === option.value
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border bg-background hover:bg-accent'
                  )}
                >
                  <span className="block text-[11px] font-semibold tracking-wide">{option.label}</span>
                  <span className="mt-1.5 block text-lg leading-none">{option.preview}</span>
                </button>
              ))}
            </div>
          </SettingsFieldRow>

          <SettingsFieldRow
            label={t('settingsPage.general.display.previewLabel')}
            description={t('settingsPage.general.display.previewDescription')}
          >
            <div className="rounded-2xl border border-border/70 bg-card/60 p-5">
              <p className={cn(
                'text-foreground',
                previewFontSize === 'xs' && 'text-xs',
                previewFontSize === 'sm' && 'text-sm',
                previewFontSize === 'md' && 'text-base',
                previewFontSize === 'lg' && 'text-lg',
                previewFontSize === 'xl' && 'text-xl'
              )}
              >
                {t('settingsPage.general.display.previewBody')}
              </p>
            </div>
          </SettingsFieldRow>
        </div>
      </SettingsContentLayout>
    </SettingsDirtyGuard>
  )
}
