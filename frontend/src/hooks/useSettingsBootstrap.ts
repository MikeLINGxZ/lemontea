import { Settings } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings'
import { Config } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/config'
import { Provider } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider'
import type { ProviderWrapper } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto/models'
import { useEffect } from 'react'
import { useSettingsStore } from '@/store/settingsStore'
import type { ProviderItem, SettingsBootstrap } from '@/types/settings'

/** 回退初始化数据 */
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

/** 转换包装终点模型服务商条目列表，接收包装对象列表，返回对应的处理结果 */
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

/** 加载设置窗口所需的初始数据，绑定不可用时使用本地默认值，无返回值 */
export function useSettingsBootstrap() {
  const hydrate = useSettingsStore((state) => state.hydrate) // 从状态仓库读取恢复
  const setLocaleOptions = useSettingsStore((state) => state.setLocaleOptions) // 从状态仓库读取集合地区选项列表

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    let cancelled = false // 已取消

    /** 加载处理，完成后结束异步任务 */
    const load = async () => {
      try {
        /** 保存载荷结果、语言列表结果、地区列表结果、服务商列表结果 */
        const [payloadResult, languagesResult, regionsResult, providersResult] = await Promise.all([
          Settings.LoadBootstrap({}),
          Config.LanguageList({}),
          Config.RegionList({}),
          Provider.ListProviders({}),
        ])
        if (!cancelled && payloadResult?.bootstrap) {
          /** 服务商列表 */
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
        // 后端绑定不可用时回退到本地初始设置
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
