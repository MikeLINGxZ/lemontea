import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { Service } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service';

export interface EmbeddingConfig {
  provider: string;
  baseUrl: string;
  apiKey: string;
  model: string;
}

interface LabState {
  memorySystemEnabled: boolean;
  setMemorySystemEnabled: (enabled: boolean) => void;
  vectorSearchEnabled: boolean;
  setVectorSearchEnabled: (enabled: boolean) => void;
  embeddingConfig: EmbeddingConfig;
  setEmbeddingConfig: (config: Partial<EmbeddingConfig>) => void;
  isHydrated: boolean;
  setPreferences: (preferences: { memorySystemEnabled: boolean; vectorSearchEnabled: boolean; embeddingConfig: Partial<EmbeddingConfig> }) => void;
  markHydrated: () => void;
}

export const useLabStore = create<LabState>()(
  persist(
    (set) => ({
      memorySystemEnabled: false,
      setMemorySystemEnabled: (enabled: boolean) => set({ memorySystemEnabled: enabled }),
      vectorSearchEnabled: false,
      setVectorSearchEnabled: (enabled: boolean) => set({ vectorSearchEnabled: enabled }),
      embeddingConfig: {
        provider: 'ollama',
        baseUrl: 'http://localhost:11434',
        apiKey: '',
        model: 'bge-m3',
      },
      setEmbeddingConfig: (config) =>
        set((state) => ({
          embeddingConfig: { ...state.embeddingConfig, ...config },
        })),
      isHydrated: false,
      setPreferences: ({ memorySystemEnabled, vectorSearchEnabled, embeddingConfig }) =>
        set((state) => ({
          memorySystemEnabled,
          vectorSearchEnabled,
          embeddingConfig: { ...state.embeddingConfig, ...embeddingConfig },
        })),
      markHydrated: () => set({ isHydrated: true }),
    }),
    {
      name: 'lab-settings',
    }
  )
);

export async function hydrateLabPreferences() {
  try {
    const preferences = await Service.GetAppPreferences();
    if (preferences) {
      useLabStore.getState().setPreferences({
        memorySystemEnabled: preferences.memory_system_enabled,
        vectorSearchEnabled: preferences.vector_search_enabled,
        embeddingConfig: {
          provider: preferences.embedding_provider,
          baseUrl: preferences.embedding_base_url,
          apiKey: preferences.embedding_api_key,
          model: preferences.embedding_model,
        },
      });
    }
  } catch (error) {
    console.error('Failed to hydrate lab preferences from backend:', error);
  } finally {
    useLabStore.getState().markHydrated();
  }
}
