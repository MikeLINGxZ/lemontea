package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 待办写入工具名称
const TodoWriteToolName = "todo_write"

// 描述待办写入条目使用的数据结构
type todoWriteItem struct {
	// 标题
	Title string `json:"title" jsonschema:"description=Short task title,required"`
	// 当前状态
	Status string `json:"status" jsonschema:"description=Task status: pending, in_progress, or completed,required"`
}

// 封装处理待办写入所需的请求参数
type todoWriteInput struct {
	// 条目列表
	Items []todoWriteItem `json:"items" jsonschema:"description=Current task list,required"`
}

// 封装处理待办写入返回给调用方的数据
type todoWriteOutput struct {
	// 总量
	Total int `json:"total"`
	// 已完成
	Completed int `json:"completed"`
	// 输入进度
	InProgress int `json:"in_progress"`
	// 待处理
	Pending int `json:"pending"`
	// 条目列表
	Items []todoWriteItem `json:"items"`
	// 摘要
	Summary string `json:"summary"`
}

// BuildTodoWriteTool 返回供注册表使用的任务进度工具元数据
func BuildTodoWriteTool() ToolMeta {
	return ToolMeta{
		Name:        TodoWriteToolName,
		Description: "Track progress for a multi-step user task and keep the assistant organized.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收原始值消息，返回文本 */ func(_ json.RawMessage) string {
			return "Update task progress"
		},
	}
}

// todoWriteFunc 校验任务列表并返回简明的进度摘要
// 接收调用上下文、待办写入输入，返回操作结果及可能发生的错误
func todoWriteFunc(_ context.Context, input todoWriteInput) (todoWriteOutput, error) {
	if len(input.Items) == 0 {
		return todoWriteOutput{}, errors.New("at least one todo item is required")
	}
	out := todoWriteOutput{Total: len(input.Items), Items: input.Items} // 本次处理的输出
	// 遍历条目列表，逐项取得当前条目
	for _, item := range input.Items {
		switch strings.TrimSpace(item.Status) {
		case "completed":
			out.Completed++
		case "in_progress":
			out.InProgress++
		default:
			out.Pending++
		}
	}
	out.Summary = fmt.Sprintf("Progress: %d completed, %d in progress, %d pending.", out.Completed, out.InProgress, out.Pending)
	return out, nil
}

// NewTodoWriteTool 创建用于跟踪任务进度的函数工具
// 返回Function工具待办写入输入待办写入输出
func NewTodoWriteTool() *function.FunctionTool[todoWriteInput, todoWriteOutput] {
	meta := BuildTodoWriteTool() // 元信息
	return function.NewFunctionTool(
		todoWriteFunc,
		function.WithName(TodoWriteToolName),
		function.WithDescription(meta.Description),
	)
}
