import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Events } from '@wailsio/runtime'
import { Onboarding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/onboarding/index'
import { Runtime } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/runtime/index'
import { cn } from '@/lib/utils'

/** 阶段的可选值 */
type Phase = 'download' | 'extract' | 'verify'

/** 运行环境视图的状态数据 */
type RuntimeViewState =
  | { /* 类别标识 */ kind: 'idle' }
  | { /* 类别标识 */ kind: 'downloading'; /* 阶段 */ phase: Phase; /* 已接收数量 */ received: number; /* 总量 */ total: number; /* 百分比 */ percent: number }
  | { /* 类别标识 */ kind: 'ready'; /* 版本 */ version: string; /* 安装目录 */ installDir: string }
  | { /* 类别标识 */ kind: 'failed'; /* 消息 */ message: string }

/** 定义进度事件数据的数据结构 */
type ProgressEventData = {
  /** 阶段 */
  phase: Phase
  /** 已接收数量 */
  received: number
  /** 总量 */
  total: number
  /** 百分比 */
  percent: number
}

/** 首次引导步骤运行环境组件接收的属性 */
type Props = {
  /** 处理返回事件，无返回值 */
  onBack: () => void
}

/** 展示运行环境下载进度，并处理安装、取消及稍后继续，接收组件输入属性，返回组件内容 */
export function OnboardingStepRuntime(props: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [state, setState] = useState<RuntimeViewState>({ kind: 'idle' }) // 当前状态及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    let cancelled = false // 已取消

    Runtime.GetStatus({})
      .then((res) => {
        if (cancelled || !res) {
          return
        }
        if (res.state === 'ready') {
          setState({ kind: 'ready', version: res.version, installDir: res.install_dir })
          return
        }
        if (res.state === 'downloading') {
          setState({ kind: 'downloading', phase: 'download', received: 0, total: 0, percent: 0 })
          return
        }
        if (res.state === 'failed') {
          setState({ kind: 'failed', message: res.error_msg || '' })
          return
        }
        setState({ kind: 'idle' })
      })
      .catch(() => undefined)

    /** 订阅 runtime.node.progress 并保存解绑方法 */
    const off = Events.On('runtime.node.progress', (evt: { /* 事件携带的数据 */ data: ProgressEventData }) => {
      const progress = evt.data // 进度
      /** 基于当前状态合并变化 */
      setState((prev) => {
        if (prev.kind === 'ready' || prev.kind === 'failed') {
          return prev
        }
        return {
          kind: 'downloading',
          phase: progress.phase,
          received: progress.received,
          total: progress.total,
          percent: progress.percent,
        }
      })

      if (progress.phase === 'verify' && progress.percent === 100) {
        Runtime.GetStatus({})
          .then((res) => {
            if (!res) {
              return
            }
            if (res.state === 'ready') {
              setState({ kind: 'ready', version: res.version, installDir: res.install_dir })
            } else if (res.state === 'failed') {
              setState({ kind: 'failed', message: res.error_msg || '' })
            }
          })
          .catch(() => undefined)
      }
    })

    return () => {
      cancelled = true
      off()
    }
  }, [])

  /** 启动下载，无返回值 */
  const startDownload = () => {
    setState({ kind: 'downloading', phase: 'download', received: 0, total: 0, percent: 0 })
    void Runtime.DownloadNode({})
  }

  /** 下载稍后处理，完成后结束异步任务 */
  const downloadLater = async () => {
    await Runtime.MarkDownloadLater({})
    await Onboarding.EnterHome({})
  }

  /** 取消下载，无返回值 */
  const cancelDownload = () => {
    void Runtime.CancelDownload({})
    setState({ kind: 'idle' })
  }

  /** 进入主界面，无返回值 */
  const enterHome = () => {
    void Onboarding.EnterHome({})
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">{t('onboarding.runtime.title')}</h2>
        <p className="text-xs text-muted-foreground">{t('onboarding.runtime.intro')}</p>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto rounded-2xl border border-border bg-card p-6">
        {state.kind === 'idle' && (
          <div className="flex flex-col items-center justify-center gap-3 py-8 text-center">
            <div className="text-sm text-muted-foreground">{t('onboarding.runtime.stateMissing')}</div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={downloadLater}
                className="rounded-xl border border-border px-5 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent"
              >
                {t('onboarding.actions.downloadLater')}
              </button>
              <button
                type="button"
                onClick={startDownload}
                className="rounded-xl bg-primary px-5 py-2 text-sm font-medium text-primary-foreground"
              >
                {t('onboarding.actions.downloadNow')}
              </button>
            </div>
          </div>
        )}

        {state.kind === 'downloading' && (
          <div className="space-y-3">
            <div className="text-sm font-medium">{t(`onboarding.runtime.phase${capitalize(state.phase)}`)}</div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className={cn('h-full bg-primary transition-all')}
                style={{ width: `${state.percent}%` }}
              />
            </div>
            <div className="text-xs text-muted-foreground">
              {formatBytes(state.received)} / {state.total > 0 ? formatBytes(state.total) : '…'} ({state.percent}%)
            </div>
            <div className="pt-2">
              <button
                type="button"
                onClick={cancelDownload}
                className="rounded-xl border border-border px-4 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-accent"
              >
                {t('onboarding.actions.cancel')}
              </button>
            </div>
          </div>
        )}

        {state.kind === 'ready' && (
          <div className="flex flex-col items-center justify-center gap-3 py-8 text-center">
            <div className="text-sm font-semibold text-primary">{t('onboarding.runtime.stateReady')}</div>
            <div className="text-xs text-muted-foreground">{state.version}</div>
            <div className="text-xs text-muted-foreground/70">{state.installDir}</div>
          </div>
        )}

        {state.kind === 'failed' && (
          <div className="flex flex-col items-center justify-center gap-3 py-8 text-center">
            <div className="text-sm font-semibold text-destructive">{t('onboarding.runtime.stateFailed')}</div>
            <div className="text-xs text-muted-foreground">{state.message}</div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={downloadLater}
                className="rounded-xl border border-border px-4 py-1.5 text-xs font-medium text-foreground hover:bg-accent"
              >
                {t('onboarding.actions.skip')}
              </button>
              <button
                type="button"
                onClick={startDownload}
                className="rounded-xl bg-primary px-4 py-1.5 text-xs font-medium text-primary-foreground"
              >
                {t('onboarding.actions.retry')}
              </button>
            </div>
          </div>
        )}
      </div>

      <div className="flex shrink-0 items-center justify-between border-t border-border pt-4">
        <button
          type="button"
          disabled={state.kind === 'downloading'}
          onClick={props.onBack}
          className="rounded-xl border border-border px-5 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:opacity-40"
        >
          {t('onboarding.actions.back')}
        </button>
        <button
          type="button"
          disabled={state.kind === 'downloading'}
          onClick={enterHome}
          className="rounded-xl bg-primary px-5 py-2 text-sm font-medium text-primary-foreground transition-opacity disabled:opacity-40"
        >
          {t('onboarding.actions.enterHome')}
        </button>
      </div>
    </div>
  )
}

/** 将首字母转为大写处理，接收值，返回处理后的文本 */
function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1)
}

/** 将字节数转换为带单位的显示文本，接收值，返回处理后的文本 */
function formatBytes(value: number): string {
  if (value <= 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB'] // 单位列表
  let unitIndex = 0 // 单位索引
  let current = value // 当前
  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024
    unitIndex += 1
  }
  return `${current.toFixed(1)} ${units[unitIndex]}`
}
