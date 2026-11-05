package skills

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	// SkillFileName 是技能 Markdown 清单的约定文件名
	SkillFileName = "SKILL.md"
	// SidecarFileName 是随每项磁盘技能保存的 JSON 元数据文件名
	SidecarFileName = ".lemontea.json"
)

// LoadFromDir 扫描 root 下的技能子目录，并按名称排序后返回技能
// 缺少可解析 SKILL.md 的子目录会被静默跳过，
// 以免一项损坏的技能阻止其余技能加载
// 接收根目录，返回技能列表及可能发生的错误
func LoadFromDir(root string) ([]Skill, error) {
	entries, err := os.ReadDir(root) // 条目列表、读取目录时的错误
	if err != nil {
		// 根目录不存在表示用户尚无技能，无需报错
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	// 技能列表
	var skills []Skill
	// 遍历条目列表，逐项取得条目
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name()) // 目录
		skill, err := loadOneDir(dir)            // 技能、加载One目录时的错误
		if err != nil {
			// 跳过无法读取或格式无效的技能目录，继续扫描
			continue
		}
		skills = append(skills, skill)
	}
	sort.Slice(skills /* 比较两个索引对应的条目，返回前一项是否应排在前面 */, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

// loadOneDir 从单个技能目录读取 SKILL.md 和可选元数据文件并返回 Skill，
// 加载时 Disabled 一律为 false
// 接收目录，返回技能及可能发生的错误
func loadOneDir(dir string) (Skill, error) {
	raw, err := os.ReadFile(filepath.Join(dir, SkillFileName)) // 尚未解析的原始数据、读取文件内容时的错误
	if err != nil {
		return Skill{}, err
	}
	skill, err := Parse(raw) // 技能
	if err != nil {
		return Skill{}, err
	}
	// 默认来源为用户，可由元数据文件覆盖
	skill.Source = SourceUser
	sidecar, sErr := ReadSidecar(dir) // 辅助进程、读取辅助进程时的错误
	if sErr == nil {
		if sidecar.Source != "" {
			skill.Source = sidecar.Source
		}
		if !sidecar.UpdatedAt.IsZero() {
			skill.UpdatedAt = sidecar.UpdatedAt
		}
	} else {
		// 没有元数据文件时，使用 SKILL.md 的修改时间
		info, statErr := os.Stat(filepath.Join(dir, SkillFileName))
		if statErr == nil {
			skill.UpdatedAt = info.ModTime()
		}
	}
	return skill, nil
}

// ReadSidecar 从 dir 读取可选的 .lemontea.json 元数据文件
// 文件缺失或格式无效时返回错误，
// 由调用方决定是否使用默认值
// 接收目录，返回辅助进程清单及可能发生的错误
func ReadSidecar(dir string) (SidecarManifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, SidecarFileName)) // 尚未解析的原始数据、读取文件内容时的错误
	if err != nil {
		return SidecarManifest{}, err
	}
	// sc 对应的数据
	var sc SidecarManifest
	decodeErr := json.Unmarshal(raw, &sc) // 解析序列化内容时的错误
	if decodeErr != nil {
		return SidecarManifest{}, decodeErr
	}
	return sc, nil
}

// WriteSidecar 将元数据以缩进的 JSON 格式保存到 dir
// 调用方可省略 UpdatedAt，零值会自动填入 time.Now()
// 接收目录、sc 对应的数据，返回操作错误
func WriteSidecar(dir string, sc SidecarManifest) error {
	if sc.UpdatedAt.IsZero() {
		sc.UpdatedAt = time.Now()
	}
	raw, err := json.MarshalIndent(sc, "", "  ") // 尚未解析的原始数据、序列化Indent 对应的数据时的错误
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, SidecarFileName), raw, 0o644)
}

// WriteSkill 将 SKILL.md 和元数据文件写入 {root}/{skill.Name}/
// 目录不存在时会创建，并保留现有元数据文件中的 CreatedAt，
// 以免重复写入时重置创建时间
// 接收根目录、技能，返回操作错误
func WriteSkill(root string, s Skill) error {
	validateErr := Validate(s) // 校验当前数据时的错误
	if validateErr != nil {
		return validateErr
	}
	dir := filepath.Join(root, s.Name)  // 目录
	mkdirErr := os.MkdirAll(dir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return mkdirErr
	}
	rendered, err := Render(s) // rendered 对应的数据、渲染当前数据时的错误
	if err != nil {
		return err
	}
	writeErr := os.WriteFile(filepath.Join(dir, SkillFileName), rendered, 0o644) // 写入文件内容时的错误
	if writeErr != nil {
		return writeErr
	}
	source := s.Source // 来源
	if source == "" {
		source = SourceUser
	}
	sc := SidecarManifest{Source: source, UpdatedAt: time.Now()} // sc 对应的数据
	// 已有元数据文件时保留原始 CreatedAt
	existing, readErr := ReadSidecar(dir)
	if readErr == nil {
		sc.CreatedAt = existing.CreatedAt
	}
	if sc.CreatedAt.IsZero() {
		sc.CreatedAt = time.Now()
	}
	return WriteSidecar(dir, sc)
}

// DeleteSkill 删除整个技能目录 {root}/{name}/
// 目录不存在时返回 nil
// 接收根目录、当前名称，返回操作错误
func DeleteSkill(root, name string) error {
	return os.RemoveAll(filepath.Join(root, name))
}
