import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Skills as SkillsBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills'
import { File as FileBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file'
import { ImportSkillInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills/skills_dto/models'
import { SelectFileInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file/file_dto/models'

/** 技能导入对话框组件接收的属性 */
type Props = {
  /** 展开 */
  open: boolean
  /** 处理关闭事件，无返回值 */
  onClose: () => void
  /** 处理已导入事件，无返回值 */
  onImported: () => void
}

/** 渲染技能导入对话框界面，接收展开、on关闭、on已导入，返回组件内容 */
export function SkillImportDialog({ open, onClose, onImported }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [filePath, setFilePath] = useState('') // 文件路径及其更新方法
  const [error, setError] = useState<string | null>(null) // 错误及其更新方法
  const [submitting, setSubmitting] = useState(false) // 正在提交及其更新方法

  if (!open) return null

  /** 处理选择文件，完成后结束异步任务 */
  const handleSelectFile = async () => {
    const result = await FileBinding.SelectFile(new SelectFileInput()) // 保存本次调用的结果
    if (result?.file_path) {
      setFilePath(result.file_path)
      setError(null)
    }
  }

  /** 处理提交，完成后结束异步任务 */
  const handleSubmit = async () => {
    setSubmitting(true)
    setError(null)
    try {
      await SkillsBinding.ImportSkill(new ImportSkillInput({ path: filePath }))
      setFilePath('')
      onImported()
      onClose()
    } catch (err: unknown) {
      setError((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-[600px] max-w-[90vw] rounded-md bg-card p-4 shadow-xl">
        <h2 className="mb-3 text-sm font-semibold">{t('settingsPage.skills.importDialog.title')}</h2>
        <p className="mb-2 text-xs text-muted-foreground">{t('settingsPage.skills.importDialog.hint')}</p>
        <div className="flex items-center gap-2">
          <input
            className="flex-1 rounded-md border px-2 py-1.5 text-xs"
            readOnly
            value={filePath}
            placeholder={t('settingsPage.skills.importDialog.placeholder')}
          />
          <button
            type="button"
            className="shrink-0 rounded-md border px-3 py-1.5 text-sm hover:bg-muted"
            onClick={() => { void handleSelectFile() }}
          >
            {t('settingsPage.skills.import')}
          </button>
        </div>
        {error && <p className="mt-2 text-xs text-red-600">{error}</p>}
        <div className="mt-3 flex justify-end gap-2">
          <button type="button" className="rounded-md border px-3 py-1 text-sm" onClick={onClose}>
            {t('settingsPage.skills.editor.cancel')}
          </button>
          <button
            type="button"
            className="rounded-md bg-primary px-3 py-1 text-sm text-primary-foreground disabled:opacity-50"
            onClick={() => { void handleSubmit() }}
            disabled={submitting || !filePath}
          >
            {t('settingsPage.skills.importDialog.submit')}
          </button>
        </div>
      </div>
    </div>
  )
}
