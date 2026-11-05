import { PanelLeftClose } from 'lucide-react'
import { isMac } from '@/lib/utils'

/** 侧栏页头组件接收的属性 */
interface SidebarHeaderProps {
  /** 处理折叠事件，无返回值 */
  onCollapse: () => void
}

/** 渲染侧栏页头界面，接收on折叠，返回组件内容 */
export function SidebarHeader({ onCollapse }: SidebarHeaderProps) {
  return (
    <div className="flex h-12 items-center gap-2.5 px-4 select-none shrink-0">
      {!isMac && (
        <>
          <div className="group cursor-pointer">
            <svg
              width="32"
              height="32"
              viewBox="0 0 32 32"
              fill="none"
              className="group-hover:animate-bounce"
            >
              <circle cx="16" cy="16" r="14" fill="hsl(var(--primary))" />
              <circle cx="16" cy="16" r="7" fill="hsl(var(--primary-foreground))" opacity="0.9" />
              <circle cx="16" cy="9" r="2.3" fill="hsl(var(--primary-foreground))" />
            </svg>
          </div>
          <span className="text-lg font-semibold tracking-tight text-foreground">
            lemontea
          </span>
        </>
      )}
      <button
        type="button"
        aria-label="折叠侧边栏"
        onClick={onCollapse}
        className="ml-auto rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <PanelLeftClose size={16} />
      </button>
    </div>
  )
}
