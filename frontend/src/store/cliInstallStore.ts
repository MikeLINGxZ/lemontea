import { create } from 'zustand'
import type { CliInstallItem } from '@/types/cliInstall'

/** 当前值的状态数据 */
type State = {
  /** 条目列表 */
  items: CliInstallItem[]
  /** 新增或更新处理，接收当前条目，无返回值 */
  upsert: (item: CliInstallItem) => void
  /** 移除处理，接收当前条目的唯一编号，无返回值 */
  remove: (id: string | number) => void
  /** 清除处理，无返回值 */
  clear: () => void
}

/** 按任务标识保存 CLI 安装进度 */
export const useCliInstallStore = create<State>((set) => ({
  items: [],
  /** 新增或更新处理，接收当前条目，返回表达式结果 */
  upsert: (item) =>
    set((s) => {
      /** 查找到的位置 */
      const idx = s.items.findIndex((i) =>
        item.request_id || i.request_id
          ? i.request_id === item.request_id
          : (item.session_id != null && i.session_id === item.session_id) ||
            (!!item.extension_id && i.extension_id === item.extension_id) ||
            (!!item.npm_package && i.npm_package === item.npm_package),
      )
      if (idx >= 0) {
        const next = [...s.items] // 更新后的值
        next[idx] = { ...next[idx], ...item }
        return { items: next }
      }
      return { items: [...s.items, item] }
    }),
  /** 移除处理，接收当前条目的唯一编号，返回表达式结果 */
  remove: (id) =>
    set((s) => ({
      items: s.items.filter((i) =>
        i.request_id
          ? i.request_id !== id
          : typeof id === 'number'
            ? i.session_id !== id
            : i.npm_package !== id && i.extension_id !== id,
      ),
    })),
  /** 清除处理，返回表达式结果 */
  clear: () => set({ items: [] }),
}))
