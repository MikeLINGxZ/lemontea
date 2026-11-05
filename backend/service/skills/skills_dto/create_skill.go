package skills_dto

// CreateSkillInput 包含新建用户技能所需的字段
type CreateSkillInput struct {
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 正文
	Body string `json:"body"`
}

// CreateSkillOutput 返回新建的技能
type CreateSkillOutput struct {
	// 技能
	Skill SkillItem `json:"skill"`
}
