import { useEffect } from 'react'

/** 输入辅助输入选择器 */
const INPUT_ASSISTANCE_SELECTOR = 'input, textarea, [contenteditable="true"]'

/** 关闭辅助输入对应元素，接收元素，无返回值 */
function disableAssistanceForElement(element: Element): void {
  element.setAttribute('autocomplete', 'off')
  element.setAttribute('autocorrect', 'off')
  element.setAttribute('autocapitalize', 'off')
  element.setAttribute('spellcheck', 'false')
}

/** 关闭输入辅助输入，接收根目录，无返回值 */
function disableInputAssistance(root: ParentNode): void {
  root.querySelectorAll(INPUT_ASSISTANCE_SELECTOR).forEach(disableAssistanceForElement)
}

/** 关闭输入控件的自动更正和拼写检查，并监听后来添加的输入控件，无返回值 */
export function useDisableInputAssistance(): void {
  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    disableInputAssistance(document)
    document.documentElement.setAttribute('spellcheck', 'false')

    /** 观察器 */
    const observer = new MutationObserver((mutations: MutationRecord[]) => {
      for (/* DOM变动 */ const mutation of mutations) {
        for (/* 节点 */ const node of mutation.addedNodes) {
          if (!(node instanceof Element)) continue
          if (node.matches(INPUT_ASSISTANCE_SELECTOR)) {
            disableAssistanceForElement(node)
          }
          disableInputAssistance(node)
        }
      }
    })

    observer.observe(document.body, { childList: true, subtree: true })
    return () => observer.disconnect()
  }, [])
}
