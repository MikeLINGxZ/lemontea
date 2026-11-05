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

/** 根据窗口入口组装主界面、设置页、快速聊天和首次引导，并挂载全局同步组件，返回组件内容 */
function App() {
  useNotificationsSubscription()
  useCliInstallSubscription()
  useDisableInputAssistance()
  const params = new URLSearchParams(window.location.search) // 参数
  const entry = params.get('entry') // 当前窗口入口

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
