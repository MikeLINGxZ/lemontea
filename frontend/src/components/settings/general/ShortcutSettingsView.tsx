import { useEffect, useState } from 'react'
import type { KeyboardEvent as ReactKeyboardEvent } from 'react'
import { Keyboard } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsActionBar } from '@/components/settings/common/SettingsActionBar'
import { SettingsContentLayout } from '@/components/settings/common/SettingsContentLayout'
import { SettingsDirtyGuard } from '@/components/settings/common/SettingsDirtyGuard'
import { SettingsFieldRow } from '@/components/settings/common/SettingsFieldRow'
import { SettingsPanelHeader } from '@/components/settings/common/SettingsPanelHeader'
import { cn } from '@/lib/utils'
import type { ShortcutValidationStatus } from '@/types/settings'

/** 定义快捷键校验的数据结构 */
type ShortcutValidation = {
  /** 状态 */
  status: ShortcutValidationStatus
  /** 消息 */
  message: string
}

/** 渲染快捷键设置视图界面，接收组件输入属性，返回组件内容 */
export function ShortcutSettingsView(props: {
  /** 快捷聊天快捷键 */
  quickChatShortcut: string
  /** 有未保存修改 */
  dirty: boolean
  /** 校验 */
  validation: ShortcutValidation
  /** 处理快捷键变更事件，接收快捷键，无返回值 */
  onShortcutChange: (shortcut: string) => void
  /** 处理应用事件，无返回值 */
  onApply: () => void
  /** 处理重置事件，无返回值 */
  onReset: () => void
}) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [recording, setRecording] = useState(false) // 正在录制及其更新方法
  const canApply = props.dirty && props.validation.status === 'available' // 能否应用
  const statusText = validationStatusText(props.validation.status, t) // 状态文本
  /** 记录提示 */
  const recordHint = recording
    ? t('settingsPage.general.shortcuts.recordingHint')
    : t('settingsPage.general.shortcuts.recordHint')

  /** 处理快捷键事件，接收触发本次回调的事件，无返回值 */
  const handleShortcutEvent = (event: ShortcutKeyboardEvent) => {
    if (event.key === 'Escape') {
      event.preventDefault()
      setRecording(false)
      return
    }

    const shortcut = shortcutFromKeyboardEvent(event) // 快捷键
    if (!shortcut) {
      event.preventDefault()
      return
    }

    event.preventDefault()
    setRecording(false)
    props.onShortcutChange(shortcut)
  }

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!recording) {
      return
    }

    /** 响应当前订阅的事件，接收触发本次回调的事件，无返回值 */
    const listener = (event: globalThis.KeyboardEvent) => {
      handleShortcutEvent(event)
      event.stopPropagation()
    }

    window.addEventListener('keydown', listener, true)
    return () => {
      window.removeEventListener('keydown', listener, true)
    }
  }, [recording])

  return (
    <SettingsDirtyGuard dirty={props.dirty}>
      <SettingsContentLayout
        header={
          <SettingsPanelHeader
            title={t('settingsPage.general.shortcuts.title')}
            description={t('settingsPage.general.shortcuts.description')}
          />
        }
        footprint={
          <SettingsActionBar
            primaryLabel={t('settingsPage.actions.apply')}
            secondaryLabel={t('settingsPage.actions.reset')}
            primaryDisabled={!canApply}
            onPrimaryClick={props.onApply}
            onSecondaryClick={props.onReset}
          />
        }
      >
        <div className="space-y-4 pt-6">
          <SettingsFieldRow
            label={t('settingsPage.general.shortcuts.quickChatLabel')}
            description={t('settingsPage.general.shortcuts.quickChatDescription')}
          >
            <div className="space-y-3">
              <button
                type="button"
                aria-label={t('settingsPage.general.shortcuts.recordAria')}
                onClick={() => setRecording(true)}
                onKeyDown={(event) => {
                  if (!recording) {
                    handleShortcutEvent(event)
                  }
                }}
                className={cn(
                  'flex w-full items-center justify-between gap-3 rounded-xl border bg-background px-4 py-3 text-left transition-colors hover:bg-accent focus:outline-hidden focus:ring-2 focus:ring-ring',
                  recording ? 'border-primary ring-2 ring-primary/20' : 'border-border',
                )}
              >
                <span className="flex min-w-0 items-center gap-3">
                  <Keyboard size={18} className="shrink-0 text-muted-foreground" />
                  <span className="min-w-0 truncate text-sm font-medium text-foreground">
                    {props.quickChatShortcut}
                  </span>
                </span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {recordHint}
                </span>
              </button>
              <div className="min-h-5">
                {statusText && (
                  <p className={cn(
                    'text-sm',
                    props.validation.status === 'available' && 'text-emerald-600 dark:text-emerald-400',
                    props.validation.status === 'conflict' && 'text-red-600 dark:text-red-400',
                    props.validation.status === 'invalid' && 'text-amber-600 dark:text-amber-400',
                  )}>
                    {statusText}
                    {props.validation.message ? `：${props.validation.message}` : ''}
                  </p>
                )}
              </div>
            </div>
          </SettingsFieldRow>
        </div>
      </SettingsContentLayout>
    </SettingsDirtyGuard>
  )
}

/** 快捷键键盘事件的数据 */
type ShortcutKeyboardEvent = Pick<
  ReactKeyboardEvent | globalThis.KeyboardEvent,
  'key' | 'code' | 'metaKey' | 'ctrlKey' | 'altKey' | 'shiftKey' | 'preventDefault'
>

/** 校验状态文本处理，接收状态、当前语言的翻译函数，返回处理后的文本 */
function validationStatusText(status: ShortcutValidationStatus, t: (key: string) => string): string {
  if (status === 'available') {
    return t('settingsPage.general.shortcuts.statusAvailable')
  }
  if (status === 'conflict') {
    return t('settingsPage.general.shortcuts.statusConflict')
  }
  if (status === 'invalid') {
    return t('settingsPage.general.shortcuts.statusInvalid')
  }
  return ''
}

/** 快捷键起点键盘事件处理，接收触发本次回调的事件，返回处理结果，无法取得时为空 */
function shortcutFromKeyboardEvent(event: ShortcutKeyboardEvent): string | null {
  const key = normalizeEventKey(event) // 键
  if (!key) {
    return null
  }

  const modifiers: string[] = [] // 修饰键
  if (event.metaKey) {
    modifiers.push('Cmd')
  }
  if (event.ctrlKey) {
    modifiers.push('Ctrl')
  }
  if (event.altKey) {
    modifiers.push('Alt')
  }
  if (event.shiftKey) {
    modifiers.push('Shift')
  }
  if (modifiers.length === 0) {
    return null
  }
  return [...modifiers, key].join('+')
}

/** 规范化事件键，接收触发本次回调的事件，返回处理结果，无法取得时为空 */
function normalizeEventKey(event: ShortcutKeyboardEvent): string | null {
  const physicalKey = normalizeEventCode(event.code) // 物理按键键
  if (physicalKey) {
    return physicalKey
  }

  const key = event.key // 键
  if (!key || key === 'Control' || key === 'Shift' || key === 'Alt' || key === 'Meta') {
    return null
  }
  if (key.length === 1) {
    return key.toUpperCase()
  }
  /** 别名列表 */
  const aliases: Record<string, string> = {
    ' ': 'Space',
    Spacebar: 'Space',
    Enter: 'Return',
    Escape: 'Escape',
    Backspace: 'Delete',
    Delete: 'Delete',
    Tab: 'Tab',
    ArrowLeft: 'Left',
    ArrowRight: 'Right',
    ArrowUp: 'Up',
    ArrowDown: 'Down',
  }
  if (aliases[key]) {
    return aliases[key]
  }
  if (/^F([1-9]|1[0-9]|20)$/.test(key)) {
    return key
  }
  return null
}

/** 规范化事件代码，接收代码，返回处理结果，无法取得时为空 */
function normalizeEventCode(code: string): string | null {
  if (/^Key[A-Z]$/.test(code)) {
    return code.slice(3)
  }
  if (/^Digit[0-9]$/.test(code)) {
    return code.slice(5)
  }
  if (/^F([1-9]|1[0-9]|20)$/.test(code)) {
    return code
  }
  /** 别名列表 */
  const aliases: Record<string, string> = {
    Space: 'Space',
    Enter: 'Return',
    Escape: 'Escape',
    Backspace: 'Delete',
    Delete: 'Delete',
    Tab: 'Tab',
    ArrowLeft: 'Left',
    ArrowRight: 'Right',
    ArrowUp: 'Up',
    ArrowDown: 'Down',
    Home: 'Home',
    End: 'End',
    PageUp: 'PageUp',
    PageDown: 'PageDown',
    Insert: 'Insert',
    Backquote: 'Backquote',
    Minus: 'Minus',
    Equal: 'Equal',
    BracketLeft: 'BracketLeft',
    BracketRight: 'BracketRight',
    Backslash: 'Backslash',
    Semicolon: 'Semicolon',
    Quote: 'Quote',
    Comma: 'Comma',
    Period: 'Period',
    Slash: 'Slash',
  }
  return aliases[code] ?? null
}
