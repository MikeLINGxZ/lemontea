import { useState, useEffect, useRef } from 'react'
import type { CSSProperties } from 'react'
import { Events } from '@wailsio/runtime'
import { useEditor, EditorContent } from '@tiptap/react'
import { Extension, InputRule } from '@tiptap/core'
import type { Editor, JSONContent } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { TextSelection } from '@tiptap/pm/state'

import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Suggestion from '@tiptap/suggestion'
import { Markdown } from 'tiptap-markdown'
import { buildSlashSuggestion } from './slashSuggestion'
import {
  Send, Square, Paperclip, Wrench, ChevronDown, Check, Search,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { useChatStore } from '@/store/chatStore'
import { useAppStore } from '@/store/appStore'
import { Provider as ProviderBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import { Plugin as PluginBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import type { ProviderModel } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model/models'
import type { Attachment } from '@/types'
import { useShallow } from 'zustand/react/shallow'
import { File as FileBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file'
import { SelectFileInput, SaveTempFileInput } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file/file_dto'
import { AttachmentChips } from './AttachmentChips'
import { ATTACHMENT_MAX_COUNT, ATTACHMENT_MAX_BYTES, inferAttachmentMeta } from '@/lib/attachments'
import { isObjectRecord } from '@/lib/guards'

/** 注册斜杠指令弹层的编辑器扩展 */
const SlashCommand = Extension.create({
  name: 'slashCommand',
  /** 注册斜杠指令建议插件，返回对应的处理结果 */
  addProseMirrorPlugins() {
    return [
      Suggestion({
        editor: this.editor,
        ...buildSlashSuggestion(),
        /** 命令处理，接收编辑器、范围、组件输入属性，无返回值 */
        command: ({ editor, range, props }: { /* 编辑器 */ editor: Editor; /* 范围 */ range: { /* 起点 */ from: number; /* 终点 */ to: number }; /* 组件输入属性 */ props: { /* 名称 */ name: string } }) => {
          editor.chain().focus().deleteRange(range).insertContent(`/${props.name} `).run()
        },
      }),
    ]
  },
})

/** 定义聊天工具选项的数据结构 */
type ChatToolOption = {
  /** 当前条目的唯一编号 */
  id: string
  /** 名称 */
  name: string
  /** 描述 */
  description: string
  /** 分类 */
  category: string
  /** 是否启用 */
  enabled: boolean
}

/** 内置工具名称和说明的国际化键 */
const TOOL_I18N_KEYS: Record<string, { /* 名称 */ name: string; /* 描述 */ description: string }> = {
  datetime: {
    name: 'input.toolNames.datetime.name',
    description: 'input.toolNames.datetime.description',
  },
  file_read: {
    name: 'input.toolNames.fileRead.name',
    description: 'input.toolNames.fileRead.description',
  },
  file_write: {
    name: 'input.toolNames.fileWrite.name',
    description: 'input.toolNames.fileWrite.description',
  },
  shell: {
    name: 'input.toolNames.shell.name',
    description: 'input.toolNames.shell.description',
  },
  web_fetch: {
    name: 'input.toolNames.webFetch.name',
    description: 'input.toolNames.webFetch.description',
  },
  web_search: {
    name: 'input.toolNames.webSearch.name',
    description: 'input.toolNames.webSearch.description',
  },
  code_exec: {
    name: 'input.toolNames.codeExec.name',
    description: 'input.toolNames.codeExec.description',
  },
}

/** 将二进制文件转换为 Base64 编码，接收二进制对象，返回处理后的文本 */
async function blobToBase64(blob: Blob): Promise<string> {
  const buf = await blob.arrayBuffer() // 保存文件的二进制缓冲区
  const bytes = new Uint8Array(buf) // 字节
  let binary = '' // 二进制文本
  for (/* 循环索引 */ let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i])
  return btoa(binary)
}

/** 将行内反引号转换为代码块的编辑器扩展 */
const SingleLineCodeBlock = Extension.create({
  name: 'singleLineCodeBlock',

  /** 注册行内代码块的输入规则，返回对应的处理结果 */
  addInputRules() {
    return [
      new InputRule({
        find: /^```\s?([^`]+?)\s?```$/,
        /** 响应当前订阅的事件，接收范围、匹配项、编辑器可调用的命令，无返回值 */
        handler: ({ range, match, commands }) => {
          const code = match[1]?.trim() ?? '' // 代码

          commands.insertContentAt(range, {
            type: 'codeBlock',
            content: code ? [{ type: 'text', text: code }] : [],
          })
        },
      }),
    ]
  },
})

/** 将源代码文本包装为编辑器代码块节点，接收代码，返回对应的处理结果 */
const createCodeBlockContent = (code: string): JSONContent => ({
  type: 'codeBlock',
  content: code ? [{ type: 'text', text: code }] : [],
})

/** 读取编辑器节点文本并保留换行，接收节点，返回表达式结果 */
const getNodeTextWithBreaks = (node: ProseMirrorNode) => (
  node.textBetween(0, node.content.size, '\n', '\n')
)

/** 把段落中的 Markdown 围栏转换为编辑器代码块，接收编辑器，返回对应的处理结果 */
const convertFencedCodeBlock = (editor: Editor) => {
  let converted = false // 是否已转换

  editor.state.doc.descendants((node, pos) => {
    if (converted || node.type.name !== 'paragraph') return false

    const match = getNodeTextWithBreaks(node).match(/^```[^\n]*\n([\s\S]*?)\n```$/) // 匹配项
    if (!match) return true

    editor.commands.insertContentAt(
      { from: pos, to: pos + node.nodeSize },
      createCodeBlockContent(match[1].trim())
    )
    converted = true

    return false
  })

  if (converted) return true

  /** 顶部级别节点列表 */
  const topLevelNodes: Array<{
    /** 节点 */
    node: ProseMirrorNode
    /** 字符位置 */
    pos: number
  }> = []

  /** 逐项处理列表内容 */
  editor.state.doc.forEach((node, offset) => {
    topLevelNodes.push({ node, pos: offset + 1 })
  })

  for (/* 开始 */ let start = 0; start < topLevelNodes.length; start += 1) {
    const startText = getNodeTextWithBreaks(topLevelNodes[start].node).trim() // 开始文本
    if (topLevelNodes[start].node.type.name !== 'paragraph' || !/^```[^\n]*$/.test(startText)) {
      continue
    }

    for (/* 结束 */ let end = start + 1; end < topLevelNodes.length; end += 1) {
      const endText = getNodeTextWithBreaks(topLevelNodes[end].node).trim() // 结束文本

      if (topLevelNodes[end].node.type.name !== 'paragraph' || endText !== '```') continue

      /** 代码 */
      const code = topLevelNodes
        .slice(start + 1, end)
        .map(item => getNodeTextWithBreaks(item.node))
        .join('\n')
        .trim()

      editor.commands.insertContentAt(
        { from: topLevelNodes[start].pos, to: topLevelNodes[end].pos + topLevelNodes[end].node.nodeSize },
        createCodeBlockContent(code)
      )

      return true
    }
  }

  return false
}

/** 聊天输入组件接收的属性 */
type ChatInputProps = {
  /** 会话编号 */
  sessionId?: number
  /** 展示变体 */
  variant?: 'default' | 'quick'
}

/** 选择初始模型编号，接收模型服务商模型列表，返回处理结果，无法取得时为空 */
export function pickInitialModelId(providerModels: ProviderModel[]): number | null {
  const enabledProviders = providerModels.filter(pm => pm.provider.enabled) // 启用状态服务商列表
  /** 默认模型服务商 */
  const defaultProvider = enabledProviders.find(pm => pm.provider.is_default) ?? enabledProviders[0]
  /** 默认模型 */
  const defaultModel = defaultProvider?.models.find(m => m.is_default && m.enable)
    ?? defaultProvider?.models.find(m => m.enable)
  return defaultModel?.id ?? null
}

/** 快捷no拖动样式 */
const quickNoDragStyle = {
  '--wails-draggable': 'no-drag',
  WebkitAppRegion: 'no-drag',
} as CSSProperties & Record<'--wails-draggable', string>

/** 渲染富文本输入区、模型和工具选择器，处理附件、斜杠指令及发送动作，接收会话编号、展示变体，返回组件内容 */
export function ChatInput({ sessionId, variant = 'default' }: ChatInputProps) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const isQuick = variant === 'quick' // 是否快捷
  /** 从状态仓库读取当前会话编号、是否流式输出、创建会话、发送消息、停止生成、集合当前会话 */
  const {
    currentConversationId,
    isStreaming,
    createConversation,
    sendMessage,
    stopGeneration,
    setCurrentConversation,
  } = useChatStore(useShallow((state) => {
    const resolvedSessionId = sessionId ?? state.currentConversationId // 已处理会话编号
    /** 当前会话 */
    const currentConversation = resolvedSessionId
      ? state.conversations.find((conversation) => conversation.id === resolvedSessionId)
      : undefined
    /** 会话状态 */
    const sessionStatus = resolvedSessionId
      ? (state.sessionStatuses[resolvedSessionId] ?? currentConversation?.status)
      : undefined

    return {
      currentConversationId: state.currentConversationId,
      isStreaming: sessionStatus === 'loading' || sessionStatus === 'waiting-unread',
      createConversation: state.createConversation,
      sendMessage: state.sendMessage,
      stopGeneration: state.stopGeneration,
      setCurrentConversation: state.setCurrentConversation,
    }
  }))
  const [tools, setTools] = useState<ChatToolOption[]>([]) // 工具列表及其更新方法
  const [providerModels, setProviderModels] = useState<ProviderModel[]>([]) // 模型服务商模型列表及其更新方法
  const [selectedModelId, setSelectedModelId] = useState<number | null>(null) // 选中项模型编号及其更新方法
  const [toolsOpen, setToolsOpen] = useState(false) // 工具列表展开及其更新方法
  const [modelOpen, setModelOpen] = useState(false) // 模型展开及其更新方法
  const [modelQuery, setModelQuery] = useState('') // 模型搜索词及其更新方法
  const [isEmpty, setIsEmpty] = useState(true) // 是否空及其更新方法
  const [attachments, setAttachments] = useState<Attachment[]>([]) // 附件列表及其更新方法
  const attachmentsRef = useRef<Attachment[]>([]) // 附件列表引用，供后续读取当前值
  attachmentsRef.current = attachments
  const [isDraggingOver, setIsDraggingOver] = useState(false) // 是否正在拖动经过及其更新方法
  const targetSessionId = sessionId ?? currentConversationId // 目标会话编号

  const enabledToolsCount = tools.filter(x => x.enabled).length // 启用状态工具列表数量

  /** 获取工具分类标签，接收分类，返回对应的处理结果 */
  const getToolCategoryLabel = (category: string) => {
    if (category === 'builtin') return t('input.toolCategories.builtin')
    if (category === 'mcp') return t('input.toolCategories.mcp')
    if (category === 'user') return t('input.toolCategories.user')
    return t('input.toolCategories.other')
  }

  /** 获取工具显示名称，接收工具，返回对应的处理结果 */
  const getToolDisplayName = (tool: ChatToolOption) => {
    const key = TOOL_I18N_KEYS[tool.id] // 键
    return key ? t(key.name) : tool.name
  }

  /** 获取工具显示描述，接收工具，返回对应的处理结果 */
  const getToolDisplayDescription = (tool: ChatToolOption) => {
    const key = TOOL_I18N_KEYS[tool.id] // 键
    return key ? t(key.description) : tool.description
  }

  /** 退出列表终点段落处理，返回对应的处理结果 */
  const exitListToParagraph = () => {
    if (!editor) return false

    const { state, view } = editor // 取出当前状态、视图
    const { $from } = state.selection // 起点
    let listDepth: number | null = null // 列表深度

    for (/* 深度 */ let depth = $from.depth; depth > 0; depth -= 1) {
      const nodeName = $from.node(depth).type.name // 节点名称

      if (nodeName === 'bulletList' || nodeName === 'orderedList') {
        listDepth = depth
        break
      }
    }

    if (listDepth === null) return false

    const insertPos = $from.after(listDepth) // 插入位置
    const paragraph = state.schema.nodes.paragraph.create() // 段落
    const tr = state.tr.insert(insertPos, paragraph) // 当前处理位置的值

    tr.setSelection(TextSelection.near(tr.doc.resolve(insertPos + 1)))
    view.dispatch(tr.scrollIntoView())
    view.focus()

    return true
  }

  /** 编辑器 */
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3] },
      }),
      SingleLineCodeBlock,
      Placeholder.configure({
        placeholder: t('input.placeholder'),
      }),
      Markdown.configure({
        html: false,
        breaks: true,
        transformPastedText: true,
      }),
      SlashCommand,
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'tiptap-input focus:outline-hidden px-4 pt-3 pb-2 min-h-[5.25rem] max-h-48 overflow-y-auto text-foreground',
      },
      /** 处理键向下，接收视图、触发本次回调的事件，返回对应的处理结果 */
      handleKeyDown: (_view, event) => {
        if (event.key === 'Enter' && event.shiftKey) {
          if (editor?.isActive('heading')) {
            event.preventDefault()
            editor.chain().focus().splitBlock().setParagraph().run()
            return true
          }

          if (editor?.isActive('listItem')) {
            event.preventDefault()
            return exitListToParagraph()
          }
        }

        if (event.key === 'Enter' && !event.shiftKey) {
          event.preventDefault()
          handleSend()
          return true
        }
        return false
      },
    },
    /** 处理更新事件，接收编辑器，无返回值 */
    onUpdate: ({ editor }) => {
      if (convertFencedCodeBlock(editor)) {
        setIsEmpty(false)
        return
      }

      setIsEmpty(editor.isEmpty)
    },
  })

  /** 初次加载模型时选中默认项，重新获得焦点时只更新列表 */
  useEffect(() => {
    /** 加载处理，接收是否初始，完成后结束异步任务 */
    const load = async (isInitial: boolean) => {
      try {
        const result = await ProviderBinding.ProviderAndModelList({}) // 保存本次调用的结果
        if (!result?.provider_models) return
        setProviderModels(result.provider_models)
        if (isInitial) {
          setSelectedModelId(pickInitialModelId(result.provider_models))
        }
      } catch {
        // 忽略本次失败并保留当前界面状态
      }
    }
    void load(true)
    /** 处理焦点，返回表达式结果 */
    const handleFocus = () => void load(false)
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  /** 刷新可选工具并恢复用户此前启用的工具集合 */
  useEffect(() => {
    /** 加载工具列表，完成后结束异步任务 */
    const loadTools = async () => {
      try {
        const result = await PluginBinding.ListAvailableTools({}) // 保存本次调用的结果
        if (!result?.tools) return
        const selectableTools = result.tools.filter(tool => tool.category !== 'builtin') // 可选择工具列表
        const persistedIds = useAppStore.getState().enabledToolIds // 持久化编号列表
        setTools(selectableTools.map((tool) => ({
          id: tool.id,
          name: tool.name,
          description: tool.description,
          category: tool.category,
          enabled: persistedIds.includes(tool.id),
        })))
      } catch {
        // 忽略本次失败并保留当前界面状态
      }
    }

    void loadTools()
    /** 处理焦点，无返回值 */
    const handleFocus = () => { void loadTools() }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [])

  /** 切换会话时清空编辑器和附件，避免发送到其他会话 */
  useEffect(() => {
    if (!editor) return
    editor.commands.clearContent()
    setIsEmpty(true)
    setAttachments([])
  }, [targetSessionId, editor])

  /** 监听原生拖放事件，独立管理拖放外观和可接收的附件路径 */
  useEffect(() => {
    /** 保存事件解绑方法 */
    const offEnter = Events.On(Events.Types.Mac.WindowFileDraggingEntered, () => {
      setIsDraggingOver(true)
    })
    /** 保存事件解绑方法 */
    const offExit = Events.On(Events.Types.Mac.WindowFileDraggingExited, () => {
      setIsDraggingOver(false)
    })
    /** 保存事件解绑方法 */
    const offPerformed = Events.On(Events.Types.Mac.WindowFileDraggingPerformed, () => {
      setIsDraggingOver(false)
    })
    /** 订阅 files-dropped 并保存解绑方法 */
    const offDrop = Events.On('files-dropped', (event) => {
      const payload: unknown = event.data // 待校验的原生拖放事件数据
      if (!isObjectRecord(payload) || !Array.isArray(payload.files)) return

      /** 仅接收有效的文件路径字符串 */
      const files = payload.files.filter((file: unknown): file is string => typeof file === 'string')
      /** 当前会话仍可接收的附件数量 */
      const remaining = Math.max(ATTACHMENT_MAX_COUNT - attachmentsRef.current.length, 0)
      const newAtts = files.slice(0, remaining).map(path => inferAttachmentMeta(path)) // 按剩余名额转换成附件数据
      if (newAtts.length > 0) {
        setAttachments(prev => [...prev, ...newAtts])
      }
    })

    return () => {
      offEnter()
      offExit()
      offPerformed()
      offDrop()
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  /** 处理添加附件，完成后结束异步任务 */
  const handleAttach = async () => {
    if (attachments.length >= ATTACHMENT_MAX_COUNT) return
    const result = await FileBinding.SelectFile(new SelectFileInput()) // 保存本次调用的结果
    if (!result?.file_path) return
    setAttachments(prev => [...prev, inferAttachmentMeta(result.file_path)])
  }

  /** 处理发送，完成后结束异步任务 */
  const handleSend = async () => {
    if (!editor || isStreaming) return
    /** 内容 */
    const content = ((editor.storage as { /* Markdown 扩展的序列化接口 */ markdown?: {
      /* 获取编辑器内容对应的 Markdown 文本 */
      getMarkdown?: () => string
    } }).markdown?.getMarkdown?.() ?? '').trim()
    if (!content && attachments.length === 0) return

    /** 选中项模型服务商 */
    const selectedProvider = providerModels.find(pm => pm.models.some(m => m.id === selectedModelId))
    const selectedModel = selectedProvider?.models.find(m => m.id === selectedModelId) // 选中项模型
    if (!selectedProvider || !selectedModel) return

    let nextSessionId = targetSessionId // 下一项会话编号
    if (!nextSessionId) {
      nextSessionId = await createConversation()
      if (!nextSessionId) return
      setCurrentConversation(nextSessionId)
    }

    const outboundAttachments = attachments // 待发送附件列表
    editor.commands.clearContent()
    setIsEmpty(true)
    setAttachments([])

    await sendMessage({
      sessionId: nextSessionId,
      content,
      baseUrl: selectedProvider.provider.base_url,
      apiKey: selectedProvider.provider.api_key,
      modelName: selectedModel.model,
      providerType: selectedProvider.provider.provider_type,
      enabledUserTools: tools.filter(tool => tool.enabled).map(tool => tool.id),
      attachments: outboundAttachments,
    })
  }

  /** 处理停止，无返回值 */
  const handleStop = () => {
    if (targetSessionId) {
      void stopGeneration(targetSessionId)
    }
  }

  /** 处理粘贴，接收触发本次回调的事件，完成后结束异步任务 */
  const handlePaste = async (event: React.ClipboardEvent<HTMLDivElement>) => {
    const items = Array.from(event.clipboardData.items) // 条目列表
    const fileItems = items.filter(item => item.kind === 'file') // 文件条目列表
    if (fileItems.length === 0) return

    event.preventDefault()

    const remaining = ATTACHMENT_MAX_COUNT - attachmentsRef.current.length // 剩余
    /** 终点进程 */
    const toProcess = fileItems.slice(0, remaining).map(item => {
      const blob = item.getAsFile() // 二进制对象
      if (!blob) return null
      if (blob.size > ATTACHMENT_MAX_BYTES) return null
      const name = blob.name || `pasted-${Date.now()}.${item.type.split('/')[1] ?? 'bin'}` // 名称
      return { blob, name, mime: item.type || 'application/octet-stream' }
    }).filter((x): x is NonNullable<typeof x> => x !== null)

    /** 保存结果列表 */
    const results = await Promise.allSettled(
      toProcess.map(async ({ blob, name, mime }) => {
        const data = await blobToBase64(blob) // 保存数据
        /** 保存本次调用的结果 */
        const result = await FileBinding.SaveTempFile(new SaveTempFileInput({ name, data, mime }))
        if (!result?.file_path) throw new Error('no path')
        const att = inferAttachmentMeta(result.file_path) // 当前附件
        return { ...att, name }
      })
    )

    /** 新附件列表 */
    const newAtts = results
      .filter((r): r is PromiseFulfilledResult<Attachment> => r.status === 'fulfilled')
      .map(r => r.value)

    if (newAtts.length > 0) {
      setAttachments(prev => [...prev, ...newAtts])
    }
  }

  /** 处理拖动经过，接收触发本次回调的事件，无返回值 */
  const handleDragOver = (event: React.DragEvent<HTMLDivElement>) => {
    if (!event.dataTransfer.types.includes('Files')) return
    event.preventDefault()
    setIsDraggingOver(true)
  }

  /** 处理拖动离开，接收触发本次回调的事件，无返回值 */
  const handleDragLeave = (event: React.DragEvent<HTMLDivElement>) => {
    if ((event.currentTarget as HTMLDivElement).contains(event.relatedTarget as Node)) return
    setIsDraggingOver(false)
  }

  /** 处理拖放，接收触发本次回调的事件，完成后结束异步任务 */
  const handleDrop = async (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    setIsDraggingOver(false)

    const dropped = Array.from(event.dataTransfer.files) // 已拖放
    const remaining = ATTACHMENT_MAX_COUNT - attachmentsRef.current.length // 剩余
    /** 终点进程 */
    const toProcess = dropped.slice(0, remaining).filter(file => file.size <= ATTACHMENT_MAX_BYTES)

    /** 保存结果列表 */
    const results = await Promise.allSettled(
      toProcess.map(async (file) => {
        const data = await blobToBase64(file) // 保存数据
        /** 保存本次调用的结果 */
        const result = await FileBinding.SaveTempFile(
          new SaveTempFileInput({ name: file.name, data, mime: file.type || 'application/octet-stream' })
        )
        if (!result?.file_path) throw new Error('no path')
        const att = inferAttachmentMeta(result.file_path) // 当前附件
        return { ...att, name: file.name }
      })
    )

    /** 新附件列表 */
    const newAtts = results
      .filter((r): r is PromiseFulfilledResult<Attachment> => r.status === 'fulfilled')
      .map(r => r.value)

    if (newAtts.length > 0) {
      setAttachments(prev => [...prev, ...newAtts])
    }
  }

  /** 关闭模型选择器，无返回值 */
  const closeModelSelector = () => {
    setModelOpen(false)
    setModelQuery('')
  }

  /** 切换工具，接收当前条目的唯一编号，无返回值 */
  const toggleTool = (id: string) => {
    setTools((prev) => {
      const next = prev.map(tl => tl.id === id ? { ...tl, enabled: !tl.enabled } : tl) // 更新后的值
      const enabledIds = next.filter(t => t.enabled).map(t => t.id) // 启用状态编号列表
      useAppStore.getState().setEnabledToolIds(enabledIds)
      return next
    })
  }

  /** 已分组工具列表 */
  const groupedTools = tools.reduce<Record<string, ChatToolOption[]>>((acc, tool) => {
    const key = tool.category || 'other' // 键
    acc[key] = acc[key] ? [...acc[key], tool] : [tool]
    return acc
  }, {})

  /** 启用状态服务商列表 */
  const enabledProviders = providerModels
    .filter(pm => pm.provider.enabled)
    .map(pm => ({
      id: pm.provider.id,
      name: pm.provider.provider_name,
      models: pm.models
        .filter(m => m.enable)
        .map(m => ({ id: m.id, name: m.alias || m.model })),
    }))
    .filter(pm => pm.models.length > 0)

  /** 选中项模型对象 */
  const selectedModelObj = enabledProviders.flatMap(p => p.models).find(m => m.id === selectedModelId)

  const normalizedModelQuery = modelQuery.trim().toLowerCase() // 规范化值模型搜索词
  /** 筛选结果服务商列表 */
  const filteredProviders = enabledProviders
    .map(provider => {
      const providerMatches = provider.name.toLowerCase().includes(normalizedModelQuery) // 模型服务商匹配项
      /** 模型列表 */
      const models = normalizedModelQuery && !providerMatches
        ? provider.models.filter(model => model.name.toLowerCase().includes(normalizedModelQuery))
        : provider.models
      return { ...provider, models }
    })
    .filter(provider => provider.models.length > 0)

  return (
    <div
      style={isQuick ? quickNoDragStyle : undefined}
      className={cn('chat-input-area shrink-0', isQuick ? 'px-3 pb-3 pt-1' : 'pb-4 pt-2')}
      data-file-drop-target
      onPaste={handlePaste}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <div className={cn('mx-auto w-full', isQuick ? 'max-w-none px-0' : 'max-w-[calc(66rem)] px-4')}>
        <div className={cn(
          'border border-border bg-background shadow-xs focus-within:border-primary/40 transition-colors relative',
          isQuick ? 'rounded-lg' : 'rounded-2xl',
          isDraggingOver && "border-primary"
        )}>
          {isDraggingOver && (
            <div className={cn(
              'absolute inset-0 z-10 bg-primary/5 flex items-center justify-center pointer-events-none',
              isQuick ? 'rounded-lg' : 'rounded-2xl',
            )}>
              <span className="text-sm text-primary font-medium">{t('input.dropFiles')}</span>
            </div>
          )}
        {/* 富文本编辑区域 */}
        <AttachmentChips
          items={attachments}
          variant="input"
          onRemove={(idx) => setAttachments(prev => prev.filter((_, i) => i !== idx))}
        />
        <EditorContent editor={editor} />

        {/* 聊天输入工具栏 */}
        <div className="flex items-center gap-1 px-2 pb-2">
          {/* 添加附件 */}
          <button
            aria-label={t('input.attach')}
            onClick={handleAttach}
            disabled={attachments.length >= ATTACHMENT_MAX_COUNT}
            className="p-1.5 rounded-lg hover:bg-accent text-muted-foreground hover:text-foreground transition-colors disabled:opacity-40"
          >
            <Paperclip size={16} />
          </button>

          {/* 工具选择和启用状态 */}
          <div className="relative">
            <button
              onClick={() => { setToolsOpen(v => !v); setModelOpen(false) }}
              className={cn(
                'flex items-center gap-1 px-2 py-1 rounded-lg text-xs transition-colors',
                'hover:bg-accent text-muted-foreground hover:text-foreground',
                toolsOpen && 'bg-accent text-foreground'
              )}
            >
              <Wrench size={14} />
              <span>
                {enabledToolsCount > 0
                  ? t('input.toolsSelected', { count: enabledToolsCount })
                  : t('input.tools')}
              </span>
              <ChevronDown size={10} />
            </button>
            {toolsOpen && (
              <>
                <div className="fixed inset-0 z-10" onClick={() => setToolsOpen(false)} />
                <div className="absolute bottom-full left-0 mb-1 z-20 w-64 rounded-xl border border-border bg-popover shadow-lg">
                  <div className="max-h-72 overflow-y-auto py-1">
                    {Object.entries(groupedTools).map(([category, categoryTools]) => (
                      <div key={category}>
                        <div className="px-3 py-1.5 text-[11px] font-semibold uppercase text-muted-foreground bg-muted/50">
                          {getToolCategoryLabel(category)}
                        </div>
                        {categoryTools.map(tool => (
                          <button
                            key={tool.id}
                            onClick={() => toggleTool(tool.id)}
                            className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent text-left"
                          >
                            <div className={cn(
                              'w-4 h-4 rounded border flex items-center justify-center shrink-0',
                              tool.enabled ? 'bg-primary border-primary' : 'border-border'
                            )}>
                              {tool.enabled && <Check size={10} className="text-primary-foreground" />}
                            </div>
                            <div className="flex-1 min-w-0">
                              <div className="text-foreground text-xs font-medium">{getToolDisplayName(tool)}</div>
                              <div className="text-muted-foreground text-xs truncate">{getToolDisplayDescription(tool)}</div>
                            </div>
                          </button>
                        ))}
                      </div>
                    ))}
                  </div>
                  <div className="border-t border-border p-2">
                    <button
                      type="button"
                      onClick={() => {
                        setToolsOpen(false)
                        void Window.OpenSettings({ tab: 'plugins' })
                      }}
                      className="w-full rounded-lg border border-border px-3 py-2 text-left text-xs text-foreground transition-colors hover:bg-accent"
                    >
                      {t('input.manageToolsPlugins')}
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>

          {/* 模型选择器 */}
          <div className="relative">
            <button
              onClick={() => {
                setModelOpen(v => {
                  if (v) setModelQuery('')
                  return !v
                })
                setToolsOpen(false)
              }}
              className={cn(
                'flex items-center gap-1 px-2 py-1 rounded-lg text-xs transition-colors',
                'hover:bg-accent text-muted-foreground hover:text-foreground',
                modelOpen && 'bg-accent text-foreground'
              )}
            >
              <span className="max-w-28 truncate">{selectedModelObj?.name ?? t('input.model')}</span>
              <ChevronDown size={10} />
            </button>
            {modelOpen && (
              <>
                <div className="fixed inset-0 z-10" onClick={closeModelSelector} />
                <div className="absolute bottom-full left-0 mb-1 z-20 flex max-h-80 w-64 flex-col rounded-xl border border-border bg-popover shadow-lg">
                  <div className="min-h-0 overflow-y-auto py-1">
                    {filteredProviders.map(provider => (
                      <div key={provider.id}>
                        <div className="px-3 py-1.5 text-[11px] font-semibold text-muted-foreground bg-muted/50">
                          {provider.name}
                        </div>
                        {provider.models.map(model => (
                          <button
                            key={model.id}
                            onClick={() => { setSelectedModelId(model.id); closeModelSelector() }}
                            className={cn(
                              'w-full flex items-center gap-2 px-3 py-1.5 text-xs hover:bg-accent text-left',
                              selectedModelId === model.id && 'text-primary font-medium'
                            )}
                          >
                            {model.name}
                            {selectedModelId === model.id && <Check size={12} className="ml-auto" />}
                          </button>
                        ))}
                      </div>
                    ))}
                    {filteredProviders.length === 0 && (
                      <div className="px-3 py-6 text-center text-[11px] text-muted-foreground">
                        {t('input.noModels')}
                      </div>
                    )}
                  </div>
                  <div className="border-t border-border p-2">
                    <div className="flex items-center gap-2 rounded-lg border border-input bg-background px-2">
                      <Search size={14} className="text-muted-foreground" />
                      <input
                        value={modelQuery}
                        onChange={e => setModelQuery(e.target.value)}
                        placeholder={t('input.searchModel')}
                        className="h-8 min-w-0 flex-1 bg-transparent text-sm outline-hidden placeholder:text-muted-foreground"
                      />
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>

          <div className="flex-1" />

          {/* 按生成状态显示发送或停止按钮 */}
          <button
            aria-label={isStreaming ? t('input.stop') : t('input.send')}
            onClick={() => {
              void (isStreaming ? Promise.resolve(handleStop()) : handleSend())
            }}
            disabled={!isStreaming && isEmpty && attachments.length === 0}
            className={cn(
              'p-1.5 rounded-lg transition-colors',
              isStreaming
                ? 'bg-destructive/10 text-destructive hover:bg-destructive/20'
                : (!isEmpty || attachments.length > 0)
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'text-muted-foreground cursor-not-allowed'
            )}
          >
            {isStreaming ? <Square size={16} /> : <Send size={16} />}
          </button>
        </div>
        </div>
      </div>
    </div>
  )
}
