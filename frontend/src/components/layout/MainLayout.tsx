import { startTransition, useEffect, useRef, useState } from 'react'
import { Events } from '@wailsio/runtime'
import { Group as PanelGroup, Panel, Separator as PanelResizeHandle } from 'react-resizable-panels'
import { Sidebar } from '@/components/sidebar/Sidebar'
import { ChatArea } from '@/components/chat/ChatArea'
import { isH5Width } from '@/lib/responsive'
import { isMac } from '@/lib/utils'
import { useChatStore } from '@/store/chatStore'

/** 快捷聊天选中项事件 */
const QUICK_CHAT_SELECTED_EVENT = 'quick_chat:selected'

/** 组合会话侧栏和聊天区域，并根据窗口宽度管理侧栏状态，返回组件内容 */
export function MainLayout() {
  const loadSessions = useChatStore((state) => state.loadSessions) // 按游标加载会话列表，重置时从第一页开始
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false) // 侧栏折叠及其更新方法
  const [isNarrow, setIsNarrow] = useState(false) // 是否窄屏及其更新方法
  const [windowFocused, setWindowFocused] = useState(true) // 窗口获得焦点及其更新方法
  const [animateWindowControlSpace, setAnimateWindowControlSpace] = useState(false) // 动画窗口控制区域及其更新方法
  const autoCollapsedRef = useRef(false) // 自动折叠引用，供后续读取当前值
  const wasNarrowRef = useRef(false) // 曾经窄屏引用，供后续读取当前值
  const showWindowControlSpace = isMac && windowFocused // 显示窗口控制区域

  /** 收起侧栏，接收自动，无返回值 */
  const collapseSidebar = (auto = false) => {
    autoCollapsedRef.current = auto
    setSidebarCollapsed(true)
  }

  /** 展开侧栏，无返回值 */
  const expandSidebar = () => {
    autoCollapsedRef.current = false
    setSidebarCollapsed(false)
  }

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 处理调整尺寸，无返回值 */
    const handleResize = () => {
      const isNarrow = isH5Width(window.innerWidth) // 是否窄屏

      setIsNarrow(isNarrow)

      if (isNarrow && !wasNarrowRef.current) {
        autoCollapsedRef.current = true
        setSidebarCollapsed(true)
      }
      if (!isNarrow && wasNarrowRef.current && autoCollapsedRef.current) {
        autoCollapsedRef.current = false
        setSidebarCollapsed(false)
      }

      wasNarrowRef.current = isNarrow
    }

    handleResize()
    window.addEventListener('resize', handleResize)

    return () => window.removeEventListener('resize', handleResize)
  }, [])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    void loadSessions(true)
  }, [loadSessions])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 保存事件解绑方法 */
    const offQuickChatSelected = Events.On(QUICK_CHAT_SELECTED_EVENT, (event: { /* 事件携带的数据 */ data?: { /* 会话编号 */ sessionId?: number } }) => {
      const sessionId = event.data?.sessionId // 会话编号
      if (typeof sessionId !== 'number' || sessionId <= 0) return
      startTransition(() => {
        useChatStore.getState().setCurrentConversation(sessionId)
      })
    })

    return () => {
      if (typeof offQuickChatSelected === 'function') {
        offQuickChatSelected()
      }
    }
  }, [])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!isMac) return

    let timeoutId: number | undefined // 超时编号
    /** 处理焦点变更，接收获得焦点，无返回值 */
    const handleFocusChange = (focused: boolean) => {
      if (timeoutId) window.clearTimeout(timeoutId)
      setAnimateWindowControlSpace(true)
      setWindowFocused(focused)
      timeoutId = window.setTimeout(() => setAnimateWindowControlSpace(false), 320)
    }

    /** 保存事件解绑方法 */
    const offFocus = Events.On(Events.Types.Common.WindowFocus, () => handleFocusChange(true))
    /** 保存事件解绑方法 */
    const offLostFocus = Events.On(Events.Types.Common.WindowLostFocus, () => handleFocusChange(false))

    return () => {
      if (timeoutId) window.clearTimeout(timeoutId)
      offFocus()
      offLostFocus()
    }
  }, [])

  return (
    <div className="h-screen w-screen overflow-hidden bg-background">
      {!sidebarCollapsed && isNarrow ? (
        <Sidebar onCollapse={() => collapseSidebar()} />
      ) : (
        <PanelGroup orientation="horizontal" className="h-full">
          {!sidebarCollapsed && (
            <>
              <Panel
                id="sidebar"
                defaultSize="260px"
                minSize="240px"
                maxSize="500px"
                className="relative z-20 flex flex-col overflow-visible"
              >
                <Sidebar onCollapse={() => collapseSidebar()} />
              </Panel>
              <PanelResizeHandle className="w-[3px] bg-border hover:bg-primary/40 transition-colors cursor-col-resize" />
            </>
          )}
          <Panel id="chat" className="relative z-0 flex flex-col min-w-0">
            <ChatArea
              sidebarCollapsed={sidebarCollapsed}
              onExpandSidebar={expandSidebar}
              showWindowControlSpace={showWindowControlSpace}
              animateWindowControlSpace={animateWindowControlSpace}
            />
          </Panel>
        </PanelGroup>
      )}
    </div>
  )
}
