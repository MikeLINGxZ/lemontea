import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Events } from '@wailsio/runtime'
import { Terminal as TerminalBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/terminal'
import type { Info } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal/models'
import { InteractiveTerminalBlock } from '@/components/chat/InteractiveTerminalBlock'

/** 定义终端视图的数据结构 */
type TerminalView = Info & {
  /** 输出 */
  output: string
  /** 游标 */
  cursor: number
}

/** 输出事件的数据 */
type OutputEvent = {
  /** 终端编号 */
  terminal_id: string
  /** 会话编号 */
  session_id: number
  /** 游标 */
  cursor: number
  /** 事件携带的数据 */
  data: string
  /** 状态 */
  status: string
  /** 可见 */
  visible?: boolean
}

/** 退出事件的数据 */
type ExitEvent = {
  /** 终端编号 */
  terminal_id: string
  /** 会话编号 */
  session_id: number
  /** 状态 */
  status: string
  /** 退出代码 */
  exit_code: number
  /** 可见 */
  visible?: boolean
}

/** 终端会话面板组件接收的属性 */
type Props = {
  /** 会话编号 */
  sessionId: number | null
}

/** 按终端会话显示输出面板并转发输入和尺寸变化，接收会话编号，返回组件内容 */
export function TerminalSessionPanels({ sessionId }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [terminals, setTerminals] = useState<Record<string, TerminalView>>({}) // 终端列表及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!sessionId) {
      setTerminals({})
      return
    }

    let cancelled = false // 已取消
    void TerminalBinding.ListTerminals({ session_id: sessionId }).then(async (result) => {
      if (cancelled) return
      const visible = (result?.items ?? []).filter((item) => item.visible) // 可见
      /** 保存条目列表 */
      const entries = await Promise.all(visible.map(async (item) => {
        /** 保存输出 */
        const output = await TerminalBinding.ReadTerminalOutput({
          terminal_id: item.id,
          cursor: 0,
        })
        const text = (output?.chunks ?? []).map((chunk) => chunk.data).join('') // 文本
        /** 游标 */
        const cursor = (output?.chunks ?? []).reduce((max, chunk) => Math.max(max, chunk.cursor_end), 0)
        return [item.id, { ...item, output: text, cursor }] as const
      }))
      if (!cancelled) {
        setTerminals(Object.fromEntries(entries))
      }
    })

    return () => {
      cancelled = true
    }
  }, [sessionId])

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!sessionId) return

    /** 订阅 terminal.output 并保存解绑方法 */
    const offOutput = Events.On('terminal.output', (event: { /* 事件携带的数据 */ data: OutputEvent | OutputEvent[] }) => {
      const items = Array.isArray(event.data) ? event.data : [event.data] // 条目列表
      setTerminals((previous) => {
        let next = previous // 更新后的值
        for (/* 当前条目 */ const item of items) {
          if (!item || item.session_id !== sessionId) continue
          const current = next[item.terminal_id] // 当前
          if (!current) {
            if (item.visible !== true) continue
            next = {
              ...next,
              [item.terminal_id]: {
                id: item.terminal_id,
                session_id: item.session_id,
                title: t('chat.interactiveTerminal'),
                command: '',
                args: '',
                cwd: '',
                status: item.status || 'active',
                visible: item.visible,
                pid: 0,
                current_cursor: item.cursor,
                output: item.data,
                cursor: item.cursor,
              },
            }
            continue
          }
          if (item.cursor <= current.cursor) continue
          next = {
            ...next,
            [item.terminal_id]: {
              ...current,
              output: current.output + item.data,
              cursor: item.cursor,
              status: item.status || current.status,
              visible: item.visible ?? current.visible,
            },
          }
        }
        return next
      })
    })

    /** 订阅 terminal.exited 并保存解绑方法 */
    const offExit = Events.On('terminal.exited', (event: { /* 事件携带的数据 */ data: ExitEvent | ExitEvent[] }) => {
      const items = Array.isArray(event.data) ? event.data : [event.data] // 条目列表
      setTerminals((previous) => {
        let next = previous // 更新后的值
        for (/* 当前条目 */ const item of items) {
          if (!item || item.session_id !== sessionId || !next[item.terminal_id]) continue
          next = {
            ...next,
            [item.terminal_id]: {
              ...next[item.terminal_id],
              status: item.status,
              exit_code: item.exit_code,
              visible: item.visible ?? next[item.terminal_id].visible,
            },
          }
        }
        return next
      })
    })

    return () => {
      offOutput()
      offExit()
    }
  }, [sessionId, t])

  /** 已排序 */
  const ordered = useMemo(
    () => Object.values(terminals).filter((item) => item.visible),
    [terminals]
  )

  if (ordered.length === 0) return null

  return (
    <div className="flex flex-col gap-2">
      {ordered.map((terminal) => (
        <InteractiveTerminalBlock
          key={terminal.id}
          output={terminal.output}
          active={terminal.status === 'active' || terminal.status === 'waiting'}
          onInput={(data) => {
            void TerminalBinding.WriteTerminalInput({
              terminal_id: terminal.id,
              data,
            })
          }}
          onResize={(rows, cols) => {
            void TerminalBinding.ResizeTerminal({
              terminal_id: terminal.id,
              rows,
              cols,
            })
          }}
        />
      ))}
    </div>
  )
}
