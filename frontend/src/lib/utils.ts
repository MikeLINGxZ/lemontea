import { isObjectRecord } from '@/lib/guards'
import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import type { Message, Conversation, DisplayMessage } from '../types'

/** 合并条件类名并解决 Tailwind 工具类冲突，接收输入列表，返回对应的处理结果 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** 是否macOS */
export const isMac =
  typeof navigator !== 'undefined' &&
  (/Mac/i.test(navigator.platform) || /Mac/i.test(navigator.userAgent))

/** 终点日期处理，接收值，返回对应的处理结果 */
function toDate(value: string | Date): Date {
  return value instanceof Date ? value : new Date(value)
}

/** 判断是否需要显示时间戳，接收消息列表、索引，返回是否满足条件 */
export function shouldShowTimestamp(messages: Message[], index: number): boolean {
  if (index === 0) return true
  const prev = toDate(messages[index - 1].createdAt) // 更新前的值
  const curr = toDate(messages[index].createdAt) // 当前项
  const prevDay = new Date(prev.getFullYear(), prev.getMonth(), prev.getDate()) // 比较用的前一天
  const currDay = new Date(curr.getFullYear(), curr.getMonth(), curr.getDate()) // 比较用的当前日期
  return prevDay.getTime() !== currDay.getTime()
}

/** 格式化时间戳，接收日期值、语言，返回处理后的文本 */
export function formatTimestamp(dateValue: string | Date, language: string): string {
  const date = toDate(dateValue) // 日期
  const now = new Date() // 当前时间
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()) // 今天
  const yesterday = new Date(today.getTime() - 86400000) // 昨天
  const msgDay = new Date(date.getFullYear(), date.getMonth(), date.getDate()) // 消息所在的日期

  /** 时间文本 */
  const timeStr = date.toLocaleTimeString(language === 'zh-CN' ? 'zh-CN' : 'en-US', {
    hour: '2-digit',
    minute: '2-digit',
  })

  const isZh = language === 'zh-CN' // 是否中文

  if (msgDay.getTime() === today.getTime()) {
    return isZh ? `今天 ${timeStr}` : `Today ${timeStr}`
  }
  if (msgDay.getTime() === yesterday.getTime()) {
    return isZh ? `昨天 ${timeStr}` : `Yesterday ${timeStr}`
  }
  if (date.getFullYear() === now.getFullYear()) {
    return isZh
      ? `${date.getMonth() + 1}月${date.getDate()}日 ${timeStr}`
      : `${date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} ${timeStr}`
  }
  return isZh
    ? `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日 ${timeStr}`
    : `${date.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })} ${timeStr}`
}

/** 同一轮消息中各内容类型的显示顺序 */
const CONTENT_TYPE_PRIORITY: Record<string, number> = {
  thinking: 0,
  tool_call: 1,
  text: 2,
}

/** 把工具结果和确认记录附加到工具调用，并重组可显示的消息顺序，接收消息列表，返回对应的处理结果 */
export function groupMessagesForDisplay(messages: Message[]): DisplayMessage[] {
  const toolResultMap = new Map<number, string>() // 工具结果映射
  /** 工具确认映射 */
  const toolConfirmMap = new Map<number, { /* 操作 */ action: 'approve' | 'reject' | 'comment'; /* 备注 */ comment?: string }>()

  for (/* 循环索引 */ let i = 0; i < messages.length; i++) {
    const m = messages[i] // 正在处理的消息
    if (m.contentType === 'tool_result') {
      for (/* 向前查找的索引 */ let j = i - 1; j >= 0; j--) {
        if (messages[j].contentType === 'tool_call') {
          toolResultMap.set(messages[j].id, m.content)
          break
        }
      }
      continue
    }

    if (m.contentType === 'confirm_response') {
      try {
        const parsed: unknown = JSON.parse(m.extra || '{}') // 保存待校验的 JSON 解析结果
        if (!isObjectRecord(parsed)) continue
        if (parsed.action === 'approve' || parsed.action === 'reject' || parsed.action === 'comment') {
          for (/* 向前查找的索引 */ let j = i - 1; j >= 0; j--) {
            if (messages[j].contentType === 'tool_call') {
              toolConfirmMap.set(messages[j].id, {
                action: parsed.action,
                comment: typeof parsed.comment === 'string' ? parsed.comment : undefined,
              })
              break
            }
          }
        }
      } catch {
        // 跳过无法解析的确认记录
      }
    }
  }

  /** 筛选结果 */
  const filtered = messages.filter(
    m => m.contentType !== 'tool_result' && m.contentType !== 'confirm_response'
  )

  const result: DisplayMessage[] = [] // 当前步骤的处理结果
  let roundStart = 0 // 轮次开始

  for (/* 循环索引 */ let i = 0; i <= filtered.length; i++) {
    const isEnd = i === filtered.length // 是否结束
    /** 是否用户文本 */
    const isUserText = !isEnd && filtered[i].role === 'user' && filtered[i].contentType === 'text'

    if ((isEnd || isUserText) && i > roundStart) {
      const round = filtered.slice(roundStart, i) // 轮次
      /** 排序结果 */
      const sorted = [...round].sort((a, b) => {
        const pa = CONTENT_TYPE_PRIORITY[a.contentType] ?? 3 // 当前处理位置的值
        const pb = CONTENT_TYPE_PRIORITY[b.contentType] ?? 3 // 当前处理位置的值
        return pa - pb
      })

      for (/* 正在处理的消息 */ const m of sorted) {
        const display: DisplayMessage = { ...m } // 显示
        if (m.contentType === 'tool_call') {
          display.toolResult = toolResultMap.get(m.id)
          const confirmMeta = toolConfirmMap.get(m.id) // 确认元信息
          if (confirmMeta) {
            display.toolConfirmAction = confirmMeta.action
            display.toolConfirmComment = confirmMeta.comment
          }
        }
        result.push(display)
      }
    }

    if (!isEnd && isUserText) {
      result.push({ ...filtered[i] })
      roundStart = i + 1
    }
  }

  const grouped: DisplayMessage[] = [] // 已分组
  let toolGroup: DisplayMessage[] = [] // 工具分组

  /** 提交并清空工具分组，无返回值 */
  const flushToolGroup = () => {
    if (toolGroup.length === 0) return
    if (toolGroup.length === 1) {
      grouped.push(toolGroup[0])
    } else {
      const first = toolGroup[0] // 第一项
      grouped.push({
        ...first,
        id: first.id,
        isToolGroup: true,
        groupedTools: toolGroup,
      })
    }
    toolGroup = []
  }

  for (/* 正在处理的消息 */ const m of result) {
    if (m.contentType === 'tool_call') {
      toolGroup.push(m)
    } else {
      flushToolGroup()
      grouped.push(m)
    }
  }
  flushToolGroup()

  return grouped
}

/** 分组会话列表按日期，接收会话列表、语言，返回对应的处理结果 */
export function groupConversationsByDate(
  conversations: Conversation[],
  language: string
): { /* 标签 */ label: string; /* 条目列表 */ items: Conversation[] }[] {
  const now = new Date() // 当前时间
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()) // 今天
  const yesterday = new Date(today.getTime() - 86400000) // 昨天
  const sevenDaysAgo = new Date(today.getTime() - 7 * 86400000) // 七天之前

  /** 分组 */
  const groups: Record<string, Conversation[]> = {
    today: [],
    yesterday: [],
    week: [],
    earlier: [],
  }

  for (/* 会话 */ const conv of conversations) {
    const updatedAt = toDate(conv.updatedAt) // 更新时间
    /** 当前处理位置的值 */
    const d = new Date(
      updatedAt.getFullYear(),
      updatedAt.getMonth(),
      updatedAt.getDate()
    )
    if (d >= today) groups.today.push(conv)
    else if (d >= yesterday) groups.yesterday.push(conv)
    else if (d >= sevenDaysAgo) groups.week.push(conv)
    else groups.earlier.push(conv)
  }

  /** 显示标签 */
  const labels = language === 'zh-CN'
    ? { today: '今天', yesterday: '昨天', week: '过去7天', earlier: '更久以前' }
    : { today: 'Today', yesterday: 'Yesterday', week: 'Last 7 days', earlier: 'Earlier' }

  return Object.entries(groups)
    .filter(([, items]) => items.length > 0)
    .map(([key, items]) => ({ label: labels[key as keyof typeof labels], items }))
}
