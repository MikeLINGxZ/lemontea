import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { ConfirmDialog } from '@/components/settings/common/ConfirmDialog'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { CliToolsSection } from '@/components/settings/plugins/CliToolsSection'
import { ExtensionDetailInfo, RuntimeStatusBadge } from '@/components/settings/plugins/ExtensionDetailInfo'
import { getJsonError } from '@/components/settings/plugins/jsonError'
import { isObjectRecord } from '@/lib/guards'
import type { ExtensionToolItem } from '@/types/settings'
import type { ExtensionItem } from '@/types/settings'

/** 展示扩展详情，编辑 JSON 清单并处理 CLI 配置和重置操作，接收组件输入属性，返回组件内容 */
export function PluginToolDetailView(props: {
  /** 扩展 */
  extension: ExtensionItem | null
  /** 配置文本 */
  configText: string
  /** 加载中 */
  loading: boolean
  /** 处理重新加载事件，无返回值 */
  onReload: () => void
  /** 处理保存事件，接收配置文本，无返回值 */
  onSave: (configText: string) => void
  /** 处理重置数据事件，无返回值 */
  onResetData?: () => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [draft, setDraft] = useState(props.configText) // 草稿及其更新方法
  const [confirmResetOpen, setConfirmResetOpen] = useState(false) // 确认重置展开及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    setDraft(props.configText)
  }, [props.configText, props.extension?.id])

  const dirty = useMemo(() => draft !== props.configText, [draft, props.configText]) // 有未保存修改
  const jsonError = useMemo(() => getJsonError(draft), [draft]) // JSON错误
  const jsonValid = jsonError.valid // JSON有效
  const errorLine = jsonError.valid ? null : jsonError.errorLine // 错误行
  /** CLI工具列表 */
  const cliTools = useMemo(() => deriveCliTools(props.extension, draft, props.configText), [draft, props.configText, props.extension])
  /** 类别徽标 */
  const kindBadge = props.extension ? (
    <span className={cn(
      'rounded-full px-2.5 py-0.5 text-xs font-medium',
      props.extension.kind === 'mcp'
        ? 'bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400'
        : props.extension.kind === 'cli'
          ? 'bg-purple-100 text-purple-700 dark:bg-purple-500/20 dark:text-purple-400'
          : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400'
    )}>
      {props.extension.kind === 'mcp'
        ? t('settingsPage.plugins.kindMcp')
        : props.extension.kind === 'cli'
          ? t('settingsPage.plugins.kindCli')
          : t('settingsPage.plugins.kindPlugin')}
    </span>
  ) : undefined

  if (!props.extension) {
    return (
      <SettingsContentLayout
        header={(
          <SettingsPanelHeader
            title={t('settingsPage.plugins.title')}
            description={t('settingsPage.plugins.emptyDescription')}
          />
        )}
      />
    )
  }

  // 仅在 CLI 或 MCP 有配置路径时显示配置操作
  const hasEditableConfig: boolean = (props.extension.kind === 'mcp' || props.extension.kind === 'cli') &&
    Boolean(props.extension.config_file_path)
  const canResetData: boolean = props.extension.kind === 'cli' && Boolean(props.onResetData) // 当前 CLI 是否支持重置数据

  return (
    <>
      <SettingsContentLayout
        header={(
          <SettingsPanelHeader
            title={props.extension.name}
            description={props.extension.description || props.extension.author || props.extension.version}
            badge={(
              <>
                {kindBadge}
                {props.extension.kind === 'cli' && <RuntimeStatusBadge status={props.extension.runtime_status} />}
              </>
            )}
          />
        )}
        footprint={hasEditableConfig || canResetData ? (
          <SettingsActionBar
            primaryLabel={hasEditableConfig ? t('settingsPage.actions.apply') : undefined}
            secondaryLabel={hasEditableConfig ? t('settingsPage.actions.reset') : undefined}
            primaryDisabled={!dirty || props.loading || !jsonValid}
            secondaryDisabled={!dirty || props.loading}
            onPrimaryClick={() => props.onSave(draft)}
            onSecondaryClick={() => setDraft(props.configText)}
            reloadLabel={hasEditableConfig ? t('settingsPage.plugins.reload') : undefined}
            onReloadClick={props.onReload}
            dangerLabel={canResetData ? t('settingsPage.plugins.cliResetData') : undefined}
            onDangerClick={() => setConfirmResetOpen(true)}
          />
        ) : undefined}
      >
        {props.extension.kind === 'cli' && props.extension.runtime_message && (
          <div className="mt-4 rounded-xl border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
            <span className="font-medium">{t('settingsPage.plugins.runtimeMessageLabel')}:</span>{' '}
            {props.extension.runtime_message}
          </div>
        )}
        {props.extension.kind === 'cli' ? (
          <CliToolsSection
            tools={cliTools}
            busy={props.loading}
            jsonValid={jsonValid}
            onToggleEnabled={(toolName, enabled) => {
              setDraft(updateCliToolDraft(draft, toolName, { enabled }))
            }}
            onToggleRequiresConfirm={(toolName, requiresConfirm) => {
              setDraft(updateCliToolDraft(draft, toolName, { requires_confirm: requiresConfirm }))
            }}
          />
        ) : null}
        <ExtensionDetailInfo
          extension={props.extension}
          draft={draft}
          jsonValid={jsonValid}
          errorLine={errorLine}
          onDraftChange={setDraft}
        />
      </SettingsContentLayout>
      <ConfirmDialog
        open={confirmResetOpen}
        title={t('settingsPage.plugins.cliResetDataConfirmTitle')}
        description={props.extension.name}
        confirmLabel={t('settingsPage.plugins.cliResetData')}
        cancelLabel={t('settingsPage.providers.cancel')}
        confirmTone="danger"
        onConfirm={() => {
          if (props.onResetData) {
            props.onResetData()
          }
          setConfirmResetOpen(false)
        }}
        onCancel={() => setConfirmResetOpen(false)}
      />
    </>
  )
}

/** 从现有数据提取CLI工具列表，接收扩展、草稿、回退配置文本，返回对应的处理结果 */
function deriveCliTools(extension: ExtensionItem | null, draft: string, fallbackConfigText: string): ExtensionToolItem[] {
  if (!extension || extension.kind !== 'cli') {
    return []
  }
  const parsedDraft = parseCliToolItems(draft) // 解析结果草稿
  if (parsedDraft) {
    return parsedDraft
  }
  const parsedFallback = parseCliToolItems(fallbackConfigText) // 解析结果回退
  if (parsedFallback) {
    return parsedFallback
  }
  return extension.tools
}

/** 解析CLI工具条目列表，接收配置文本，返回处理结果，无法取得时为空 */
function parseCliToolItems(configText: string): ExtensionToolItem[] | null {
  try {
    const parsed: unknown = JSON.parse(configText) // 保存待校验的 JSON 解析结果
    if (!isObjectRecord(parsed) || !Array.isArray(parsed.tools)) return []

    return parsed.tools.filter(isObjectRecord).map((tool) => ({
      tool_id: typeof tool.name === 'string' ? tool.name : '',
      server_id: '',
      name: typeof tool.name === 'string' ? tool.name : '',
      description: typeof tool.description === 'string' ? tool.description : '',
      enabled: Boolean(tool.enabled),
      requires_confirm: Boolean(tool.requires_confirm),
    }))
  } catch {
    return null
  }
}

/** 更新CLI工具草稿，接收草稿、工具名称、要写入的字段和值，返回处理后的文本 */
function updateCliToolDraft(
  draft: string,
  toolName: string,
  changes: { /* 是否启用 */ enabled?: boolean; /* 需要确认 */ requires_confirm?: boolean },
): string {
  try {
    const parsed: unknown = JSON.parse(draft) // 保存待校验的 JSON 解析结果
    if (!isObjectRecord(parsed) || !Array.isArray(parsed.tools)) return draft

    // 仅更新匹配工具的指定字段，保留清单中的扩展属性
    parsed.tools = parsed.tools.map((tool: unknown) =>
      isObjectRecord(tool) && tool.name === toolName ? { ...tool, ...changes } : tool
    )
    return `${JSON.stringify(parsed, null, 2)}\n`
  } catch {
    return draft
  }
}
