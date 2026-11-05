import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Theme, FontSize, Language } from '../types'
import i18n from '../i18n'

/** 应用状态仓库保存的数据和操作 */
interface AppStore {
  /** 主题 */
  theme: Theme
  /** 字体尺寸 */
  fontSize: FontSize
  /** 语言 */
  language: Language
  /** 启用状态工具编号列表 */
  enabledToolIds: string[]
  /** 设置主题，接收主题，无返回值 */
  setTheme: (theme: Theme) => void
  /** 设置字体尺寸，接收尺寸，无返回值 */
  setFontSize: (size: FontSize) => void
  /** 设置语言，接收语言，无返回值 */
  setLanguage: (lang: Language) => void
  /** 设置启用状态工具编号列表，接收编号列表，无返回值 */
  setEnabledToolIds: (ids: string[]) => void
}

/** 持久化应用主题、字号、语言和工具偏好 */
export const useAppStore = create<AppStore>()(
  persist(
    (set) => ({
      theme: 'auto',
      fontSize: 'md',
      language: 'zh-CN',
      enabledToolIds: [] as string[],
      /** 设置主题，接收主题，返回表达式结果 */
      setTheme: (theme) => set({ theme }),
      /** 设置字体尺寸，接收字体尺寸，返回表达式结果 */
      setFontSize: (fontSize) => set({ fontSize }),
      /** 设置语言，接收语言，无返回值 */
      setLanguage: (language) => {
        i18n.changeLanguage(language)
        set({ language })
      },
      /** 设置启用状态工具编号列表，接收启用状态工具编号列表，返回表达式结果 */
      setEnabledToolIds: (enabledToolIds) => set({ enabledToolIds }),
    }),
    {
      name: 'lemontea-app-settings',
      /** 处理恢复持久状态存储事件，返回表达式结果 */
      onRehydrateStorage: () => (state) => {
        if (state?.language) {
          i18n.changeLanguage(state.language)
        }
      },
    }
  )
)
