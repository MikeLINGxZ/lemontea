import { Settings } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings'
import { File } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file'
import { DisplaySettingsView } from '@/components/settings/general/DisplaySettingsView'
import { FileSettingsView } from '@/components/settings/general/FileSettingsView'
import { LocaleSettingsView } from '@/components/settings/general/LocaleSettingsView'
import { ShortcutSettingsView } from '@/components/settings/general/ShortcutSettingsView'
import { useAppStore } from '@/store/appStore'
import { useSettingsStore } from '@/store/settingsStore'

/** 按当前二级标签切换显示、语言、文件或快捷键设置，返回组件内容 */
export function GeneralSettingsPanel() {
  const generalTab = useSettingsStore((state) => state.generalTab) // 从状态仓库读取常规标签页
  const fontSize = useSettingsStore((state) => state.fontSize) // 从状态仓库读取字体尺寸
  const dataDir = useSettingsStore((state) => state.dataDir) // 从状态仓库读取数据目录
  const quickChatShortcut = useSettingsStore((state) => state.quickChatShortcut) // 从状态仓库读取快捷聊天快捷键
  const displayDraft = useSettingsStore((state) => state.displayDraft) // 从状态仓库读取显示草稿
  const localeDraft = useSettingsStore((state) => state.localeDraft) // 从状态仓库读取地区草稿
  const fileDraft = useSettingsStore((state) => state.fileDraft) // 从状态仓库读取文件草稿
  const shortcutDraft = useSettingsStore((state) => state.shortcutDraft) // 从状态仓库读取快捷键草稿
  const localeDirty = useSettingsStore((state) => state.localeDirty) // 从状态仓库读取地区有未保存修改
  const fileDirty = useSettingsStore((state) => state.fileDirty) // 从状态仓库读取文件有未保存修改
  const shortcutDirty = useSettingsStore((state) => state.shortcutDirty) // 从状态仓库读取快捷键有未保存修改
  const shortcutValidation = useSettingsStore((state) => state.shortcutValidation) // 从状态仓库读取快捷键校验
  const languages = useSettingsStore((state) => state.languages) // 从状态仓库读取语言列表
  const regions = useSettingsStore((state) => state.regions) // 从状态仓库读取地区列表
  const setDisplayDraft = useSettingsStore((state) => state.setDisplayDraft) // 从状态仓库读取集合显示草稿
  const setLocaleDraft = useSettingsStore((state) => state.setLocaleDraft) // 从状态仓库读取集合地区草稿
  const setFileDraft = useSettingsStore((state) => state.setFileDraft) // 从状态仓库读取集合文件草稿
  const setShortcutDraft = useSettingsStore((state) => state.setShortcutDraft) // 从状态仓库读取集合快捷键草稿
  /** 从状态仓库读取集合快捷键校验 */
  const setShortcutValidation = useSettingsStore((state) => state.setShortcutValidation)
  const applyDisplaySettings = useSettingsStore((state) => state.applyDisplaySettings) // 应用主题和字号草稿
  const applyLocaleSettings = useSettingsStore((state) => state.applyLocaleSettings) // 应用地区和语言草稿
  /** 从状态仓库读取应用快捷键设置 */
  const applyShortcutSettings = useSettingsStore((state) => state.applyShortcutSettings)
  const setFontSize = useAppStore((state) => state.setFontSize) // 从状态仓库读取集合字体尺寸
  const setLanguage = useAppStore((state) => state.setLanguage) // 从状态仓库读取集合语言

  if (generalTab === 'locale') {
    return (
      <LocaleSettingsView
        locale={localeDraft.locale}
        language={localeDraft.language}
        languages={languages}
        regions={regions}
        dirty={localeDirty}
        onLocaleChange={(value) => setLocaleDraft({ locale: value })}
        onLanguageChange={(value) => setLocaleDraft({ language: value })}
        onApply={async () => {
          await Settings.SaveLocaleSettings({ locale: localeDraft.locale, language: localeDraft.language })
          applyLocaleSettings({ locale: localeDraft.locale, language: localeDraft.language })
          setLanguage(localeDraft.language)
        }}
      />
    )
  }

  if (generalTab === 'file') {
    return (
      <FileSettingsView
        dataDir={fileDraft.dataDir}
        currentDataDir={dataDir}
        dirty={fileDirty}
        onSelectFolder={async () => {
          const result = await File.SelectFolder({ folder_path: fileDraft.dataDir }) // 保存本次调用的结果
          if (result?.folder_path) {
            setFileDraft({ dataDir: result.folder_path })
          }
        }}
        onApply={() => undefined}
      />
    )
  }

  if (generalTab === 'shortcuts') {
    return (
      <ShortcutSettingsView
        quickChatShortcut={shortcutDraft.quickChatShortcut}
        dirty={shortcutDirty}
        validation={shortcutValidation}
        onShortcutChange={async (shortcut) => {
          setShortcutDraft({ quickChatShortcut: shortcut })
          setShortcutValidation({ status: 'idle', message: '' })
          try {
            /** 保存本次调用的结果 */
            const result = await Settings.ValidateShortcutSettings({ quick_chat_shortcut: shortcut })
            if (result) {
              const nextShortcut = result.quick_chat_shortcut || shortcut // 下一项快捷键
              setShortcutDraft({ quickChatShortcut: nextShortcut })
              setShortcutValidation({
                status: result.status as typeof shortcutValidation.status,
                message: result.message ?? '',
              })
            }
          } catch {
            setShortcutValidation({ status: 'available', message: '' })
          }
        }}
        onApply={async () => {
          try {
            /** 保存本次调用的结果 */
            const result = await Settings.SaveShortcutSettings({ quick_chat_shortcut: shortcutDraft.quickChatShortcut })
            if (result) {
              setShortcutValidation({
                status: result.status as typeof shortcutValidation.status,
                message: result.message ?? '',
              })
              if (result.status === 'available') {
                applyShortcutSettings(result.quick_chat_shortcut || shortcutDraft.quickChatShortcut)
              }
            }
          } catch {
            applyShortcutSettings(shortcutDraft.quickChatShortcut)
          }
        }}
        onReset={() => {
          setShortcutDraft({ quickChatShortcut: quickChatShortcut })
          setShortcutValidation({ status: 'idle', message: '' })
        }}
      />
    )
  }

  return (
    <DisplaySettingsView
      value={fontSize}
      draft={displayDraft.fontSize}
      onDraftChange={(value) => setDisplayDraft({ fontSize: value })}
      onApply={async () => {
        await Settings.SaveDisplaySettings({ font_size: displayDraft.fontSize })
        applyDisplaySettings(displayDraft.fontSize)
        setFontSize(displayDraft.fontSize)
      }}
      onReset={() => setDisplayDraft({ fontSize })}
    />
  )
}
