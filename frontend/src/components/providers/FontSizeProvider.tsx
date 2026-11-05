import { useEffect } from 'react'
import { useAppStore } from '@/store/appStore'
import type { FontSize } from '@/types'

/** 字体尺寸映射 */
const FONT_SIZE_MAP: Record<FontSize, string> = {
  xs: '13px',
  sm: '14px',
  md: '16px',
  lg: '18px',
  xl: '20px',
}

/** 将字号偏好映射为根节点样式，接收要渲染的子内容，返回组件内容 */
export function FontSizeProvider({ children }: { /* 要渲染的子内容 */ children: React.ReactNode }) {
  const fontSize = useAppStore(s => s.fontSize) // 从状态仓库读取字体尺寸

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    document.documentElement.style.fontSize = FONT_SIZE_MAP[fontSize]
  }, [fontSize])

  return <>{children}</>
}
