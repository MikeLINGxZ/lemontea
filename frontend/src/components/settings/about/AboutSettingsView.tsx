import { useTranslation } from 'react-i18next'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'

/** 渲染关于设置视图界面，接收组件输入属性，返回组件内容 */
export function AboutSettingsView(props: { /* 版本 */ version: string }) {
  const { t } = useTranslation() // 获取当前语言的翻译方法

  return (
    <SettingsContentLayout
      header={
        <SettingsPanelHeader
          title="Lemontea"
          description={t('settingsPage.about.description')}
          aside={<span className="rounded-full bg-primary/10 px-3 py-1 text-sm text-primary">{props.version}</span>}
        />
      }
    />
  )
}
