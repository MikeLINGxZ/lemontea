import { useTranslation } from 'react-i18next'
import type { CliInstallItem, CliInstallPhase } from '@/types/cliInstall'

/** 安装阶段对应的进度刻度，按阶段推进 */
const phaseProgress: Record<CliInstallPhase, number> = {
  pending: 10,
  downloading: 30,
  generating: 60,
  registering: 75,
  installed: 80,
  initializing: 85,
  waiting_auth: 90,
  verifying: 95,
  done: 100,
  failed: 0,
}

/** 显示 CLI 安装的当前阶段和进度条，接收安装记录，返回组件内容 */
export function CliInstallProgress({ install }: { /** 安装记录 */ install: CliInstallItem }) {
  const { t } = useTranslation() // 文案翻译方法
  const label = t(`settingsPage.plugins.cliInstallPhase.${install.phase}`) // 当前阶段的可读名称

  return (
    <div className="mt-2 space-y-1.5">
      <div role="status" className="text-xs text-muted-foreground">{label}</div>
      <progress
        aria-label={label}
        aria-valuetext={label}
        max={100}
        value={phaseProgress[install.phase]}
        className="block h-1.5 w-full appearance-none overflow-hidden rounded-full bg-muted [&::-webkit-progress-bar]:bg-muted [&::-webkit-progress-value]:rounded-full [&::-webkit-progress-value]:bg-primary [&::-webkit-progress-value]:transition-all [&::-moz-progress-bar]:bg-primary"
      />
      {install.detail && <div className="break-words text-xs text-muted-foreground">{install.detail}</div>}
    </div>
  )
}
