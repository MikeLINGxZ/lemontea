import type { Conversation, Message, Tool, ModelProvider } from '../types'

/** 当前时间 */
const now = new Date()
/** 小时之前处理，接收小时，返回表达式结果 */
const hoursAgo = (hours: number) => new Date(now.getTime() - hours * 3600000).toISOString()

/** 模拟会话列表 */
export const mockConversations: Conversation[] = [
  {
    id: 101,
    title: 'React Performance',
    kind: 'user',
    createdAt: hoursAgo(3),
    updatedAt: hoursAgo(1),
    starred: true,
    status: 'idle',
  },
]

/** 模拟消息列表 */
export const mockMessages: Record<number, Message[]> = {
  101: [
    {
      id: 1,
      sessionId: 101,
      parentId: null,
      role: 'user',
      contentType: 'text',
      content: 'React performance tips?',
      modelName: '',
      agentName: '',
      tokensIn: 0,
      tokensOut: 0,
      extra: '',
      createdAt: hoursAgo(3),
    },
    {
      id: 2,
      sessionId: 101,
      parentId: null,
      role: 'assistant',
      contentType: 'thinking',
      content: 'Need a concise overview with practical items.',
      modelName: 'claude-sonnet-4-5',
      agentName: 'main',
      tokensIn: 10,
      tokensOut: 50,
      extra: '',
      createdAt: hoursAgo(2),
    },
    {
      id: 3,
      sessionId: 101,
      parentId: null,
      role: 'assistant',
      contentType: 'text',
      content: 'Use memoization, list virtualization, and lazy loading.',
      modelName: 'claude-sonnet-4-5',
      agentName: 'main',
      tokensIn: 10,
      tokensOut: 50,
      extra: '',
      createdAt: hoursAgo(2),
    },
  ],
}

/** 模拟工具列表 */
export const mockTools: Tool[] = [
  { id: 'web_fetch', name: '网页抓取', description: '访问任意 URL 并获取页面内容', enabled: true },
  { id: 'code_exec', name: '代码执行', description: '执行代码片段', enabled: true },
  { id: 'file_read', name: '文件读取', description: '读取本地文件内容', enabled: false },
  { id: 'shell', name: 'Shell 命令', description: '执行 Shell 命令（需审批）', enabled: false },
]

/** 模拟服务商列表 */
export const mockProviders: ModelProvider[] = [
  {
    id: 'anthropic',
    name: 'Anthropic',
    models: [
      { id: 'claude-sonnet-4-5', name: 'Claude Sonnet 4.5', providerId: 'anthropic' },
    ],
  },
]

/** 默认模型编号 */
export const DEFAULT_MODEL_ID = 'claude-sonnet-4-5'
