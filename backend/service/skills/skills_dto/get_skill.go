package skills_dto

// GetSkillInput 按名称指定要查询的技能
type GetSkillInput struct {
	// 名称
	Name string `json:"name"`
}

// GetSkillOutput 返回查询到的技能
type GetSkillOutput struct {
	// 技能
	Skill SkillItem `json:"skill"`
}
