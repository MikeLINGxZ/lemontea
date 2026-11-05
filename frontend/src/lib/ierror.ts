/** 定义解析结果i错误的数据结构 */
export type ParsedIError = {
  /** 可展示的错误消息 */
  msg: string
  /** 详情 */
  detail: string
}

/** 定义未校验记录的数据结构 */
type UnknownRecord = Record<string, unknown>

/** 判断是否为记录，接收值，返回是否满足条件 */
function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null
}

/** 规范化文本，接收值，返回处理结果，无法取得时为空 */
function normalizeString(value: unknown): string | null {
  return typeof value === 'string' && value.length > 0 ? value : null
}

/** 解析结构化数据i错误，接收值，返回处理结果，无法取得时为空 */
function parseStructuredIError(value: unknown): ParsedIError | null {
  if (!isRecord(value)) {
    return null
  }

  const msg = normalizeString(value.msg) // 可展示的错误消息
  const detail = normalizeString(value.detail) // 详情
  if (!msg && !detail) {
    return null
  }

  return {
    msg: msg ?? detail ?? 'Unknown error',
    detail: detail ?? msg ?? 'Unknown error',
  }
}

/** 解析序列化 i错误，接收值，返回处理结果，无法取得时为空 */
function parseSerializedIError(value: string): ParsedIError | null {
  try {
    const parsed = JSON.parse(value) as unknown // 保存待校验的 JSON 解析结果
    return parseUnknownIError(parsed)
  } catch {
    return null
  }
}

/** 从异常对象或序列化文本中提取可展示的消息和详情，接收值，返回对应的处理结果 */
export function parseUnknownIError(value: unknown): ParsedIError {
  const structured = parseStructuredIError(value) // 结构化数据
  if (structured) {
    return structured
  }

  if (typeof value === 'string') {
    const parsed = parseSerializedIError(value) // 从文本解析出的数据
    if (parsed) {
      return parsed
    }

    return { msg: value, detail: value }
  }

  if (value instanceof Error) {
    const parsedFromMessage = parseSerializedIError(value.message) // 解析结果起点消息
    if (parsedFromMessage) {
      return parsedFromMessage
    }

    if ('cause' in value) {
      return parseUnknownIError((value as Error & { /* 原因 */ cause?: unknown }).cause ?? value.message)
    }

    return { msg: value.message, detail: value.message }
  }

  if (isRecord(value)) {
    const messageValue = value.message // 消息值
    if (typeof messageValue === 'string') {
      const parsed = parseSerializedIError(messageValue) // 从文本解析出的数据
      if (parsed) {
        return parsed
      }

      if (isRecord(value.cause)) {
        return parseUnknownIError(value.cause)
      }

      return { msg: messageValue, detail: messageValue }
    }

    if (messageValue !== undefined) {
      return parseUnknownIError(messageValue)
    }

    if (value.cause !== undefined) {
      return parseUnknownIError(value.cause)
    }
  }

  const fallback = String(value) // 回退
  return { msg: fallback, detail: fallback }
}
