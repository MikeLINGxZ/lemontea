import { ChatHeader } from './ChatHeader'
import { ChatMessages } from './ChatMessages'
import { ChatInput } from './ChatInput'

/** 聊天内容组件接收的属性 */
type ChatContentProps = {
  /** 会话编号 */
  sessionId?: number
  /** 展示变体 */
  variant?: 'default' | 'quick'
}

/** 按当前会话状态组织聊天区域的主要内容，接收会话编号、展示变体，返回组件内容 */
export function ChatContent({ sessionId, variant = 'default' }: ChatContentProps) {
  return (
    <>
      <ChatMessages sessionId={sessionId} variant={variant} />
      <ChatInput sessionId={sessionId} variant={variant} />
    </>
  )
}

/** 聊天区域组件接收的属性 */
interface ChatAreaProps {
  /** 侧栏折叠 */
  sidebarCollapsed: boolean
  /** 处理展开侧栏事件，无返回值 */
  onExpandSidebar: () => void
}

/** 组合聊天页标题、消息列表和输入区，接收侧栏折叠状态和展开回调，返回组件内容 */
export function ChatArea({
  sidebarCollapsed,
  onExpandSidebar,
}: ChatAreaProps) {
  return (
    <div className="flex flex-col h-full bg-background">
      <ChatHeader
        sidebarCollapsed={sidebarCollapsed}
        onExpandSidebar={onExpandSidebar}
      />
      <ChatContent />
    </div>
  )
}
