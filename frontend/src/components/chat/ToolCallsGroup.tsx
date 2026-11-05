import { useState } from 'react'
import { ChevronDown, Wrench } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ToolCallBlock } from './ToolCallBlock'
import { InteractiveTerminalBlock } from '@/components/chat/InteractiveTerminalBlock'
import type { DisplayMessage } from '@/types'
import { isObjectRecord } from '@/lib/guards'
import {
  getInteractiveTerminalState,
  isInteractiveTerminalControl,
  type InteractiveTerminalState,
} from '@/lib/terminalOutput'

/** 工具调用分组组件接收的属性 */
type Props = {
  /** 工具列表 */
  tools: DisplayMessage[]
}

/** 将工具调用折叠展示，并单独渲染最近仍可见的交互终端，接收工具列表并返回组件内容 */
export function ToolCallsGroup({ tools }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [expanded, setExpanded] = useState(false) // 展开及其更新方法
  const terminalState = latestInteractiveTerminalState(tools) // 终端状态
  /** 折叠工具列表 */
  const collapsedTools = tools.filter((tool) => !isInteractiveTerminalControl(tool.toolResult ?? ''))

  return (
    <div className="my-0 flex flex-col gap-0">
      {terminalState?.visible && (
        <InteractiveTerminalBlock
          output={terminalState.output}
          active={terminalState.active}
        />
      )}

      {collapsedTools.length > 0 && (
        <div>
          <button
            onClick={() => setExpanded((v) => !v)}
            className="flex w-full min-w-0 items-center gap-1.5 rounded-md py-0.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            <Wrench size={14} className="shrink-0" />
            <span className="shrink-0 text-foreground">
              {t('toolCall.toolsUsed', { count: collapsedTools.length })}
            </span>
            <ChevronDown
              size={14}
              className={expanded ? 'shrink-0 rotate-180 transition-transform' : 'shrink-0 rotate-0 transition-transform'}
            />
            <span className="min-w-0 flex-1" />
          </button>
          {expanded && (
            <div className="ml-1.5 mt-1 space-y-1 border-l border-border pl-3">
              {collapsedTools.map((tool) => {
                /** 工具名称和可展示的参数文本，解析失败时保留原始参数 */
                let toolName = '', args = tool.content
                try {
                  const parsed: unknown = JSON.parse(tool.content) // 待校验的工具调用数据
                  if (isObjectRecord(parsed)) {
                    toolName = typeof parsed.name === 'string' ? parsed.name : ''
                    args = JSON.stringify(parsed.args ?? {}, null, 2)
                  }
                } catch {
                  // 老消息可能直接保存参数文本
                }

                /** 将确认记录转换为工具卡片的展示状态 */
                const status = tool.toolConfirmAction === 'comment' ? 'user_commented'
                  : tool.toolConfirmAction === 'reject' ? 'user_rejected' : 'completed'

                return <ToolCallBlock key={tool.id} toolName={toolName} purpose={tool.extra} args={args}
                  result={tool.toolResult ?? ''} userComment={tool.toolConfirmComment} status={status} />
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

/** 按工具调用顺序查找最后一次终端状态变更，没有终端事件时返回空值 */
function latestInteractiveTerminalState(tools: DisplayMessage[]): InteractiveTerminalState | null {
  let latest: InteractiveTerminalState | null = null // 最新
  for (/* 工具 */ const tool of tools) {
    const state = getInteractiveTerminalState(tool.toolResult ?? '') // 当前状态
    if (state) latest = state
  }
  return latest
}
