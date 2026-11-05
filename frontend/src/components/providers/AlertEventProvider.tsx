import { useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { APP_ALERT_EVENT, normalizeAlertEventPayload } from '@/alert/event'
import { useAlertStore } from '@/alert/store'

/** 提醒事件模型服务商组件接收的属性 */
type AlertEventProviderProps = {
  /** 要渲染的子内容 */
  children: React.ReactNode
}

/** 监听 Wails 提醒事件并将有效事件加入提醒仓库，接收要渲染的子内容，返回组件内容 */
export function AlertEventProvider({ children }: AlertEventProviderProps) {
  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 保存事件解绑方法 */
    const off = Events.On(APP_ALERT_EVENT, (event: { /* 事件携带的数据 */ data: unknown }) => {
      const payload = normalizeAlertEventPayload(event.data) // 载荷
      if (!payload) {
        return
      }

      useAlertStore.getState().pushAlert(payload)
    })

    return () => off()
  }, [])

  return <>{children}</>
}
