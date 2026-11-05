import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { AlertViewport } from '@/components/alert/AlertViewport'
import { AppSettingsSyncProvider } from '@/components/providers/AppSettingsSyncProvider'
import { AlertEventProvider } from '@/components/providers/AlertEventProvider'
import { FontSizeProvider } from '@/components/providers/FontSizeProvider'
import { ThemeProvider } from '@/components/providers/ThemeProvider'
import { SkillEditor } from './SkillEditor'
import { Skills as SkillsBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills'
import { CreateSkillInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills/skills_dto/models'
import { toSkillName } from '@/lib/skills'

/** 保存的编辑草稿 */
type SaveDraft = {
  /** 名称 */
  name: string
  /** 描述 */
  description: string
  /** 正文 */
  body: string
}

/** 新增技能应用，返回组件内容 */
export function AddSkillApp() {
  return (
    <AppSettingsSyncProvider>
      <AlertEventProvider>
        <ThemeProvider>
          <FontSizeProvider>
            <AddSkillWindow />
            <AlertViewport />
          </FontSizeProvider>
        </ThemeProvider>
      </AlertEventProvider>
    </AppSettingsSyncProvider>
  )
}

/** 新增技能窗口，返回组件内容 */
function AddSkillWindow() {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [saving, setSaving] = useState(false) // 保存中及其更新方法
  const [nameError, setNameError] = useState<string | null>(null) // 名称错误及其更新方法

  /** 处理提交，接收草稿，完成后结束异步任务 */
  const handleSubmit = async (draft: SaveDraft) => {
    const name = toSkillName(draft.name) // 名称
    if (!name) {
      setNameError(t('settingsPage.skills.editor.nameInvalid'))
      return
    }
    setNameError(null)
    setSaving(true)
    try {
      await SkillsBinding.CreateSkill(new CreateSkillInput({
        ...draft,
        name,
      }))
      await Window.CloseAddSkill({})
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background text-foreground">
      <div className="shrink-0 px-6 pb-3 pt-12">
        <h1 className="text-lg font-semibold">{t('settingsPage.skills.editor.createTitle')}</h1>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-6">
        <SkillEditor
          initial={{}}
          saving={saving}
          nameError={nameError}
          onSubmit={handleSubmit}
          onCancel={() => { void Window.CloseAddSkill({}) }}
        />
      </div>
    </div>
  )
}
