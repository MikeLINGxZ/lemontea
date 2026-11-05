/** CLI安装阶段的可选值 */
export type CliInstallPhase =
  | 'pending'
  | 'downloading'
  | 'installed'
  | 'generating'
  | 'initializing'
  | 'waiting_auth'
  | 'verifying'
  | 'done'
  | 'failed'

/** 定义CLI安装条目的数据结构 */
export type CliInstallItem = {
  /** npm包名 */
  npm_package: string
  /** 名称 */
  name: string
  /** 阶段 */
  phase: CliInstallPhase
  /** 详情 */
  detail: string
  /** 扩展编号 */
  extension_id?: string
  /** 会话编号 */
  session_id?: number
  /** 操作地址 */
  action_url?: string
  /** 操作标签 */
  action_label?: string
  /** 过期时间 */
  expires_at?: string
}
