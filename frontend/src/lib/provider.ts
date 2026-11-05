import { Type as ProviderType } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider/models'

/** 将服务商标识 value 匹配到后端枚举，缺失或无效时返回后端定义的空值 */
export function resolveProviderType(value: string | undefined): ProviderType {
  return Object.values(ProviderType).find((type) => type === value) ?? ProviderType.$zero
}
