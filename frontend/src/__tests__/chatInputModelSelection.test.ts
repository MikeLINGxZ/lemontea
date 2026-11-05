import { describe, expect, it } from 'vitest'
import { pickInitialModelId } from '@/components/chat/ChatInput'

describe('ChatInput model selection', () => {
  it('falls back to the first enabled model when no default provider is marked', () => {
    const selected = pickInitialModelId([
      {
        provider: {
          id: 1,
          provider_name: 'First',
          provider_type: 'deepseek',
          base_url: 'https://example.test',
          api_key: '',
          enabled: true,
          is_default: false,
          icon: '',
        },
        models: [
          { id: 10, provider_id: 1, model: 'disabled', owned_by: '', object: 'model', enable: false, alias: null, is_custom: false, is_default: false },
          { id: 11, provider_id: 1, model: 'first-enabled', owned_by: '', object: 'model', enable: true, alias: null, is_custom: false, is_default: false },
        ],
      },
    ] as any)

    expect(selected).toBe(11)
  })

  it('prefers the default enabled model from the default enabled provider', () => {
    const selected = pickInitialModelId([
      {
        provider: {
          id: 1,
          provider_name: 'First',
          provider_type: 'deepseek',
          base_url: 'https://example.test',
          api_key: '',
          enabled: true,
          is_default: false,
          icon: '',
        },
        models: [
          { id: 11, provider_id: 1, model: 'first-enabled', owned_by: '', object: 'model', enable: true, alias: null, is_custom: false, is_default: false },
        ],
      },
      {
        provider: {
          id: 2,
          provider_name: 'Default',
          provider_type: 'openai_compatibility',
          base_url: 'https://example.test',
          api_key: '',
          enabled: true,
          is_default: true,
          icon: '',
        },
        models: [
          { id: 20, provider_id: 2, model: 'ordinary', owned_by: '', object: 'model', enable: true, alias: null, is_custom: false, is_default: false },
          { id: 21, provider_id: 2, model: 'default-model', owned_by: '', object: 'model', enable: true, alias: null, is_custom: false, is_default: true },
        ],
      },
    ] as any)

    expect(selected).toBe(21)
  })
})
