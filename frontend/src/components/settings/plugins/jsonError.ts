/** 定义JSON错误的数据结构 */
export type JsonError =
  | { /* 有效 */ valid: true }
  | { /* 有效 */ valid: false, /* 错误行 */ errorLine: number, /* 错误消息 */ errorMessage: string }

/** 检查 JSON 文本并返回错误所在行及说明，接收文本，返回对应的处理结果 */
export function getJsonError(text: string): JsonError {
  if (text.trim() === '') return { valid: false, errorLine: 1, errorMessage: 'empty' }

  try {
    JSON.parse(text)
    return { valid: true }
  } catch (err) {
    if (err instanceof SyntaxError) {
      const pos = extractErrorPos(text, err.message) // 字符位置
      const line = getLineNumberAtPos(text, pos) // 行
      return { valid: false, errorLine: Math.max(line, 1), errorMessage: err.message }
    }

    return { valid: false, errorLine: 1, errorMessage: 'parse error' }
  }
}

/** 获取行数值时间位置，接收文本、字符位置，返回计算后的数值 */
function getLineNumberAtPos(text: string, pos: number): number {
  return text.slice(0, Math.min(Math.max(pos, 0), text.length)).split('\n').length
}

/** 获取位置起点行and列，接收文本、行、列，返回计算后的数值 */
function getPosFromLineAndColumn(text: string, line: number, column: number): number {
  if (line <= 1) return Math.max(column - 1, 0)

  let currentLine = 1 // 当前行

  for (/* 循环索引 */ let i = 0; i < text.length; i++) {
    if (currentLine === line) {
      return i + Math.max(column - 1, 0)
    }
    if (text[i] === '\n') currentLine += 1
  }

  return text.length
}

/** 提取错误位置处理，接收文本、消息，返回计算后的数值 */
function extractErrorPos(text: string, message: string): number {
  const positionMatch = message.match(/position\s+(\d+)/i) // 位置匹配项
  if (positionMatch) return parseInt(positionMatch[1], 10)

  const lineColumnMatch = message.match(/line\s+(\d+)\s+column\s+(\d+)/i) // 行列匹配项
  if (lineColumnMatch) {
    const [, line, column] = lineColumnMatch // 取出行、列
    return getPosFromLineAndColumn(text, parseInt(line, 10), parseInt(column, 10))
  }

  return scanErrorPos(text)
}

/** 扫描错误位置，接收文本，返回计算后的数值 */
function scanErrorPos(text: string): number {
  const scanner = new JsonScanner(text) // 扫描器
  return scanner.findErrorPos()
}

/** 定义JSON扫描器的数据结构 */
class JsonScanner {
  /** 索引 */
  private index = 0

  /** 保存待扫描的 JSON 文本并初始化扫描器 */
  constructor(private readonly text: string) {}

  /** 返回 JSON 中第一个非法字符的位置，返回计算后的数值 */
  findErrorPos(): number {
    this.skipWhitespace()
    if (this.index >= this.text.length) return this.text.length

    const valueError = this.parseValue() // 值错误
    if (valueError !== null) return valueError

    this.skipWhitespace()
    return this.index < this.text.length ? this.index : this.text.length
  }

  /** 解析值，返回处理结果，无法取得时为空 */
  private parseValue(): number | null {
    if (this.index >= this.text.length) return this.text.length

    const ch = this.text[this.index] // 当前位置的字符
    if (ch === '{') return this.parseObject()
    if (ch === '[') return this.parseArray()
    if (ch === '"') return this.parseString()
    if (ch === '-' || this.isDigit(ch)) return this.parseNumber()
    if (this.text.startsWith('true', this.index)) {
      this.index += 4
      return null
    }
    if (this.text.startsWith('false', this.index)) {
      this.index += 5
      return null
    }
    if (this.text.startsWith('null', this.index)) {
      this.index += 4
      return null
    }

    return this.index
  }

  /** 解析对象，返回处理结果，无法取得时为空 */
  private parseObject(): number | null {
    this.index += 1
    this.skipWhitespace()

    if (this.peek() === '}') {
      this.index += 1
      return null
    }

    while (this.index < this.text.length) {
      if (this.peek() !== '"') return this.index

      const keyError = this.parseString() // 键错误
      if (keyError !== null) return keyError

      this.skipWhitespace()
      if (this.peek() !== ':') return this.index
      this.index += 1

      this.skipWhitespace()
      const valueError = this.parseValue() // 值错误
      if (valueError !== null) return valueError

      this.skipWhitespace()
      const ch = this.peek() // 当前位置的字符
      if (ch === ',') {
        this.index += 1
        this.skipWhitespace()
        continue
      }
      if (ch === '}') {
        this.index += 1
        return null
      }
      return this.index
    }

    return this.text.length
  }

  /** 解析数组，返回处理结果，无法取得时为空 */
  private parseArray(): number | null {
    this.index += 1
    this.skipWhitespace()

    if (this.peek() === ']') {
      this.index += 1
      return null
    }

    while (this.index < this.text.length) {
      const valueError = this.parseValue() // 值错误
      if (valueError !== null) return valueError

      this.skipWhitespace()
      const ch = this.peek() // 当前位置的字符
      if (ch === ',') {
        this.index += 1
        this.skipWhitespace()
        continue
      }
      if (ch === ']') {
        this.index += 1
        return null
      }
      return this.index
    }

    return this.text.length
  }

  /** 解析文本，返回处理结果，无法取得时为空 */
  private parseString(): number | null {
    this.index += 1

    while (this.index < this.text.length) {
      const ch = this.text[this.index] // 当前位置的字符

      if (ch === '"') {
        this.index += 1
        return null
      }

      if (ch === '\\') {
        this.index += 1
        if (this.index >= this.text.length) return this.text.length

        const escaped = this.text[this.index] // 转义字符
        if (escaped === 'u') {
          for (/* 循环索引 */ let i = 1; i <= 4; i++) {
            const hex = this.text[this.index + i] // 当前十六进制字符
            if (hex === undefined || !/[0-9a-fA-F]/.test(hex)) return this.index + i
          }
          this.index += 5
          continue
        }

        this.index += 1
        continue
      }

      if (ch === '\n' || ch === '\r') return this.index
      this.index += 1
    }

    return this.text.length
  }

  /** 解析数值，返回处理结果，无法取得时为空 */
  private parseNumber(): number | null {
    const start = this.index // 开始

    if (this.peek() === '-') this.index += 1
    if (!this.isDigit(this.peek())) return this.index

    if (this.peek() === '0') {
      this.index += 1
    } else {
      this.consumeDigits()
    }

    if (this.peek() === '.') {
      this.index += 1
      if (!this.isDigit(this.peek())) return this.index
      this.consumeDigits()
    }

    if (this.peek() === 'e' || this.peek() === 'E') {
      this.index += 1
      if (this.peek() === '+' || this.peek() === '-') this.index += 1
      if (!this.isDigit(this.peek())) return this.index
      this.consumeDigits()
    }

    return this.index === start ? start : null
  }

  /** 读取并推进数字序列，无返回值 */
  private consumeDigits() {
    while (this.isDigit(this.peek())) {
      this.index += 1
    }
  }

  /** 跳过 空白字符处理，无返回值 */
  private skipWhitespace() {
    while (/\s/.test(this.peek())) {
      this.index += 1
    }
  }

  /** 查看当前位置字符处理，返回处理后的文本 */
  private peek(): string {
    return this.text[this.index] ?? ''
  }

  /** 判断是否为数字，接收当前位置的字符，返回是否满足条件 */
  private isDigit(ch: string): boolean {
    return ch >= '0' && ch <= '9'
  }
}
