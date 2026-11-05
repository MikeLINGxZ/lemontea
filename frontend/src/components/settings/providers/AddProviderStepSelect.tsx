import { useEffect, useState } from 'react'
import { Config } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/config'
import { cn } from '@/lib/utils'
import type { SupportedProvider } from '@/types/settings'

/** 新增模型服务商步骤选择，接收组件输入属性，返回组件内容 */
export function AddProviderStepSelect(props: {
  /** 选中项 */
  selected: SupportedProvider | null
  /** 处理选择事件，接收当前条目，无返回值 */
  onSelect: (p: SupportedProvider) => void
}) {
  const [providers, setProviders] = useState<SupportedProvider[]>([]) // 服务商列表及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    let cancelled = false // 已取消
    Config.SupportedProviderList({})
      .then((result) => {
        if (!cancelled && result?.supported_providers) {
          setProviders(
            result.supported_providers.map((p) => ({
              type: p.type,
              icon: p.icon,
              name: p.name,
              description: p.description,
              base_url: p.base_url,
            })),
          )
        }
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="space-y-2 pb-4 pt-2">
      {providers.map((provider) => (
        <button
          key={provider.type}
          type="button"
          onClick={() => props.onSelect(provider)}
          aria-label={`Select ${provider.name}`}
          className={cn(
            'flex w-full items-start gap-4 rounded-2xl border p-4 text-left transition-colors',
            props.selected?.type === provider.type
              ? 'border-primary bg-primary/10'
              : 'border-border bg-background hover:bg-accent',
          )}
        >
          {provider.icon && (
            <img
              src={provider.icon}
              alt={provider.name}
              className="h-10 w-10 rounded-xl object-contain"
              onError={(e) => {
                ;(e.currentTarget as HTMLImageElement).style.display = 'none'
              }}
            />
          )}
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold text-foreground">{provider.name}</div>
            <div className="mt-0.5 text-xs text-muted-foreground">{provider.description}</div>
            {provider.base_url && (
              <div className="mt-1 text-xs text-muted-foreground/70">{provider.base_url}</div>
            )}
          </div>
        </button>
      ))}
    </div>
  )
}
