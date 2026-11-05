import { useTranslation } from 'react-i18next'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsDirtyGuard } from '@/components/settings/common/SettingsDirtyGuard'
import { SettingsFieldRow } from '@/components/settings/common/SettingsFieldRow'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { SettingsSelect } from '@/components/settings/common/SettingsSelect'
import type { Language } from '@/types'
import type { SettingsOption } from '@/types/settings'

/** 渲染地区设置视图界面，接收组件输入属性，返回组件内容 */
export function LocaleSettingsView(props: {
  /** 地区 */
  locale: string
  /** 语言 */
  language: Language
  /** 地区列表 */
  regions: SettingsOption[]
  /** 语言列表 */
  languages: SettingsOption[]
  /** 有未保存修改 */
  dirty: boolean
  /** 处理地区变更事件，接收值，无返回值 */
  onLocaleChange: (value: string) => void
  /** 处理语言变更事件，接收值，无返回值 */
  onLanguageChange: (value: Language) => void
  /** 处理应用事件，无返回值 */
  onApply: () => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  /** 区域选项列表 */
  const regionOptions = props.regions.map((item) => ({ value: item.id, label: item.name, icon: item.icon }))
  const languageOptions = props.languages.map((item) => ({ value: item.id, label: item.name })) // 语言选项列表

  return (
    <SettingsDirtyGuard dirty={props.dirty}>
      <SettingsContentLayout
        header={
          <SettingsPanelHeader
            title={t('settingsPage.general.locale.title')}
            description={t('settingsPage.general.locale.description')}
          />
        }
        footprint={
          <SettingsActionBar
            primaryLabel={t('settingsPage.actions.apply')}
            primaryDisabled={!props.dirty}
            onPrimaryClick={props.onApply}
          />
        }
      >
        <div className="space-y-4 pt-6">
          <SettingsFieldRow
            label={t('settingsPage.general.locale.regionLabel')}
            description={t('settingsPage.general.locale.regionDescription')}
          >
            <SettingsSelect
              ariaLabel={t('settingsPage.general.locale.regionLabel')}
              value={props.locale}
              options={[...regionOptions]}
              onChange={props.onLocaleChange}
            />
          </SettingsFieldRow>

          <SettingsFieldRow
            label={t('settingsPage.general.locale.languageLabel')}
            description={t('settingsPage.general.locale.languageDescription')}
          >
            <SettingsSelect
              ariaLabel={t('settingsPage.general.locale.languageLabel')}
              value={props.language}
              options={[...languageOptions]}
              onChange={(value) => {
                if (value === 'zh-CN' || value === 'en') props.onLanguageChange(value)
              }}
            />
          </SettingsFieldRow>
        </div>
      </SettingsContentLayout>
    </SettingsDirtyGuard>
  )
}
