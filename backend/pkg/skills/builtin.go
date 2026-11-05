package skills

import (
	"embed"
	"io/fs"
	"path"
	"sort"
)

// 内置FS
//
//go:embed all:_builtin
var builtinFS embed.FS

// LoadBuiltin 返回嵌入可执行文件的全部技能，并将来源标为 SourceBuiltin
// _builtin/ 下每个包含 SKILL.md 的一级子目录对应一项技能
// 缺少可解析 SKILL.md 的子目录会被静默跳过，
// 以免一项损坏的技能阻止其余技能加载
func LoadBuiltin() ([]Skill, error) {
	entries, err := fs.ReadDir(builtinFS, "_builtin") // 条目列表、读取目录时的错误
	if err != nil {
		return nil, err
	}
	// 技能列表
	var skills []Skill
	// 遍历条目列表，逐项取得条目
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// 只有包含 SKILL.md 的目录才会作为技能加载
		mdPath := path.Join("_builtin", entry.Name(), SkillFileName)
		raw, err := fs.ReadFile(builtinFS, mdPath) // 尚未解析的原始数据、读取文件内容时的错误
		if err != nil {
			continue
		}
		skill, err := Parse(raw) // 技能
		if err != nil {
			continue
		}
		skill.Source = SourceBuiltin
		skills = append(skills, skill)
	}
	sort.Slice(skills /* 比较两个索引对应的条目，返回前一项是否应排在前面 */, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}
