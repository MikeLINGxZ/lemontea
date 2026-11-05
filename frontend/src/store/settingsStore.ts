import { create } from 'zustand'
import type { StateCreator } from 'zustand'
import type { FontSize, Language } from '@/types'
import type { ExtensionItem, GeneralSettingsTab, ProviderItem, SettingsBootstrap, SettingsOption, SettingsPrimaryTab, ShortcutValidationStatus } from '@/types/settings'
import type { SkillItem } from '@/types/skills'

/** 显示的编辑草稿 */
type DisplayDraft = {
  /** 字体尺寸 */
  fontSize: FontSize
}

/** 地区的编辑草稿 */
type LocaleDraft = {
  /** 地区 */
  locale: string
  /** 语言 */
  language: Language
}

/** 文件的编辑草稿 */
type FileDraft = {
  /** 数据目录 */
  dataDir: string
}

/** 快捷键的编辑草稿 */
type ShortcutDraft = {
  /** 快捷聊天快捷键 */
  quickChatShortcut: string
}

/** 定义快捷键校验的数据结构 */
type ShortcutValidation = {
  /** 状态 */
  status: ShortcutValidationStatus
  /** 消息 */
  message: string
}

/** 设置的状态数据 */
type SettingsState = {
  /** 当前标签页 */
  activeTab: SettingsPrimaryTab
  /** 常规标签页 */
  generalTab: GeneralSettingsTab
  /** 地区 */
  locale: string
  /** 语言 */
  language: Language
  /** 字体尺寸 */
  fontSize: FontSize
  /** 数据目录 */
  dataDir: string
  /** 快捷聊天快捷键 */
  quickChatShortcut: string
  /** 日志级别 */
  logLevel: string
  /** 默认模型服务商编号 */
  defaultProviderId: number
  /** 选中项模型服务商编号 */
  selectedProviderId: number | null
  /** 选中项扩展编号 */
  selectedExtensionId: string | null
  /** 选中项技能名称 */
  selectedSkillName: string | null
  /** 版本 */
  version: string
  /** 服务商列表 */
  providers: SettingsBootstrap['providers']
  /** 扩展列表 */
  extensions: ExtensionItem[]
  /** 技能列表 */
  skills: SkillItem[]
  /** 语言列表 */
  languages: SettingsOption[]
  /** 地区列表 */
  regions: SettingsOption[]
  /** 显示草稿 */
  displayDraft: DisplayDraft
  /** 地区草稿 */
  localeDraft: LocaleDraft
  /** 文件草稿 */
  fileDraft: FileDraft
  /** 快捷键草稿 */
  shortcutDraft: ShortcutDraft
  /** 快捷键校验 */
  shortcutValidation: ShortcutValidation
  /** 显示有未保存修改 */
  displayDirty: boolean
  /** 地区有未保存修改 */
  localeDirty: boolean
  /** 文件有未保存修改 */
  fileDirty: boolean
  /** 快捷键有未保存修改 */
  shortcutDirty: boolean
  /** 设置服务商列表，接收服务商列表，无返回值 */
  setProviders: (providers: SettingsBootstrap['providers']) => void
  /** 设置扩展列表，接收扩展列表，无返回值 */
  setExtensions: (extensions: ExtensionItem[]) => void
  /** 更新模型服务商，接收模型服务商，无返回值 */
  updateProvider: (provider: ProviderItem) => void
  /** 更新扩展，接收扩展，无返回值 */
  updateExtension: (extension: ExtensionItem) => void
  /** 删除模型服务商，接收当前条目的唯一编号，无返回值 */
  deleteProvider: (id: number) => void
  /** 删除扩展，接收当前条目的唯一编号，无返回值 */
  deleteExtension: (id: string) => void
  /** 设置技能列表，接收条目列表，无返回值 */
  setSkills: (items: SkillItem[]) => void
  /** 设置选中项技能名称，接收名称，无返回值 */
  setSelectedSkillName: (name: string | null) => void
  /** 更新技能，接收当前条目，无返回值 */
  updateSkill: (item: SkillItem) => void
  /** 移除技能，接收名称，无返回值 */
  removeSkill: (name: string) => void
  /** 设置默认模型服务商，接收模型服务商编号，无返回值 */
  setDefaultProvider: (providerId: number) => void
  /** 恢复处理，接收载荷，无返回值 */
  hydrate: (payload: SettingsBootstrap) => void
  /** 设置当前标签页，接收标签页，无返回值 */
  setActiveTab: (tab: SettingsPrimaryTab) => void
  /** 设置常规标签页，接收标签页，无返回值 */
  setGeneralTab: (tab: GeneralSettingsTab) => void
  /** 设置选中项模型服务商编号，接收模型服务商编号，无返回值 */
  setSelectedProviderId: (providerId: number) => void
  /** 设置选中项扩展编号，接收扩展编号，无返回值 */
  setSelectedExtensionId: (extensionId: string) => void
  /** 设置地区选项列表，接收载荷，无返回值 */
  setLocaleOptions: (payload: { /* 语言列表 */ languages: SettingsOption[]; /* 地区列表 */ regions: SettingsOption[] }) => void
  /** 应用主题和字号草稿，接收字体尺寸，无返回值 */
  applyDisplaySettings: (fontSize: FontSize) => void
  /** 应用地区和语言草稿，接收载荷，无返回值 */
  applyLocaleSettings: (payload: { /* 地区 */ locale: string; /* 语言 */ language: Language }) => void
  /** 应用快捷键设置，接收快捷聊天快捷键，无返回值 */
  applyShortcutSettings: (quickChatShortcut: string) => void
  /** 设置显示草稿，接收草稿，无返回值 */
  setDisplayDraft: (draft: Partial<DisplayDraft>) => void
  /** 设置地区草稿，接收草稿，无返回值 */
  setLocaleDraft: (draft: Partial<LocaleDraft>) => void
  /** 设置文件草稿，接收草稿，无返回值 */
  setFileDraft: (draft: Partial<FileDraft>) => void
  /** 设置快捷键草稿，接收草稿，无返回值 */
  setShortcutDraft: (draft: Partial<ShortcutDraft>) => void
  /** 设置快捷键校验，接收校验，无返回值 */
  setShortcutValidation: (validation: ShortcutValidation) => void
}

/** 创建设置状态，接收更新状态，返回构造后的数据 */
const createSettingsState: StateCreator<SettingsState> = (set) => ({
    activeTab: 'general',
    generalTab: 'display',
    locale: 'zh-CN',
    language: 'zh-CN',
    fontSize: 'md',
    dataDir: '',
    quickChatShortcut: 'CmdOrCtrl+Shift+Space',
    logLevel: 'info',
    defaultProviderId: 0,
    selectedProviderId: null,
    selectedExtensionId: null,
    selectedSkillName: null,
    version: 'v0.0.1-dev',
    providers: [],
    extensions: [],
    skills: [],
    languages: [],
    regions: [],
    displayDraft: { fontSize: 'md' },
    localeDraft: { locale: 'zh-CN', language: 'zh-CN' },
    fileDraft: { dataDir: '' },
    shortcutDraft: { quickChatShortcut: 'CmdOrCtrl+Shift+Space' },
    shortcutValidation: { status: 'idle', message: '' },
    displayDirty: false,
    localeDirty: false,
    fileDirty: false,
    shortcutDirty: false,
    /** 恢复处理，接收载荷，返回表达式结果 */
    hydrate: (payload) => set((state) => ({
      locale: payload.locale,
      language: payload.language,
      fontSize: payload.font_size,
      dataDir: payload.data_dir,
      quickChatShortcut: payload.quick_chat_shortcut || 'CmdOrCtrl+Shift+Space',
      logLevel: payload.log_level,
      defaultProviderId: payload.default_provider_id,
      selectedProviderId: payload.providers?.[0]?.id ?? null,
      selectedExtensionId: state.selectedExtensionId,
      version: payload.version,
      providers: payload.providers,
      extensions: state.extensions,
      skills: state.skills,
      languages: payload.languages ?? [],
      regions: payload.regions ?? [],
      displayDraft: { fontSize: payload.font_size },
      localeDraft: { locale: payload.locale, language: payload.language },
      fileDraft: { dataDir: payload.data_dir },
      shortcutDraft: { quickChatShortcut: payload.quick_chat_shortcut || 'CmdOrCtrl+Shift+Space' },
      shortcutValidation: { status: 'idle', message: '' },
      displayDirty: false,
      localeDirty: false,
      fileDirty: false,
      shortcutDirty: false,
    })),
    /** 设置当前标签页，接收标签页，返回表达式结果 */
    setActiveTab: (tab) => set({ activeTab: tab }),
    /** 设置常规标签页，接收标签页，返回表达式结果 */
    setGeneralTab: (tab) => set({ generalTab: tab }),
    /** 设置选中项模型服务商编号，接收模型服务商编号，返回表达式结果 */
    setSelectedProviderId: (providerId) => set({ selectedProviderId: providerId }),
    /** 设置选中项扩展编号，接收扩展编号，返回表达式结果 */
    setSelectedExtensionId: (extensionId) => set({ selectedExtensionId: extensionId }),
    /** 设置地区选项列表，接收载荷，返回表达式结果 */
    setLocaleOptions: (payload) => set({
      languages: payload.languages,
      regions: payload.regions,
    }),
    /** 应用主题和字号草稿，接收字体尺寸，返回表达式结果 */
    applyDisplaySettings: (fontSize) => set({
      fontSize,
      displayDraft: { fontSize },
      displayDirty: false,
    }),
    /** 应用地区和语言草稿，接收载荷，返回表达式结果 */
    applyLocaleSettings: (payload) => set({
      locale: payload.locale,
      language: payload.language,
      localeDraft: payload,
      localeDirty: false,
    }),
    /** 应用快捷键设置，接收快捷聊天快捷键，返回表达式结果 */
    applyShortcutSettings: (quickChatShortcut) => set({
      quickChatShortcut,
      shortcutDraft: { quickChatShortcut },
      shortcutValidation: { status: 'available', message: '' },
      shortcutDirty: false,
    }),
    /** 设置显示草稿，接收草稿，返回表达式结果 */
    setDisplayDraft: (draft) => set((state) => {
      const nextDraft = { ...state.displayDraft, ...draft } // 下一项草稿
      return {
        displayDraft: nextDraft,
        displayDirty: nextDraft.fontSize !== state.fontSize,
      }
    }),
    /** 设置地区草稿，接收草稿，返回表达式结果 */
    setLocaleDraft: (draft) => set((state) => {
      const nextDraft = { ...state.localeDraft, ...draft } // 下一项草稿
      return {
        localeDraft: nextDraft,
        localeDirty: nextDraft.locale !== state.locale || nextDraft.language !== state.language,
      }
    }),
    /** 设置文件草稿，接收草稿，返回表达式结果 */
    setFileDraft: (draft) => set((state) => {
      const nextDraft = { ...state.fileDraft, ...draft } // 下一项草稿
      return {
        fileDraft: nextDraft,
        fileDirty: nextDraft.dataDir !== state.dataDir,
      }
    }),
    /** 设置快捷键草稿，接收草稿，返回表达式结果 */
    setShortcutDraft: (draft) => set((state) => {
      const nextDraft = { ...state.shortcutDraft, ...draft } // 下一项草稿
      return {
        shortcutDraft: nextDraft,
        shortcutDirty: nextDraft.quickChatShortcut !== state.quickChatShortcut,
      }
    }),
    /** 设置快捷键校验，接收校验，返回表达式结果 */
    setShortcutValidation: (validation) => set({ shortcutValidation: validation }),
    /** 设置服务商列表，接收服务商列表，返回表达式结果 */
    setProviders: (providers) => set({ providers }),
    /** 设置扩展列表，接收扩展列表，返回表达式结果 */
    setExtensions: (extensions) => set((state) => {
      // 刷新时沿用当前选中项，原选中项已移除或为空时回退到第一个扩展
      const stillExists = state.selectedExtensionId !== null
        && extensions.some((item) => item.id === state.selectedExtensionId)
      return {
        extensions,
        selectedExtensionId: stillExists ? state.selectedExtensionId : (extensions[0]?.id ?? null),
      }
    }),
    /** 更新模型服务商，接收模型服务商，返回表达式结果 */
    updateProvider: (provider) => set((state) => ({
      providers: state.providers.map(p => p.id === provider.id ? provider : p),
    })),
    /** 更新扩展，接收扩展，返回表达式结果 */
    updateExtension: (extension) => set((state) => ({
      extensions: state.extensions.map(item => item.id === extension.id ? extension : item),
    })),
    /** 删除模型服务商，接收当前条目的唯一编号，返回表达式结果 */
    deleteProvider: (id) => set((state) => {
      const providers = state.providers.filter(p => p.id !== id) // 服务商列表
      return {
        providers,
        selectedProviderId: state.selectedProviderId === id
          ? (providers[0]?.id ?? null)
          : state.selectedProviderId,
      }
    }),
    /** 删除扩展，接收当前条目的唯一编号，返回表达式结果 */
    deleteExtension: (id) => set((state) => {
      const extensions = state.extensions.filter(item => item.id !== id) // 扩展列表
      return {
        extensions,
        selectedExtensionId: state.selectedExtensionId === id
          ? (extensions[0]?.id ?? null)
          : state.selectedExtensionId,
      }
    }),
    /** 设置技能列表，接收条目列表，返回表达式结果 */
    setSkills: (items) => set((state) => {
      /** 仍然是否存在 */
      const stillExists = state.selectedSkillName !== null
        && items.some((item) => item.name === state.selectedSkillName)
      return {
        skills: items,
        selectedSkillName: stillExists ? state.selectedSkillName : (items[0]?.name ?? null),
      }
    }),
    /** 设置选中项技能名称，接收名称，返回表达式结果 */
    setSelectedSkillName: (name) => set({ selectedSkillName: name }),
    /** 更新技能，接收当前条目，返回表达式结果 */
    updateSkill: (item) => set((state) => {
      const exists = state.skills.some(s => s.name === item.name) // 是否存在
      /** 技能列表 */
      const skills = exists
        ? state.skills.map(s => s.name === item.name ? item : s)
        : [...state.skills, item].sort((a, b) => a.name.localeCompare(b.name))
      return { skills }
    }),
    /** 移除技能，接收名称，返回表达式结果 */
    removeSkill: (name) => set((state) => {
      const skills = state.skills.filter(s => s.name !== name) // 技能列表
      return {
        skills,
        selectedSkillName: state.selectedSkillName === name
          ? (skills[0]?.name ?? null)
          : state.selectedSkillName,
      }
    }),
    /** 设置默认模型服务商，接收模型服务商编号，返回表达式结果 */
    setDefaultProvider: (providerId) => set((state) => ({
      providers: state.providers.map(p => ({ ...p, is_default: p.id === providerId })),
    })),
})

/** 管理设置页选中项、编辑草稿和服务商及扩展列表 */
export const useSettingsStore = create<SettingsState>(createSettingsState)

/** 获取设置初始状态，返回对应的处理结果 */
export function getSettingsInitialState(): SettingsState {
  return useSettingsStore.getInitialState()
}
