/** 提醒类别的可选值 */
export type AlertKind = 'error' | 'info' | 'success' | 'warning'

/** 提醒展示位置的可选值 */
export type AlertPlacement = 'toast' | 'banner'

/** 提醒操作样式的可选值 */
export type AlertActionStyle = 'primary' | 'secondary' | 'danger'

/** 提醒关闭原因的可选值 */
export type AlertCloseReason =
  | 'user'
  | 'timeout'
  | 'programmatic'
  | 'replaced'
  | 'action'

/** 提醒无障碍优先级的可选值 */
export type AlertAriaPriority = 'assertive' | 'polite'

/** 提醒来源的可选值 */
export type AlertSource = 'frontend' | 'backend'

/** 提醒去重行为的可选值 */
export type AlertDedupeBehavior = 'count' | 'replace'

/** 定义提醒操作的数据结构 */
export type AlertAction = {
  /** 当前条目的唯一编号 */
  id: string
  /** 标签 */
  label: string
  /** 样式 */
  style: AlertActionStyle
  /** 关闭on点击 */
  closeOnClick: boolean
  /** 链接地址 */
  href?: string
  /** 处理点击事件，无返回值 */
  onClick?: () => void
}

/** 定义提醒输入的数据结构 */
export type AlertInput = {
  /** 类别标识 */
  kind: AlertKind
  /** 展示位置 */
  placement: AlertPlacement
  /** 标题 */
  title: string
  /** 消息 */
  message: string
  /** 详情 */
  detail?: string | null
  /** 代码 */
  code?: string | null
  /** 操作列表 */
  actions?: AlertAction[]
  /** 可关闭 */
  dismissible?: boolean
  /** 自动关闭 */
  autoClose?: boolean
  /** 持续时间，单位为毫秒 */
  durationMs?: number | null
  /** 去重键 */
  dedupeKey?: string
  /** 去重行为 */
  dedupeBehavior?: AlertDedupeBehavior
  /** 显示数量 */
  showCount?: boolean
  /** 来源 */
  source?: AlertSource
}

/** 定义提醒条目的数据结构 */
export type AlertItem = {
  /** 当前条目的唯一编号 */
  id: string
  /** 类别标识 */
  kind: AlertKind
  /** 展示位置 */
  placement: AlertPlacement
  /** 标题 */
  title: string
  /** 消息 */
  message: string
  /** 详情 */
  detail: string | null
  /** 代码 */
  code: string | null
  /** 操作列表 */
  actions: AlertAction[]
  /** 可关闭 */
  dismissible: boolean
  /** 自动关闭 */
  autoClose: boolean
  /** 持续时间，单位为毫秒 */
  durationMs: number | null
  /** 数量 */
  count: number
  /** 去重键 */
  dedupeKey: string
  /** 去重行为 */
  dedupeBehavior: AlertDedupeBehavior
  /** 显示数量 */
  showCount: boolean
  /** 创建时间 */
  createdAt: number
  /** 更新时间 */
  updatedAt: number
  /** 无障碍优先级 */
  ariaPriority: AlertAriaPriority
  /** 来源 */
  source: AlertSource
}

/** 定义关闭提醒记录的数据结构 */
export type ClosedAlertRecord = {
  /** 当前条目的唯一编号 */
  id: string
  /** 原因 */
  reason: AlertCloseReason
}
