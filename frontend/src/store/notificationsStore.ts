import { create } from 'zustand'
import type { NotificationItem } from '@/types/notifications'

/** 通知列表状态仓库保存的数据和操作 */
type NotificationsStore = {
  /** 条目列表 */
  items: NotificationItem[]
  /** 设置条目列表，接收条目列表，无返回值 */
  setItems: (items: NotificationItem[]) => void
  /** 前插处理，接收当前条目，无返回值 */
  prepend: (item: NotificationItem) => void
  /** 移除处理，接收当前条目的唯一编号，无返回值 */
  remove: (id: number) => void
}

/** 维护通知列表并按标识更新条目 */
export const useNotificationsStore = create<NotificationsStore>((set) => ({
  items: [],
  /** 设置条目列表，接收条目列表，返回表达式结果 */
  setItems: (items) => set({ items }),
  /** 前插处理，接收当前条目，返回表达式结果 */
  prepend: (item) => set((state) => ({
    items: [item, ...state.items.filter((current) => current.id !== item.id)],
  })),
  /** 移除处理，接收当前条目的唯一编号，返回表达式结果 */
  remove: (id) => set((state) => ({
    items: state.items.filter((item) => item.id !== id),
  })),
}))
