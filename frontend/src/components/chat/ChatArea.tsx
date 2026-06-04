import { ChatHeader } from './ChatHeader'
import { ChatMessages } from './ChatMessages'
import { ChatInput } from './ChatInput'

type ChatContentProps = {
  sessionId?: number
  variant?: 'default' | 'quick'
}

export function ChatContent({ sessionId, variant = 'default' }: ChatContentProps) {
  return (
    <>
      <ChatMessages sessionId={sessionId} variant={variant} />
      <ChatInput sessionId={sessionId} variant={variant} />
    </>
  )
}

interface ChatAreaProps {
  sidebarCollapsed: boolean
  onExpandSidebar: () => void
  showWindowControlSpace: boolean
  animateWindowControlSpace: boolean
}

export function ChatArea({
  sidebarCollapsed,
  onExpandSidebar,
  showWindowControlSpace,
  animateWindowControlSpace,
}: ChatAreaProps) {
  return (
    <div className="flex flex-col h-full bg-background">
      <ChatHeader
        sidebarCollapsed={sidebarCollapsed}
        onExpandSidebar={onExpandSidebar}
        showWindowControlSpace={showWindowControlSpace}
        animateWindowControlSpace={animateWindowControlSpace}
      />
      <ChatContent />
    </div>
  )
}
