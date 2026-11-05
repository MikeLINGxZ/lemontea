package skills_dto

// DeleteSkillInput 按名称指定要删除的技能
type DeleteSkillInput struct {
	// 名称
	Name string `json:"name"`
}

// DeleteSkillOutput 表示指定技能已删除
type DeleteSkillOutput struct{}
