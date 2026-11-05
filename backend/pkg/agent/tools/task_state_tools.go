package tools

import (
	"context"
	"encoding/json"
	"errors"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// TaskStateStore 定义持久化隐藏或自动化会话中可恢复的任务状态的接口
type TaskStateStore interface {
	// 保存任务状态，接收会话编号、键、文本，返回操作错误
	SaveTaskState(sessionID uint, key, value string) error
	// 加载任务状态，接收会话编号、键，返回文本、是否找到、操作错误
	LoadTaskState(sessionID uint, key string) (value string, found bool, err error)
}

const (
	// 保存任务状态工具名称
	SaveTaskStateToolName = "SaveTaskState"
	// 加载任务状态工具名称
	LoadTaskStateToolName = "LoadTaskState"
)

// 封装保存任务状态所需的请求参数
type saveTaskStateInput struct {
	// 键
	Key string `json:"key" jsonschema:"description=state key,required"`
	// 值
	Value string `json:"value" jsonschema:"description=state value string,required"`
}

// 封装加载任务状态所需的请求参数
type loadTaskStateInput struct {
	// 键
	Key string `json:"key" jsonschema:"description=state key,required"`
}

// 构造保存任务状态工具，返回工具元信息
func BuildSaveTaskStateTool() ToolMeta {
	return ToolMeta{
		Name:        SaveTaskStateToolName,
		Description: "Persist resumable state for the current task session. Use for device codes, verification URLs, expiry timestamps, and other cross-turn init state.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input saveTaskStateInput
			_ = json.Unmarshal(args, &input)
			return "Save task state: " + input.Key
		},
	}
}

// 构造加载任务状态工具，返回工具元信息
func BuildLoadTaskStateTool() ToolMeta {
	return ToolMeta{
		Name:        LoadTaskStateToolName,
		Description: "Load previously persisted state for the current task session.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input loadTaskStateInput
			_ = json.Unmarshal(args, &input)
			return "Load task state: " + input.Key
		},
	}
}

// 调用保存任务状态，接收本次调用的上下文、状态仓库、会话编号、参数，返回文本及可能发生的错误
func InvokeSaveTaskState(ctx context.Context, store TaskStateStore, sessionID uint, args json.RawMessage) (string, error) {
	// 请求参数
	var input saveTaskStateInput
	decodeErr := json.Unmarshal(args, &input) // 解析序列化内容时的错误
	if decodeErr != nil {
		return "", decodeErr
	}
	if input.Key == "" {
		return "", errors.New("key is required")
	}
	saveErr := store.SaveTaskState(sessionID, input.Key, input.Value) // 保存任务状态时的错误
	if saveErr != nil {
		return "", saveErr
	}
	return "saved state for key: " + input.Key, nil
}

// 调用加载任务状态，接收本次调用的上下文、状态仓库、会话编号、参数，返回文本及可能发生的错误
func InvokeLoadTaskState(ctx context.Context, store TaskStateStore, sessionID uint, args json.RawMessage) (string, error) {
	// 请求参数
	var input loadTaskStateInput
	decodeErr := json.Unmarshal(args, &input) // 解析序列化内容时的错误
	if decodeErr != nil {
		return "", decodeErr
	}
	if input.Key == "" {
		return "", errors.New("key is required")
	}
	// 待处理的值、是否找到、加载任务状态时的错误
	value, found, err := store.LoadTaskState(sessionID, input.Key)
	if err != nil {
		return "", err
	}
	// 载荷
	payload, err := json.Marshal(map[string]any{
		"key":   input.Key,
		"found": found,
		"value": value,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// 构建保存任务状态工具，接收状态仓库、会话编号，返回Function工具保存任务状态输入string
func NewSaveTaskStateTool(store TaskStateStore, sessionID uint) *function.FunctionTool[saveTaskStateInput, string] {
	meta := BuildSaveTaskStateTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input saveTaskStateInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, err := json.Marshal(input) // 载荷、将数据序列化时的错误
			if err != nil {
				return "", err
			}
			return InvokeSaveTaskState(ctx, store, sessionID, payload)
		},
		function.WithName(SaveTaskStateToolName),
		function.WithDescription(meta.Description),
	)
}

// 构建加载任务状态工具，接收状态仓库、会话编号，返回Function工具加载任务状态输入string
func NewLoadTaskStateTool(store TaskStateStore, sessionID uint) *function.FunctionTool[loadTaskStateInput, string] {
	meta := BuildLoadTaskStateTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input loadTaskStateInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, err := json.Marshal(input) // 载荷、将数据序列化时的错误
			if err != nil {
				return "", err
			}
			return InvokeLoadTaskState(ctx, store, sessionID, payload)
		},
		function.WithName(LoadTaskStateToolName),
		function.WithDescription(meta.Description),
	)
}
