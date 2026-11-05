import { useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { Notification as NotificationBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/notification'
import { useNotificationsStore } from '@/store/notificationsStore'
import type { NotificationItem } from '@/types/notifications'

/** 通知创建事件的数据 */
type NotificationCreatedEvent = {
  /** 事件携带的数据 */
  data: NotificationItem | NotificationItem[]
}

/** 通知已处理事件的数据 */
type NotificationResolvedEvent = {
  /** 事件携带的数据 */
  data: { /* 当前条目的唯一编号 */ id: number } | { /* 当前条目的唯一编号 */ id: number }[]
}

/** 同步初始通知并订阅通知创建和处理事件，无返回值 */
export function useNotificationsSubscription() {
  const setItems = useNotificationsStore((state) => state.setItems) // 从状态仓库读取集合条目列表
  const prepend = useNotificationsStore((state) => state.prepend) // 从状态仓库读取前插
  const remove = useNotificationsStore((state) => state.remove) // 从状态仓库读取移除

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    void NotificationBinding.ListNotifications({ include_resolved: false }).then((result) => {
      setItems((result?.items ?? []).flatMap((item): NotificationItem[] => {
        const { kind } = item // 类别标识
        if (kind !== 'needs_attention' && kind !== 'info') return []

        return [{ ...item, kind, resolved_at: item.resolved_at ?? null }]
      }))
    })

    /** 订阅 notification.created 并保存解绑方法 */
    const offCreate = Events.On('notification.created', (event: NotificationCreatedEvent) => {
      const data = Array.isArray(event.data) ? event.data[0] : event.data // 数据
      if (data) prepend(data)
    })
    /** 订阅 notification.resolved 并保存解绑方法 */
    const offResolve = Events.On('notification.resolved', (event: NotificationResolvedEvent) => {
      const data = Array.isArray(event.data) ? event.data[0] : event.data // 数据
      if (data) remove(data.id)
    })

    return () => {
      offCreate()
      offResolve()
    }
  }, [prepend, remove, setItems])
}
