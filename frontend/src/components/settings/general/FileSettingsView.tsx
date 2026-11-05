import { FolderOpen } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsDirtyGuard } from '@/components/settings/common/SettingsDirtyGuard'
import { SettingsFieldRow } from '@/components/settings/common/SettingsFieldRow'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'

/** 渲染文件设置视图界面，接收组件输入属性，返回组件内容 */
export function FileSettingsView(props: {
  /** 数据目录 */
  dataDir: string
  /** 当前数据目录 */
  currentDataDir: string
  /** 有未保存修改 */
  dirty: boolean
  /** 处理选择文件夹事件，无返回值 */
  onSelectFolder: () => void
  /** 处理应用事件，无返回值 */
  onApply: () => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法

  return (
    <SettingsDirtyGuard dirty={props.dirty}>
      <SettingsContentLayout
        header={
          <SettingsPanelHeader
            title={t('settingsPage.general.file.title')}
            description={t('settingsPage.general.file.description')}
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
            label={t('settingsPage.general.file.dataDirLabel')}
            description={t('settingsPage.general.file.dataDirDescription')}
          >
            <div className="flex items-center gap-3">
              <input
                value={props.dataDir}
                readOnly
                className="w-full rounded-xl border border-border bg-background px-3 py-2"
                placeholder={props.currentDataDir}
              />
              <button
                type="button"
                aria-label="Select folder"
                onClick={props.onSelectFolder}
                className="inline-flex h-10 shrink-0 items-center justify-center rounded-xl border border-border px-3 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                <FolderOpen size={16} />
              </button>
            </div>
          </SettingsFieldRow>
        </div>
      </SettingsContentLayout>
    </SettingsDirtyGuard>
  )
}
