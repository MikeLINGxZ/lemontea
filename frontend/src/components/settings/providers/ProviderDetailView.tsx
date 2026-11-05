import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw, Plus, Eye, EyeOff, Search, X, Check, Info } from 'lucide-react'
import { Provider as ProviderBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import { EditProviderInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import { Model as ViewModel } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model/models'
import { ConfirmDialog } from '@/components/settings/common/ConfirmDialog'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { cn } from '@/lib/utils'
import type { ProviderItem, ProviderModel } from '@/types/settings'

/** 加载选中的模型服务商，并显示配置和模型列表，接收组件输入属性，返回组件内容 */
export function ProviderDetailView(props: {
  /** 模型服务商 */
  provider: ProviderItem | null
  /** 处理更新事件，返回对应的处理结果 */
  onUpdated: () => void | Promise<void>
  /** 处理已删除事件，接收当前条目的唯一编号，无返回值 */
  onDeleted: (id: number) => void
  /** 处理集合默认事件，接收当前条目的唯一编号，无返回值 */
  onSetDefault?: (id: number) => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法

  if (!props.provider) {
    return (
      <SettingsContentLayout
        header={
          <SettingsPanelHeader
            title={t('settingsPage.primary.providers')}
            description={t('settingsPage.addProvider.doneMessage')}
          />
        }
      />
    )
  }

  return (
    <ProviderDetailInner
      provider={props.provider}
      onUpdated={props.onUpdated}
      onDeleted={props.onDeleted}
      onSetDefault={props.onSetDefault}
    />
  )
}

/** 编辑模型服务商的连接信息、自定义模型和默认项，接收组件输入属性，返回组件内容 */
function ProviderDetailInner(props: {
  /** 模型服务商 */
  provider: ProviderItem
  /** 处理更新事件，返回对应的处理结果 */
  onUpdated: () => void | Promise<void>
  /** 处理已删除事件，接收当前条目的唯一编号，无返回值 */
  onDeleted: (id: number) => void
  /** 处理集合默认事件，接收当前条目的唯一编号，无返回值 */
  onSetDefault?: (id: number) => void
}) {
  const { provider } = props // 模型服务商
  const { t } = useTranslation() // 获取当前语言的翻译方法

  const [enable, setEnable] = useState(provider.enabled) // 启用及其更新方法
  const [name, setName] = useState(provider.provider_name) // 名称及其更新方法
  const [baseUrl, setBaseUrl] = useState(provider.base_url) // 服务的基础地址及其更新方法
  const [apiKey, setApiKey] = useState(provider.api_key) // 服务认证使用的 API 密钥及其更新方法
  const [showApiKey, setShowApiKey] = useState(false) // 显示API键及其更新方法
  /** 默认模型及其更新方法 */
  const [defaultModel, setDefaultModel] = useState(
    provider.models.find(m => m.is_default)?.model ?? provider.models[0]?.model ?? ''
  )
  const [models, setModels] = useState<ProviderModel[]>(provider.models) // 模型列表及其更新方法
  const [refreshing, setRefreshing] = useState(false) // 正在刷新及其更新方法
  const [applying, setApplying] = useState(false) // 正在应用及其更新方法
  const [deleting, setDeleting] = useState(false) // 删除中及其更新方法
  const [settingDefault, setSettingDefault] = useState(false) // 设置默认及其更新方法
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false) // 确认删除展开及其更新方法
  const [addingCustom, setAddingCustom] = useState(false) // 正在添加自定义及其更新方法
  const [customModelInput, setCustomModelInput] = useState('') // 自定义模型输入及其更新方法
  const [searchOpen, setSearchOpen] = useState(false) // 搜索展开及其更新方法
  const [searchQuery, setSearchQuery] = useState('') // 搜索词及其更新方法
  const searchInputRef = useRef<HTMLInputElement>(null) // 搜索输入引用，供后续读取当前值
  const customInputRef = useRef<HTMLInputElement>(null) // 自定义输入引用，供后续读取当前值

  const initialRef = useRef(buildInitialState(provider)) // 初始引用，供后续读取当前值

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    const nextInitial = buildInitialState(provider) // 下一项初始
    setEnable(nextInitial.enable)
    setName(nextInitial.name)
    setBaseUrl(nextInitial.baseUrl)
    setApiKey(nextInitial.apiKey)
    setDefaultModel(nextInitial.defaultModel)
    setModels(provider.models)
    setShowApiKey(false)
    setRefreshing(false)
    setApplying(false)
    setDeleting(false)
    setSettingDefault(false)
    setConfirmDeleteOpen(false)
    setAddingCustom(false)
    setCustomModelInput('')
    setSearchOpen(false)
    setSearchQuery('')
    initialRef.current = nextInitial
  }, [provider])

  /** 有未保存修改 */
  const dirty =
    enable !== initialRef.current.enable ||
    name !== initialRef.current.name ||
    baseUrl !== initialRef.current.baseUrl ||
    apiKey !== initialRef.current.apiKey ||
    defaultModel !== initialRef.current.defaultModel ||
    models.some(m => m.id === 0) ||
    models.filter(m => m.id > 0).length !== initialRef.current.modelIds.size

  /** 筛选结果模型列表 */
  const filteredModels = searchOpen && searchQuery
    ? models.filter(m => m.model.toLowerCase().includes(searchQuery.toLowerCase()))
    : models

  /** 处理刷新模型列表，完成后结束异步任务 */
  const handleRefreshModels = async () => {
    if (!baseUrl || refreshing) return
    setRefreshing(true)
    try {
      /** 保存本次调用的结果 */
      const result = await ProviderBinding.RequestProviderModelList({ base_url: baseUrl, api_key: apiKey })
      if (result?.models) {
        /** 后端返回的数据 */
        const fetched = result.models.map((m) => ({
          id: 0, provider_id: provider.id, model: m.model,
          owned_by: m.owned_by, object: m.object, enable: true,
          alias: null, is_custom: false, is_default: false,
        }))
        const existing = new Map<string, ProviderModel>() // 已有匹配项
        for (/* 当前模型 */ const m of models) existing.set(m.model, m)
        for (/* 当前模型 */ const m of fetched) {
          if (!existing.has(m.model)) existing.set(m.model, m)
        }
        const merged = Array.from(existing.values()) // 合并
        setModels(merged)
        if (merged.length > 0 && !defaultModel) setDefaultModel(merged[0].model)
      }
    } catch {
      // 忽略本次失败并保留当前界面状态
    } finally {
      setRefreshing(false)
    }
  }

  /** 处理集合默认模型，接收正在处理的消息，完成后结束异步任务 */
  const handleSetDefaultModel = async (m: ProviderModel) => {
    const prev = defaultModel // 更新前的值
    setDefaultModel(m.model)
    if (m.id > 0) {
      try {
        await ProviderBinding.SetDefault({ provider_id: provider.id, model_id: m.id })
        initialRef.current = { ...initialRef.current, defaultModel: m.model }
      } catch {
        setDefaultModel(prev)
      }
    }
  }

  /** 处理应用，完成后结束异步任务 */
  const handleApply = async () => {
    if (!dirty || applying) return
    setApplying(true)
    try {
      const currentIds = new Set(models.filter(m => m.id > 0).map(m => m.id)) // 当前会话编号集合
      for (/* 当前条目的唯一编号 */ const id of initialRef.current.modelIds) {
        if (!currentIds.has(id)) {
          await ProviderBinding.DeleteModel({ model_id: id })
        }
      }

      /** 新模型列表 */
      const newModels = models
        .filter(m => m.id === 0)
        .map(m => new ViewModel({
          id: 0, provider_id: provider.id, model: m.model,
          owned_by: m.owned_by, object: m.object,
          enable: m.enable, alias: m.alias, is_custom: m.is_custom, is_default: false,
        }))

      await ProviderBinding.EditProvider(new EditProviderInput({
        provider_id: provider.id,
        provider_name: name,
        base_url: baseUrl,
        api_key: apiKey,
        enable,
        default_model: defaultModel || null,
        models: newModels,
      }))

      await props.onUpdated()
    } catch {
      // 保留表单和输入，供用户重试
    } finally {
      setApplying(false)
    }
  }

  /** 处理删除，完成后结束异步任务 */
  const handleDelete = async () => {
    if (deleting) return
    setDeleting(true)
    try {
      await ProviderBinding.DeleteProvider({ provider_id: provider.id })
      props.onDeleted(provider.id)
    } catch {
      setDeleting(false)
    }
  }

  /** 处理集合默认模型服务商，完成后结束异步任务 */
  const handleSetDefaultProvider = async () => {
    if (settingDefault) return
    setSettingDefault(true)
    try {
      await ProviderBinding.SetDefault({ provider_id: provider.id, model_id: null })
      props.onSetDefault?.(provider.id)
    } catch {
      // 忽略本次失败并保留当前界面状态
    } finally {
      setSettingDefault(false)
    }
  }

  /** 打开搜索，无返回值 */
  const openSearch = () => {
    setSearchOpen(true)
    requestAnimationFrame(() => searchInputRef.current?.focus())
  }

  /** 关闭搜索，无返回值 */
  const closeSearch = () => {
    setSearchOpen(false)
    setSearchQuery('')
  }

  /** 处理新增自定义模型，接收保留展开，无返回值 */
  const handleAddCustomModel = (keepOpen = false) => {
    const trimmed = customModelInput.trim() // 已去除空白
    if (trimmed && !models.some(m => m.model === trimmed)) {
      setModels(prev => [...prev, {
        id: 0, provider_id: provider.id, model: trimmed,
        owned_by: '', object: '', enable: true,
        alias: null, is_custom: true, is_default: false,
      }])
    }
    setCustomModelInput('')
    if (keepOpen) {
      requestAnimationFrame(() => customInputRef.current?.focus())
    } else {
      setAddingCustom(false)
    }
  }

  /** 处理开始新增自定义，无返回值 */
  const handleStartAddCustom = () => {
    if (addingCustom) {
      if (customModelInput.trim() === '') return
      handleAddCustomModel(true)
      return
    }
    setAddingCustom(true)
    setSearchOpen(false)
  }

  /** 处理取消，无返回值 */
  const handleCancel = () => {
    const s = initialRef.current // 当前状态或候选项
    setEnable(s.enable)
    setName(s.name)
    setBaseUrl(s.baseUrl)
    setApiKey(s.apiKey)
    setDefaultModel(s.defaultModel)
    setModels(provider.models)
    setAddingCustom(false)
    setCustomModelInput('')
  }

  /** 处理删除模型，接收正在处理的消息，无返回值 */
  const handleDeleteModel = (m: ProviderModel) => {
    setModels(prev => prev.filter(p => p.model !== m.model))
    if (defaultModel === m.model) {
      const remaining = models.filter(p => p.model !== m.model) // 剩余
      setDefaultModel(remaining[0]?.model ?? '')
    }
  }

  return (
    <SettingsContentLayout
      noContentScroll
      header={
      <SettingsPanelHeader
          title={name}
        />
      }
      footprint={
        <SettingsActionBar
          primaryLabel={t('settingsPage.actions.apply')}
          primaryDisabled={!dirty || applying}
          onPrimaryClick={() => { void handleApply() }}
          secondaryLabel={dirty ? t('settingsPage.providers.cancel') : (!provider.is_default ? t('settingsPage.providers.setDefault') : undefined)}
          secondaryDisabled={dirty ? false : (!provider.is_default ? settingDefault : undefined)}
          onSecondaryClick={dirty ? handleCancel : (!provider.is_default ? () => { void handleSetDefaultProvider() } : undefined)}
          dangerLabel={t('settingsPage.providers.delete')}
          onDangerClick={() => setConfirmDeleteOpen(true)}
        />
      }
    >
      <div className="flex min-h-0 flex-1 flex-col space-y-5 pb-4 pt-2">
        {/* API 密钥存储和隐私说明 */}
        <div className="flex shrink-0 items-start gap-2 rounded-xl bg-blue-500/10 px-3 py-2.5 text-sm text-blue-700 dark:text-blue-400">
          <Info size={14} className="mt-0.5 shrink-0" />
          <span>{t('settingsPage.providers.apiKeyPrivacyNotice')}</span>
        </div>

        {/* 服务商启用开关 */}
        <div className="shrink-0 flex flex-col gap-2">
          <span className="text-sm font-medium text-foreground">{t('settingsPage.addProvider.form.enable')}</span>
          <button
            type="button"
            role="switch"
            aria-checked={enable}
            aria-label={t('settingsPage.addProvider.form.enable')}
            onClick={() => setEnable(v => !v)}
            className={cn(
              'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors duration-200 ease-in-out',
              enable ? 'bg-switch-on' : 'bg-switch-off'
            )}
          >
            <span className={cn(
              'inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform duration-200 ease-in-out',
              enable ? 'translate-x-6' : 'translate-x-1'
            )} />
          </button>
        </div>

        {/* 服务商名称输入框 */}
        <div className="shrink-0 space-y-1.5">
          <label className="text-sm font-medium text-foreground">
            {t('settingsPage.addProvider.form.name')} <span className="text-destructive">*</span>
          </label>
          <input
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm outline-hidden focus:border-primary"
          />
        </div>

        {/* 模型服务地址 */}
        <div className="shrink-0 space-y-1.5">
          <label className="text-sm font-medium text-foreground">
            {t('settingsPage.addProvider.form.baseUrl')} <span className="text-destructive">*</span>
          </label>
          <input
            required
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm outline-hidden focus:border-primary"
          />
        </div>

        {/* 模型服务密钥及显示切换按钮 */}
        <div className="shrink-0 space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('settingsPage.addProvider.form.apiKey')}</label>
          <div className="flex items-center rounded-xl border border-border bg-background px-3 py-2 focus-within:border-primary">
            <input
              type={showApiKey ? 'text' : 'password'}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="min-w-0 flex-1 bg-transparent text-sm outline-hidden"
            />
            <button
              type="button"
              onClick={() => setShowApiKey(v => !v)}
              title={showApiKey ? t('settingsPage.providers.hideApiKey') : t('settingsPage.providers.showApiKey')}
              className="ml-2 shrink-0 text-muted-foreground transition-colors hover:text-foreground"
            >
              {showApiKey ? <EyeOff size={14} /> : <Eye size={14} />}
            </button>
          </div>
        </div>

        {/* 可配置的模型列表 */}
        <div className="flex min-h-0 flex-1 flex-col space-y-1.5">
          <div className="flex shrink-0 items-center justify-between">
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-foreground">{t('settingsPage.addProvider.form.modelList')}</label>
              <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                {t('settingsPage.providers.modelsCount', { count: models.length })}
              </span>
            </div>
            <div className="flex items-center gap-1">
              {/* 从右向左展开的模型搜索框 */}
              <div className={cn(
                'flex items-center overflow-hidden rounded-lg border border-border bg-background px-2 transition-all duration-200',
                searchOpen ? 'w-36 opacity-100' : 'w-0 border-0 opacity-0'
              )}>
                <input
                  ref={searchInputRef}
                  tabIndex={searchOpen ? 0 : -1}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder={t('settingsPage.providers.searchModelPlaceholder')}
                  className="min-w-0 flex-1 bg-transparent py-1 text-xs outline-hidden"
                />
                <button
                  type="button"
                  onClick={closeSearch}
                  className="ml-1 shrink-0 text-muted-foreground hover:text-foreground"
                >
                  <X size={11} />
                </button>
              </div>
              {/* 模型搜索入口 */}
              {!searchOpen && (
                <button
                  type="button"
                  onClick={openSearch}
                  title={t('settingsPage.providers.searchModelPlaceholder')}
                  className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent"
                >
                  <Search size={14} />
                </button>
              )}
              <button
                type="button"
                onClick={() => { void handleRefreshModels() }}
                disabled={refreshing}
                title={t('settingsPage.addProvider.form.refreshModels')}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent disabled:opacity-40"
              >
                <RefreshCw size={14} className={refreshing ? 'animate-spin' : ''} />
              </button>
              <button
                type="button"
                onClick={handleStartAddCustom}
                title={t('settingsPage.addProvider.form.addCustomModel')}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent"
              >
                <Plus size={14} />
              </button>
            </div>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto rounded-xl border border-border">
            {models.length === 0 && (
              <div className="flex items-center justify-center gap-1.5 px-3 py-4 text-xs text-muted-foreground">
                {refreshing ? (
                  <RefreshCw size={12} className="animate-spin" />
                ) : (
                  <>
                    {t('settingsPage.addProvider.form.noModels')}
                    <span
                      className="cursor-pointer underline hover:text-foreground"
                      onClick={() => { void handleRefreshModels() }}
                    >
                      {t('settingsPage.addProvider.form.refreshModels')}
                    </span>
                  </>
                )}
              </div>
            )}
            {models.length > 0 && filteredModels.length === 0 && searchQuery && (
              <div className="flex items-center justify-center px-3 py-4 text-xs text-muted-foreground">
                {t('settingsPage.providers.noSearchResults')}
              </div>
            )}
            {filteredModels.map((m, idx) => (
              <div key={`${m.is_custom ? 'custom' : 'fetched'}-${m.model}-${idx}`} className="group flex items-center gap-2 px-3 py-2 hover:bg-accent">
                <span className="min-w-0 flex-1 truncate text-sm">{m.model}</span>
                <span className="flex shrink-0 items-center gap-1.5">
                  {defaultModel !== m.model && (
                    <button
                      type="button"
                      onClick={() => { void handleSetDefaultModel(m) }}
                      className="hidden rounded px-1.5 py-0.5 text-[10px] text-muted-foreground hover:text-foreground group-hover:inline-block"
                    >
                      {t('settingsPage.addProvider.form.setDefault')}
                    </button>
                  )}
                  {m.is_custom && (
                    <button
                      type="button"
                      onClick={() => handleDeleteModel(m)}
                      className="hidden rounded px-1.5 py-0.5 text-[10px] text-destructive/70 hover:text-destructive group-hover:inline-block"
                    >
                      {t('settingsPage.providers.delete')}
                    </button>
                  )}
                  {defaultModel === m.model && (
                    <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] text-primary">
                      {t('settingsPage.addProvider.form.defaultTag')}
                    </span>
                  )}
                  {m.is_custom && (
                    <span className="rounded border border-border px-1.5 py-0.5 text-[10px] text-muted-foreground">
                      {t('settingsPage.addProvider.form.customTag')}
                    </span>
                  )}
                </span>
              </div>
            ))}
            {addingCustom && (
              <div className="flex items-center gap-2 border-t border-border px-3 py-2">
                <input
                  autoFocus
                  ref={customInputRef}
                  value={customModelInput}
                  onChange={(e) => setCustomModelInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleAddCustomModel()
                    if (e.key === 'Escape') { setAddingCustom(false); setCustomModelInput('') }
                  }}
                  placeholder={t('settingsPage.addProvider.form.customModelPlaceholder')}
                  className="min-w-0 flex-1 bg-transparent text-sm outline-hidden"
                />
                <button
                  type="button"
                  onClick={() => handleAddCustomModel()}
                  disabled={customModelInput.trim() === ''}
                  className="flex items-center justify-center text-primary disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <Check size={13} />
                </button>
                <button
                  type="button"
                  onClick={() => { setAddingCustom(false); setCustomModelInput('') }}
                  className="flex items-center justify-center text-muted-foreground hover:text-foreground"
                >
                  <X size={13} />
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={confirmDeleteOpen}
        title={t('settingsPage.providers.confirmDelete')}
        description={name}
        confirmLabel={t('settingsPage.providers.delete')}
        cancelLabel={t('settingsPage.providers.cancel')}
        confirmTone="danger"
        busy={deleting}
        onConfirm={() => { void handleDelete() }}
        onCancel={() => {
          if (!deleting) setConfirmDeleteOpen(false)
        }}
      />
    </SettingsContentLayout>
  )
}

/** 构造初始状态，接收模型服务商，返回对应的处理结果 */
function buildInitialState(provider: ProviderItem) {
  return {
    enable: provider.enabled,
    name: provider.provider_name,
    baseUrl: provider.base_url,
    apiKey: provider.api_key,
    defaultModel: provider.models.find(m => m.is_default)?.model ?? provider.models[0]?.model ?? '',
    modelIds: new Set(provider.models.map(m => m.id).filter(id => id > 0)),
  }
}
