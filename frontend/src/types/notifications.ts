/** 通知类别的可选值 */
export type NotificationKind = 'needs_attention' | 'info'

/** 定义通知条目的数据结构 */
export type NotificationItem = {
  /** 当前条目的唯一编号 */
  id: number
  /** 会话编号 */
  session_id: number
  /** 类别标识 */
  kind: NotificationKind
  /** 标题 */
  title: string
  /** 消息 */
  message: string
  /** 创建时间 */
  created_at: string
  /** 处理完成的时间 */
  resolved_at: string | null
}
