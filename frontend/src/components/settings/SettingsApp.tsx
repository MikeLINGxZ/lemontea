import { useCallback, useEffect, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { Globe2, Keyboard, Languages, MonitorSmartphone } from 'lucide-react'
import { Events } from '@wailsio/runtime'
import { AboutSettingsView } from '@/components/settings/about/AboutSettingsView'
import { SettingsShell } from '@/components/settings/SettingsShell'
import { SettingsSectionLayout } from '@/components/settings/common/SettingsSectionLayout'
import { SettingsSubmenuList } from '@/components/settings/common/SettingsSubmenuList'
import { GeneralSettingsPanel } from '@/components/settings/general/GeneralSettingsPanel'
import { MemoryList, MemorySettingsView, useMemoryController } from '@/components/settings/memory/MemorySettingsView'
import { CliInstallModal } from '@/components/settings/plugins/CliInstallModal'
import { CliLoginDialog } from '@/components/settings/plugins/CliLoginDialog'
import { PluginToolDetailView } from '@/components/settings/plugins/PluginToolDetailView'
import { PluginToolList } from '@/components/settings/plugins/PluginToolList'
import { ProviderDetailView } from '@/components/settings/providers/ProviderDetailView'
import { ProviderList } from '@/components/settings/providers/ProviderList'
import { SkillsList } from '@/components/settings/skills/SkillsList'
import { SkillsDetailPane } from '@/components/settings/skills/SkillsDetailPane'
import { useSettingsBootstrap } from '@/hooks/useSettingsBootstrap'
import { useAlertStore } from '@/alert/store'
import { parseUnknownIError } from '@/lib/ierror'
import { useCliInstallStore } from '@/store/cliInstallStore'
import { useChatStore } from '@/store/chatStore'
import { useSettingsStore } from '@/store/settingsStore'
import { useTranslation } from 'react-i18next'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { Provider as ProviderBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import { Plugin as PluginBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin'
import { File as FileBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file'
import { Runtime } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/runtime/index'
import { EditProviderInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import { CancelLoginCliInput, GenerateCliManifestInput, ImportExtensionInput, InstallCliFromNpmInput, LoginCliInput, ReloadExtensionInput, ResetCliDataInput, SaveExtensionConfigInput, ToggleExtensionInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto/models'
import { SelectFolderInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file/file_dto/models'
import type { ProviderWrapper } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import type { ExtensionItem as BindingExtensionItem } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models/models'
import type { ExtensionItem, ProviderItem } from '@/types/settings'

type RuntimePhase = 'download' | 'extract' | 'verify'

type RuntimeProgressEvent = {
  phase: RuntimePhase
  received: number
  total: number
  percent: number
}

type LoginDialogState = {
  id: string
  title: string
}

const RUNTIME_TOAST_KEY = 'plugins-runtime-download-toast'

function mapProviderItem(w: ProviderWrapper): ProviderItem {
  return {
    id: w.providers.id,
    provider_name: w.providers.provider_name,
    provider_type: w.providers.provider_type,
    base_url: w.providers.base_url,
    api_key: w.providers.api_key,
    enabled: w.providers.enabled,
    is_default: w.providers.is_default,
    model_count: w.models.length,
    icon: w.providers.icon,
    models: w.models.map((m) => ({
      id: m.id,
      provider_id: m.provider_id,
      model: m.model,
      owned_by: m.owned_by,
      object: m.object,
      enable: m.enable,
      alias: m.alias,
      is_custom: m.is_custom,
      is_default: m.is_default,
    })),
  }
}

function mapExtensionItem(item: BindingExtensionItem): ExtensionItem {
  const kind: 'mcp' | 'plugin' | 'cli' = item.kind === 'plugin' || item.kind === 'cli' ? item.kind : 'mcp'
  return {
    ...item,
    kind,
    tools: item.tools ?? [],
  }
}

export function SettingsApp() {
  useSettingsBootstrap()
  const { t } = useTranslation()

  const setProviders = useSettingsStore((state) => state.setProviders)
  const setExtensions = useSettingsStore((state) => state.setExtensions)
  const updateProvider = useSettingsStore((state) => state.updateProvider)
  const updateExtension = useSettingsStore((state) => state.updateExtension)
  const deleteProvider = useSettingsStore((state) => state.deleteProvider)
  const deleteExtension = useSettingsStore((state) => state.deleteExtension)
  const setDefaultProvider = useSettingsStore((state) => state.setDefaultProvider)
  const cliInstallUpsert = useCliInstallStore((state) => state.upsert)
  const createConversation = useChatStore((state) => state.createConversation)
  const sendMessage = useChatStore((state) => state.sendMessage)
  const setCurrentConversation = useChatStore((state) => state.setCurrentConversation)
  const [selectedExtensionConfig, setSelectedExtensionConfig] = useState<string>('')
  const [pluginLoading, setPluginLoading] = useState(false)
  const [cliModalOpen, setCliModalOpen] = useState(false)
  const [loginDialog, setLoginDialog] = useState<LoginDialogState | null>(null)
  const loginDialogRef = useRef<LoginDialogState | null>(null)

  const refreshProviders = useCallback(async () => {
    try {
      const result = await ProviderBinding.ListProviders({})
      if (result?.providers) {
        setProviders(result.providers.map(mapProviderItem))
      }
    } catch {
      // ignore
    }
  }, [setProviders])

  const refreshExtensions = useCallback(async () => {
    try {
      const result = await PluginBinding.ListExtensions({})
      if (result?.extensions) {
        setExtensions(result.extensions.map(mapExtensionItem))
      }
    } catch {
      // ignore
    }
  }, [setExtensions])

  useEffect(() => {
    const handleFocus = () => {
      void refreshProviders()
      void refreshExtensions()
    }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [refreshExtensions, refreshProviders])

  const handleToggleEnable = useCallback(async (item: ProviderItem) => {
    const updated = { ...item, enabled: !item.enabled }
    updateProvider(updated)
    try {
      await ProviderBinding.EditProvider(new EditProviderInput({
        provider_id: item.id,
        provider_name: item.provider_name,
        base_url: item.base_url,
        api_key: item.api_key,
        enable: !item.enabled,
        default_model: null,
        models: [],
      }))
    } catch {
      updateProvider(item)
    }
  }, [updateProvider])

  const handleSetDefault = useCallback(async (id: number) => {
    try {
      await ProviderBinding.SetDefault({ provider_id: id, model_id: null })
      setDefaultProvider(id)
    } catch {
      // ignore
    }
  }, [setDefaultProvider])

  const handleDeleteFromList = useCallback(async (id: number) => {
    try {
      await ProviderBinding.DeleteProvider({ provider_id: id })
      deleteProvider(id)
    } catch {
      // ignore
    }
  }, [deleteProvider])

  const activeTab = useSettingsStore((state) => state.activeTab)
  const generalTab = useSettingsStore((state) => state.generalTab)
  const setActiveTab = useSettingsStore((state) => state.setActiveTab)
  const setGeneralTab = useSettingsStore((state) => state.setGeneralTab)
  const version = useSettingsStore((state) => state.version)
  const providers = useSettingsStore((state) => state.providers)
  const defaultProviderId = useSettingsStore((state) => state.defaultProviderId)
  const extensions = useSettingsStore((state) => state.extensions)
  const selectedProviderId = useSettingsStore((state) => state.selectedProviderId)
  const selectedExtensionId = useSettingsStore((state) => state.selectedExtensionId)
  const setSelectedProviderId = useSettingsStore((state) => state.setSelectedProviderId)
  const setSelectedExtensionId = useSettingsStore((state) => state.setSelectedExtensionId)
  const pushAlert = useAlertStore((state) => state.pushAlert)
  const memoryController = useMemoryController(activeTab === 'memory')

  const generalItems = [
    { key: 'display' as const, label: t('settingsPage.general.display.title'), icon: <MonitorSmartphone size={16} /> },
    { key: 'locale' as const, label: t('settingsPage.general.locale.title'), icon: <Languages size={16} /> },
    { key: 'shortcuts' as const, label: t('settingsPage.general.shortcuts.title'), icon: <Keyboard size={16} /> },
    { key: 'file' as const, label: t('settingsPage.general.file.title'), icon: <Globe2 size={16} className="rotate-45" /> },
  ]

  useEffect(() => {
    const tab = new URLSearchParams(window.location.search).get('tab')
    if (tab === 'general' || tab === 'providers' || tab === 'plugins' || tab === 'skills' || tab === 'memory' || tab === 'about') {
      setActiveTab(tab)
    }
  }, [setActiveTab])

  useEffect(() => {
    void refreshExtensions()
  }, [refreshExtensions])

  useEffect(() => {
    let cancelled = false

    if (activeTab !== 'plugins') {
      return
    }

    Runtime.GetStatus({})
      .then((result) => {
        if (cancelled || !result || !shouldShowRuntimeReminder(result.state)) {
          return
        }

        pushAlert({
          kind: 'info',
          placement: 'toast',
          title: t('onboarding.runtime.title'),
          message: t('onboarding.runtime.intro'),
          dedupeKey: RUNTIME_TOAST_KEY,
          dedupeBehavior: 'replace',
          showCount: false,
          autoClose: false,
          actions: [
            {
              id: 'download-runtime-later',
              label: t('onboarding.actions.downloadLater'),
              style: 'secondary',
              closeOnClick: true,
            },
            {
              id: 'download-runtime-now',
              label: t('onboarding.actions.downloadNow'),
              style: 'primary',
              closeOnClick: false,
              onClick: () => {
                pushAlert({
                  kind: 'info',
                  placement: 'toast',
                  title: t('onboarding.runtime.title'),
                  message: buildRuntimeProgressMessage(t, {
                    phase: 'download',
                    received: 0,
                    total: 0,
                    percent: 0,
                  }),
                  detail: null,
                  dedupeKey: RUNTIME_TOAST_KEY,
                  dedupeBehavior: 'replace',
                  showCount: false,
                  autoClose: false,
                  actions: [],
                })
                void Runtime.DownloadNode({})
              },
            },
          ],
        })
      })
      .catch(() => undefined)

    return () => {
      cancelled = true
    }
  }, [activeTab, pushAlert, t])

  useEffect(() => {
    if (activeTab !== 'plugins') {
      return
    }

    const off = Events.On('runtime.node.progress', (event: { data: RuntimeProgressEvent }) => {
      const progress = event.data
      pushAlert({
        kind: 'info',
        placement: 'toast',
        title: t('onboarding.runtime.title'),
        message: buildRuntimeProgressMessage(t, progress),
        detail: null,
        dedupeKey: RUNTIME_TOAST_KEY,
        dedupeBehavior: 'replace',
        showCount: false,
        autoClose: false,
        actions: [],
      })

      if (progress.phase === 'verify' && progress.percent === 100) {
        Runtime.GetStatus({})
          .then((result) => {
            if (!result || result.state !== 'ready') {
              return
            }

            pushAlert({
              kind: 'success',
              placement: 'toast',
              title: t('onboarding.runtime.title'),
              message: t('onboarding.runtime.stateReady'),
              detail: result.version || null,
              dedupeKey: RUNTIME_TOAST_KEY,
              dedupeBehavior: 'replace',
              showCount: false,
              autoClose: true,
              durationMs: 4000,
              actions: [],
            })
          })
          .catch(() => undefined)
      }
    })

    return () => {
      off()
    }
  }, [activeTab, pushAlert, t])

  useEffect(() => {
    const loadDetail = async () => {
      if (!selectedExtensionId) {
        setSelectedExtensionConfig('')
        return
      }
      try {
        const result = await PluginBinding.GetExtensionDetail({ id: selectedExtensionId })
        setSelectedExtensionConfig(result?.config_text ?? '')
      } catch {
        setSelectedExtensionConfig('')
      }
    }
    void loadDetail()
  }, [selectedExtensionId, extensions])

  useEffect(() => {
    loginDialogRef.current = loginDialog
  }, [loginDialog])

  useEffect(() => {
    return () => {
      const activeDialog = loginDialogRef.current
      if (!activeDialog) {
        return
      }
      void Promise.resolve(
        PluginBinding.CancelLoginCli(new CancelLoginCliInput({ id: activeDialog.id })),
      ).catch(() => undefined)
    }
  }, [])

  let content: ReactNode

  if (activeTab === 'about') {
    content = (
      <SettingsSectionLayout>
        <AboutSettingsView version={version} />
      </SettingsSectionLayout>
    )
  } else if (activeTab === 'providers') {
    const selectedProvider = providers.find((provider) => provider.id === selectedProviderId) ?? providers[0] ?? null

    content = (
      <SettingsSectionLayout
        sidebarClassName="w-80"
        sidebar={(
          <ProviderList
            items={providers}
            selectedId={selectedProvider?.id ?? null}
            onSelect={setSelectedProviderId}
            onCreate={() => { void Window.OpenAddProvider({}) }}
            onToggleEnable={handleToggleEnable}
            onSetDefault={handleSetDefault}
            onDelete={handleDeleteFromList}
          />
        )}
      >
        <ProviderDetailView
          provider={selectedProvider}
          onUpdated={refreshProviders}
          onDeleted={deleteProvider}
          onSetDefault={handleSetDefault}
        />
      </SettingsSectionLayout>
    )
  } else if (activeTab === 'plugins') {
    const selectedExtension = extensions.find((extension) => extension.id === selectedExtensionId) ?? extensions[0] ?? null

    const handleNpmSubmit = async (npmPackage: string, name: string) => {
      try {
        const out = await PluginBinding.InstallCliFromNpm(
          new InstallCliFromNpmInput({ npm_package: npmPackage, name }),
        )
        if (out?.extension) {
          await refreshExtensions()
          setSelectedExtensionId(out.extension.id)
        }
      } catch (err) {
        pushAlert({ kind: 'error', placement: 'banner', title: '', message: String(err) })
      }
    }

    const handleSmartSubmit = async (content: string) => {
      const defaultProvider = providers.find((p) => p.id === defaultProviderId) ?? providers.find((p) => p.is_default) ?? providers[0]
      const defaultModel = defaultProvider?.models.find((m) => m.is_default) ?? defaultProvider?.models[0]
      try {
        const sessionId = await createConversation({
          title: t('settingsPage.plugins.cliFromDocsTask'),
          tags: ['task'],
        })
        if (sessionId) {
          setCurrentConversation(sessionId)
          cliInstallUpsert({
            session_id: sessionId,
            npm_package: `task:${sessionId}`,
            name: t('settingsPage.plugins.cliFromDocsTask'),
            phase: 'pending',
            detail: t('settingsPage.plugins.cliTaskPending'),
          })
          await sendMessage({
            sessionId,
            content,
            systemPrompt: '',
            skillName: 'install-cli-from-docs',
            baseUrl: defaultProvider?.base_url ?? '',
            apiKey: defaultProvider?.api_key ?? '',
            modelName: defaultModel?.model ?? '',
            providerType: (defaultProvider?.provider_type ?? '') as any,
            enabledUserTools: [],
            attachments: [],
          })
        }
        pushAlert({
          kind: 'success',
          placement: 'toast',
          title: '',
          message: t('settingsPage.plugins.cliTaskCreatedToast'),
        })
      } catch (err) {
        pushAlert({ kind: 'error', placement: 'banner', title: '', message: String(err) })
      }
    }

    const handleCreateExtension = async (kind: 'mcp' | 'plugin' | 'cli') => {
      if (kind === 'cli') {
        setCliModalOpen(true)
        return
      }
      const result = await FileBinding.SelectFolder(new SelectFolderInput())
      if (!result?.folder_path) return
      const imported = await PluginBinding.ImportExtension(new ImportExtensionInput({
        kind,
        path: result.folder_path,
      }))
      if (!imported?.extension) return
      await refreshExtensions()
      setSelectedExtensionId(imported.extension.id)
    }

    const handleToggleExtension = async (item: ExtensionItem) => {
      updateExtension({ ...item, enabled: !item.enabled })
      try {
        const result = await PluginBinding.ToggleExtension(new ToggleExtensionInput({
          id: item.id,
          enabled: !item.enabled,
        }))
        if (result?.extension) {
          updateExtension(mapExtensionItem(result.extension))
        }
      } catch {
        updateExtension(item)
      }
    }

    const handleReloadExtension = async (id: string) => {
      try {
        const result = await PluginBinding.ReloadExtension(new ReloadExtensionInput({ id }))
        if (result?.extension) {
          updateExtension(mapExtensionItem(result.extension))
        }
        if (selectedExtensionId === id) {
          const detail = await PluginBinding.GetExtensionDetail({ id })
          setSelectedExtensionConfig(detail?.config_text ?? '')
        }
      } catch {
        // ignore
      }
    }

    const handleSaveExtensionConfig = async (configText: string) => {
      if (!selectedExtension) return
      setPluginLoading(true)
      try {
        const result = await PluginBinding.SaveExtensionConfig(new SaveExtensionConfigInput({
          id: selectedExtension.id,
          config_text: configText,
        }))
        if (result?.extension) {
          updateExtension(mapExtensionItem(result.extension))
          setSelectedExtensionConfig(configText)
        }
      } finally {
        setPluginLoading(false)
      }
    }

    const handleDeleteExtension = async (id: string) => {
      try {
        await PluginBinding.DeleteExtension({ id })
        deleteExtension(id)
      } catch {
        // ignore
      }
    }

    const handleResetCliData = async (): Promise<void> => {
      if (!selectedExtension || selectedExtension.kind !== 'cli') return
      try {
        await PluginBinding.ResetCliData(new ResetCliDataInput({ id: selectedExtension.id }))
        await refreshExtensions()
      } catch (err: unknown) {
        console.error('reset cli data failed:', err)
      }
    }

    const handleRegenerateCliManifest = async (): Promise<void> => {
      if (!selectedExtension || selectedExtension.kind !== 'cli') return
      setPluginLoading(true)
      try {
        const result = await PluginBinding.GenerateCliManifest(new GenerateCliManifestInput({ id: selectedExtension.id }))
        if (result?.extension) {
          updateExtension(mapExtensionItem(result.extension))
        }
        const detail = await PluginBinding.GetExtensionDetail({ id: selectedExtension.id })
        setSelectedExtensionConfig(detail?.config_text ?? '')
      } finally {
        setPluginLoading(false)
      }
    }

    const handleLoginCli = async (): Promise<void> => {
      if (!selectedExtension || selectedExtension.kind !== 'cli') return
      setLoginDialog({
        id: selectedExtension.id,
        title: t('settingsPage.plugins.cliLoginDialogTitle', { name: selectedExtension.name }),
      })
    }

    const handleLoginStartError = (err: unknown): void => {
      const parsed = parseUnknownIError(err)
      pushAlert({
        kind: 'error',
        placement: 'banner',
        title: '',
        message: parsed.msg,
        detail: parsed.detail !== parsed.msg ? parsed.detail : null,
      })
    }

    const handleCancelLogin = async (): Promise<void> => {
      if (!loginDialog) return
      try {
        await PluginBinding.CancelLoginCli(new CancelLoginCliInput({ id: loginDialog.id }))
      } finally {
        setLoginDialog(null)
      }
    }

    content = (
      <>
        <SettingsSectionLayout
          sidebarClassName="w-80"
          sidebar={(
            <PluginToolList
              items={extensions}
              selectedId={selectedExtension?.id ?? null}
              onSelect={setSelectedExtensionId}
              onCreate={handleCreateExtension}
              onToggleEnable={handleToggleExtension}
              onReload={handleReloadExtension}
              onDelete={handleDeleteExtension}
            />
          )}
        >
          <PluginToolDetailView
            extension={selectedExtension}
            configText={selectedExtensionConfig}
            loading={pluginLoading}
            onReload={() => {
              if (selectedExtension) {
                void handleReloadExtension(selectedExtension.id)
              }
            }}
            onLoginCli={() => { void handleLoginCli() }}
            onSave={(configText) => { void handleSaveExtensionConfig(configText) }}
            onResetData={() => { void handleResetCliData() }}
            onRegenerateCliManifest={() => { void handleRegenerateCliManifest() }}
          />
        </SettingsSectionLayout>
        <CliInstallModal
          open={cliModalOpen}
          onClose={() => setCliModalOpen(false)}
          onNpmSubmit={(npmPackage, name) => { void handleNpmSubmit(npmPackage, name) }}
          onSmartSubmit={(content) => { void handleSmartSubmit(content) }}
        />
        {loginDialog ? (
          <CliLoginDialog
            open={true}
            extensionId={loginDialog.id}
            title={loginDialog.title}
            onStart={() => PluginBinding.LoginCli(new LoginCliInput({ id: loginDialog.id })).then(() => undefined)}
            onClose={() => setLoginDialog(null)}
            onCancel={() => { void handleCancelLogin() }}
            onStartError={handleLoginStartError}
            onDone={(exitCode: number) => {
              pushAlert({
                kind: exitCode === 0 ? 'success' : 'error',
                placement: 'toast',
                title: selectedExtension?.name ?? loginDialog.title,
                message: exitCode === 0
                  ? t('settingsPage.plugins.cliLoginDoneSuccess')
                  : t('settingsPage.plugins.cliLoginDoneFailureShort', { code: exitCode }),
                detail: null,
                autoClose: true,
                durationMs: 4000,
              })
            }}
          />
        ) : null}
      </>
    )
  } else if (activeTab === 'skills') {
    content = (
      <SettingsSectionLayout
        sidebarClassName="w-80"
        sidebar={<SkillsList />}
      >
        <SkillsDetailPane />
      </SettingsSectionLayout>
    )
  } else if (activeTab === 'memory') {
    content = (
      <SettingsSectionLayout
        sidebarClassName="w-56 lg:w-64 xl:w-80"
        sidebar={<MemoryList controller={memoryController} />}
      >
        <MemorySettingsView controller={memoryController} />
      </SettingsSectionLayout>
    )
  } else {
    content = (
      <SettingsSectionLayout
        sidebar={
          <SettingsSubmenuList
            title={t('settingsPage.primary.general')}
            items={generalItems}
            value={generalTab}
            onChange={setGeneralTab}
          />
        }
      >
        <GeneralSettingsPanel />
      </SettingsSectionLayout>
    )
  }

  return <SettingsShell>{content}</SettingsShell>
}

function buildRuntimeProgressDetail(
  t: (key: string) => string,
  progress: RuntimeProgressEvent,
): string {
  return [
    t(`onboarding.runtime.phase${capitalize(progress.phase)}`),
    `${formatBytes(progress.received)} / ${progress.total > 0 ? formatBytes(progress.total) : '…'}`,
  ].join(' · ')
}

function buildRuntimeProgressMessage(
  t: (key: string) => string,
  progress: RuntimeProgressEvent,
): string {
  return `${t('onboarding.runtime.stateDownloading')} ${progress.percent}% · ${buildRuntimeProgressDetail(t, progress)}`
}

function shouldShowRuntimeReminder(state: string): boolean {
  return state === 'pending_later' || state === 'missing'
}

function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1)
}

function formatBytes(value: number): string {
  if (value <= 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB']
  let unitIndex = 0
  let current = value

  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024
    unitIndex += 1
  }

  return `${current.toFixed(1)} ${units[unitIndex]}`
}
