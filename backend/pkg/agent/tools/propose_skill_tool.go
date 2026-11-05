package tools

import (
	"context"
	"encoding/json"
	"errors"

	pkgskills "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/skills"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// SkillCreator 定义保存新技能提案的接口
type SkillCreator interface {
	// 创建技能创建器，接收技能，返回技能及可能发生的错误
	Create(skill pkgskills.Skill) (pkgskills.Skill, error)
}

// ProposeSkillToolName 是暴露给任务会话的内置工具名称
const ProposeSkillToolName = "ProposeSkill"

// 封装处理提议技能所需的请求参数
type proposeSkillInput struct {
	// 名称
	Name string `json:"name" jsonschema:"description=Kebab-case skill name,required"`
	// 描述
	Description string `json:"description" jsonschema:"description=Short description shown to the user,required"`
	// 正文
	Body string `json:"body" jsonschema:"description=Markdown skill body,required"`
}

// BuildProposeSkillTool 返回供注册表使用的技能提案工具元数据
func BuildProposeSkillTool() ToolMeta {
	return ToolMeta{
		Name:        ProposeSkillToolName,
		Description: "Propose a new AI-generated skill. The user must confirm before it is saved.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input proposeSkillInput
			_ = json.Unmarshal(args, &input)
			return "Propose skill: " + input.Name
		},
	}
}

// InvokeProposeSkill 在保存 AI 生成的技能前请求用户确认
// 接收本次调用的上下文、gate 对应的数据、创建器、会话编号、参数，返回文本及可能发生的错误
func InvokeProposeSkill(ctx context.Context, gate AttentionRequester, creator SkillCreator, sessionID uint, args json.RawMessage) (string, error) {
	// 请求参数
	var input proposeSkillInput
	decodeErr := json.Unmarshal(args, &input) // 解析序列化内容时的错误
	if decodeErr != nil {
		return "", decodeErr
	}
	if input.Name == "" || input.Description == "" || input.Body == "" {
		return "", errors.New("name, description, body are required")
	}
	// 通知编号、通知注意提醒时的错误
	notificationID, err := gate.NotifyAttention(ctx, sessionID, "Confirm new skill: "+input.Name, input.Description)
	if err != nil {
		return "", err
	}
	waitErr := gate.WaitForResolution(ctx, notificationID) // 等待对应处理结果时的错误
	if waitErr != nil {
		return "", waitErr
	}
	// 创建当前数据时的错误
	_, createErr := creator.Create(pkgskills.Skill{
		Name:        input.Name,
		Description: input.Description,
		Body:        input.Body,
		Source:      pkgskills.SourceAI,
	})
	if createErr != nil {
		return "", createErr
	}
	return "skill " + input.Name + " saved as ai-generated", nil
}

// NewProposeSkillTool 创建绑定到单个任务会话的函数工具
// 接收gate 对应的数据、创建器、会话编号，返回Function工具提议技能输入string
func NewProposeSkillTool(gate AttentionRequester, creator SkillCreator, sessionID uint) *function.FunctionTool[proposeSkillInput, string] {
	meta := BuildProposeSkillTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input proposeSkillInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, err := json.Marshal(input) // 载荷、将数据序列化时的错误
			if err != nil {
				return "", err
			}
			return InvokeProposeSkill(ctx, gate, creator, sessionID, payload)
		},
		function.WithName(ProposeSkillToolName),
		function.WithDescription(meta.Description),
	)
}
