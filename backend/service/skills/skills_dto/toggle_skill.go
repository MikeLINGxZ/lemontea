package skills_dto

// ToggleSkillInput 指定技能及其目标禁用状态
type ToggleSkillInput struct {
	// 名称
	Name string `json:"name"`
	// 禁用状态
	Disabled bool `json:"disabled"`
}

// ToggleSkillOutput 返回禁用状态更新后的技能
type ToggleSkillOutput struct {
	// 技能
	Skill SkillItem `json:"skill"`
}
