import { create } from 'zustand'
import { createStore } from 'zustand/vanilla'
import type { StoreApi } from 'zustand'
import type {
  AlertAriaPriority,
  AlertCloseReason,
  AlertInput,
  AlertItem,
  ClosedAlertRecord,
} from '@/alert/types'

/** 提醒默认时长毫秒 */
export const ALERT_DEFAULT_DURATION_MS = 5000
/** 提醒上限时长毫秒 */
export const ALERT_MAX_DURATION_MS = 15000
/** 提醒浮动提示上限 */
export const ALERT_TOAST_LIMIT = 3
/** 提醒横幅上限 */
export const ALERT_BANNER_LIMIT = 1

/** 提醒的状态数据 */
type AlertState = {
  /** 浮动提醒列表 */
  toasts: AlertItem[]
  /** 横幅提醒列表 */
  banners: AlertItem[]
  /** 最后一个关闭 */
  lastClosed: ClosedAlertRecord | null
  /** 加入提醒处理，接收输入，返回处理后的文本 */
  pushAlert: (input: AlertInput) => string
  /** 关闭提醒，接收当前条目的唯一编号、原因，无返回值 */
  closeAlert: (id: string, reason: AlertCloseReason) => void
  /** 重置处理，无返回值 */
  reset: () => void
}

/** 定义定时器注册表的数据结构 */
type TimerRegistry = Map<string, ReturnType<typeof setTimeout>>

/** 下一项提醒编号 */
let nextAlertId = 0

/** 构造去重键，接收输入，返回处理后的文本 */
function buildDedupeKey(input: AlertInput): string {
  if (input.dedupeKey && input.dedupeKey.trim().length > 0) {
    return input.dedupeKey
  }

  return [
    input.kind,
    input.placement,
    input.title,
    input.message,
    input.detail ?? '',
  ].join('::')
}

/** 解析无障碍优先级，接收类别标识，返回对应的处理结果 */
function resolveAriaPriority(kind: AlertInput['kind']): AlertAriaPriority {
  return kind === 'error' ? 'assertive' : 'polite'
}

/** 解析自动关闭，接收输入，返回是否满足条件 */
function resolveAutoClose(input: AlertInput): boolean {
  if (typeof input.autoClose === 'boolean') {
    return input.autoClose
  }

  const actionCount = input.actions?.length ?? 0 // 操作数量
  if (actionCount > 0) {
    return false
  }

  return input.kind !== 'error'
}

/** 解析持续时间，单位为毫秒，接收输入、自动关闭，返回处理结果，无法取得时为空 */
function resolveDurationMs(input: AlertInput, autoClose: boolean): number | null {
  if (!autoClose) {
    return null
  }

  const candidate = input.durationMs ?? ALERT_DEFAULT_DURATION_MS // 候选值
  if (candidate <= 0) {
    return ALERT_DEFAULT_DURATION_MS
  }

  return Math.min(candidate, ALERT_MAX_DURATION_MS)
}

/** 规范化提醒，接收输入，返回对应的处理结果 */
function normalizeAlert(input: AlertInput): AlertItem {
  const now = Date.now() // 当前时间
  const autoClose = resolveAutoClose(input) // 自动关闭
  return {
    id: `alert-${now}-${nextAlertId += 1}`,
    kind: input.kind,
    placement: input.placement,
    title: input.title,
    message: input.message,
    detail: input.detail ?? null,
    code: input.code ?? null,
    actions: input.actions ?? [],
    dismissible: input.dismissible ?? true,
    autoClose,
    durationMs: resolveDurationMs(input, autoClose),
    count: 1,
    dedupeKey: buildDedupeKey(input),
    dedupeBehavior: input.dedupeBehavior ?? 'count',
    showCount: input.showCount ?? true,
    createdAt: now,
    updatedAt: now,
    ariaPriority: resolveAriaPriority(input.kind),
    source: input.source ?? 'frontend',
  }
}

/** 清除提醒定时器，接收定时器列表、当前条目的唯一编号，无返回值 */
function clearAlertTimer(timers: TimerRegistry, id: string): void {
  const timer = timers.get(id) // 定时器
  if (!timer) {
    return
  }

  clearTimeout(timer)
  timers.delete(id)
}

/** 优先级顺序值处理，接收类别标识，返回计算后的数值 */
function priorityRank(kind: AlertItem['kind']): number {
  if (kind === 'error') {
    return 4
  }
  if (kind === 'warning') {
    return 3
  }
  if (kind === 'success') {
    return 2
  }
  return 1
}

/** 创建提醒状态，接收定时器列表，返回对应的处理结果 */
function createAlertState(timers: TimerRegistry) {
  return (set: StoreApi<AlertState>['setState'], get: StoreApi<AlertState>['getState']): AlertState => {
    /** 调度定时器，接收当前条目，无返回值 */
    const scheduleTimer = (item: AlertItem): void => {
      clearAlertTimer(timers, item.id)
      if (!item.autoClose || item.durationMs === null) {
        return
      }

      /** 定时器 */
      const timer = setTimeout(() => {
        get().closeAlert(item.id, 'timeout')
      }, item.durationMs)
      timers.set(item.id, timer)
    }

    /** 应用浮动提示上限，接收条目列表，返回对应的处理结果 */
    const applyToastLimit = (items: AlertItem[]): AlertItem[] => {
      if (items.length <= ALERT_TOAST_LIMIT) {
        return items
      }

      const removed = items.slice(0, items.length - ALERT_TOAST_LIMIT) // 已移除
      for (/* 当前条目 */ const item of removed) {
        clearAlertTimer(timers, item.id)
      }

      return items.slice(items.length - ALERT_TOAST_LIMIT)
    }

    return {
      toasts: [],
      banners: [],
      lastClosed: null,
      /** 加入提醒处理，接收输入，返回对应的处理结果 */
      pushAlert: (input) => {
        const existingList = input.placement === 'toast' ? get().toasts : get().banners // 已存在列表
        const dedupeKey = buildDedupeKey(input) // 去重键
        const existing = existingList.find((item) => item.dedupeKey === dedupeKey) // 已有匹配项

        if (existing) {
          // 重复提醒按策略替换正文或增加计数，两种方式都重新安排关闭时间
          const now = Date.now() // 当前时间
          const autoClose = resolveAutoClose(input) // 自动关闭
          const shouldReplace = (input.dedupeBehavior ?? existing.dedupeBehavior) === 'replace' // 是否需替换
          /** 更新 */
          const updated: AlertItem = shouldReplace
            ? {
                ...existing,
                kind: input.kind,
                title: input.title,
                message: input.message,
                detail: input.detail ?? null,
                code: input.code ?? null,
                actions: input.actions ?? [],
                dismissible: input.dismissible ?? true,
                autoClose,
                durationMs: resolveDurationMs(input, autoClose),
                count: 1,
                updatedAt: now,
                ariaPriority: resolveAriaPriority(input.kind),
                source: input.source ?? existing.source,
                dedupeBehavior: input.dedupeBehavior ?? existing.dedupeBehavior,
                showCount: input.showCount ?? existing.showCount,
              }
            : {
                ...existing,
                count: existing.count + 1,
                updatedAt: now,
              }
          const nextList = existingList.map((item) => item.id === existing.id ? updated : item) // 更新后的列表
          if (input.placement === 'toast') {
            set({ toasts: nextList })
          } else {
            set({ banners: nextList })
          }
          scheduleTimer(updated)
          return existing.id
        }

        const item = normalizeAlert(input) // 统一补全提醒显示属性后再分流到横幅或浮动队列
        if (item.placement === 'toast') {
          const nextToasts = applyToastLimit([...get().toasts, item]) // 下一项浮动提醒列表
          set({ toasts: nextToasts })
        } else {
          const currentBanner = get().banners[0] // 当前横幅
          if (!currentBanner || priorityRank(item.kind) >= priorityRank(currentBanner.kind)) {
            if (currentBanner) {
              clearAlertTimer(timers, currentBanner.id)
            }
            set({
              banners: [item],
              lastClosed: currentBanner ? { id: currentBanner.id, reason: 'replaced' } : get().lastClosed,
            })
          }
        }

        scheduleTimer(item)
        return item.id
      },
      /** 关闭提醒，接收当前条目的唯一编号、原因，无返回值 */
      closeAlert: (id, reason) => {
        clearAlertTimer(timers, id)
        set((state) => ({
          toasts: state.toasts.filter((item) => item.id !== id),
          banners: state.banners.filter((item) => item.id !== id),
          lastClosed: { id, reason },
        }))
      },
      /** 重置处理，无返回值 */
      reset: () => {
        for (/* 当前条目的唯一编号 */ const id of Array.from(timers.keys())) {
          clearAlertTimer(timers, id)
        }
        set({
          toasts: [],
          banners: [],
          lastClosed: null,
        })
      },
    }
  }
}

/** 创建提醒状态仓库，返回对应的处理结果 */
export function createAlertStore(): StoreApi<AlertState> {
  const timers: TimerRegistry = new Map<string, ReturnType<typeof setTimeout>>() // 定时器列表
  return createStore<AlertState>(createAlertState(timers))
}

/** 提醒状态仓库 */
const alertStore = createAlertStore()

/** 保存提醒队列并管理去重、计时和关闭操作 */
export const useAlertStore = create<AlertState>()((set, get) =>
  createAlertState(new Map<string, ReturnType<typeof setTimeout>>())(set, get)
)

/** 重置提醒状态仓库，无返回值 */
export function resetAlertStore(): void {
  useAlertStore.getState().reset()
}

export { alertStore }
export type { AlertState }
