import { AppSettingsSyncProvider } from '@/components/providers/AppSettingsSyncProvider'
import { AlertEventProvider } from '@/components/providers/AlertEventProvider'
import { ThemeProvider } from '@/components/providers/ThemeProvider'
import { FontSizeProvider } from '@/components/providers/FontSizeProvider'
import { MainLayout } from '@/components/layout/MainLayout'
import { SettingsApp } from '@/components/settings/SettingsApp'
import { AddMemoryApp } from '@/components/settings/memory/AddMemoryApp'
import { AddProviderApp } from '@/components/settings/providers/AddProviderApp'
import { AddSkillApp } from '@/components/settings/skills/AddSkillApp'
import { QuickChatApp } from '@/components/quick-chat/QuickChatApp'
import { OnboardingApp } from '@/components/onboarding/OnboardingApp'
import { AlertViewport } from '@/components/alert/AlertViewport'
import { useNotificationsSubscription } from '@/hooks/useNotificationsSubscription'
import { useCliInstallSubscription } from '@/hooks/useCliInstallSubscription'
import { useDisableInputAssistance } from '@/hooks/useDisableInputAssistance'

function App() {
  useNotificationsSubscription()
  useCliInstallSubscription()
  useDisableInputAssistance()
  const params = new URLSearchParams(window.location.search)
  const entry = params.get('entry')

  if (entry === 'onboarding') {
    return <OnboardingApp />
  }

  return (
    <AppSettingsSyncProvider>
      <AlertEventProvider>
        <ThemeProvider>
          <FontSizeProvider>
            {entry === 'settings'
              ? <SettingsApp />
              : entry === 'add_provider'
                ? <AddProviderApp />
                : entry === 'add_skill'
                  ? <AddSkillApp />
                  : entry === 'add_memory'
                    ? <AddMemoryApp />
                    : entry === 'quick_chat'
                      ? <QuickChatApp />
                      : <MainLayout />}
            <AlertViewport />
          </FontSizeProvider>
        </ThemeProvider>
      </AlertEventProvider>
    </AppSettingsSyncProvider>
  )
}

export default App
