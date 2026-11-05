import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { ConfirmDialog } from '@/components/settings/common/ConfirmDialog'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { CliToolsSection } from '@/components/settings/plugins/CliToolsSection'
import { ExtensionDetailInfo } from '@/components/settings/plugins/ExtensionDetailInfo'
import { getJsonError } from '@/components/settings/plugins/jsonError'
import { isObjectRecord } from '@/lib/guards'
import type { ExtensionToolItem } from '@/types/settings'
import type { ExtensionItem } from '@/types/settings'

/** 展示扩展详情，编辑 JSON 清单并处理 CLI 登录、配置和重置操作，接收组件输入属性，返回组件内容 */
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
  /** 处理登录CLI事件，无返回值 */
  onLoginCli?: () => void
  /** 处理重置数据事件，无返回值 */
  onResetData?: () => void
  /** 处理重新生成CLI清单事件，无返回值 */
  onRegenerateCliManifest?: () => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [draft, setDraft] = useState(props.configText) // 草稿及其更新方法
  const [confirmResetOpen, setConfirmResetOpen] = useState(false) // 确认重置展开及其更新方法
  const [confirmRegenerateOpen, setConfirmRegenerateOpen] = useState(false) // 确认重新生成展开及其更新方法

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
  /** CLI清单 */
  const cliManifest = useMemo(() => deriveCliManifest(props.extension, draft, props.configText), [draft, props.configText, props.extension])

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

  return (
    <>
      <SettingsContentLayout
        header={(
          <SettingsPanelHeader
            title={props.extension.name}
            description={props.extension.description || props.extension.author || props.extension.version}
            badge={kindBadge}
          />
        )}
        footprint={(props.extension.kind === 'mcp' || props.extension.kind === 'cli') && props.extension.config_file_path ? (
          <SettingsActionBar
            primaryLabel={t('settingsPage.actions.apply')}
            secondaryLabel={t('settingsPage.actions.reset')}
            primaryDisabled={!dirty || props.loading || !jsonValid}
            secondaryDisabled={!dirty || props.loading}
            onPrimaryClick={() => props.onSave(draft)}
            onSecondaryClick={() => setDraft(props.configText)}
            reloadLabel={t('settingsPage.plugins.reload')}
            onReloadClick={props.onReload}
            regenerateLabel={props.extension.kind === 'cli' ? (props.loading ? t('settingsPage.plugins.cliRegenerating') : t('settingsPage.plugins.cliRegenerate')) : undefined}
            regenerateLoading={props.loading}
            regenerateDisabled={props.loading || !props.onRegenerateCliManifest}
            onRegenerateClick={props.extension.kind === 'cli' ? () => setConfirmRegenerateOpen(true) : undefined}
          />
        ) : undefined}
      >
        <ExtensionDetailInfo
          extension={props.extension}
          draft={draft}
          jsonValid={jsonValid}
          errorLine={errorLine}
          onDraftChange={setDraft}
        />
        {props.extension.kind === 'cli' && cliManifest ? (
          <section className="mt-6 space-y-3 border-t border-border pt-4">
            <div className="flex items-center justify-between gap-4">
              <div className="space-y-1">
                <h3 className="text-sm font-semibold text-foreground">
                  {t('settingsPage.plugins.cliLoginTitle')}
                </h3>
                <p className="text-xs text-muted-foreground">
                  {t('settingsPage.plugins.cliLoginHint')}
                </p>
              </div>
              <button
                type="button"
                disabled={
                  props.loading
                  || !props.onLoginCli
                  || cliManifest.login_command.length === 0
                  || dirty
                  || !jsonValid
                }
                onClick={() => props.onLoginCli?.()}
                className="rounded-xl bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {t('settingsPage.plugins.cliLoginAction')}
              </button>
            </div>
            <label className="flex flex-col gap-1 text-sm text-foreground">
              <span className="text-xs text-muted-foreground">
                {t('settingsPage.plugins.cliLoginCommandLabel')}
              </span>
              <input
                type="text"
                placeholder={t('settingsPage.plugins.cliLoginCommandPlaceholder')}
                value={cliManifest.login_command.join(' ')}
                onChange={(event) => {
                  setDraft(updateCliManifestDraftLoginCommand(
                    draft,
                    parseLoginCommandInput(event.target.value),
                  ))
                }}
                className="rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm focus:border-primary focus:outline-hidden disabled:opacity-60"
                disabled={props.loading || !jsonValid}
              />
              {cliManifest.login_command.length === 0 ? (
                <span className="text-xs text-muted-foreground">
                  {t('settingsPage.plugins.cliLoginCommandEmptyHint')}
                </span>
              ) : dirty ? (
                <span className="text-xs text-amber-600 dark:text-amber-400">
                  {t('settingsPage.plugins.cliLoginCommandDirtyHint')}
                </span>
              ) : null}
            </label>
            <label className="flex items-center gap-2 text-sm text-foreground">
              <input
                type="checkbox"
                checked={cliManifest.isolation === 'shared'}
                onChange={(event) => {
                  setDraft(updateCliManifestDraftIsolation(
                    draft,
                    event.target.checked ? 'shared' : 'isolated',
                  ))
                }}
                disabled={props.loading || !jsonValid}
              />
              <span>{t('settingsPage.plugins.cliUseSharedIsolation')}</span>
            </label>
          </section>
        ) : null}
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
        {props.extension.kind === 'cli' && props.onResetData ? (
          <div className="mt-6 flex gap-3 border-t border-border pt-4">
            <button
              type="button"
              onClick={() => setConfirmResetOpen(true)}
              className="rounded-lg border border-red-300 bg-red-50 px-4 py-2 text-sm text-red-700 hover:bg-red-100 dark:border-red-700 dark:bg-red-500/10 dark:text-red-400"
            >
              {t('settingsPage.plugins.cliResetData')}
            </button>
          </div>
        ) : null}
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
      <ConfirmDialog
        open={confirmRegenerateOpen}
        title={t('settingsPage.plugins.cliRegenerateConfirmTitle')}
        description={t('settingsPage.plugins.cliRegenerateConfirmDescription', {
          name: props.extension.name,
        })}
        confirmLabel={t('settingsPage.plugins.cliRegenerate')}
        cancelLabel={t('settingsPage.providers.cancel')}
        busy={props.loading}
        onConfirm={() => {
          if (props.onRegenerateCliManifest) {
            props.onRegenerateCliManifest()
          }
          setConfirmRegenerateOpen(false)
        }}
        onCancel={() => setConfirmRegenerateOpen(false)}
      />
    </>
  )
}

/** CLI隔离模式的可选值 */
type CliIsolationMode = 'isolated' | 'shared'

/** 定义草稿CLI清单的数据结构 */
type DraftCliManifest = {
  /** 登录命令 */
  login_command: string[]
  /** 隔离 */
  isolation: CliIsolationMode
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

/** 从现有数据提取CLI清单，接收扩展、草稿、回退配置文本，返回处理结果，无法取得时为空 */
function deriveCliManifest(extension: ExtensionItem | null, draft: string, fallbackConfigText: string): DraftCliManifest | null {
  if (!extension || extension.kind !== 'cli') {
    return null
  }

  const parsedDraft = parseCliManifestDraft(draft) // 解析结果草稿
  if (parsedDraft) {
    return parsedDraft
  }

  return parseCliManifestDraft(fallbackConfigText)
}

/** 解析CLI清单草稿，接收配置文本，返回处理结果，无法取得时为空 */
function parseCliManifestDraft(configText: string): DraftCliManifest | null {
  try {
    const parsed: unknown = JSON.parse(configText) // 保存待校验的 JSON 解析结果
    if (!isObjectRecord(parsed)) return null

    /** 登录命令 */
    const loginCommand: string[] = Array.isArray(parsed.login_command)
      ? parsed.login_command.filter((part: unknown): part is string => typeof part === 'string')
      : []
    const isolation: CliIsolationMode = parsed.isolation === 'shared' ? 'shared' : 'isolated' // 隔离

    return {
      login_command: loginCommand,
      isolation,
    }
  } catch {
    return null
  }
}

/** 更新CLI清单草稿隔离，接收草稿、隔离，返回处理后的文本 */
function updateCliManifestDraftIsolation(draft: string, isolation: CliIsolationMode): string {
  try {
    const parsed: unknown = JSON.parse(draft) // 保存待校验的 JSON 解析结果
    if (!isObjectRecord(parsed)) return draft

    parsed.isolation = isolation
    return `${JSON.stringify(parsed, null, 2)}\n`
  } catch {
    return draft
  }
}

/** 更新CLI清单草稿登录命令，接收草稿、登录命令，返回处理后的文本 */
function updateCliManifestDraftLoginCommand(draft: string, loginCommand: string[]): string {
  try {
    const parsed: unknown = JSON.parse(draft) // 保存待校验的 JSON 解析结果
    if (!isObjectRecord(parsed)) return draft

    parsed.login_command = loginCommand
    return `${JSON.stringify(parsed, null, 2)}\n`
  } catch {
    return draft
  }
}

/** 解析登录命令输入，接收值，返回对应的处理结果 */
function parseLoginCommandInput(value: string): string[] {
  return value.trim().split(/\s+/).filter((token) => token.length > 0)
}
