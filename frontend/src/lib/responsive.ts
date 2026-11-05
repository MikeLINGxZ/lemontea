/** 窄屏断点 */
export const H5_BREAKPOINT = 820

/** 判断是否为窄屏宽度，接收宽度，返回是否满足条件 */
export function isH5Width(width: number): boolean {
  return width <= H5_BREAKPOINT
}
