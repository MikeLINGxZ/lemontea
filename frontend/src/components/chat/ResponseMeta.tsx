import { ArrowDownToLine, ArrowUpFromLine } from 'lucide-react'

/** 响应元信息组件接收的属性 */
interface Props {
  /** 模型名称 */
  modelName?: string
  /** 输入 Token 数 */
  tokensIn?: number
  /** 输出 Token 数 */
  tokensOut?: number
}

/** 渲染响应元信息界面，接收模型名称、输入 Token 数、输出 Token 数，返回组件内容 */
export function ResponseMeta({ modelName = '', tokensIn = 0, tokensOut = 0 }: Props) {
  if (!modelName && tokensIn <= 0 && tokensOut <= 0) {
    return null
  }

  return (
    <div className="mt-0.5 flex w-full items-center gap-3 text-xs text-muted-foreground">
      {modelName && <span className="opacity-70">{modelName}</span>}
      {tokensIn > 0 && (
        <span className="flex items-center gap-1">
          <ArrowUpFromLine size={10} />
          {tokensIn}
        </span>
      )}
      {tokensOut > 0 && (
        <span className="flex items-center gap-1">
          <ArrowDownToLine size={10} />
          {tokensOut}
        </span>
      )}
    </div>
  )
}
