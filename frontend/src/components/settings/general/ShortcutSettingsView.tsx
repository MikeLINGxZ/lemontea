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

type ShortcutValidation = {
  status: ShortcutValidationStatus
  message: string
}

export function ShortcutSettingsView(props: {
  quickChatShortcut: string
  dirty: boolean
  validation: ShortcutValidation
  onShortcutChange: (shortcut: string) => void
  onApply: () => void
  onReset: () => void
}) {
  const { t } = useTranslation()
  const [recording, setRecording] = useState(false)
  const canApply = props.dirty && props.validation.status === 'available'
  const statusText = validationStatusText(props.validation.status, t)
  const recordHint = recording
    ? t('settingsPage.general.shortcuts.recordingHint')
    : t('settingsPage.general.shortcuts.recordHint')

  const handleShortcutEvent = (event: ShortcutKeyboardEvent) => {
    if (event.key === 'Escape') {
      event.preventDefault()
      setRecording(false)
      return
    }

    const shortcut = shortcutFromKeyboardEvent(event)
    if (!shortcut) {
      event.preventDefault()
      return
    }

    event.preventDefault()
    setRecording(false)
    props.onShortcutChange(shortcut)
  }

  useEffect(() => {
    if (!recording) {
      return
    }

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
                  'flex w-full items-center justify-between gap-3 rounded-xl border bg-background px-4 py-3 text-left transition-colors hover:bg-accent focus:outline-none focus:ring-2 focus:ring-ring',
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

type ShortcutKeyboardEvent = Pick<
  ReactKeyboardEvent | globalThis.KeyboardEvent,
  'key' | 'metaKey' | 'ctrlKey' | 'altKey' | 'shiftKey' | 'preventDefault'
>

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

function shortcutFromKeyboardEvent(event: ShortcutKeyboardEvent): string | null {
  const key = normalizeEventKey(event)
  if (!key) {
    return null
  }

  const modifiers: string[] = []
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

function normalizeEventKey(event: ShortcutKeyboardEvent): string | null {
  const key = event.key
  if (!key || key === 'Control' || key === 'Shift' || key === 'Alt' || key === 'Meta') {
    return null
  }
  if (key.length === 1) {
    return key.toUpperCase()
  }
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
