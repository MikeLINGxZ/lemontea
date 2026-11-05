package tools

import (
	"context"
	"encoding/json"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 交互式终端工具名称
const InteractiveTerminalToolName = "InteractiveTerminal"

// 封装处理交互式终端所需的请求参数
type interactiveTerminalInput struct {
	// 当前状态
	Status string `json:"status" jsonschema:"enum=active,enum=done,description=active shows the terminal panel; done hides it when the user-assisted step is complete,required"`
	// 输出
	Output string `json:"output" jsonschema:"description=terminal text to show when status is active"`
}

// 封装处理交互式终端返回给调用方的数据
type interactiveTerminalOutput struct {
	// 交互式终端
	InteractiveTerminal bool `json:"interactive_terminal"`
	// 终端状态
	TerminalStatus string `json:"terminal_status"`
	// 终端输出
	TerminalOutput string `json:"terminal_output,omitempty"`
}

// 构造交互式终端工具，返回工具元信息
func BuildInteractiveTerminalTool() ToolMeta {
	return ToolMeta{
		Name:        InteractiveTerminalToolName,
		Description: "Control a user-facing interactive terminal panel. Use status=active when the user must scan a QR code, open a login URL, copy a code, or provide input outside chat. Call status=done once that user-assisted step is complete so the panel can hide.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input interactiveTerminalInput
			_ = json.Unmarshal(args, &input)
			if strings.EqualFold(strings.TrimSpace(input.Status), "done") {
				return "Hide interactive terminal"
			}
			return "Show interactive terminal"
		},
	}
}

// 构建交互式终端工具，返回Function工具交互式终端输入交互式终端输出
func NewInteractiveTerminalTool() *function.FunctionTool[interactiveTerminalInput, interactiveTerminalOutput] {
	meta := BuildInteractiveTerminalTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input interactiveTerminalInput) (interactiveTerminalOutput, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回操作结果及可能发生的错误
			status := strings.ToLower(strings.TrimSpace(input.Status)) // 状态
			if status != "done" {
				status = "active"
			}
			return interactiveTerminalOutput{
				InteractiveTerminal: true,
				TerminalStatus:      status,
				TerminalOutput:      input.Output,
			}, nil
		},
		function.WithName(InteractiveTerminalToolName),
		function.WithDescription(meta.Description),
	)
}
