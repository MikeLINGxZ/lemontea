package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 询问工具名称
const QuestionToolName = "question"

// 描述询问选项使用的数据结构
type questionOption struct {
	// 标签
	Label string `json:"label" jsonschema:"description=Short option label,required"`
	// 描述
	Description string `json:"description" jsonschema:"description=What choosing this option means"`
}

// 封装处理询问所需的请求参数
type questionInput struct {
	// 询问
	Question string `json:"question" jsonschema:"description=Clear user-facing question to ask,required"`
	// 上下文
	Context string `json:"context" jsonschema:"description=Brief reason why the question is needed"`
	// 选项列表
	Options []questionOption `json:"options" jsonschema:"description=Optional choices for the user"`
}

// 封装处理询问返回给调用方的数据
type questionOutput struct {
	// 询问
	Question string `json:"question"`
	// 上下文
	Context string `json:"context,omitempty"`
	// 选项列表
	Options []questionOption `json:"options,omitempty"`
	// 指令
	Instruction string `json:"instruction"`
}

// BuildQuestionTool 返回供注册表使用的澄清工具元数据
func BuildQuestionTool() ToolMeta {
	return ToolMeta{
		Name:        QuestionToolName,
		Description: "Ask the user a clear clarification question when the request is incomplete, risky, or needs a preference choice.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收原始值消息，返回文本 */ func(_ json.RawMessage) string {
			return "Ask the user a clarification question"
		},
	}
}

// questionFunc 校验并格式化要展示给用户的澄清请求
// 接收调用上下文、询问输入，返回操作结果及可能发生的错误
func questionFunc(_ context.Context, input questionInput) (questionOutput, error) {
	question := strings.TrimSpace(input.Question) // 询问
	if question == "" {
		return questionOutput{}, errors.New("question is required")
	}
	return questionOutput{
		Question:    question,
		Context:     strings.TrimSpace(input.Context),
		Options:     input.Options,
		Instruction: "Ask the user this question directly, then wait for their answer before continuing.",
	}, nil
}

// NewQuestionTool 创建用于组织用户澄清请求的函数工具
// 返回Function工具询问输入询问输出
func NewQuestionTool() *function.FunctionTool[questionInput, questionOutput] {
	meta := BuildQuestionTool() // 元信息
	return function.NewFunctionTool(
		questionFunc,
		function.WithName(QuestionToolName),
		function.WithDescription(meta.Description),
	)
}
