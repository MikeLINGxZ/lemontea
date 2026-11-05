import { useCallback, useContext, useEffect, useState } from 'react'
import { LockKeyhole } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsMenuContext } from '@/components/settings/SettingsShell'
import { ConfirmDialog } from '@/components/settings/common/ConfirmDialog'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { normalizeSkill, toSkillName } from '@/lib/skills'
import { useSettingsStore } from '@/store/settingsStore'
import { Skills as SkillsBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills'
import { DeleteSkillInput, GetSkillInput, UpdateSkillInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills/skills_dto/models'
import { SkillEditor } from './SkillEditor'
import { cn } from '@/lib/utils'
import type { SkillFull } from '@/types/skills'

/** 加载当前技能详情并处理保存、改名和删除，返回组件内容 */
export function SkillsDetailPane() {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const { isH5 } = useContext(SettingsMenuContext) // 是否采用窄屏布局
  const selectedSkillName = useSettingsStore((s) => s.selectedSkillName) // 从状态仓库读取选中项技能名称
  const updateSkill = useSettingsStore((s) => s.updateSkill) // 从状态仓库读取更新技能
  const removeSkill = useSettingsStore((s) => s.removeSkill) // 从状态仓库读取移除技能
  const setSelectedSkillName = useSettingsStore((s) => s.setSelectedSkillName) // 从状态仓库读取集合选中项技能名称
  const [fullSkill, setFullSkill] = useState<SkillFull | null>(null) // 完整技能及其更新方法
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false) // 确认删除展开及其更新方法
  const [deleting, setDeleting] = useState(false) // 删除中及其更新方法
  const [nameError, setNameError] = useState<string | null>(null) // 名称错误及其更新方法

  /** 刷新处理，完成后结束异步任务 */
  const refresh = useCallback(async () => {
    if (!selectedSkillName) {
      setFullSkill(null)
      return
    }
    /** 保存本次调用的结果 */
    const result = await SkillsBinding.GetSkill(new GetSkillInput({ name: selectedSkillName }))
    if (result?.skill) {
      setFullSkill(normalizeSkill(result.skill))
      setNameError(null)
    }
  }, [selectedSkillName])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => { void refresh() }, [refresh])

  /** 处理更新，接收草稿，完成后结束异步任务 */
  const handleUpdate = useCallback(async (draft: { /* 名称 */ name: string; /* 描述 */ description: string; /* 正文 */ body: string }) => {
    if (!fullSkill) return
    const nextName = toSkillName(draft.name) // 下一项名称
    if (!nextName) {
      setNameError(t('settingsPage.skills.editor.nameInvalid'))
      return
    }
    setNameError(null)
    /** 保存本次调用的结果 */
    const result = await SkillsBinding.UpdateSkill(new UpdateSkillInput({
      name: fullSkill.name,
      new_name: nextName,
      description: draft.description,
      body: draft.body,
    }))
    if (result?.skill) {
      const nextSkill = normalizeSkill(result.skill) // 下一项技能
      if (nextSkill.name !== fullSkill.name) {
        removeSkill(fullSkill.name)
      }
      updateSkill(nextSkill)
      setFullSkill(nextSkill)
      setSelectedSkillName(nextSkill.name)
    }
  }, [fullSkill, removeSkill, setSelectedSkillName, t, updateSkill])

  /** 处理删除，完成后结束异步任务 */
  const handleDelete = useCallback(async () => {
    if (!fullSkill) return
    if (deleting) return
    setDeleting(true)
    try {
      await SkillsBinding.DeleteSkill(new DeleteSkillInput({ name: fullSkill.name }))
      removeSkill(fullSkill.name)
      setSelectedSkillName(null)
      setConfirmDeleteOpen(false)
    } finally {
      setDeleting(false)
    }
  }, [deleting, fullSkill, removeSkill, setSelectedSkillName])

  if (!fullSkill) {
    return (
      <div className={cn('flex h-full items-center justify-center text-sm opacity-60', isH5 ? 'p-5' : 'p-10')}>
        {t('settingsPage.skills.selectHint')}
      </div>
    )
  }

  const readOnly = fullSkill.source === 'builtin' // 已读仅
  /** 来源标签 */
  const sourceLabel = t(`settingsPage.skills.source${fullSkill.source === 'builtin' ? 'Builtin' : fullSkill.source === 'user' ? 'User' : 'AI'}`)

  return (
    <div className="flex h-full flex-col">
      <div className={cn('pb-4 pt-4', isH5 ? 'px-5' : 'px-10')}>
        <SettingsPanelHeader
          title={fullSkill.name}
          badge={
            <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
              {sourceLabel}
            </span>
          }
        />
      </div>
      {readOnly && (
        <div className={cn(
          'mb-4 flex shrink-0 items-start gap-2 rounded-xl bg-amber-500/10 px-3 py-2.5 text-sm text-amber-700 dark:text-amber-400',
          isH5 ? 'mx-5' : 'mx-10',
        )}>
          <LockKeyhole size={14} className="mt-0.5 shrink-0" />
          <span>
            <span className="font-medium">{t('settingsPage.skills.builtinLockedTitle')}</span>
            {' '}
            {t('settingsPage.skills.builtinLockedMsg')}
          </span>
        </div>
      )}
      <div className={cn('min-h-0 flex-1', isH5 ? 'px-5' : 'px-10')}>
        <SkillEditor
          initial={fullSkill}
          readOnly={readOnly}
          nameError={nameError}
          dangerLabel={t('settingsPage.skills.delete')}
          onDangerClick={() => setConfirmDeleteOpen(true)}
          onSubmit={handleUpdate}
          onCancel={() => setSelectedSkillName(null)}
        />
      </div>
      <ConfirmDialog
        open={confirmDeleteOpen}
        title={t('settingsPage.skills.deleteConfirm')}
        description={fullSkill.name}
        confirmLabel={t('settingsPage.skills.delete')}
        cancelLabel={t('settingsPage.skills.editor.cancel')}
        confirmTone="danger"
        busy={deleting}
        onConfirm={() => { void handleDelete() }}
        onCancel={() => {
          if (!deleting) setConfirmDeleteOpen(false)
        }}
      />
    </div>
  )
}
