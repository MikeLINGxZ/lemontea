import { startTransition, useEffect, useRef, useState } from 'react'
import { Events } from '@wailsio/runtime'
import { Group as PanelGroup, Panel, Separator as PanelResizeHandle } from 'react-resizable-panels'
import { Sidebar } from '@/components/sidebar/Sidebar'
import { ChatArea } from '@/components/chat/ChatArea'
import { isH5Width } from '@/lib/responsive'
import { isMac } from '@/lib/utils'
import { useChatStore } from '@/store/chatStore'

const QUICK_CHAT_SELECTED_EVENT = 'quick_chat:selected'

export function MainLayout() {
  const loadSessions = useChatStore((state) => state.loadSessions)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [isNarrow, setIsNarrow] = useState(false)
  const [windowFocused, setWindowFocused] = useState(true)
  const [animateWindowControlSpace, setAnimateWindowControlSpace] = useState(false)
  const autoCollapsedRef = useRef(false)
  const wasNarrowRef = useRef(false)
  const showWindowControlSpace = isMac && windowFocused

  const collapseSidebar = (auto = false) => {
    autoCollapsedRef.current = auto
    setSidebarCollapsed(true)
  }

  const expandSidebar = () => {
    autoCollapsedRef.current = false
    setSidebarCollapsed(false)
  }

  useEffect(() => {
    const handleResize = () => {
      const isNarrow = isH5Width(window.innerWidth)

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

  useEffect(() => {
    void loadSessions(true)
  }, [loadSessions])

  useEffect(() => {
    const offQuickChatSelected = Events.On(QUICK_CHAT_SELECTED_EVENT, (event: { data?: { sessionId?: number } }) => {
      const sessionId = event.data?.sessionId
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

  useEffect(() => {
    if (!isMac) return

    let timeoutId: number | undefined
    const handleFocusChange = (focused: boolean) => {
      if (timeoutId) window.clearTimeout(timeoutId)
      setAnimateWindowControlSpace(true)
      setWindowFocused(focused)
      timeoutId = window.setTimeout(() => setAnimateWindowControlSpace(false), 320)
    }

    const offFocus = Events.On(Events.Types.Common.WindowFocus, () => handleFocusChange(true))
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
