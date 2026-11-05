import { ReactRenderer } from '@tiptap/react'
import tippy, { type Instance as TippyInstance } from 'tippy.js'
import type { SuggestionProps, SuggestionKeyDownProps } from '@tiptap/suggestion'
import { Skills as SkillsBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills'
import { SlashSuggestionPopup, type SlashSuggestionHandle, type SlashSuggestionItem } from './SlashSuggestionPopup'

/** 创建斜杠指令建议配置，查询技能并管理弹层生命周期，返回对应的处理结果 */
export function buildSlashSuggestion() {
  return {
    char: '/',
    startOfLine: false,
    /** 条目列表处理，接收搜索词，返回对应的处理结果 */
    items: async ({ query }: { /* 搜索词 */ query: string }) => {
      try {
        const list = await SkillsBinding.ListSkills({}) // 保存列表
        const enabled = (list?.skills ?? []).filter((s) => !s.disabled) // 是否启用
        return enabled
          .filter((s) => s.name.toLowerCase().startsWith(query.toLowerCase()))
          .slice(0, 10)
          .map((s) => ({ name: s.name, description: s.description })) as SlashSuggestionItem[]
      } catch {
        return []
      }
    },
    /** 构造渲染与清理回调处理，返回对应的处理结果 */
    render: () => {
      let component: ReactRenderer<SlashSuggestionHandle> // 组件
      let popup: TippyInstance[] = [] // 弹层
      return {
        /** 处理开始事件，接收组件输入属性，无返回值 */
        onStart: (props: SuggestionProps<SlashSuggestionItem, SlashSuggestionItem>) => {
          component = new ReactRenderer(SlashSuggestionPopup, { props, editor: props.editor })
          popup = tippy('body', {
            getReferenceClientRect: props.clientRect as () => DOMRect,
            /** 追加终点，返回表达式结果 */
            appendTo: () => document.body,
            content: component.element,
            showOnCreate: true,
            interactive: true,
            trigger: 'manual',
            placement: 'top-start',
          })
        },
        /** 处理更新事件，接收组件输入属性，无返回值 */
        onUpdate: (props: SuggestionProps<SlashSuggestionItem, SlashSuggestionItem>) => {
          component.updateProps(props)
          popup[0]?.setProps({ getReferenceClientRect: props.clientRect as () => DOMRect })
        },
        /** 处理键向下事件，接收组件输入属性，返回表达式结果 */
        onKeyDown: (props: SuggestionKeyDownProps) => component.ref?.onKeyDown(props) ?? false,
        /** 处理退出事件，无返回值 */
        onExit: () => {
          popup[0]?.destroy()
          component.destroy()
        },
      }
    },
  }
}
