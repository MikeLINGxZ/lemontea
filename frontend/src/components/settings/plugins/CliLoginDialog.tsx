import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Events } from '@wailsio/runtime'
import { Plugin as PluginBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin'
import { ResizeLoginCliInput, SendLoginStdinInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto/models'
import 'xterm/css/xterm.css'

/** 登录输出事件的数据 */
type LoginOutputEvent = {
  /** 事件携带的数据 */
  data: {
    /** 当前条目的唯一编号 */
    id: string
    /** 事件携带的数据 */
    data: string
  }
}

/** 登录完成事件的数据 */
type LoginDoneEvent = {
  /** 事件携带的数据 */
  data: {
    /** 当前条目的唯一编号 */
    id: string
    /** 退出代码 */
    exit_code: number
    /** 错误 */
    error: string
  }
}

/** 完成的状态数据 */
type DoneState = {
  /** 退出代码 */
  exitCode: number
  /** 错误 */
  error: string
}

/** CLI登录对话框组件接收的属性 */
export type CliLoginDialogProps = {
  /** 展开 */
  open: boolean
  /** 扩展编号 */
  extensionId: string
  /** 标题 */
  title: string
  /** 处理开始事件，完成后结束异步任务 */
  onStart: () => Promise<void>
  /** 处理关闭事件，无返回值 */
  onClose: () => void
  /** 处理取消事件，无返回值 */
  onCancel: () => void
  /** 处理完成事件，接收退出代码，无返回值 */
  onDone?: (exitCode: number) => void
  /** 处理开始错误事件，接收错误，无返回值 */
  onStartError?: (error: unknown) => void
}

/** 展示交互式 CLI 登录终端，转发终端输入、输出、尺寸和退出状态，接收组件输入属性，返回组件内容 */
export function CliLoginDialog(props: CliLoginDialogProps) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const terminalHostRef = useRef<HTMLDivElement | null>(null) // 终端宿主引用，供后续读取当前值
  const onStartRef = useRef(props.onStart) // on开始引用，供后续读取当前值
  const onCloseRef = useRef(props.onClose) // on关闭引用，供后续读取当前值
  const onDoneRef = useRef(props.onDone) // on完成引用，供后续读取当前值
  const onStartErrorRef = useRef(props.onStartError) // on开始错误引用，供后续读取当前值
  const [doneState, setDoneState] = useState<DoneState | null>(null) // 完成状态及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    onStartRef.current = props.onStart
    onCloseRef.current = props.onClose
    onDoneRef.current = props.onDone
    onStartErrorRef.current = props.onStartError
  }, [props.onClose, props.onDone, props.onStart, props.onStartError])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!props.open || !terminalHostRef.current) {
      return
    }

    let disposed = false // 已销毁
    let terminalDispose: (() => void) | null = null // 终端销毁
    let doneTimer: number | null = null // 完成定时器
    let terminalWriter: ((data: string) => void) | null = null // 将收到的输出数据写入终端
    const pendingOutput: string[] = [] // 等待终端初始化后写入的输出

    /** 订阅 cli.login.output 并保存解绑方法 */
    const offOutput = Events.On('cli.login.output', (event: LoginOutputEvent) => {
      if (event.data.id !== props.extensionId) {
        return
      }
      if (terminalWriter) {
        terminalWriter(event.data.data)
        return
      }
      pendingOutput.push(event.data.data)
    })

    /** 订阅 cli.login.done 并保存解绑方法 */
    const offDone = Events.On('cli.login.done', (event: LoginDoneEvent) => {
      if (event.data.id !== props.extensionId) {
        return
      }
      setDoneState({
        exitCode: event.data.exit_code,
        error: event.data.error,
      })
      onDoneRef.current?.(event.data.exit_code)
      doneTimer = window.setTimeout(() => {
        onCloseRef.current()
      }, 3000)
    })

    /** 初始化终端处理，完成后结束异步任务 */
    const setupTerminal = async (): Promise<void> => {
      /** 保存终端、适配扩展模块、网页链接扩展模块 */
      const [{ Terminal }, { FitAddon }, { WebLinksAddon }] = await Promise.all([
        import('xterm'),
        import('xterm-addon-fit'),
        import('xterm-addon-web-links'),
      ])

      if (disposed || !terminalHostRef.current) {
        return
      }

      /** 终端 */
      const terminal = new Terminal({
        convertEol: true,
        fontFamily: 'Menlo, Consolas, monospace',
        fontSize: 13,
        theme: {
          background: '#1b2636',
          foreground: '#f8fafc',
        },
      })
      const fitAddon = new FitAddon() // 适配扩展模块
      const webLinksAddon = new WebLinksAddon() // 网页链接扩展模块

      terminal.open(terminalHostRef.current)
      terminal.loadAddon(fitAddon)
      terminal.loadAddon(webLinksAddon)
      fitAddon.fit()
      terminal.focus()
      terminalWriter = (data: string) => terminal.write(data)

      for (/* 分片 */ const chunk of pendingOutput.splice(0)) {
        terminal.write(chunk)
      }

      /** 通知调整尺寸，无返回值 */
      const emitResize = (): void => {
        void Promise.resolve(
          PluginBinding.ResizeLoginCli(new ResizeLoginCliInput({
            id: props.extensionId,
            rows: Math.max(terminal.rows, 1),
            cols: Math.max(terminal.cols, 1),
          })),
        ).catch(() => undefined)
      }

      emitResize()

      /** 数据订阅 */
      const dataSubscription = terminal.onData((data: string) => {
        void Promise.resolve(
          PluginBinding.SendLoginStdin(new SendLoginStdinInput({
            id: props.extensionId,
            data,
          })),
        ).catch(() => undefined)
      })

      let resizeObserver: ResizeObserver | null = null // 调整尺寸观察器
      if (typeof ResizeObserver !== 'undefined' && terminalHostRef.current) {
        resizeObserver = new ResizeObserver(() => {
          fitAddon.fit()
          emitResize()
        })
        resizeObserver.observe(terminalHostRef.current)
      }

      await onStartRef.current().catch((error: unknown) => {
        onStartErrorRef.current?.(error)
        onCloseRef.current()
      })

      terminalDispose = () => {
        if (doneTimer !== null) {
          window.clearTimeout(doneTimer)
        }
        terminalWriter = null
        dataSubscription.dispose()
        resizeObserver?.disconnect()
        terminal.dispose()
      }
    }

    void setupTerminal()

    return () => {
      disposed = true
      offOutput()
      offDone()
      terminalDispose?.()
    }
  }, [props.extensionId, props.open])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!props.open) {
      setDoneState(null)
    }
  }, [props.open])

  if (!props.open) {
    return null
  }

  const doneSuccess = doneState?.exitCode === 0 // 完成成功

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        aria-label={t('settingsPage.providers.cancel')}
        className="absolute inset-0 bg-black/45"
        onClick={props.onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="cli-login-dialog-title"
        className="relative z-10 flex h-[min(80vh,720px)] w-full max-w-4xl flex-col overflow-hidden rounded-3xl border border-border bg-background shadow-2xl"
      >
        <div className="flex items-center justify-between border-b border-border px-5 py-4">
          <h3 id="cli-login-dialog-title" className="text-base font-semibold text-foreground">
            {props.title}
          </h3>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={props.onCancel}
              className="rounded-xl border border-border px-3 py-1.5 text-sm text-foreground transition-colors hover:bg-accent"
            >
              {t('settingsPage.plugins.cliLoginCancel')}
            </button>
            <button
              type="button"
              onClick={props.onClose}
              className="rounded-xl border border-border px-3 py-1.5 text-sm text-foreground transition-colors hover:bg-accent"
            >
              {t('settingsPage.plugins.cliLoginClose')}
            </button>
          </div>
        </div>
        {doneState ? (
          <div className={doneSuccess
            ? 'border-b border-emerald-200 bg-emerald-50 px-5 py-3 text-sm text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300'
            : 'border-b border-red-200 bg-red-50 px-5 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300'}
          >
            {doneSuccess
              ? t('settingsPage.plugins.cliLoginDoneSuccess')
              : t('settingsPage.plugins.cliLoginDoneFailure', {
                code: doneState.exitCode,
                detail: doneState.error || t('settingsPage.plugins.cliLoginDoneFailureUnknown'),
              })}
          </div>
        ) : null}
        <div className="min-h-0 flex-1 bg-slate-950 px-3 py-3">
          <div ref={terminalHostRef} className="h-full w-full overflow-hidden rounded-2xl" />
        </div>
      </div>
    </div>
  )
}
