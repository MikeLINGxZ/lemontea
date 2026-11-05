import { useState } from 'react'
import { PanelLeftOpen, Pencil, Check, X, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Provider as ProviderBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import type { ProviderWrapper } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import { resolveProviderType } from '@/lib/provider'
import { useChatStore } from '@/store/chatStore'
import { useSettingsStore } from '@/store/settingsStore'
import { cn, isMac } from '@/lib/utils'
import { useShallow } from 'zustand/react/shallow'
import type { Message } from '@/types'
import type { ProviderItem } from '@/types/settings'
import { NotificationBell } from '@/components/notifications/NotificationBell'

/** 空消息列表 */
const EMPTY_MESSAGES: Message[] = []

/** 聊天页头组件接收的属性 */
interface ChatHeaderProps {
  /** 侧栏折叠 */
  sidebarCollapsed: boolean
  /** 处理展开侧栏事件，无返回值 */
  onExpandSidebar: () => void
}

/** 选择模型服务商模型，接收模型服务商、首选模型名称，返回对应的处理结果 */
function pickProviderModel(provider: ProviderItem, preferredModelName?: string) {
  if (preferredModelName) {
    /** 已匹配模型 */
    const matchedModel = provider.models.find(
      (model) => model.enable && model.model === preferredModelName
    )
    if (matchedModel) return matchedModel
  }

  return (
    provider.models.find((model) => model.enable && model.is_default) ??
    provider.models.find((model) => model.enable) ??
    null
  )
}

/** 转换包装终点模型服务商条目列表，接收包装对象列表，返回对应的处理结果 */
function mapWrapperToProviderItems(wrappers: ProviderWrapper[]): ProviderItem[] {
  return wrappers.map((wrapper) => ({
    id: wrapper.providers.id,
    provider_name: wrapper.providers.provider_name,
    provider_type: wrapper.providers.provider_type,
    base_url: wrapper.providers.base_url,
    api_key: wrapper.providers.api_key,
    enabled: wrapper.providers.enabled,
    is_default: wrapper.providers.is_default,
    model_count: wrapper.models.length,
    icon: wrapper.providers.icon,
    models: wrapper.models.map((model) => ({
      id: model.id,
      provider_id: model.provider_id,
      model: model.model,
      owned_by: model.owned_by,
      object: model.object,
      enable: model.enable,
      alias: model.alias,
      is_custom: model.is_custom,
      is_default: model.is_default,
    })),
  }))
}

/** 显示会话标题和当前模型，并处理重命名和标题生成，接收侧栏折叠状态和展开回调，返回组件内容 */
export function ChatHeader({
  sidebarCollapsed,
  onExpandSidebar,
}: ChatHeaderProps) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  /** 从状态仓库读取当前会话编号、重命名会话、生成标题草稿、标题、是否有消息列表、会话运行配置、消息列表 */
  const {
    currentConversationId,
    renameConversation,
    generateTitleDraft,
    title,
    hasMessages,
    sessionRunConfig,
    messages,
  } = useChatStore(useShallow((state) => {
    /** 当前会话 */
    const currentConversation = state.conversations.find(
      conversation => conversation.id === state.currentConversationId
    )
    /** 会话消息列表 */
    const conversationMessages = state.currentConversationId
      ? (state.messages[state.currentConversationId] ?? EMPTY_MESSAGES)
      : EMPTY_MESSAGES

    return {
      currentConversationId: state.currentConversationId,
      renameConversation: state.renameConversation,
      generateTitleDraft: state.generateTitleDraft,
      title: currentConversation?.title,
      hasMessages: conversationMessages.length > 0,
      sessionRunConfig: state.currentConversationId
        ? state.sessionRunConfigs[state.currentConversationId]
        : undefined,
      messages: conversationMessages,
    }
  }))
  const providers = useSettingsStore((state) => state.providers) // 从状态仓库读取服务商列表
  const [editing, setEditing] = useState(false) // 正在编辑及其更新方法
  const [editValue, setEditValue] = useState('') // 编辑值及其更新方法
  const [isGeneratingTitle, setIsGeneratingTitle] = useState(false) // 是否正在生成标题及其更新方法

  const displayTitle = title ?? t('chat.newChat') // 显示标题

  /** 启动编辑，无返回值 */
  const startEdit = () => {
    setEditValue(displayTitle)
    setEditing(true)
  }

  /** 保存编辑，无返回值 */
  const saveEdit = () => {
    if (editValue.trim() && currentConversationId) {
      void renameConversation(currentConversationId, editValue.trim())
    }
    setEditing(false)
  }

  /** 解析标题配置起点服务商列表，接收模型服务商列表，返回对应的处理结果 */
  const resolveTitleConfigFromProviders = (providerList: ProviderItem[]) => {
    /** 最新模型名称 */
    const latestModelName = [...messages]
      .reverse()
      .find((message) => message.modelName)?.modelName

    const enabledProviders = providerList.filter((provider) => provider.enabled) // 启用状态服务商列表
    if (enabledProviders.length === 0) return null

    /** 匹配模型服务商 */
    const matchingProvider = latestModelName
      ? enabledProviders.find((provider) =>
          provider.models.some((model) => model.enable && model.model === latestModelName)
        )
      : null

    /** 选中项模型服务商 */
    const selectedProvider =
      matchingProvider ??
      enabledProviders.find((provider) => provider.is_default) ??
      enabledProviders[0]

    if (!selectedProvider) return null

    const selectedModel = pickProviderModel(selectedProvider, latestModelName) // 选中项模型
    if (!selectedModel) return null

    return {
      baseUrl: selectedProvider.base_url,
      apiKey: selectedProvider.api_key,
      modelName: selectedModel.model,
      providerType: resolveProviderType(selectedProvider.provider_type),
    }
  }

  const fallbackTitleConfig = sessionRunConfig ?? resolveTitleConfigFromProviders(providers) // 回退标题配置

  /** 重新生成标题，完成后结束异步任务 */
  const regenerateTitle = async () => {
    if (!currentConversationId || isGeneratingTitle) return

    setIsGeneratingTitle(true)
    try {
      let titleConfig = fallbackTitleConfig // 标题配置
      if (!titleConfig) {
        const providersResult = await ProviderBinding.ListProviders({}) // 保存服务商列表结果
        titleConfig = providersResult?.providers
          ? resolveTitleConfigFromProviders(mapWrapperToProviderItems(providersResult.providers))
          : null
      }

      if (!titleConfig) return

      const nextTitle = await generateTitleDraft(currentConversationId, titleConfig) // 保存下一项标题
      if (nextTitle) {
        setEditValue(nextTitle)
      }
    } finally {
      setIsGeneratingTitle(false)
    }
  }

  return (
    <div className={cn(
      'flex items-center gap-2 px-4 py-3 border-b border-border h-14 shrink-0',
      isMac && 'window-titlebar',
      // 窗口失焦时仍需为灰色原生按钮预留位置
      sidebarCollapsed && isMac && 'pl-20',
    )}>
      {sidebarCollapsed && (
        <button
          type="button"
          aria-label="展开侧边栏"
          onClick={onExpandSidebar}
          className="shrink-0 rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <PanelLeftOpen size={16} />
        </button>
      )}
      {editing ? (
        <>
          <input
            className="flex-1 bg-transparent border-b border-primary outline-hidden text-lg font-medium text-foreground"
            value={editValue}
            onChange={e => setEditValue(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter') saveEdit()
              if (e.key === 'Escape' && !isGeneratingTitle) setEditing(false)
            }}
            autoFocus
          />
          <button onClick={saveEdit} className="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground">
            <Check size={14} />
          </button>
          <button
            onClick={() => void regenerateTitle()}
            disabled={isGeneratingTitle}
            aria-label={t('chat.regenerateTitle')}
            title={t('chat.regenerateTitle')}
            className="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
          >
            <RefreshCw size={14} className={cn(isGeneratingTitle && 'animate-spin')} />
          </button>
          <button
            onClick={() => setEditing(false)}
            disabled={isGeneratingTitle}
            className="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
          >
            <X size={14} />
          </button>
        </>
      ) : (
        <div className="flex max-w-full min-w-1 items-center gap-3">
          <h1 className="text-lg font-medium text-foreground truncate">{displayTitle}</h1>
          {hasMessages && (
            <button
              onClick={startEdit}
              className="shrink-0 p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground"
            >
              <Pencil size={14} />
            </button>
          )}
        </div>
      )}
      <div className="ml-auto flex shrink-0 items-center [--wails-draggable:no-drag]">
        <NotificationBell />
      </div>
    </div>
  )
}
