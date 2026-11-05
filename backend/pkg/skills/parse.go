package skills

import (
	"bytes"
	"errors"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// skillNameRe 约束技能名称采用 kebab-case，最多 64 个小写字母、数字或连字符
var skillNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

var (
	// ErrInvalidSkillName 表示技能名称缺失或不符合 kebab-case 规则
	ErrInvalidSkillName = errors.New("skill name must be kebab-case (a-z0-9- only, max 64 chars)")
	// ErrInvalidSkillContent 表示技能缺少必需内容
	ErrInvalidSkillContent = errors.New("skill description is required")
	// ErrSkillNameTaken 表示请求使用的名称已被另一项技能占用
	ErrSkillNameTaken = errors.New("skill name is already taken")
)

// Frontmatter 是每个 SKILL.md 开头的 YAML 元数据
type Frontmatter struct {
	// 名称
	Name string `yaml:"name"`
	// 描述
	Description string `yaml:"description"`
}

// Parse 将 SKILL.md 的字节切片解析为 Skill，Source、Disabled 和 UpdatedAt 留待填充
// 调用方负责根据加载上下文填充这些字段
// 接收尚未解析的原始数据，返回技能及可能发生的错误
// 解析序列化内容
// 去掉首尾空白
func Parse(raw []byte) (Skill, error) {
	// 检查开头的 "---\n" 分隔符
	const delim = "---"
	trimmed := bytes.TrimLeft(raw, "\r\n ") // 已去除空白
	if !bytes.HasPrefix(trimmed, []byte(delim)) {
		return Skill{}, errors.New("missing YAML frontmatter")
	}
	// 跳过开头的分隔符
	body := trimmed[len(delim):]
	// 跳过首个分隔符后的可选换行
	body = bytes.TrimLeft(body, "\r\n")
	// 找到独占一行的结束分隔符
	end := bytes.Index(body, []byte("\n"+delim))
	if end < 0 {
		return Skill{}, errors.New("frontmatter not closed")
	}
	headerBytes := body[:end]          // 页头字节
	rest := body[end+len("\n"+delim):] // rest 对应的数据
	rest = bytes.TrimLeft(rest, "\r\n")

	// 解析 YAML 元数据块
	var fm Frontmatter
	err := yaml.Unmarshal(headerBytes, &fm) // 解析序列化内容时的错误
	if err != nil {
		return Skill{}, err
	}
	fm.Name = strings.TrimSpace(fm.Name)
	fm.Description = strings.TrimSpace(fm.Description)
	if fm.Name == "" {
		return Skill{}, ErrInvalidSkillName
	}
	if fm.Description == "" {
		return Skill{}, ErrInvalidSkillContent
	}
	if !skillNameRe.MatchString(fm.Name) {
		return Skill{}, ErrInvalidSkillName
	}
	return Skill{
		Name:        fm.Name,
		Description: fm.Description,
		Body:        string(rest),
	}, nil
}

// IsValidName 判断 name 是否符合技能的 kebab-case 命名规则
// 接收当前名称，返回是否满足条件
func IsValidName(name string) bool {
	return skillNameRe.MatchString(strings.TrimSpace(name))
}

// Validate 检查持久化 Skill 所需的字段
// 接收技能，返回操作错误
func Validate(s Skill) error {
	if !IsValidName(s.Name) {
		return ErrInvalidSkillName
	}
	if strings.TrimSpace(s.Description) == "" {
		return ErrInvalidSkillContent
	}
	return nil
}

// Render 生成写入磁盘的 SKILL.md 字节，包含元数据和正文
// 接收技能，返回字节数据及可能发生的错误
func Render(s Skill) ([]byte, error) {
	header, err := yaml.Marshal(Frontmatter{Name: s.Name, Description: s.Description}) // 页头、将数据序列化时的错误
	if err != nil {
		return nil, err
	}
	// 缓冲区
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(header)
	buf.WriteString("---\n\n")
	buf.WriteString(strings.TrimRight(s.Body, "\n"))
	buf.WriteString("\n")
	return buf.Bytes(), nil
}
