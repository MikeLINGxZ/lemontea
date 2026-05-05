import { Events } from '@wailsio/runtime';
import { Model } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models';

export const DEFAULT_MODEL_KEY = 'chat_default_model';

export interface DefaultModelConfig {
  modelId: number;
  modelName: string;
}

function getProviderDisplayName(model: Model): string {
  const providerName = model.provider_name?.trim();
  if (providerName) {
    return providerName;
  }
  return `Provider #${model.provider_id}`;
}

function compareModels(a: Model, b: Model): number {
  const providerNameCompare = getProviderDisplayName(a).localeCompare(
    getProviderDisplayName(b),
    undefined,
    { sensitivity: 'base', numeric: true }
  );
  if (providerNameCompare !== 0) {
    return providerNameCompare;
  }

  if (a.provider_id !== b.provider_id) {
    return a.provider_id - b.provider_id;
  }

  return a.model.localeCompare(b.model, undefined, {
    sensitivity: 'base',
    numeric: true,
  });
}

export function getDefaultModelConfig(): DefaultModelConfig | null {
  try {
    const raw = localStorage.getItem(DEFAULT_MODEL_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as DefaultModelConfig;
  } catch {
    return null;
  }
}

export function setDefaultModelConfig(config: DefaultModelConfig) {
  localStorage.setItem(DEFAULT_MODEL_KEY, JSON.stringify(config));
  window.dispatchEvent(new CustomEvent('chat-default-model-changed', { detail: config }));
  void Events.Emit('chat-default-model-changed', config);
}

export function clearDefaultModelConfig() {
  localStorage.removeItem(DEFAULT_MODEL_KEY);
  window.dispatchEvent(new CustomEvent('chat-default-model-changed', { detail: null }));
  void Events.Emit('chat-default-model-changed', null);
}

export function resolvePreferredModel(models: Model[]): Model | null {
  if (models.length === 0) {
    return null;
  }

  const config = getDefaultModelConfig();
  if (config) {
    const configuredModel = models.find(model => model.id === config.modelId);
    if (configuredModel) {
      return configuredModel;
    }
  }

  return [...models].sort(compareModels)[0] ?? null;
}
