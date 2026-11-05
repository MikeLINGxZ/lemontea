// skills 包定义磁盘技能模型并负责技能文件读写
package skills

import "time"

// Source 标识技能来源
type Source string

const (
	// SourceBuiltin 表示随应用发布的内置技能
	SourceBuiltin Source = "builtin"
	// SourceUser 表示由用户安装的技能
	SourceUser Source = "user"
	// SourceAI 表示由 AI 助手生成的技能
	SourceAI Source = "ai"
)

// Skill 是一项技能在内存中的表示
// Body 是 YAML 元数据之后的 Markdown 正文
type Skill struct {
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 正文
	Body string `json:"body"`
	// 来源
	Source Source `json:"source"`
	// 禁用状态
	Disabled bool `json:"disabled"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// SidecarManifest 表示与 SKILL.md 同目录保存的 .lemontea.json 元数据
// 内置技能没有独立的元数据文件
type SidecarManifest struct {
	// 来源
	Source Source `json:"source"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}
