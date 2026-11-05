import { useRef, useState } from 'react'
import { RefreshCw, Plus, Check, X, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Onboarding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/onboarding/index'
import { Provider as ProviderBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import { CreateProviderInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import { Model as ViewModel } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model/models'
import { resolveProviderType } from '@/lib/provider'
import { useAlertStore } from '@/alert/store'
import { parseUnknownIError } from '@/lib/ierror'
import { cn } from '@/lib/utils'
import type { SupportedProvider } from '@/types/settings'

/** 定义后端返回的数据模型的数据结构 */
type FetchedModel = { /* 模型 */ model: string; /* 模型所属服务商 */ owned_by: string; /* 对象 */ object: string; /* 来源 */ source: 'fetched' | 'custom' }

/** 填写服务商连接参数并验证可选模型，接收组件输入属性，返回组件内容 */
export function AddProviderStepForm(props: {
  /** 模型服务商 */
  provider: SupportedProvider
  /** 处理完成事件，无返回值 */
  onDone: () => void
  /** 模式 */
  mode?: 'settings' | 'onboarding'
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const pushAlert = useAlertStore((state) => state.pushAlert) // 从状态仓库读取加入提醒
  const [enable, setEnable] = useState(true) // 启用及其更新方法
  const [name, setName] = useState(props.provider.name) // 名称及其更新方法
  const [apiKey, setApiKey] = useState('') // 服务认证使用的 API 密钥及其更新方法
  const [baseUrl, setBaseUrl] = useState(props.provider.base_url) // 服务的基础地址及其更新方法
  const [defaultModel, setDefaultModel] = useState('') // 默认模型及其更新方法
  const [models, setModels] = useState<FetchedModel[]>([]) // 模型列表及其更新方法
  const [refreshing, setRefreshing] = useState(false) // 正在刷新及其更新方法
  const [submitting, setSubmitting] = useState(false) // 正在提交及其更新方法
  const [addingCustom, setAddingCustom] = useState(false) // 正在添加自定义及其更新方法
  const [customModelInput, setCustomModelInput] = useState('') // 自定义模型输入及其更新方法
  const customInputRef = useRef<HTMLInputElement>(null) // 自定义输入引用，供后续读取当前值
  const customModelsRef = useRef<FetchedModel[]>([]) // 自定义模型列表引用，供后续读取当前值
  const searchInputRef = useRef<HTMLInputElement>(null) // 搜索输入引用，供后续读取当前值
  const [searchOpen, setSearchOpen] = useState(false) // 搜索展开及其更新方法
  const [searchQuery, setSearchQuery] = useState('') // 搜索词及其更新方法

  /** 处理刷新模型列表，完成后结束异步任务 */
  const handleRefreshModels = async () => {
    if (!baseUrl || refreshing) return
    setRefreshing(true)
    try {
      /** 保存本次调用的结果 */
      const result = await ProviderBinding.RequestProviderModelList({ base_url: baseUrl, api_key: apiKey })
      if (result?.models) {
        /** 后端返回的数据 */
        const fetched = result.models.map((m) => ({ model: m.model, owned_by: m.owned_by, object: m.object, source: 'fetched' as const }))
        const existing = new Map<string, FetchedModel>() // 已有匹配项
        for (/* 当前模型 */ const m of customModelsRef.current) existing.set(m.model, m)
        for (/* 当前模型 */ const m of fetched) {
          if (!existing.has(m.model)) existing.set(m.model, m)
        }
        const merged = Array.from(existing.values()) // 合并
        setModels(merged)
        if (merged.length > 0 && !defaultModel) setDefaultModel(merged[0].model)
      }
    } catch (err) {
      const parsed = parseUnknownIError(err) // 从文本解析出的数据
      pushAlert({
        kind: 'error',
        placement: 'banner',
        title: '',
        message: parsed.msg || t('settingsPage.addProvider.errors.fetchModels'),
        detail: parsed.detail !== parsed.msg ? parsed.detail : null,
      })
    } finally {
      setRefreshing(false)
    }
  }

  /** 处理确认新增自定义，接收保留展开，无返回值 */
  const handleConfirmAddCustom = (keepOpen = false) => {
    const trimmed = customModelInput.trim() // 已去除空白
    if (!trimmed || models.some((m) => m.model === trimmed)) {
      if (keepOpen) {
        setCustomModelInput('')
        requestAnimationFrame(() => customInputRef.current?.focus())
      } else {
        setAddingCustom(false)
        setCustomModelInput('')
      }
      return
    }
    /** 自定义 */
    const custom: FetchedModel = { model: trimmed, owned_by: '', object: 'model', source: 'custom' }
    customModelsRef.current = [...customModelsRef.current, custom]
    setModels((prev) => [...prev, custom])
    if (!defaultModel) setDefaultModel(trimmed)
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
      handleConfirmAddCustom(true)
      return
    }
    setAddingCustom(true)
    setCustomModelInput('')
    requestAnimationFrame(() => customInputRef.current?.focus())
  }

  /** 处理取消新增自定义，无返回值 */
  const handleCancelAddCustom = () => {
    setAddingCustom(false)
    setCustomModelInput('')
  }

  /** 处理删除自定义模型，接收模型名称，无返回值 */
  const handleDeleteCustomModel = (modelName: string) => {
    customModelsRef.current = customModelsRef.current.filter((m) => m.model !== modelName)
    setModels((prev) => {
      const next = prev.filter((m) => !(m.source === 'custom' && m.model === modelName)) // 更新后的值
      if (defaultModel === modelName && next.length > 0) setDefaultModel(next[0].model)
      else if (defaultModel === modelName) setDefaultModel('')
      return next
    })
  }

  /** 处理提交，接收当前事件或条目，完成后结束异步任务 */
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    setSubmitting(true)

    try {
      /** 全部模型列表 */
      const allModels: ViewModel[] = models.map((m) => new ViewModel({
        id: 0,
        provider_id: 0,
        model: m.model,
        owned_by: m.owned_by,
        object: m.object,
        enable: true,
        alias: null,
        is_custom: m.source === 'custom',
        is_default: false,
      }))

      /** 输入 */
      const input = new CreateProviderInput({
        provider_name: name,
        provider_type: resolveProviderType(props.provider.type),
        base_url: baseUrl,
        api_key: apiKey,
        enable,
        default_model: defaultModel || null,
        models: allModels,
      })

      if (props.mode === 'onboarding') {
        await Onboarding.SaveProviderAndMarkInitialized({
          provider_name: input.provider_name,
          provider_type: input.provider_type,
          base_url: input.base_url,
          api_key: input.api_key,
          enable: input.enable,
          default_model: input.default_model,
          models: input.models,
        })
      } else {
        await ProviderBinding.CreateProvider(input)
      }
      props.onDone()
    } catch (err) {
      const parsed = parseUnknownIError(err) // 从文本解析出的数据
      pushAlert({
        kind: 'error',
        placement: 'banner',
        title: '',
        message: parsed.msg || t('settingsPage.addProvider.errors.createProvider'),
        detail: parsed.detail !== parsed.msg ? parsed.detail : null,
      })
    } finally {
      setSubmitting(false)
    }
  }

  /** 筛选结果模型列表 */
  const filteredModels = searchOpen && searchQuery
    ? models.filter(m => m.model.toLowerCase().includes(searchQuery.toLowerCase()))
    : models

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

  return (
    <form id="add-provider-form" onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col space-y-5 pb-4 pt-2">
      {/* 服务商启用开关 */}
      <div className="flex shrink-0 flex-col gap-2">
        <span className="text-sm font-medium text-foreground">{t('settingsPage.addProvider.form.enable')}</span>
        <button
          type="button"
          role="switch"
          aria-checked={enable}
          onClick={() => setEnable((v) => !v)}
          className={cn(
            'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors',
            enable ? 'bg-switch-on' : 'bg-switch-off'
          )}
        >
          <span className={cn(
            'inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform',
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

      {/* 模型服务的 API 密钥 */}
      <div className="shrink-0 space-y-1.5">
        <label className="text-sm font-medium text-foreground">{t('settingsPage.addProvider.form.apiKey')}</label>
        <input
          type="password"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm outline-hidden focus:border-primary"
        />
      </div>

      {/* 可配置的模型列表 */}
      <div className="flex min-h-0 flex-1 flex-col space-y-1.5">
        <div className="flex shrink-0 items-center justify-between">
          <label className="text-sm font-medium text-foreground">{t('settingsPage.addProvider.form.modelList')}</label>
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
              onClick={handleRefreshModels}
              disabled={refreshing}
              title={t('settingsPage.addProvider.form.refreshModels')}
              aria-label={t('settingsPage.addProvider.form.refreshModels')}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent disabled:opacity-40"
            >
              <RefreshCw size={14} className={refreshing ? 'animate-spin' : ''} />
            </button>
            <button
              type="button"
              onClick={handleStartAddCustom}
              title={t('settingsPage.addProvider.form.addCustomModel')}
              aria-label={t('settingsPage.addProvider.form.addCustomModel')}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent"
            >
              <Plus size={14} />
            </button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto rounded-xl border border-border">
          {addingCustom && (
            <div className="flex items-center gap-1.5 border-b border-border px-3 py-2">
              <input
                ref={customInputRef}
                value={customModelInput}
                onChange={(e) => setCustomModelInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleConfirmAddCustom(); else if (e.key === 'Escape') handleCancelAddCustom() }}
                placeholder={t('settingsPage.addProvider.form.customModelPlaceholder')}
                className="min-w-0 flex-1 bg-transparent text-sm outline-hidden"
              />
              <button
                type="button"
                onClick={() => handleConfirmAddCustom()}
                disabled={customModelInput.trim() === ''}
                className="flex h-6 w-6 items-center justify-center rounded text-primary hover:bg-accent disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent"
              >
                <Check size={14} />
              </button>
              <button type="button" onClick={handleCancelAddCustom} className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground hover:bg-accent">
                <X size={14} />
              </button>
            </div>
          )}
          {models.length === 0 && !addingCustom && (
            <div className="flex items-center justify-center gap-1.5 px-3 py-4 text-xs text-muted-foreground">
              {refreshing ? (
                <RefreshCw size={12} className="animate-spin" />
              ) : (
                <>
                  {t('settingsPage.addProvider.form.noModels')}
                  <span
                    className="cursor-pointer underline hover:text-foreground"
                    onClick={handleRefreshModels}
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
            <div
              key={`${m.source}-${m.model}-${idx}`}
              className="group flex items-center gap-2 px-3 py-2 hover:bg-accent"
            >
              <span className="min-w-0 flex-1 truncate text-sm">{m.model}</span>
              <span className="flex shrink-0 items-center gap-1.5">
                {defaultModel !== m.model && (
                  <button
                    type="button"
                    onClick={() => setDefaultModel(m.model)}
                    className="hidden rounded px-1.5 py-0.5 text-[10px] text-muted-foreground hover:bg-accent hover:text-foreground group-hover:inline-block"
                  >
                    {t('settingsPage.addProvider.form.setDefault')}
                  </button>
                )}
                {m.source === 'custom' && (
                  <button
                    type="button"
                    onClick={() => handleDeleteCustomModel(m.model)}
                    className="hidden rounded px-1.5 py-0.5 text-[10px] text-destructive/70 hover:bg-accent hover:text-destructive group-hover:inline-block"
                  >
                    {t('settingsPage.providers.delete')}
                  </button>
                )}
                {defaultModel === m.model && (
                  <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] text-primary">
                    {t('settingsPage.addProvider.form.defaultTag')}
                  </span>
                )}
                {m.source === 'custom' && (
                  <span className="rounded border border-border px-1.5 py-0.5 text-[10px] text-muted-foreground">
                    {t('settingsPage.addProvider.form.customTag')}
                  </span>
                )}
              </span>
            </div>
          ))}
        </div>
      </div>
    </form>
  )
}
