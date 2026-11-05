import { useState } from 'react'
import { MoreHorizontal, Star, Pencil, Trash2, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { useChatStore } from '@/store/chatStore'
import type { Conversation } from '@/types'

/** 会话条目组件接收的属性 */
interface Props {
  /** 会话 */
  conversation: Conversation
  /** 是否当前 */
  isActive: boolean
  /** 处理点击事件，无返回值 */
  onClick: () => void
}

/** 渲染会话条目界面，接收会话、是否当前、on点击，返回组件内容 */
export function ConversationItem({ conversation, isActive, onClick }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const deleteConversation = useChatStore((state) => state.deleteConversation) // 从状态仓库读取删除会话
  const renameConversation = useChatStore((state) => state.renameConversation) // 从状态仓库读取重命名会话
  const toggleStar = useChatStore((state) => state.toggleStar) // 从状态仓库读取切换收藏
  const [menuOpen, setMenuOpen] = useState(false) // 菜单展开及其更新方法
  const [isRenaming, setIsRenaming] = useState(false) // 是否正在重命名及其更新方法
  const [renameValue, setRenameValue] = useState(conversation.title) // 重命名值及其更新方法

  const { id, title, status, starred } = conversation // 取出当前条目的唯一编号、标题、状态、已收藏
  const isTask = (conversation.tags ?? []).includes('task') || conversation.kind === 'task' // 是否任务

  /** 处理重命名提交，无返回值 */
  const handleRenameSubmit = () => {
    if (renameValue.trim()) renameConversation(id, renameValue.trim())
    setIsRenaming(false)
  }

  return (
    <div
      className={cn(
        'group relative flex items-center gap-1.5 px-2 py-1.5 rounded-lg cursor-pointer select-none',
        'hover:bg-accent transition-colors',
        isActive && 'bg-accent'
      )}
      onClick={() => !isRenaming && onClick()}
    >
      {(() => {
        if (status === 'loading') return <Loader2 size={12} className="animate-spin text-muted-foreground shrink-0" />
        if (status === 'done-unread') return <span className="w-2 h-2 rounded-full bg-green-500 shrink-0" />
        if (status === 'error-unread') return <span className="w-2 h-2 rounded-full bg-red-500 shrink-0" />
        if (status === 'waiting-unread') return <span className="w-2 h-2 rounded-full bg-blue-500 shrink-0" />
        return <span className="w-2 h-2 shrink-0" />
      })()}

      {isRenaming ? (
        <input
          className="flex-1 min-w-0 bg-transparent border-b border-primary outline-hidden text-sm text-foreground"
          value={renameValue}
          onChange={e => setRenameValue(e.target.value)}
          onBlur={handleRenameSubmit}
          onKeyDown={e => {
            if (e.key === 'Enter') handleRenameSubmit()
            if (e.key === 'Escape') setIsRenaming(false)
          }}
          autoFocus
          onClick={e => e.stopPropagation()}
        />
      ) : (
        <span className="flex-1 min-w-0 flex items-center gap-1.5 truncate">
          {isTask && (
            <span className="shrink-0 rounded px-1 py-0.5 text-[10px] font-medium bg-purple-100 text-purple-700 dark:bg-purple-500/20 dark:text-purple-400">
              {t('sidebar.taskBadge')}
            </span>
          )}
          <span className="truncate text-sm text-foreground">{title}</span>
        </span>
      )}

      <button
        className={cn(
          'shrink-0 p-0.5 rounded text-muted-foreground hover:text-foreground',
          'opacity-0 group-hover:opacity-100 transition-opacity',
          menuOpen && 'opacity-100'
        )}
        onClick={e => {
          e.stopPropagation()
          setMenuOpen(v => !v)
        }}
      >
        <MoreHorizontal size={14} />
      </button>

      {menuOpen && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={() => setMenuOpen(false)}
          />
          <div className="absolute right-0 top-full mt-1 z-20 w-36 rounded-lg border border-border bg-popover shadow-md py-1">
            <button
              className="w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent text-left"
              onClick={e => {
                e.stopPropagation()
                void toggleStar(id, !starred)
                setMenuOpen(false)
              }}
            >
              <Star size={14} />
              {starred ? t('conversation.unstar') : t('conversation.star')}
            </button>
            <button
              className="w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent text-left"
              onClick={e => {
                e.stopPropagation()
                setRenameValue(title)
                setIsRenaming(true)
                setMenuOpen(false)
              }}
            >
              <Pencil size={14} />
              {t('conversation.rename')}
            </button>
            <button
              className="w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent text-destructive text-left"
              onClick={e => {
                e.stopPropagation()
                deleteConversation(id)
                setMenuOpen(false)
              }}
            >
              <Trash2 size={14} />
              {t('conversation.delete')}
            </button>
          </div>
        </>
      )}
    </div>
  )
}
