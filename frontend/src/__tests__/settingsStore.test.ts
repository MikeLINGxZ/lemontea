import { beforeEach, describe, expect, it } from 'vitest'
import { useSettingsStore, getSettingsInitialState } from '@/store/settingsStore'

beforeEach(() => {
  useSettingsStore.setState(getSettingsInitialState())
})

describe('settingsStore', () => {
  it('marks display settings dirty when draft font size differs from applied font size', () => {
    useSettingsStore.getState().hydrate({
      locale: 'zh-CN',
      language: 'zh-CN',
      font_size: 'md',
      data_dir: '/tmp/a',
      log_level: 'info',
      quick_chat_shortcut: 'CmdOrCtrl+Shift+Space',
      default_provider_id: 0,
      version: 'v0.0.1-dev',
      providers: [],
      languages: [],
      regions: [],
    })

    useSettingsStore.getState().setDisplayDraft({ fontSize: 'xl' })

    expect(useSettingsStore.getState().displayDirty).toBe(true)
  })

  it('marks shortcut settings dirty when draft shortcut differs from applied shortcut', () => {
    useSettingsStore.getState().hydrate({
      locale: 'zh-CN',
      language: 'zh-CN',
      font_size: 'md',
      data_dir: '/tmp/a',
      log_level: 'info',
      quick_chat_shortcut: 'CmdOrCtrl+Shift+Space',
      default_provider_id: 0,
      version: 'v0.0.1-dev',
      providers: [],
      languages: [],
      regions: [],
    })

    useSettingsStore.getState().setShortcutDraft({ quickChatShortcut: 'CmdOrCtrl+Shift+K' })

    expect(useSettingsStore.getState().shortcutDirty).toBe(true)
  })

  it('keeps imported extensions during bootstrap hydration', () => {
    useSettingsStore.setState({
      ...useSettingsStore.getState(),
      extensions: [{
        id: 'mcp:mysql:default',
        name: 'mysql',
        description: '',
        author: '',
      version: '',
      kind: 'mcp',
      enabled: true,
      runtime_status: 'ready',
      runtime_message: '',
      root_dir: '/tmp/mysql',
      source_dir: '/tmp/source',
      config_file_path: '/tmp/mysql/mcp.json',
        tools: [],
      }],
      selectedExtensionId: 'mcp:mysql:default',
    })

    useSettingsStore.getState().hydrate({
      locale: 'zh-CN',
      language: 'zh-CN',
      font_size: 'md',
      data_dir: '/tmp/a',
      log_level: 'info',
      quick_chat_shortcut: 'CmdOrCtrl+Shift+Space',
      default_provider_id: 0,
      version: 'v0.0.1-dev',
      providers: [],
      languages: [],
      regions: [],
    })

    expect(useSettingsStore.getState().extensions).toHaveLength(1)
    expect(useSettingsStore.getState().selectedExtensionId).toBe('mcp:mysql:default')
  })
})
