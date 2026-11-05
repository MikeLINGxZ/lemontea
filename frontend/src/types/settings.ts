import type { FontSize, Language } from '@/types'

/** 设置主操作标签页的可选值 */
export type SettingsPrimaryTab = 'general' | 'providers' | 'plugins' | 'skills' | 'memory' | 'about'
/** 常规设置标签页的可选值 */
export type GeneralSettingsTab = 'display' | 'locale' | 'file' | 'shortcuts'
/** 快捷键校验状态的可选值 */
export type ShortcutValidationStatus = 'available' | 'conflict' | 'invalid' | 'idle'

/** 定义设置选项的数据结构 */
export interface SettingsOption {
  /** 当前条目的唯一编号 */
  id: string
  /** 名称 */
  name: string
  /** 图标 */
  icon?: string
}

/** 定义模型服务商模型的数据结构 */
export interface ProviderModel {
  /** 当前条目的唯一编号 */
  id: number
  /** 模型服务商编号 */
  provider_id: number
  /** 模型 */
  model: string
  /** 模型所属服务商 */
  owned_by: string
  /** 对象 */
  object: string
  /** 启用 */
  enable: boolean
  /** 别名 */
  alias: string | null
  /** 是否自定义 */
  is_custom: boolean
  /** 是否默认 */
  is_default: boolean
}

/** 定义模型服务商条目的数据结构 */
export interface ProviderItem {
  /** 当前条目的唯一编号 */
  id: number
  /** 模型服务商名称 */
  provider_name: string
  /** 模型服务商类型 */
  provider_type: string
  /** 基础地址 */
  base_url: string
  /** API键 */
  api_key: string
  /** 是否启用 */
  enabled: boolean
  /** 是否默认 */
  is_default: boolean
  /** 模型数量 */
  model_count: number
  /** 图标 */
  icon: string
  /** 模型列表 */
  models: ProviderModel[]
}

/** 定义受支持模型服务商的数据结构 */
export type SupportedProvider = {
  /** 类型 */
  type: string
  /** 图标 */
  icon: string
  /** 名称 */
  name: string
  /** 描述 */
  description: string
  /** 基础地址 */
  base_url: string
}

/** 定义设置初始化数据的数据结构 */
export interface SettingsBootstrap {
  /** 地区 */
  locale: string
  /** 语言 */
  language: Language
  /** 字体尺寸 */
  font_size: FontSize
  /** 数据目录 */
  data_dir: string
  /** 日志级别 */
  log_level: string
  /** 快捷聊天快捷键 */
  quick_chat_shortcut: string
  /** 默认模型服务商编号 */
  default_provider_id: number
  /** 版本 */
  version: string
  /** 服务商列表 */
  providers: ProviderItem[]
  /** 语言列表 */
  languages?: SettingsOption[]
  /** 地区列表 */
  regions?: SettingsOption[]
}

/** 定义扩展工具条目的数据结构 */
export type ExtensionToolItem = {
  /** 工具编号 */
  tool_id: string
  /** 服务编号 */
  server_id: string
  /** 名称 */
  name: string
  /** 描述 */
  description: string
  /** 是否启用 */
  enabled: boolean
  /** 需要确认 */
  requires_confirm: boolean
}

/** 定义扩展条目的数据结构 */
export type ExtensionItem = {
  /** 当前条目的唯一编号 */
  id: string
  /** 名称 */
  name: string
  /** 描述 */
  description: string
  /** 作者 */
  author: string
  /** 版本 */
  version: string
  /** 类别标识 */
  kind: 'mcp' | 'plugin' | 'cli'
  /** 是否启用 */
  enabled: boolean
  /** 运行环境状态 */
  runtime_status: string
  /** 运行环境消息 */
  runtime_message: string
  /** 根目录目录 */
  root_dir: string
  /** 来源目录 */
  source_dir: string
  /** 配置文件路径 */
  config_file_path: string
  /** 工具列表 */
  tools: ExtensionToolItem[]
}
