package skills_dto

// ListSkillsInput 是查询所有技能的请求
type ListSkillsInput struct{}

// ListSkillsOutput 返回所有已知技能
type ListSkillsOutput struct {
	// 技能列表
	Skills []SkillItem `json:"skills"`
}

// SkillItem 表示返回给前端的一项技能
type SkillItem struct {
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 正文
	Body string `json:"body,omitempty"`
	// 来源
	Source string `json:"source"`
	// 禁用状态
	Disabled bool `json:"disabled"`
}
