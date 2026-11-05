import { createContext, type ReactNode, useEffect, useRef, useState } from 'react'
import { SettingsPrimaryMenu } from '@/components/settings/SettingsPrimaryMenu'
import { isH5Width } from '@/lib/responsive'
import { useSettingsStore } from '@/store/settingsStore'

/** 设置菜单上下文 */
export const SettingsMenuContext = createContext<{
  /** 是否采用窄屏布局 */
  isH5: boolean
  /** 处理展开菜单事件，无返回值 */
  onOpenMenu?: () => void
  /** 二级菜单折叠 */
  submenuCollapsed: boolean
  /** 处理折叠二级菜单事件，无返回值 */
  onCollapseSubmenu: () => void
  /** 处理展开二级菜单事件，无返回值 */
  onOpenSubmenu?: () => void
}>({
  isH5: false,
  submenuCollapsed: false,
  /** 处理折叠二级菜单事件，无返回值 */
  onCollapseSubmenu: () => {},
})

/** 提供设置页分栏和窄屏导航上下文，接收组件输入属性，返回组件内容 */
export function SettingsShell(props: { /* 要渲染的子内容 */ children: ReactNode }) {
  const activeTab = useSettingsStore((state) => state.activeTab) // 从状态仓库读取当前标签页
  const setActiveTab = useSettingsStore((state) => state.setActiveTab) // 从状态仓库读取集合当前标签页
  const [menuCollapsed, setMenuCollapsed] = useState(false) // 菜单折叠及其更新方法
  const [submenuCollapsed, setSubmenuCollapsed] = useState(false) // 二级菜单折叠及其更新方法
  const [isH5, setIsH5] = useState(false) // 是否采用窄屏布局及其更新方法
  const autoCollapsedRef = useRef(false) // 自动折叠引用，供后续读取当前值
  const wasNarrowRef = useRef(false) // 曾经窄屏引用，供后续读取当前值

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 处理调整尺寸，无返回值 */
    const handleResize = () => {
      const narrow = isH5Width(window.innerWidth) // 窄屏
      setIsH5(narrow)

      if (narrow && !wasNarrowRef.current) {
        autoCollapsedRef.current = true
        setMenuCollapsed(true)
        setSubmenuCollapsed(true)
      }
      if (!narrow && wasNarrowRef.current && autoCollapsedRef.current) {
        autoCollapsedRef.current = false
        setMenuCollapsed(false)
        setSubmenuCollapsed(false)
      }

      wasNarrowRef.current = narrow
    }

    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  /** 处理变更标签页，接收标签页，无返回值 */
  const handleChangeTab = (tab: typeof activeTab) => {
    setActiveTab(tab)
    if (isH5) {
      setMenuCollapsed(true)
      setSubmenuCollapsed(false)
    }
  }

  /** on展开菜单 */
  const onOpenMenu = (menuCollapsed && isH5)
    ? () => { autoCollapsedRef.current = false; setMenuCollapsed(false) }
    : undefined

  /** on展开二级菜单 */
  const onOpenSubmenu = (submenuCollapsed && isH5)
    ? () => setSubmenuCollapsed(false)
    : undefined

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground [--settings-top-padding:2.5rem]">
      {!menuCollapsed && !isH5 && <SettingsPrimaryMenu activeTab={activeTab} onChange={handleChangeTab} />}
      {!menuCollapsed && isH5 ? (
        <main className="relative flex h-full w-full flex-col bg-background">
          <SettingsPrimaryMenu activeTab={activeTab} onChange={handleChangeTab} isH5 />
        </main>
      ) : (
        <main className="relative flex min-h-0 min-w-0 flex-1 flex-col">
          <SettingsMenuContext.Provider value={{
            isH5,
            onOpenMenu,
            submenuCollapsed,
            /** 处理折叠二级菜单事件，返回表达式结果 */
            onCollapseSubmenu: () => setSubmenuCollapsed(true),
            onOpenSubmenu,
          }}>
            <div className="flex min-h-0 min-w-0 flex-1 flex-col">
              {props.children}
            </div>
          </SettingsMenuContext.Provider>
        </main>
      )}
    </div>
  )
}
