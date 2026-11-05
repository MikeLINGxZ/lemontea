package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pkgskills "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/skills"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// SkillProvider 抽象技能管理器，让智能体可以列出并加载技能
// 通过接口访问，避免直接导入技能管理器的实现包
type SkillProvider interface {
	// Enabled 返回智能体可用的全部已启用技能
	Enabled() []pkgskills.Skill
	// Get 按名称返回单个技能，找不到时返回 false
	Get(name string) (pkgskills.Skill, bool)
}

// SkillToolName 是智能体使用的 Skill 元工具的标准名称
const SkillToolName = "Skill"

// skillInput 定义 Skill 工具接收的 JSON 参数
type skillInput struct {
	// 名称
	Name string `json:"name" jsonschema:"description=Name of the skill to load,required"`
}

// BuildSkillTool 构建 Skill 元工具的 ToolMeta
// 每次调用都重新生成工具描述
// 使描述始终反映当前启用的技能
// 接收提供商，返回工具元信息
func BuildSkillTool(provider SkillProvider) ToolMeta {
	enabled := provider.Enabled() // 是否启用
	// 文本构建器
	var sb strings.Builder
	sb.WriteString("Load a skill (a set of natural-language instructions for a specific task). Call with {\"name\": \"<skill-name>\"}. Available skills:\n")
	if len(enabled) == 0 {
		sb.WriteString("(none)\n")
	}
	// 遍历是否启用，逐项取得当前值
	for _, s := range enabled {
		sb.WriteString("- ")
		sb.WriteString(s.Name)
		sb.WriteString(": ")
		sb.WriteString(s.Description)
		sb.WriteString("\n")
	}
	return ToolMeta{
		Name:        SkillToolName,
		Description: sb.String(),
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 解析结果
			var parsed skillInput
			_ = json.Unmarshal(args, &parsed)
			return "Loading skill: " + parsed.Name
		},
	}
}

// InvokeSkill 返回指定技能的 SKILL.md 正文
// 模型调用 Skill({name}) 时由智能体的工具调用处理器使用
// 接收调用上下文、提供商、参数，返回文本及可能发生的错误
func InvokeSkill(_ context.Context, provider SkillProvider, args json.RawMessage) (string, error) {
	// 解析结果
	var parsed skillInput
	err := json.Unmarshal(args, &parsed) // 解析序列化内容时的错误
	if err != nil {
		return "", err
	}
	if parsed.Name == "" {
		return "", errors.New("skill name is required")
	}
	sk, ok := provider.Get(parsed.Name) // 技能、是否成功取得对应值
	if !ok {
		return "", errors.New("skill not found: " + parsed.Name)
	}
	if sk.Disabled {
		return "", errors.New("skill is disabled: " + parsed.Name)
	}
	return sk.Body, nil
}

// NewSkillTool 将 Skill 元工具构建为供模型调用的函数工具
// 根据提供者当前启用的技能生成工具描述
// 使模型始终看到最新的技能列表
// 接收提供商，返回Function工具技能输入string
func NewSkillTool(provider SkillProvider) *function.FunctionTool[skillInput, string] {
	meta := BuildSkillTool(provider) // 元信息
	return function.NewFunctionTool(
		/* 构建Function工具时调用的回调，接收调用上下文、技能输入，返回文本及可能发生的错误 */ func(_ context.Context, input skillInput) (string, error) {
			if input.Name == "" {
				return "", errors.New("skill name is required")
			}
			sk, ok := provider.Get(input.Name) // 技能、是否成功取得对应值
			if !ok {
				return "", errors.New("skill not found: " + input.Name)
			}
			if sk.Disabled {
				return "", errors.New("skill is disabled: " + input.Name)
			}
			return sk.Body, nil
		},
		function.WithName(SkillToolName),
		function.WithDescription(meta.Description),
	)
}
