import { SidebarHeader } from './SidebarHeader'
import { ConversationList } from './ConversationList'
import { SidebarFooter } from './SidebarFooter'

/** 侧栏组件接收的属性 */
interface SidebarProps {
  /** 处理折叠事件，无返回值 */
  onCollapse: () => void
}

/** 渲染侧栏界面，接收on折叠，返回组件内容 */
export function Sidebar({ onCollapse }: SidebarProps) {
  return (
    <div className="flex flex-col h-full overflow-visible bg-background">
      <SidebarHeader onCollapse={onCollapse} />
      <ConversationList />
      <SidebarFooter />
    </div>
  )
}
