import { isObjectRecord } from '@/lib/guards'

/** 交互式终端的状态数据 */
export type InteractiveTerminalState = {
  /** 可见 */
  visible: boolean
  /** 输出 */
  output: string
  /** 当前 */
  active: boolean
}

/** 清理终端输出处理，接收输出，返回处理后的文本 */
export function cleanTerminalOutput(output: string): string {
  const extracted = extractTerminalOutput(output, 0) // 已提取
  return stripProgressEnvelope(extracted)
    .replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, '')
    .replace(/^\[(stdout|stderr)\]\n?/gim, '')
    .replace(/^Tool confirmation:.*\n*/gim, '')
    .replace(/^\s*\n+/, '')
    .trimEnd()
}

/** 提取终端输出处理，接收输出、深度，返回处理后的文本 */
function extractTerminalOutput(output: string, depth: number): string {
  if (depth > 4) return output

  const trimmed = output.trim() // 已去除空白
  const jsonPayload = findJSONObject(trimmed) // JSON载荷
  if (!jsonPayload) return output

  try {
    const parsed: unknown = JSON.parse(jsonPayload) // 保存待校验的 JSON 解析结果
    if (!isObjectRecord(parsed)) return output
    if (typeof parsed.terminal_output === 'string') return extractTerminalOutput(parsed.terminal_output, depth + 1)
    if (typeof parsed.output === 'string') return extractTerminalOutput(parsed.output, depth + 1)
    if (typeof parsed.qr === 'string') return extractTerminalOutput(parsed.qr, depth + 1)
    if (typeof parsed.result === 'string') return extractTerminalOutput(parsed.result, depth + 1)

    const stdout = typeof parsed.stdout === 'string' ? parsed.stdout : '' // 标准输出
    const stderr = typeof parsed.stderr === 'string' ? parsed.stderr : '' // 标准错误
    if (stdout || stderr) return extractTerminalOutput([stdout, stderr].filter(Boolean).join('\n'), depth + 1)
  } catch {
    // 无法解析终端包装数据时展示原始工具输出
  }
  return output
}

/** 查找JSON对象，接收文本，返回处理结果，无法取得时为空 */
function findJSONObject(text: string): string | null {
  const start = text.indexOf('{') // 开始
  const end = text.lastIndexOf('}') // 结束
  if (start < 0 || end <= start) return null
  return text.slice(start, end + 1)
}

/** 移除进度包装内容，接收输出，返回处理后的文本 */
function stripProgressEnvelope(output: string): string {
  let text = output.replace(/\r\n/g, '\n') // 文本

  if (text.includes('\\n') && /[█▀▄]|OUTPUT_SO_FAR|TIMEOUT_REACHED/.test(text)) {
    text = text.replace(/\\n/g, '\n')
  }

  const outputSoFar = text.match(/OUTPUT_SO_FAR:\s*([\s\S]*)/i) // 输出当前已获得
  if (outputSoFar?.[1]) {
    text = outputSoFar[1]
  }

  text = text.replace(/^TIMEOUT_REACHED\s*\n?/gim, '')
  return text
}

/** 获取交互式终端状态，接收输出，返回处理结果，无法取得时为空 */
export function getInteractiveTerminalState(output: string): InteractiveTerminalState | null {
  const payload = parseToolJSON(output) // 载荷
  if (!payload || payload.interactive_terminal !== true) return null

  const status = String(payload.terminal_status ?? payload.status ?? 'active').toLowerCase() // 状态
  if (status === 'done' || status === 'hidden' || status === 'hide' || status === 'complete' || payload.visible === false) {
    return { visible: false, output: '', active: false }
  }

  /** 终端输出 */
  const terminalOutput = stringValue(payload.terminal_output)
    ?? stringValue(payload.output)
    ?? stringValue(payload.qr)
    ?? [stringValue(payload.stdout), stringValue(payload.stderr)].filter(Boolean).join('\n')

  return {
    visible: true,
    output: cleanTerminalOutput(terminalOutput),
    active: status === 'active' || status === 'waiting',
  }
}

/** 判断是否为交互式终端控制，接收输出，返回是否满足条件 */
export function isInteractiveTerminalControl(output: string): boolean {
  return getInteractiveTerminalState(output) !== null
}

/** 判断是否为交互式终端输出，接收输出，返回是否满足条件 */
export function isInteractiveTerminalOutput(output: string): boolean {
  return getInteractiveTerminalState(output)?.visible === true
}

/** 解析工具JSON，接收输出，返回处理结果，无法取得时为空 */
function parseToolJSON(output: string): Record<string, unknown> | null {
  if (!output) return null
  const jsonPayload = findJSONObject(output.trim()) // JSON载荷
  if (!jsonPayload) return null
  try {
    const parsed: unknown = JSON.parse(jsonPayload) // 保存待校验的 JSON 解析结果
    return isObjectRecord(parsed) ? parsed : null
  } catch {
    return null
  }
}

/** 文本值处理，接收值，返回处理结果，无法取得时为空 */
function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}
