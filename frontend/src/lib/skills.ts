import type { SkillItem as BoundSkillItem } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills/skills_dto/models'
import type { SkillFull } from '@/types/skills'

/** 技能名称匹配规则 */
const skillNamePattern = /^[a-z0-9][a-z0-9-]{0,63}$/

/** 判断是否为有效技能名称，接收值，返回是否满足条件 */
export function isValidSkillName(value: string): boolean {
  return skillNamePattern.test(value.trim())
}

/** 终点技能名称处理，接收值，返回处理后的文本 */
export function toSkillName(value: string): string {
  const name = value.trim() // 名称
  return isValidSkillName(name) ? name : ''
}

/** 将后端技能转换为列表和编辑页共用的数据，接收当前条目，返回对应的处理结果 */
export function normalizeSkill(item: BoundSkillItem): SkillFull {
  // 内置和 AI 技能保留专属来源，其他来源按可编辑的用户技能展示
  const source = item.source === 'builtin' || item.source === 'ai' ? item.source : 'user'
  const body = item.body ?? '' // 正文

  return { name: item.name, description: item.description, source, disabled: item.disabled, has_body: body.length > 0, body }
}
