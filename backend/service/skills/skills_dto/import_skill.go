package skills_dto

// ImportSkillInput 提供待导入的 SKILL.md 本地路径
type ImportSkillInput struct {
	// 路径
	Path string `json:"path"`
}

// ImportSkillOutput 返回导入的技能
type ImportSkillOutput struct {
	// 技能
	Skill SkillItem `json:"skill"`
}
