import { useEffect } from 'react'
import i18n from '@/i18n'
import { useAppStore } from '@/store/appStore'
import { isObjectRecord } from '@/lib/guards'

/** 监听跨窗口的存储变化并同步主题、字号、语言和工具设置，接收要渲染的子内容，返回组件内容 */
export function AppSettingsSyncProvider({ children }: { /* 要渲染的子内容 */ children: React.ReactNode }) {
  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 处理存储，接收触发本次回调的事件，无返回值 */
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== 'lemontea-app-settings' || !event.newValue) {
        return
      }

      try {
        const parsed: unknown = JSON.parse(event.newValue) // 保存待校验的 JSON 解析结果
        if (!isObjectRecord(parsed) || !isObjectRecord(parsed.state)) return

        // 校验其他窗口写入的字段，防止旧配置和无效值覆盖当前状态
        const { theme, fontSize, language, enabledToolIds } = parsed.state
        const validLanguage = language === 'zh-CN' || language === 'en' ? language : undefined // 有效语言
        /** 有效工具编号列表 */
        const validToolIds = Array.isArray(enabledToolIds) && enabledToolIds.every((id: unknown): id is string => typeof id === 'string')
          ? enabledToolIds : undefined

        if (validLanguage) void i18n.changeLanguage(validLanguage)

        useAppStore.setState((state) => ({
          theme: theme === 'auto' || theme === 'light' || theme === 'dark' ? theme : state.theme,
          fontSize: fontSize === 'xs' || fontSize === 'sm' || fontSize === 'md' || fontSize === 'lg' || fontSize === 'xl'
            ? fontSize : state.fontSize,
          language: validLanguage ?? state.language,
          enabledToolIds: validToolIds ?? state.enabledToolIds,
        }))
      } catch {
        // 忽略其他窗口写入的无效配置
      }
    }

    window.addEventListener('storage', handleStorage)
    return () => window.removeEventListener('storage', handleStorage)
  }, [])

  return <>{children}</>
}
