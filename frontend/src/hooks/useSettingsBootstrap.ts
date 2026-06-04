import { Settings } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings'
import { Config } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/config'
import { Provider } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import type { ProviderWrapper } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import { useEffect } from 'react'
import { useSettingsStore } from '@/store/settingsStore'
import type { ProviderItem, SettingsBootstrap } from '@/types/settings'

const fallbackBootstrap: SettingsBootstrap = {
  locale: 'zh-CN',
  language: 'zh-CN',
  font_size: 'md',
  data_dir: '',
  log_level: 'info',
  quick_chat_shortcut: 'CmdOrCtrl+Shift+Space',
  default_provider_id: 0,
  version: 'v0.0.1-dev',
  providers: [],
  languages: [
    { id: 'zh-CN', name: '简体中文' },
    { id: 'en', name: 'English' },
  ],
  regions: [
    { id: 'zh-CN', name: '中国', icon: '🇨🇳' },
    { id: 'en-US', name: '美国', icon: '🇺🇸' },
  ],
}

function mapWrapperToProviderItems(wrappers: ProviderWrapper[]): ProviderItem[] {
  return wrappers.map((w) => ({
    id: w.providers.id,
    provider_name: w.providers.provider_name,
    provider_type: w.providers.provider_type,
    base_url: w.providers.base_url,
    api_key: w.providers.api_key,
    enabled: w.providers.enabled,
    is_default: w.providers.is_default,
    model_count: w.models.length,
    icon: w.providers.icon,
    models: w.models.map((m) => ({
      id: m.id,
      provider_id: m.provider_id,
      model: m.model,
      owned_by: m.owned_by,
      object: m.object,
      enable: m.enable,
      alias: m.alias,
      is_custom: m.is_custom,
      is_default: m.is_default,
    })),
  }))
}

export function useSettingsBootstrap() {
  const hydrate = useSettingsStore((state) => state.hydrate)
  const setLocaleOptions = useSettingsStore((state) => state.setLocaleOptions)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      try {
        const [payloadResult, languagesResult, regionsResult, providersResult] = await Promise.all([
          Settings.LoadBootstrap({}),
          Config.LanguageList({}),
          Config.RegionList({}),
          Provider.ListProviders({}),
        ])
        if (!cancelled && payloadResult?.bootstrap) {
          const providers = providersResult?.providers
            ? mapWrapperToProviderItems(providersResult.providers)
            : []
          hydrate({
            ...(payloadResult.bootstrap as SettingsBootstrap),
            providers,
            languages: languagesResult?.languages ?? fallbackBootstrap.languages,
            regions: regionsResult?.regions ?? fallbackBootstrap.regions,
          })
          setLocaleOptions({
            languages: languagesResult?.languages ?? fallbackBootstrap.languages!,
            regions: regionsResult?.regions ?? fallbackBootstrap.regions!,
          })
          return
        }
      } catch {
        // Fall back to a local bootstrap payload when backend bindings are unavailable.
      }

      if (!cancelled) {
        hydrate(fallbackBootstrap)
        setLocaleOptions({
          languages: fallbackBootstrap.languages!,
          regions: fallbackBootstrap.regions!,
        })
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  }, [hydrate, setLocaleOptions])
}
