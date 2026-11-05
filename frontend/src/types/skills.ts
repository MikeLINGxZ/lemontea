/** 技能来源的可选值 */
export type SkillSource = 'builtin' | 'user' | 'ai'

/** 定义技能条目的数据结构 */
export type SkillItem = {
  /** 名称 */
  name: string
  /** 描述 */
  description: string
  /** 来源 */
  source: SkillSource
  /** 是否禁用 */
  disabled: boolean
  /** 是否包含技能正文 */
  has_body: boolean
}

/** 定义技能完整的数据结构 */
export type SkillFull = SkillItem & {
  /** 正文 */
  body: string
}
