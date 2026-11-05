import { useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { useCliInstallStore } from '@/store/cliInstallStore'
import type { CliInstallItem } from '@/types/cliInstall'

/** 进度事件的数据 */
type ProgressEvent = {
  /** 从设置页提交的安装请求标识 */
  request_id?: string
  /** 会话编号 */
  session_id?: number
  /** npm包名 */
  npm_package: string
  /** 名称 */
  name: string
  /** 阶段 */
  phase: CliInstallItem['phase']
  /** 详情 */
  detail: string
  /** 当前条目的唯一编号 */
  id?: string
  /** 扩展编号 */
  extension_id?: string
  /** 操作地址 */
  action_url?: string
  /** 操作标签 */
  action_label?: string
  /** 过期时间 */
  expires_at?: string
}

/** 完成事件的数据 */
type DoneEvent = {
  /** 扩展 */
  extension: { /* 当前条目的唯一编号 */ id: string; /* 名称 */ name: string }
}

/** 订阅 CLI 安装进度和完成事件并更新任务仓库，无返回值 */
export function useCliInstallSubscription() {
  const upsert = useCliInstallStore((s) => s.upsert) // 从状态仓库读取新增或更新
  const remove = useCliInstallStore((s) => s.remove) // 从状态仓库读取移除

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    /** 订阅 cli.install.progress 并保存解绑方法 */
    const offProgress = Events.On('cli.install.progress', (event: { /* 事件携带的数据 */ data: ProgressEvent }) => {
      const d = event.data // 当前处理位置的值
      // 设置页直接安装由 RPC 完成，忽略迟到的阶段事件
      if (d.request_id && !useCliInstallStore.getState().items.some(
        (item) => item.request_id === d.request_id && item.phase !== 'failed',
      )) return

      const extensionId = d.extension_id || d.id // 已登记的扩展编号
      upsert({
        request_id: d.request_id,
        npm_package: d.npm_package ?? d.id ?? '',
        name: d.name ?? '',
        phase: d.phase,
        detail: d.detail ?? '',
        ...(extensionId ? { extension_id: extensionId } : {}),
        session_id: d.session_id,
        action_url: d.action_url,
        action_label: d.action_label,
        expires_at: d.expires_at,
      })
      if (d.phase === 'done') {
        setTimeout(() => remove(d.session_id ?? d.extension_id ?? d.npm_package), 2000)
      }
    })
    /** 订阅 cli.install.done 并保存解绑方法 */
    const offDone = Events.On('cli.install.done', (event: { /* 事件携带的数据 */ data: DoneEvent }) => {
      const ext = event.data.extension // 扩展名
      if (ext?.name) {
        setTimeout(() => remove(ext.id ?? ext.name), 2000)
      }
    })
    return () => {
      offProgress()
      offDone()
    }
  }, [upsert, remove])
}
