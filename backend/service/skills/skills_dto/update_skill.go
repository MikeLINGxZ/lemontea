package skills_dto

// UpdateSkillInput 包含更新已有用户技能所需的字段
type UpdateSkillInput struct {
	// 名称
	Name string `json:"name"`
	// 新名称
	NewName string `json:"new_name,omitempty"`
	// 描述
	Description string `json:"description"`
	// 正文
	Body string `json:"body"`
}

// UpdateSkillOutput 返回更新后的技能
type UpdateSkillOutput struct {
	// 技能
	Skill SkillItem `json:"skill"`
}
