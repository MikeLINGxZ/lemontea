import { useEffect } from 'react'
import { useAppStore } from '@/store/appStore'

/** 将主题偏好和系统配色同步到根节点的深色样式，接收要渲染的子内容，返回组件内容 */
export function ThemeProvider({ children }: { /* 要渲染的子内容 */ children: React.ReactNode }) {
  const theme = useAppStore(s => s.theme) // 从状态仓库读取主题

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    const root = document.documentElement // 根目录
    /** 应用处理，接收深色，无返回值 */
    const apply = (dark: boolean) => {
      root.classList.toggle('dark', dark)
    }

    if (theme === 'auto') {
      const mq = window.matchMedia('(prefers-color-scheme: dark)') // 当前处理位置的值
      apply(mq.matches)
      /** 响应当前订阅的事件，接收当前事件或条目，返回表达式结果 */
      const handler = (e: MediaQueryListEvent) => apply(e.matches)
      mq.addEventListener('change', handler)
      return () => mq.removeEventListener('change', handler)
    } else {
      apply(theme === 'dark')
    }
  }, [theme])

  return <>{children}</>
}
