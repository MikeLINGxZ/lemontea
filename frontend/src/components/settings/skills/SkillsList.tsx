import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Upload } from 'lucide-react'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { ConfirmDialog } from '@/components/settings/common/ConfirmDialog'
import { useSettingsStore } from '@/store/settingsStore'
import { Skills as SkillsBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills'
import { DeleteSkillInput, ListSkillsInput, ToggleSkillInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills/skills_dto/models'
import { SkillsListItem } from './SkillsListItem'
import { SkillImportDialog } from './SkillImportDialog'
import type { SkillItem } from '@/types/skills'
import { normalizeSkill } from '@/lib/skills'

/** 刷新技能列表并处理启用、导入和删除操作，返回组件内容 */
export function SkillsList() {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const skills = useSettingsStore((s) => s.skills) // 从状态仓库读取技能列表
  const setSkills = useSettingsStore((s) => s.setSkills) // 从状态仓库读取集合技能列表
  const selectedSkillName = useSettingsStore((s) => s.selectedSkillName) // 从状态仓库读取选中项技能名称
  const setSelectedSkillName = useSettingsStore((s) => s.setSelectedSkillName) // 从状态仓库读取集合选中项技能名称
  const updateSkill = useSettingsStore((s) => s.updateSkill) // 从状态仓库读取更新技能
  const removeSkill = useSettingsStore((s) => s.removeSkill) // 从状态仓库读取移除技能
  const [importOpen, setImportOpen] = useState(false) // 导入展开及其更新方法
  const [deleteTarget, setDeleteTarget] = useState<SkillItem | null>(null) // 删除目标及其更新方法
  const [deleting, setDeleting] = useState(false) // 删除中及其更新方法

  /** 刷新处理，完成后结束异步任务 */
  const refresh = useCallback(async () => {
    const result = await SkillsBinding.ListSkills(new ListSkillsInput()) // 保存本次调用的结果
    setSkills((result?.skills ?? []).map(normalizeSkill))
  }, [setSkills])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    void refresh()
  }, [refresh])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 处理焦点，无返回值 */
    const handleFocus = () => {
      void refresh()
    }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [refresh])

  /** 处理切换，接收当前条目，完成后结束异步任务 */
  const handleToggle = useCallback(async (item: SkillItem) => {
    const nextDisabled = !item.disabled // 下一项禁用状态
    const optimistic: SkillItem = { ...item, disabled: nextDisabled } // 乐观更新
    updateSkill(optimistic)
    try {
      /** 保存本次调用的结果 */
      const result = await SkillsBinding.ToggleSkill(new ToggleSkillInput({ name: item.name, disabled: nextDisabled }))
      if (result?.skill) {
        updateSkill(normalizeSkill(result.skill))
      }
    } catch {
      updateSkill(item)
    }
  }, [updateSkill])

  /** 处理删除，完成后结束异步任务 */
  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    if (deleting) return
    setDeleting(true)
    try {
      await SkillsBinding.DeleteSkill(new DeleteSkillInput({ name: deleteTarget.name }))
      removeSkill(deleteTarget.name)
      if (selectedSkillName === deleteTarget.name) {
        setSelectedSkillName(null)
      }
      setDeleteTarget(null)
    } finally {
      setDeleting(false)
    }
  }, [deleteTarget, deleting, removeSkill, selectedSkillName, setSelectedSkillName])

  /** 处理开始新，无返回值 */
  const handleStartNew = () => { void Window.OpenAddSkill({}) }

  return (
    <div className="flex h-full flex-col gap-2 p-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">{t('settingsPage.skills.title')}</h3>
        <div className="flex gap-1">
          <button
            type="button"
            aria-label={t('settingsPage.skills.addNew')}
            onClick={handleStartNew}
            className="rounded-md p-1 hover:bg-muted"
          >
            <Plus size={16} />
          </button>
          <button
            type="button"
            aria-label={t('settingsPage.skills.import')}
            onClick={() => setImportOpen(true)}
            className="rounded-md p-1 hover:bg-muted"
          >
            <Upload size={16} />
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        {skills.length === 0 ? (
          <p className="px-2 py-6 text-center text-sm opacity-60">{t('settingsPage.skills.empty')}</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {skills.map((s) => (
              <li key={s.name}>
                <SkillsListItem
                  item={s}
                  selected={selectedSkillName === s.name}
                  onSelect={setSelectedSkillName}
                  onToggle={handleToggle}
                  onDelete={setDeleteTarget}
                />
              </li>
            ))}
          </ul>
        )}
      </div>
      <SkillImportDialog open={importOpen} onClose={() => setImportOpen(false)} onImported={() => { void refresh() }} />
      <ConfirmDialog
        open={deleteTarget !== null}
        title={t('settingsPage.skills.deleteConfirm')}
        description={deleteTarget?.name}
        confirmLabel={t('settingsPage.skills.delete')}
        cancelLabel={t('settingsPage.skills.editor.cancel')}
        confirmTone="danger"
        busy={deleting}
        onConfirm={() => { void handleDelete() }}
        onCancel={() => {
          if (!deleting) setDeleteTarget(null)
        }}
      />
    </div>
  )
}
