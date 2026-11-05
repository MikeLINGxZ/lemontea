import { startTransition, useState } from 'react'
import { Plus, Search, MessageSquare, Star } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { groupConversationsByDate } from '@/lib/utils'
import { useChatStore } from '@/store/chatStore'
import { useAppStore } from '@/store/appStore'
import { useShallow } from 'zustand/react/shallow'
import { ConversationItem } from './ConversationItem'
import type { Conversation, ConversationTab } from '@/types'

/** 判断是否为当前任务，接收c，返回是否满足条件 */
function isActiveTask(c: Conversation) {
  return ((c.tags ?? []).includes('task') || c.kind === 'task') && c.status !== 'idle'
}

/** 渲染会话列表界面，返回组件内容 */
export function ConversationList() {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const language = useAppStore(s => s.language) // 从状态仓库读取语言
  /** 从状态仓库读取会话列表、当前会话编号、集合当前会话 */
  const { conversations, currentConversationId, setCurrentConversation } =
    useChatStore(useShallow((state) => ({
      conversations: state.conversations,
      currentConversationId: state.currentConversationId,
      setCurrentConversation: state.setCurrentConversation,
    })))
  const [tab, setTab] = useState<ConversationTab>('chats') // 标签页及其更新方法
  const [search, setSearch] = useState('') // 搜索及其更新方法

  /** 当前任务列表 */
  const activeTasks = conversations.filter(c =>
    isActiveTask(c) &&
    (!search || c.title.toLowerCase().includes(search.toLowerCase()))
  )

  /** 列表条目列表 */
  const listItems = conversations.filter(c => {
    if (isActiveTask(c)) return false
    if (tab === 'favorites' && !c.starred) return false
    if (search && !c.title.toLowerCase().includes(search.toLowerCase())) return false
    return true
  })

  const groups = groupConversationsByDate(listItems, language) // 分组

  /** 处理新聊天，无返回值 */
  const handleNewChat = () => {
    startTransition(() => {
      setCurrentConversation(null)
    })
  }

  return (
    <div className="flex flex-col flex-1 min-h-0 px-2 py-2 gap-2">
      {/* 新建会话按钮 */}
      <button
        onClick={() => { void handleNewChat() }}
        className="flex items-center gap-2 w-full px-3 py-2 rounded-lg border border-border hover:bg-accent transition-colors text-sm font-medium text-foreground"
      >
        <Plus size={16} />
        {t('sidebar.newChat')}
      </button>

      {/* 切换会话与收藏列表 */}
      <div className="flex rounded-lg bg-muted p-0.5">
        {(['chats', 'favorites'] as ConversationTab[]).map(t2 => (
          <button
            key={t2}
            onClick={() => setTab(t2)}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 py-1 rounded-md text-sm font-medium transition-colors',
              tab === t2
                ? 'bg-background text-foreground shadow-xs'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {t2 === 'chats' ? <MessageSquare size={12} /> : <Star size={12} />}
            {t2 === 'chats' ? t('sidebar.chats') : t('sidebar.favorites')}
          </button>
        ))}
      </div>

      {/* 搜索会话 */}
      <div className="relative">
        <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder={t('sidebar.searchPlaceholder')}
          className="w-full pl-8 pr-3 py-1.5 rounded-lg bg-muted text-sm outline-hidden focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
        />
      </div>

      {/* 会话列表 */}
      <div className="sidebar-scrollbar flex-1 overflow-y-auto min-h-0 -mx-2 px-2">
        <div className="flex flex-col gap-0.5">
          {/* 进行中的任务在列表顶部单独显示 */}
          {activeTasks.length > 0 && (
            <div>
              <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                {t('sidebar.tasks')}
              </div>
              {activeTasks.map(conv => (
                <ConversationItem
                  key={conv.id}
                  conversation={conv}
                  isActive={conv.id === currentConversationId}
                  onClick={() => {
                    startTransition(() => { setCurrentConversation(conv.id) })
                  }}
                />
              ))}
            </div>
          )}

          {/* 普通会话与已完成任务按日期分组 */}
          {groups.map(group => (
            <div key={group.label}>
              <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                {group.label}
              </div>
              {group.items.map(conv => (
                <ConversationItem
                  key={conv.id}
                  conversation={conv}
                  isActive={conv.id === currentConversationId}
                  onClick={() => {
                    startTransition(() => { setCurrentConversation(conv.id) })
                  }}
                />
              ))}
            </div>
          ))}
        </div>

        {/* 底部显示会话总数 */}
        <div className="py-3 text-center text-xs text-muted-foreground">
          {t('sidebar.loadedAll', { count: listItems.length + activeTasks.length })}
        </div>
      </div>
    </div>
  )
}
