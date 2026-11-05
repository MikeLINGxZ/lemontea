import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AlertViewport } from '@/components/alert/AlertViewport'
import { SettingsApp } from '@/components/settings/SettingsApp'
import i18n from '@/i18n'
import { useAlertStore } from '@/alert/store'
import { getSettingsInitialState, useSettingsStore } from '@/store/settingsStore'

const {
  downloadNodeMock,
  getStatusMock,
  onMock,
  listExtensionsMock,
  getExtensionDetailMock,
  toggleExtensionMock,
  reloadExtensionMock,
  deleteExtensionMock,
  saveExtensionConfigMock,
  importExtensionMock,
} = vi.hoisted(() => ({
  downloadNodeMock: vi.fn(),
  getStatusMock: vi.fn(),
  onMock: vi.fn(),
  listExtensionsMock: vi.fn(),
  getExtensionDetailMock: vi.fn(),
  toggleExtensionMock: vi.fn(),
  reloadExtensionMock: vi.fn(),
  deleteExtensionMock: vi.fn(),
  saveExtensionConfigMock: vi.fn(),
  importExtensionMock: vi.fn(),
}))

vi.mock('@/hooks/useSettingsBootstrap', () => ({
  useSettingsBootstrap: vi.fn(),
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window', () => ({
  Window: {
    OpenAddProvider: vi.fn(),
    OpenAddSkill: vi.fn(),
    OpenSettings: vi.fn(),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider', () => ({
  Provider: {
    ListProviders: vi.fn().mockResolvedValue({ providers: [] }),
    EditProvider: vi.fn(),
    SetDefault: vi.fn(),
    DeleteProvider: vi.fn(),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin', () => ({
  Plugin: {
    ListExtensions: listExtensionsMock,
    GetExtensionDetail: getExtensionDetailMock,
    ToggleExtension: toggleExtensionMock,
    ReloadExtension: reloadExtensionMock,
    DeleteExtension: deleteExtensionMock,
    SaveExtensionConfig: saveExtensionConfigMock,
    ImportExtension: importExtensionMock,
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file', () => ({
  File: {
    SelectFolder: vi.fn(),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/runtime/index', () => ({
  Runtime: {
    GetStatus: getStatusMock,
    DownloadNode: downloadNodeMock,
  },
}))

vi.mock('@wailsio/runtime', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@wailsio/runtime')>()
  return {
    ...actual,
    Events: {
      ...actual.Events,
      On: onMock,
    },
  }
})

describe('Plugin settings', () => {
  beforeEach(async () => {
    window.history.replaceState({}, '', '/?entry=settings&tab=plugins')
    useAlertStore.getState().reset()
    useSettingsStore.setState({
      ...getSettingsInitialState(),
      activeTab: 'plugins',
    })
    getStatusMock.mockReset()
    downloadNodeMock.mockReset()
    onMock.mockReset()
    listExtensionsMock.mockReset()
    getExtensionDetailMock.mockReset()
    toggleExtensionMock.mockReset()
    reloadExtensionMock.mockReset()
    deleteExtensionMock.mockReset()
    saveExtensionConfigMock.mockReset()
    importExtensionMock.mockReset()

    listExtensionsMock.mockResolvedValue({
      extensions: [{
        id: 'mcp:filesystem:1.0.0',
        name: 'filesystem',
        description: 'Filesystem tools',
        author: 'Lemontea',
        version: '1.0.0',
        kind: 'mcp',
        enabled: true,
        root_dir: '/tmp/filesystem',
        source_dir: '/tmp/source',
        config_file_path: '/tmp/filesystem/mcp.json',
        tools: [],
      }],
    })
    getExtensionDetailMock.mockResolvedValue({
      extension: {
        id: 'mcp:filesystem:1.0.0',
        name: 'filesystem',
        description: 'Filesystem tools',
        author: 'Lemontea',
        version: '1.0.0',
        kind: 'mcp',
        enabled: true,
        root_dir: '/tmp/filesystem',
        source_dir: '/tmp/source',
        config_file_path: '/tmp/filesystem/mcp.json',
        tools: [],
      },
      config_text: '{"transport":"stdio"}',
    })
    getStatusMock.mockResolvedValue({
      state: 'pending_later',
      version: 'v22.11.0',
      install_dir: '',
      node_path: '',
      npm_path: '',
      error_msg: '',
    })
    onMock.mockReturnValue(() => {})
    await i18n.changeLanguage('en')
  })

  it('renders plugins as a primary settings tab', async () => {
    render(
      <>
        <SettingsApp />
        <AlertViewport />
      </>
    )

    expect(await screen.findByRole('heading', { name: 'Plugins & Tools' })).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: /filesystem/ })).toBeInTheDocument()
  })

  it('shows a runtime reminder toast on the plugins tab when download was deferred', async () => {
    const user = userEvent.setup()

    render(
      <>
        <SettingsApp />
        <AlertViewport />
      </>
    )

    await waitFor(() => {
      expect(screen.getByText('Plugin runtime')).toBeInTheDocument()
    })
    expect(screen.getByText('Install it to unlock more CLI and MCP plugin capabilities. You can download it now or finish setup later in Settings.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download later' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download now' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Download now' }))

    expect(downloadNodeMock).toHaveBeenCalledWith({})
  })

  it('shows a runtime reminder toast on the plugins tab when runtime is still missing', async () => {
    getStatusMock.mockResolvedValueOnce({
      state: 'missing',
      version: 'v22.11.0',
      install_dir: '',
      node_path: '',
      npm_path: '',
      error_msg: '',
    })

    render(
      <>
        <SettingsApp />
        <AlertViewport />
      </>
    )

    expect(await screen.findByText('Plugin runtime')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download later' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download now' })).toBeInTheDocument()
  })

  it('updates the runtime toast with progress and hides duplicate count badges', async () => {
    const user = userEvent.setup()

    render(
      <>
        <SettingsApp />
        <AlertViewport />
      </>
    )

    await screen.findByText('Plugin runtime')
    await user.click(screen.getByRole('button', { name: 'Download now' }))

    expect(screen.getByText('Downloading… 0% · Download · 0 B / …')).toBeInTheDocument()
    expect(screen.queryByText(/^x\d+$/)).not.toBeInTheDocument()

    const progressHandler = onMock.mock.calls.find(([name]) => name === 'runtime.node.progress')?.[1] as
      | ((event: { data: { phase: 'download' | 'extract' | 'verify'; received: number; total: number; percent: number } }) => void)
      | undefined

    expect(progressHandler).toBeTypeOf('function')

    await act(async () => {
      progressHandler?.({
        data: {
          phase: 'download',
          received: 5 * 1024 * 1024,
          total: 10 * 1024 * 1024,
          percent: 50,
        },
      })
    })

    expect(await screen.findByText('Downloading… 50% · Download · 5.0 MB / 10.0 MB')).toBeInTheDocument()
    expect(screen.queryByText(/^x\d+$/)).not.toBeInTheDocument()
  })
})
