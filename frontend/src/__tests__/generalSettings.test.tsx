import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { GeneralSettingsPanel } from '@/components/settings/general/GeneralSettingsPanel'
import { DisplaySettingsView } from '@/components/settings/general/DisplaySettingsView'
import { useAppStore } from '@/store/appStore'
import { getSettingsInitialState, useSettingsStore } from '@/store/settingsStore'

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings', () => ({
  Settings: {
    SaveDisplaySettings: vi.fn().mockResolvedValue(undefined),
    SaveShortcutSettings: vi.fn().mockResolvedValue({ quick_chat_shortcut: 'CmdOrCtrl+Shift+K', status: 'available', message: '' }),
    ValidateShortcutSettings: vi.fn().mockResolvedValue({ quick_chat_shortcut: 'CmdOrCtrl+Shift+K', status: 'available', message: '' }),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file', () => ({
  SelectFolder: vi.fn().mockResolvedValue(''),
}))

beforeEach(() => {
  useSettingsStore.setState(getSettingsInitialState())
  useAppStore.setState({
    theme: 'auto',
    fontSize: 'md',
    language: 'zh-CN',
  })
})

describe('DisplaySettingsView', () => {
  it('enables apply when font size draft changes', async () => {
    const user = userEvent.setup()
    const onApply = vi.fn()

    render(
      <DisplaySettingsView
        value="md"
        draft="md"
        onDraftChange={() => undefined}
        onApply={onApply}
        onReset={() => undefined}
      />
    )

    await user.click(screen.getByRole('button', { name: '大' }))

    expect(screen.getByRole('button', { name: '应用' })).toBeEnabled()
  })

  it('applies the font size draft when the user clicks apply', async () => {
    const user = userEvent.setup()

    useSettingsStore.setState({
      generalTab: 'display',
      fontSize: 'md',
      displayDraft: { fontSize: 'md' },
      displayDirty: false,
    })

    render(<GeneralSettingsPanel />)

    await user.click(screen.getByRole('button', { name: '大' }))
    await user.click(screen.getByRole('button', { name: '应用' }))

    await waitFor(() => {
      expect(useAppStore.getState().fontSize).toBe('lg')
      expect(useSettingsStore.getState().fontSize).toBe('lg')
      expect(useSettingsStore.getState().displayDirty).toBe(false)
    })
  })

  it('validates quick chat shortcut as soon as a shortcut is recorded', async () => {
    const { Settings } = await import('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings')
    vi.mocked(Settings.ValidateShortcutSettings).mockResolvedValueOnce({
      quick_chat_shortcut: 'CmdOrCtrl+Shift+K',
      status: 'conflict',
      message: 'shortcut already taken',
    })

    useSettingsStore.setState({
      generalTab: 'shortcuts',
      quickChatShortcut: 'CmdOrCtrl+Shift+Space',
      shortcutDraft: { quickChatShortcut: 'CmdOrCtrl+Shift+Space' },
      shortcutDirty: false,
    })

    render(<GeneralSettingsPanel />)

    const recorder = screen.getByRole('button', { name: /记录快捷键/i })
    fireEvent.keyDown(recorder, { key: 'K', code: 'KeyK', ctrlKey: true, shiftKey: true })

    await waitFor(() => {
      expect(Settings.ValidateShortcutSettings).toHaveBeenCalledWith({ quick_chat_shortcut: 'Ctrl+Shift+K' })
      expect(screen.getByText(/快捷键冲突/)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: '应用' })).toBeDisabled()
    })
  })

  it('records the next window key after the user clicks the shortcut recorder', async () => {
    const { Settings } = await import('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings')
    vi.mocked(Settings.ValidateShortcutSettings).mockResolvedValueOnce({
      quick_chat_shortcut: 'Ctrl+Shift+K',
      status: 'available',
      message: '',
    })

    useSettingsStore.setState({
      generalTab: 'shortcuts',
      quickChatShortcut: 'CmdOrCtrl+Shift+Space',
      shortcutDraft: { quickChatShortcut: 'CmdOrCtrl+Shift+Space' },
      shortcutDirty: false,
      shortcutValidation: { status: 'idle', message: '' },
    })

    render(<GeneralSettingsPanel />)

    await userEvent.click(screen.getByRole('button', { name: /记录快捷键/i }))
    expect(screen.getByText('正在录制，按新的组合键')).toBeInTheDocument()

    fireEvent.keyDown(window, { key: 'K', code: 'KeyK', ctrlKey: true, shiftKey: true })

    await waitFor(() => {
      expect(Settings.ValidateShortcutSettings).toHaveBeenCalledWith({ quick_chat_shortcut: 'Ctrl+Shift+K' })
      expect(screen.getByText('Ctrl+Shift+K')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: '应用' })).toBeEnabled()
    })
  })

  it('cancels shortcut recording when Escape is pressed', async () => {
    const { Settings } = await import('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings')
    vi.mocked(Settings.ValidateShortcutSettings).mockClear()

    useSettingsStore.setState({
      generalTab: 'shortcuts',
      quickChatShortcut: 'CmdOrCtrl+Shift+Space',
      shortcutDraft: { quickChatShortcut: 'CmdOrCtrl+Shift+Space' },
      shortcutDirty: false,
      shortcutValidation: { status: 'idle', message: '' },
    })

    render(<GeneralSettingsPanel />)

    await userEvent.click(screen.getByRole('button', { name: /记录快捷键/i }))
    fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' })

    await waitFor(() => {
      expect(screen.queryByText('正在录制，按新的组合键')).toBeNull()
      expect(Settings.ValidateShortcutSettings).not.toHaveBeenCalled()
    })
  })

  it('allows local shortcut changes when backend validation is unavailable', async () => {
    const { Settings } = await import('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings')
    vi.mocked(Settings.ValidateShortcutSettings).mockRejectedValueOnce(new Error('backend unavailable'))
    vi.mocked(Settings.SaveShortcutSettings).mockRejectedValueOnce(new Error('backend unavailable'))

    useSettingsStore.setState({
      generalTab: 'shortcuts',
      quickChatShortcut: 'CmdOrCtrl+Shift+Space',
      shortcutDraft: { quickChatShortcut: 'CmdOrCtrl+Shift+Space' },
      shortcutDirty: false,
      shortcutValidation: { status: 'idle', message: '' },
    })

    render(<GeneralSettingsPanel />)

    const recorder = screen.getByRole('button', { name: /记录快捷键/i })
    fireEvent.keyDown(recorder, { key: 'K', code: 'KeyK', ctrlKey: true, shiftKey: true })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: '应用' })).toBeEnabled()
    })

    await userEvent.click(screen.getByRole('button', { name: '应用' }))

    await waitFor(() => {
      expect(useSettingsStore.getState().quickChatShortcut).toBe('Ctrl+Shift+K')
      expect(useSettingsStore.getState().shortcutDirty).toBe(false)
    })
  })
})
