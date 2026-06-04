import { create } from 'zustand'
import type { StateCreator } from 'zustand'
import type { FontSize, Language } from '@/types'
import type { ExtensionItem, GeneralSettingsTab, ProviderItem, SettingsBootstrap, SettingsOption, SettingsPrimaryTab, ShortcutValidationStatus } from '@/types/settings'
import type { SkillItem } from '@/types/skills'

type DisplayDraft = {
  fontSize: FontSize
}

type LocaleDraft = {
  locale: string
  language: Language
}

type FileDraft = {
  dataDir: string
}

type ShortcutDraft = {
  quickChatShortcut: string
}

type ShortcutValidation = {
  status: ShortcutValidationStatus
  message: string
}

type SettingsState = {
  activeTab: SettingsPrimaryTab
  generalTab: GeneralSettingsTab
  locale: string
  language: Language
  fontSize: FontSize
  dataDir: string
  quickChatShortcut: string
  logLevel: string
  defaultProviderId: number
  selectedProviderId: number | null
  selectedExtensionId: string | null
  selectedSkillName: string | null
  version: string
  providers: SettingsBootstrap['providers']
  extensions: ExtensionItem[]
  skills: SkillItem[]
  languages: SettingsOption[]
  regions: SettingsOption[]
  displayDraft: DisplayDraft
  localeDraft: LocaleDraft
  fileDraft: FileDraft
  shortcutDraft: ShortcutDraft
  shortcutValidation: ShortcutValidation
  displayDirty: boolean
  localeDirty: boolean
  fileDirty: boolean
  shortcutDirty: boolean
  setProviders: (providers: SettingsBootstrap['providers']) => void
  setExtensions: (extensions: ExtensionItem[]) => void
  updateProvider: (provider: ProviderItem) => void
  updateExtension: (extension: ExtensionItem) => void
  deleteProvider: (id: number) => void
  deleteExtension: (id: string) => void
  setSkills: (items: SkillItem[]) => void
  setSelectedSkillName: (name: string | null) => void
  updateSkill: (item: SkillItem) => void
  removeSkill: (name: string) => void
  setDefaultProvider: (providerId: number) => void
  hydrate: (payload: SettingsBootstrap) => void
  setActiveTab: (tab: SettingsPrimaryTab) => void
  setGeneralTab: (tab: GeneralSettingsTab) => void
  setSelectedProviderId: (providerId: number) => void
  setSelectedExtensionId: (extensionId: string) => void
  setLocaleOptions: (payload: { languages: SettingsOption[]; regions: SettingsOption[] }) => void
  applyDisplaySettings: (fontSize: FontSize) => void
  applyLocaleSettings: (payload: { locale: string; language: Language }) => void
  applyShortcutSettings: (quickChatShortcut: string) => void
  setDisplayDraft: (draft: Partial<DisplayDraft>) => void
  setLocaleDraft: (draft: Partial<LocaleDraft>) => void
  setFileDraft: (draft: Partial<FileDraft>) => void
  setShortcutDraft: (draft: Partial<ShortcutDraft>) => void
  setShortcutValidation: (validation: ShortcutValidation) => void
}

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
    setActiveTab: (tab) => set({ activeTab: tab }),
    setGeneralTab: (tab) => set({ generalTab: tab }),
    setSelectedProviderId: (providerId) => set({ selectedProviderId: providerId }),
    setSelectedExtensionId: (extensionId) => set({ selectedExtensionId: extensionId }),
    setLocaleOptions: (payload) => set({
      languages: payload.languages,
      regions: payload.regions,
    }),
    applyDisplaySettings: (fontSize) => set({
      fontSize,
      displayDraft: { fontSize },
      displayDirty: false,
    }),
    applyLocaleSettings: (payload) => set({
      locale: payload.locale,
      language: payload.language,
      localeDraft: payload,
      localeDirty: false,
    }),
    applyShortcutSettings: (quickChatShortcut) => set({
      quickChatShortcut,
      shortcutDraft: { quickChatShortcut },
      shortcutValidation: { status: 'available', message: '' },
      shortcutDirty: false,
    }),
    setDisplayDraft: (draft) => set((state) => {
      const nextDraft = { ...state.displayDraft, ...draft }
      return {
        displayDraft: nextDraft,
        displayDirty: nextDraft.fontSize !== state.fontSize,
      }
    }),
    setLocaleDraft: (draft) => set((state) => {
      const nextDraft = { ...state.localeDraft, ...draft }
      return {
        localeDraft: nextDraft,
        localeDirty: nextDraft.locale !== state.locale || nextDraft.language !== state.language,
      }
    }),
    setFileDraft: (draft) => set((state) => {
      const nextDraft = { ...state.fileDraft, ...draft }
      return {
        fileDraft: nextDraft,
        fileDirty: nextDraft.dataDir !== state.dataDir,
      }
    }),
    setShortcutDraft: (draft) => set((state) => {
      const nextDraft = { ...state.shortcutDraft, ...draft }
      return {
        shortcutDraft: nextDraft,
        shortcutDirty: nextDraft.quickChatShortcut !== state.quickChatShortcut,
      }
    }),
    setShortcutValidation: (validation) => set({ shortcutValidation: validation }),
    setProviders: (providers) => set({ providers }),
    setExtensions: (extensions) => set((state) => {
      // Keep the user's current selection across refreshes (window focus, mutations).
      // Only fall back to the first extension when the previous selection is gone or unset.
      const stillExists = state.selectedExtensionId !== null
        && extensions.some((item) => item.id === state.selectedExtensionId)
      return {
        extensions,
        selectedExtensionId: stillExists ? state.selectedExtensionId : (extensions[0]?.id ?? null),
      }
    }),
    updateProvider: (provider) => set((state) => ({
      providers: state.providers.map(p => p.id === provider.id ? provider : p),
    })),
    updateExtension: (extension) => set((state) => ({
      extensions: state.extensions.map(item => item.id === extension.id ? extension : item),
    })),
    deleteProvider: (id) => set((state) => {
      const providers = state.providers.filter(p => p.id !== id)
      return {
        providers,
        selectedProviderId: state.selectedProviderId === id
          ? (providers[0]?.id ?? null)
          : state.selectedProviderId,
      }
    }),
    deleteExtension: (id) => set((state) => {
      const extensions = state.extensions.filter(item => item.id !== id)
      return {
        extensions,
        selectedExtensionId: state.selectedExtensionId === id
          ? (extensions[0]?.id ?? null)
          : state.selectedExtensionId,
      }
    }),
    setSkills: (items) => set((state) => {
      const stillExists = state.selectedSkillName !== null
        && items.some((item) => item.name === state.selectedSkillName)
      return {
        skills: items,
        selectedSkillName: stillExists ? state.selectedSkillName : (items[0]?.name ?? null),
      }
    }),
    setSelectedSkillName: (name) => set({ selectedSkillName: name }),
    updateSkill: (item) => set((state) => {
      const exists = state.skills.some(s => s.name === item.name)
      const skills = exists
        ? state.skills.map(s => s.name === item.name ? item : s)
        : [...state.skills, item].sort((a, b) => a.name.localeCompare(b.name))
      return { skills }
    }),
    removeSkill: (name) => set((state) => {
      const skills = state.skills.filter(s => s.name !== name)
      return {
        skills,
        selectedSkillName: state.selectedSkillName === name
          ? (skills[0]?.name ?? null)
          : state.selectedSkillName,
      }
    }),
    setDefaultProvider: (providerId) => set((state) => ({
      providers: state.providers.map(p => ({ ...p, is_default: p.id === providerId })),
    })),
})

export const useSettingsStore = create<SettingsState>(createSettingsState)

export function getSettingsInitialState(): SettingsState {
  return useSettingsStore.getInitialState()
}
