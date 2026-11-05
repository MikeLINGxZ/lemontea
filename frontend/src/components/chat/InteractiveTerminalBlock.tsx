import { useEffect, useRef } from 'react'
import { ScanLine, Terminal } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import 'xterm/css/xterm.css'

/** 交互式终端区块组件接收的属性 */
type Props = {
  /** 输出 */
  output: string
  /** 当前 */
  active?: boolean
  /** 处理输入事件，接收数据，无返回值 */
  onInput?: (data: string) => void
  /** 处理调整尺寸事件，接收行数、列数，无返回值 */
  onResize?: (rows: number, cols: number) => void
}

/** 渲染支持 ANSI 的终端，按差量写入输出并同步输入和尺寸，接收输出、当前、on输入、on调整尺寸，返回组件内容 */
export function InteractiveTerminalBlock({ output, active = false, onInput, onResize }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const terminalHostRef = useRef<HTMLDivElement | null>(null) // 终端宿主引用，供后续读取当前值
  const terminalWriterRef = useRef<((data: string) => void) | null>(null) // 终端写入方法引用，供后续读取当前值
  const lastOutputRef = useRef('') // 最后一个输出引用，供后续读取当前值
  const pendingOutputRef = useRef(output) // 待处理输出引用，供后续读取当前值
  const activeRef = useRef(active) // 当前引用，供后续读取当前值
  const onInputRef = useRef(onInput) // on输入引用，供后续读取当前值
  const onResizeRef = useRef(onResize) // on调整尺寸引用，供后续读取当前值

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    activeRef.current = active
    onInputRef.current = onInput
    onResizeRef.current = onResize
  }, [active, onInput, onResize])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    const writer = terminalWriterRef.current // 写入方法
    if (!writer) {
      pendingOutputRef.current = output
      return
    }

    if (output.startsWith(lastOutputRef.current)) {
      const nextChunk = output.slice(lastOutputRef.current.length) // 下一项分片
      if (nextChunk) {
        writer(nextChunk)
      }
    } else {
      writer('\x1bc')
      if (output) {
        writer(output)
      }
    }
    lastOutputRef.current = output
  }, [output])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!terminalHostRef.current) return

    let disposed = false // 已销毁
    let cleanup: (() => void) | null = null // 清理

    /** 初始化终端处理，完成后结束异步任务 */
    const setupTerminal = async (): Promise<void> => {
      /** 保存横坐标终端、适配扩展模块、网页链接扩展模块 */
      const [{ Terminal: XTerm }, { FitAddon }, { WebLinksAddon }] = await Promise.all([
        import('xterm'),
        import('xterm-addon-fit'),
        import('xterm-addon-web-links'),
      ])

      if (disposed || !terminalHostRef.current) return

      /** 终端 */
      const terminal = new XTerm({
        convertEol: true,
        cursorBlink: true,
        fontFamily: 'Menlo, Consolas, monospace',
        fontSize: 13,
        scrollback: 5000,
        theme: {
          background: '#fffbeb',
          foreground: '#0f172a',
          cursor: '#92400e',
          selectionBackground: '#fbbf24',
        },
      })
      const fitAddon = new FitAddon() // 适配扩展模块
      const webLinksAddon = new WebLinksAddon() // 网页链接扩展模块

      terminal.open(terminalHostRef.current)
      terminal.loadAddon(fitAddon)
      terminal.loadAddon(webLinksAddon)
      fitAddon.fit()
      terminalWriterRef.current = (data: string) => terminal.write(data)

      if (pendingOutputRef.current) {
        terminal.write(pendingOutputRef.current)
        lastOutputRef.current = pendingOutputRef.current
      }

      /** 通知调整尺寸，无返回值 */
      const emitResize = (): void => {
        onResizeRef.current?.(Math.max(terminal.rows, 1), Math.max(terminal.cols, 1))
      }

      emitResize()

      /** 数据订阅 */
      const dataSubscription = terminal.onData((data: string) => {
        if (activeRef.current) {
          onInputRef.current?.(data)
        }
      })

      let resizeObserver: ResizeObserver | null = null // 调整尺寸观察器
      if (typeof ResizeObserver !== 'undefined' && terminalHostRef.current) {
        resizeObserver = new ResizeObserver(() => {
          fitAddon.fit()
          emitResize()
        })
        resizeObserver.observe(terminalHostRef.current)
      }

      if (activeRef.current) {
        terminal.focus()
      }

      cleanup = () => {
        terminalWriterRef.current = null
        dataSubscription.dispose()
        resizeObserver?.disconnect()
        terminal.dispose()
      }
    }

    void setupTerminal()

    return () => {
      disposed = true
      cleanup?.()
    }
  }, [])

  return (
    <div className="my-1 overflow-hidden rounded-lg border border-amber-300 bg-amber-50/80 shadow-xs dark:border-amber-500/50 dark:bg-amber-950/20">
      <div className="flex items-center gap-2 border-b border-amber-200 px-3 py-2 text-xs text-amber-900 dark:border-amber-500/40 dark:text-amber-100">
        <ScanLine size={14} />
        <span className="font-medium">
          {t(active ? 'chat.interactiveTerminalActive' : 'chat.interactiveTerminal')}
        </span>
        <Terminal size={12} className="ml-auto opacity-70" />
      </div>
      <span className="sr-only">{stripAnsiForAccessibleText(output)}</span>
      <div
        ref={terminalHostRef}
        role="textbox"
        aria-label={t('chat.interactiveTerminalInput')}
        aria-readonly={!active}
        className="h-[280px] max-h-[420px] w-full overflow-hidden px-3 py-2 [&_.xterm]:h-full [&_.xterm-viewport]:bg-transparent [&_.xterm-screen]:bg-transparent"
      />
    </div>
  )
}

/** 移除ANSI 对应 无障碍文本文本，接收输出，返回处理后的文本 */
function stripAnsiForAccessibleText(output: string): string {
  return output.replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, '')
}
