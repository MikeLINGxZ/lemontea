import type { FontSize, Language } from '@/types'

export type SettingsPrimaryTab = 'general' | 'providers' | 'plugins' | 'skills' | 'memory' | 'about'
export type GeneralSettingsTab = 'display' | 'locale' | 'file' | 'shortcuts'
export type ShortcutValidationStatus = 'available' | 'conflict' | 'invalid' | 'idle'

export interface SettingsOption {
  id: string
  name: string
  icon?: string
}

export interface ProviderModel {
  id: number
  provider_id: number
  model: string
  owned_by: string
  object: string
  enable: boolean
  alias: string | null
  is_custom: boolean
  is_default: boolean
}

export interface ProviderItem {
  id: number
  provider_name: string
  provider_type: string
  base_url: string
  api_key: string
  enabled: boolean
  is_default: boolean
  model_count: number
  icon: string
  models: ProviderModel[]
}

export type SupportedProvider = {
  type: string
  icon: string
  name: string
  description: string
  base_url: string
}

export interface SettingsBootstrap {
  locale: string
  language: Language
  font_size: FontSize
  data_dir: string
  log_level: string
  quick_chat_shortcut: string
  default_provider_id: number
  version: string
  providers: ProviderItem[]
  languages?: SettingsOption[]
  regions?: SettingsOption[]
}

export type ExtensionToolItem = {
  tool_id: string
  server_id: string
  name: string
  description: string
  enabled: boolean
  requires_confirm: boolean
}

export type ExtensionItem = {
  id: string
  name: string
  description: string
  author: string
  version: string
  kind: 'mcp' | 'plugin' | 'cli'
  enabled: boolean
  runtime_status: string
  runtime_message: string
  root_dir: string
  source_dir: string
  config_file_path: string
  tools: ExtensionToolItem[]
}
