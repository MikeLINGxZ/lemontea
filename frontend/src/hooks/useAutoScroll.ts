import { useEffect, useLayoutEffect, useRef, useState, useCallback } from 'react'
import type { DependencyList } from 'react'

/** 滚动消息容器到其内容末尾，接收元素、平滑滚动，无返回值 */
function scrollElementToBottom(el: HTMLDivElement, smooth = false) {
  const top = el.scrollHeight // 顶部
  if (typeof el.scrollTo === 'function') {
    el.scrollTo({ top, behavior: smooth ? 'smooth' : 'auto' })
    return
  }
  el.scrollTop = top
}

/** 维护消息容器的底部状态并返回自动滚动控制方法，接收触发更新的依赖列表、变化时重置状态的标识，返回滚动容器引用、底部状态和滚动方法 */
export function useAutoScroll(deps: DependencyList, resetKey?: DependencyList[number]) {
  const containerRef = useRef<HTMLDivElement>(null) // 容器引用，供后续读取当前值
  const [isAtBottom, setIsAtBottom] = useState(true) // 是否时间底部及其更新方法
  const autoScrollRef = useRef(true) // 自动滚动引用，供后续读取当前值

  /** 滚动终点底部处理，接收平滑滚动，无返回值 */
  const scrollToBottom = useCallback((smooth = false) => {
    const el = containerRef.current // 当前处理位置的值
    if (!el) return
    scrollElementToBottom(el, smooth)
    autoScrollRef.current = true
    setIsAtBottom(true)
  }, [])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    const el = containerRef.current // 当前处理位置的值
    if (!el) return
    /** 响应当前订阅的事件，无返回值 */
    const handler = () => {
      const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50 // 是否已滚动到消息底部
      setIsAtBottom(atBottom)
      autoScrollRef.current = atBottom
    }
    el.addEventListener('scroll', handler, { passive: true })
    return () => el.removeEventListener('scroll', handler)
  }, [])

  // eslint-disable-next-line react-hooks/exhaustive-deps
  /** 在浏览器绘制前同步布局状态 */
  useLayoutEffect(() => {
    if (resetKey === undefined) return
    autoScrollRef.current = true
    const el = containerRef.current // 当前处理位置的值
    if (!el) return
    scrollElementToBottom(el)
    setIsAtBottom(true)
  }, [resetKey])

  // eslint-disable-next-line react-hooks/exhaustive-deps
  /** 在浏览器绘制前同步布局状态 */
  useLayoutEffect(() => {
    if (autoScrollRef.current) {
      const el = containerRef.current // 当前处理位置的值
      if (!el) return
      scrollElementToBottom(el)
    }
  }, deps)

  return { containerRef, isAtBottom, scrollToBottom, autoScrollRef }
}
