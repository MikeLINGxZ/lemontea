import { useCallback, useEffect, useState } from 'react'
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
import { ImportExtensionInput, InstallCliFromNpmInput, ReloadExtensionInput, ResetCliDataInput, SaveExtensionConfigInput, ToggleExtensionInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto/models'
import { SelectFolderInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file/file_dto/models'
import type { ProviderWrapper } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import type { ExtensionItem as BindingExtensionItem } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models/models'
import type { ExtensionItem, ProviderItem } from '@/types/settings'
import { resolveProviderType } from '@/lib/provider'

/** 运行环境阶段的可选值 */
type RuntimePhase = 'download' | 'extract' | 'verify'

/** 运行环境进度事件的数据 */
type RuntimeProgressEvent = {
  /** 阶段 */
  phase: RuntimePhase
  /** 已接收数量 */
  received: number
  /** 总量 */
  total: number
  /** 百分比 */
  percent: number
}

/** 运行环境浮动提示键 */
const RUNTIME_TOAST_KEY = 'plugins-runtime-download-toast'

/** 将后端服务商信息转换为设置页面条目，接收w，返回对应的处理结果 */
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

/** 将后端扩展信息转换为设置页面条目，接收当前条目，返回对应的处理结果 */
function mapExtensionItem(item: BindingExtensionItem): ExtensionItem {
  /** 类别标识 */
  const kind: 'mcp' | 'plugin' | 'cli' = item.kind === 'plugin' || item.kind === 'cli' ? item.kind : 'mcp'
  return {
    ...item,
    kind,
    tools: item.tools ?? [],
  }
}

/** 协调设置分区、服务商、扩展和运行环境的加载与保存，返回组件内容 */
export function SettingsApp() {
  useSettingsBootstrap()
  const { t } = useTranslation() // 获取当前语言的翻译方法

  const setProviders = useSettingsStore((state) => state.setProviders) // 从状态仓库读取集合服务商列表
  const setExtensions = useSettingsStore((state) => state.setExtensions) // 从状态仓库读取集合扩展列表
  const updateProvider = useSettingsStore((state) => state.updateProvider) // 从状态仓库读取更新模型服务商
  const updateExtension = useSettingsStore((state) => state.updateExtension) // 从状态仓库读取更新扩展
  const deleteProvider = useSettingsStore((state) => state.deleteProvider) // 从状态仓库读取删除模型服务商
  const deleteExtension = useSettingsStore((state) => state.deleteExtension) // 从状态仓库读取删除扩展
  const setDefaultProvider = useSettingsStore((state) => state.setDefaultProvider) // 从状态仓库读取集合默认模型服务商
  const cliInstallUpsert = useCliInstallStore((state) => state.upsert) // 从状态仓库读取CLI安装新增或更新
  const createConversation = useChatStore((state) => state.createConversation) // 从状态仓库读取创建会话
  const sendMessage = useChatStore((state) => state.sendMessage) // 将用户消息和模型配置提交到后端，并记录本轮生成状态
  const setCurrentConversation = useChatStore((state) => state.setCurrentConversation) // 从状态仓库读取集合当前会话
  const [selectedExtensionConfig, setSelectedExtensionConfig] = useState<string>('') // 选中项扩展配置及其更新方法
  const [pluginLoading, setPluginLoading] = useState(false) // 插件加载中及其更新方法
  const [cliModalOpen, setCliModalOpen] = useState(false) // CLI模态框展开及其更新方法
  /** 刷新服务商列表，完成后结束异步任务 */
  const refreshProviders = useCallback(async () => {
    try {
      const result = await ProviderBinding.ListProviders({}) // 保存本次调用的结果
      if (result?.providers) {
        setProviders(result.providers.map(mapProviderItem))
      }
    } catch {
      // 忽略本次失败并保留当前界面状态
    }
  }, [setProviders])

  /** 刷新扩展列表，完成后结束异步任务 */
  const refreshExtensions = useCallback(async () => {
    try {
      const result = await PluginBinding.ListExtensions({}) // 保存本次调用的结果
      if (result?.extensions) {
        setExtensions(result.extensions.map(mapExtensionItem))
      }
    } catch {
      // 忽略本次失败并保留当前界面状态
    }
  }, [setExtensions])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 处理焦点，无返回值 */
    const handleFocus = () => {
      void refreshProviders()
      void refreshExtensions()
    }
    window.addEventListener('focus', handleFocus)
    /** 智能安装完成后刷新扩展列表，接收进度事件，无返回值 */
    const offInstallProgress = Events.On('cli.install.progress', (event: { /** 进度数据 */ data: { /** 安装阶段 */ phase: string } }) => {
      if (event.data.phase === 'done') void refreshExtensions()
    })
    return () => {
      window.removeEventListener('focus', handleFocus)
      offInstallProgress()
    }
  }, [refreshExtensions, refreshProviders])

  /** 处理切换启用，接收当前条目，完成后结束异步任务 */
  const handleToggleEnable = useCallback(async (item: ProviderItem) => {
    const updated = { ...item, enabled: !item.enabled } // 更新
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

  /** 处理集合默认，接收当前条目的唯一编号，完成后结束异步任务 */
  const handleSetDefault = useCallback(async (id: number) => {
    try {
      await ProviderBinding.SetDefault({ provider_id: id, model_id: null })
      setDefaultProvider(id)
    } catch {
      // 忽略本次失败并保留当前界面状态
    }
  }, [setDefaultProvider])

  /** 处理删除起点列表，接收当前条目的唯一编号，完成后结束异步任务 */
  const handleDeleteFromList = useCallback(async (id: number) => {
    try {
      await ProviderBinding.DeleteProvider({ provider_id: id })
      deleteProvider(id)
    } catch {
      // 忽略本次失败并保留当前界面状态
    }
  }, [deleteProvider])

  const activeTab = useSettingsStore((state) => state.activeTab) // 从状态仓库读取当前标签页
  const generalTab = useSettingsStore((state) => state.generalTab) // 从状态仓库读取常规标签页
  const setActiveTab = useSettingsStore((state) => state.setActiveTab) // 从状态仓库读取集合当前标签页
  const setGeneralTab = useSettingsStore((state) => state.setGeneralTab) // 从状态仓库读取集合常规标签页
  const version = useSettingsStore((state) => state.version) // 从状态仓库读取版本
  const providers = useSettingsStore((state) => state.providers) // 从状态仓库读取服务商列表
  const defaultProviderId = useSettingsStore((state) => state.defaultProviderId) // 从状态仓库读取默认模型服务商编号
  const extensions = useSettingsStore((state) => state.extensions) // 从状态仓库读取扩展列表
  const selectedProviderId = useSettingsStore((state) => state.selectedProviderId) // 从状态仓库读取选中项模型服务商编号
  const selectedExtensionId = useSettingsStore((state) => state.selectedExtensionId) // 从状态仓库读取选中项扩展编号
  /** 从状态仓库读取集合选中项模型服务商编号 */
  const setSelectedProviderId = useSettingsStore((state) => state.setSelectedProviderId)
  /** 从状态仓库读取集合选中项扩展编号 */
  const setSelectedExtensionId = useSettingsStore((state) => state.setSelectedExtensionId)
  const pushAlert = useAlertStore((state) => state.pushAlert) // 从状态仓库读取加入提醒
  const memoryController = useMemoryController(activeTab === 'memory') // 记忆控制器

  /** 常规条目列表 */
  const generalItems = [
    { key: 'display' as const, label: t('settingsPage.general.display.title'), icon: <MonitorSmartphone size={16} /> },
    { key: 'locale' as const, label: t('settingsPage.general.locale.title'), icon: <Languages size={16} /> },
    { key: 'shortcuts' as const, label: t('settingsPage.general.shortcuts.title'), icon: <Keyboard size={16} /> },
    { key: 'file' as const, label: t('settingsPage.general.file.title'), icon: <Globe2 size={16} className="rotate-45" /> },
  ]

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    const tab = new URLSearchParams(window.location.search).get('tab') // 标签页
    if (tab === 'general' || tab === 'providers' || tab === 'plugins' || tab === 'skills' || tab === 'memory' || tab === 'about') {
      setActiveTab(tab)
    }
  }, [setActiveTab])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    void refreshExtensions()
  }, [refreshExtensions])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    let cancelled = false // 已取消

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
              /** 处理点击事件，无返回值 */
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

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (activeTab !== 'plugins') {
      return
    }

    /** 订阅 runtime.node.progress 并保存解绑方法 */
    const off = Events.On('runtime.node.progress', (event: { /* 事件携带的数据 */ data: RuntimeProgressEvent }) => {
      const progress = event.data // 进度
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

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 加载详情，完成后结束异步任务 */
    const loadDetail = async () => {
      if (!selectedExtensionId) {
        setSelectedExtensionConfig('')
        return
      }
      try {
        const result = await PluginBinding.GetExtensionDetail({ id: selectedExtensionId }) // 保存本次调用的结果
        setSelectedExtensionConfig(result?.config_text ?? '')
      } catch {
        setSelectedExtensionConfig('')
      }
    }
    void loadDetail()
  }, [selectedExtensionId, extensions])

  let content: ReactNode // 内容

  if (activeTab === 'about') {
    content = (
      <SettingsSectionLayout>
        <AboutSettingsView version={version} />
      </SettingsSectionLayout>
    )
  } else if (activeTab === 'providers') {
    /** 选中项模型服务商 */
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
    /** 选中项扩展 */
    const selectedExtension = extensions.find((extension) => extension.id === selectedExtensionId) ?? extensions[0] ?? null

    /** 提交 npm 安装或重试，接收包名和显示名称，返回安装处理完成的异步任务 */
    const handleNpmSubmit = async (npmPackage: string, name: string): Promise<void> => {
      const requestId = `npm:${npmPackage}` // 同一 npm 包共用一条直接安装记录
      const { items, remove } = useCliInstallStore.getState() // 当前安装记录和移除操作
      if (items.some((item) => item.request_id === requestId && item.phase !== 'failed')) return

      const install = { request_id: requestId, npm_package: npmPackage, name } // 安装记录和重试参数
      cliInstallUpsert({ ...install, phase: 'pending', detail: '' })
      try {
        const out = await PluginBinding.InstallCliFromNpm(
          new InstallCliFromNpmInput({ npm_package: npmPackage, name }),
        ) // 安装结果
        if (!out?.extension?.id) {
          throw new Error(t('settingsPage.plugins.cliNpmMissingResult'))
        }

        // 直接使用安装结果更新列表，避免等待其他扩展刷新
        const extension = mapExtensionItem(out.extension) // 新安装的扩展
        const currentExtensions = useSettingsStore.getState().extensions // 安装完成时的扩展列表
        setExtensions([...currentExtensions.filter((item) => item.id !== extension.id), extension])
        setSelectedExtensionId(extension.id)
        remove(requestId)
        pushAlert({
          kind: 'success',
          placement: 'toast',
          title: extension.name,
          message: t('settingsPage.plugins.cliNpmInstalled'),
        })
      } catch (err) {
        const { msg, detail } = parseUnknownIError(err) // 后端错误摘要和详情
        cliInstallUpsert({ ...install, phase: 'failed', detail: msg === detail ? msg : `${msg}\n${detail}` })
      }
    }

    /** 处理智能提交，接收内容，完成后结束异步任务 */
    const handleSmartSubmit = async (content: string) => {
      /** 默认模型服务商 */
      const defaultProvider = providers.find((p) => p.id === defaultProviderId) ?? providers.find((p) => p.is_default) ?? providers[0]
      /** 默认模型 */
      const defaultModel = defaultProvider?.models.find((m) => m.is_default) ?? defaultProvider?.models[0]
      try {
        /** 保存会话编号 */
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
            providerType: resolveProviderType(defaultProvider?.provider_type),
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

    /** 处理创建扩展，接收类别标识，完成后结束异步任务 */
    const handleCreateExtension = async (kind: 'mcp' | 'plugin' | 'cli') => {
      if (kind === 'cli') {
        setCliModalOpen(true)
        return
      }
      const result = await FileBinding.SelectFolder(new SelectFolderInput()) // 保存本次调用的结果
      if (!result?.folder_path) return
      /** 保存已导入 */
      const imported = await PluginBinding.ImportExtension(new ImportExtensionInput({
        kind,
        path: result.folder_path,
      }))
      if (!imported?.extension) return
      await refreshExtensions()
      setSelectedExtensionId(imported.extension.id)
    }

    /** 处理切换扩展，接收当前条目，完成后结束异步任务 */
    const handleToggleExtension = async (item: ExtensionItem) => {
      updateExtension({ ...item, enabled: !item.enabled })
      try {
        /** 保存本次调用的结果 */
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

    /** 处理重新加载扩展，接收当前条目的唯一编号，完成后结束异步任务 */
    const handleReloadExtension = async (id: string) => {
      try {
        const result = await PluginBinding.ReloadExtension(new ReloadExtensionInput({ id })) // 保存本次调用的结果
        if (result?.extension) {
          updateExtension(mapExtensionItem(result.extension))
        }
        if (selectedExtensionId === id) {
          const detail = await PluginBinding.GetExtensionDetail({ id }) // 保存详情
          setSelectedExtensionConfig(detail?.config_text ?? '')
        }
      } catch {
        // 忽略本次失败并保留当前界面状态
      }
    }

    /** 处理保存扩展配置，接收配置文本，完成后结束异步任务 */
    const handleSaveExtensionConfig = async (configText: string) => {
      if (!selectedExtension) return
      setPluginLoading(true)
      try {
        /** 保存本次调用的结果 */
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

    /** 处理删除扩展，接收当前条目的唯一编号，完成后结束异步任务 */
    const handleDeleteExtension = async (id: string) => {
      try {
        await PluginBinding.DeleteExtension({ id })
        deleteExtension(id)
      } catch {
        // 忽略本次失败并保留当前界面状态
      }
    }

    /** 处理重置CLI数据，完成后结束异步任务 */
    const handleResetCliData = async (): Promise<void> => {
      if (!selectedExtension || selectedExtension.kind !== 'cli') return
      try {
        await PluginBinding.ResetCliData(new ResetCliDataInput({ id: selectedExtension.id }))
        await refreshExtensions()
      } catch (err: unknown) {
        console.error('reset cli data failed:', err)
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
              onRetryNpm={handleNpmSubmit}
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
            onSave={(configText) => { void handleSaveExtensionConfig(configText) }}
            onResetData={() => { void handleResetCliData() }}
          />
        </SettingsSectionLayout>
        <CliInstallModal
          open={cliModalOpen}
          onClose={() => setCliModalOpen(false)}
          onNpmSubmit={handleNpmSubmit}
          onSmartSubmit={(content) => { void handleSmartSubmit(content) }}
        />
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

/** 构造运行环境进度详情，接收当前语言的翻译函数、进度，返回处理后的文本 */
function buildRuntimeProgressDetail(
  t: (key: string) => string,
  progress: RuntimeProgressEvent,
): string {
  return [
    t(`onboarding.runtime.phase${capitalize(progress.phase)}`),
    `${formatBytes(progress.received)} / ${progress.total > 0 ? formatBytes(progress.total) : '…'}`,
  ].join(' · ')
}

/** 构造运行环境进度消息，接收当前语言的翻译函数、进度，返回处理后的文本 */
function buildRuntimeProgressMessage(
  t: (key: string) => string,
  progress: RuntimeProgressEvent,
): string {
  return `${t('onboarding.runtime.stateDownloading')} ${progress.percent}% · ${buildRuntimeProgressDetail(t, progress)}`
}

/** 判断是否需要显示运行环境提醒，接收当前状态，返回是否满足条件 */
function shouldShowRuntimeReminder(state: string): boolean {
  return state === 'pending_later' || state === 'missing'
}

/** 将首字母转为大写处理，接收值，返回处理后的文本 */
function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1)
}

/** 将字节数转换为带单位的显示文本，接收值，返回处理后的文本 */
function formatBytes(value: number): string {
  if (value <= 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB'] // 单位列表
  let unitIndex = 0 // 单位索引
  let current = value // 当前

  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024
    unitIndex += 1
  }

  return `${current.toFixed(1)} ${units[unitIndex]}`
}
