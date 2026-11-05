import type { AlertAction, AlertInput, AlertKind, AlertPlacement } from '@/alert/types'

/** 应用提醒事件 */
export const APP_ALERT_EVENT = 'app:alert'

/** 定义提醒事件载荷的数据结构 */
export type AlertEventPayload = {
  /** 类别标识 */
  kind?: AlertKind
  /** 展示位置 */
  placement?: AlertPlacement
  /** 标题 */
  title?: string
  /** 消息 */
  message?: string
  /** 详情 */
  detail?: string | null
  /** 代码 */
  code?: string | null
  /** 可关闭 */
  dismissible?: boolean
  /** 自动关闭 */
  auto_close?: boolean
  /** 持续时间，单位为毫秒 */
  duration_ms?: number | null
  /** 去重键 */
  dedupe_key?: string
  /** 操作列表 */
  actions?: AlertAction[]
}

/** 判断是否为提醒类别，接收值，返回是否满足条件 */
function isAlertKind(value: unknown): value is AlertKind {
  return value === 'error' || value === 'info' || value === 'success' || value === 'warning'
}

/** 判断是否为提醒展示位置，接收值，返回是否满足条件 */
function isAlertPlacement(value: unknown): value is AlertPlacement {
  return value === 'toast' || value === 'banner'
}

/** 判断是否为提醒操作，接收值，返回是否满足条件 */
function isAlertAction(value: unknown): value is AlertAction {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  const candidate = value as Record<string, unknown> // 候选值
  const style = candidate.style // 样式
  return typeof candidate.id === 'string'
    && typeof candidate.label === 'string'
    && (style === 'primary' || style === 'secondary' || style === 'danger')
    && typeof candidate.closeOnClick === 'boolean'
    && (candidate.href === undefined || typeof candidate.href === 'string')
}

/** 验证后端提醒事件并转换为界面提醒数据，接收载荷，返回处理结果，无法取得时为空 */
export function normalizeAlertEventPayload(payload: unknown): AlertInput | null {
  if (typeof payload !== 'object' || payload === null) {
    return null
  }

  const candidate = payload as Record<string, unknown> // 候选值
  if (!isAlertKind(candidate.kind) || !isAlertPlacement(candidate.placement)) {
    return null
  }
  if (typeof candidate.title !== 'string' || typeof candidate.message !== 'string') {
    return null
  }

  /** 操作列表 */
  const actions = Array.isArray(candidate.actions)
    ? candidate.actions.filter((item) => isAlertAction(item))
    : []

  return {
    kind: candidate.kind,
    placement: candidate.placement,
    title: candidate.title,
    message: candidate.message,
    detail: typeof candidate.detail === 'string' ? candidate.detail : null,
    code: typeof candidate.code === 'string' ? candidate.code : null,
    dismissible: typeof candidate.dismissible === 'boolean' ? candidate.dismissible : undefined,
    autoClose: typeof candidate.auto_close === 'boolean' ? candidate.auto_close : undefined,
    durationMs: typeof candidate.duration_ms === 'number' ? candidate.duration_ms : undefined,
    dedupeKey: typeof candidate.dedupe_key === 'string' ? candidate.dedupe_key : undefined,
    actions,
    source: 'backend',
  }
}
