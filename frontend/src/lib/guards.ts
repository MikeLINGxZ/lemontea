/** 判断外部输入是否为可读取字段的对象，接收值，返回是否满足条件 */
export function isObjectRecord(value: unknown): value is { [key: string]: unknown } {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
