package tools

import (
	"context"
	"encoding/json"
	"errors"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// AttentionRequester 创建请求用户关注的通知，并等待用户处理
type AttentionRequester interface {
	// 通知注意提醒，接收本次调用的上下文、会话编号、标题、消息，返回无符号整数及可能发生的错误
	NotifyAttention(ctx context.Context, sessionID uint, title, message string) (uint, error)
	// 等待对应处理结果，接收本次调用的上下文、通知编号，返回操作错误
	WaitForResolution(ctx context.Context, notificationID uint) error
}

// RequestAttentionToolName 是暴露给任务会话的内置工具名称
const RequestAttentionToolName = "RequestUserAttention"

// 封装处理请求注意提醒所需的请求参数
type requestAttentionInput struct {
	// 标题
	Title string `json:"title" jsonschema:"description=Short headline shown to the user,required"`
	// 消息
	Message string `json:"message" jsonschema:"description=Detailed request for the user,required"`
}

// BuildRequestAttentionTool 返回供注册表使用的用户关注工具元数据
func BuildRequestAttentionTool() ToolMeta {
	return ToolMeta{
		Name:        RequestAttentionToolName,
		Description: "Pause the task and ask the user for help or clarification.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input requestAttentionInput
			_ = json.Unmarshal(args, &input)
			return "Ask user for help: " + input.Title
		},
	}
}

// InvokeRequestAttention 创建通知并阻塞等待用户处理
// 接收本次调用的上下文、请求器、会话编号、参数，返回文本及可能发生的错误
func InvokeRequestAttention(ctx context.Context, requester AttentionRequester, sessionID uint, args json.RawMessage) (string, error) {
	// 请求参数
	var input requestAttentionInput
	decodeErr := json.Unmarshal(args, &input) // 解析序列化内容时的错误
	if decodeErr != nil {
		return "", decodeErr
	}
	if input.Title == "" {
		return "", errors.New("title is required")
	}
	// 通知编号、通知注意提醒时的错误
	notificationID, err := requester.NotifyAttention(ctx, sessionID, input.Title, input.Message)
	if err != nil {
		return "", err
	}
	waitErr := requester.WaitForResolution(ctx, notificationID) // 等待对应处理结果时的错误
	if waitErr != nil {
		return "", waitErr
	}
	return "user replied; resume from the latest message in this session", nil
}

// NewRequestAttentionTool 创建绑定到单个任务会话的函数工具
// 接收请求器、会话编号，返回Function工具请求注意提醒输入string
func NewRequestAttentionTool(requester AttentionRequester, sessionID uint) *function.FunctionTool[requestAttentionInput, string] {
	meta := BuildRequestAttentionTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input requestAttentionInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, err := json.Marshal(input) // 载荷、将数据序列化时的错误
			if err != nil {
				return "", err
			}
			return InvokeRequestAttention(ctx, requester, sessionID, payload)
		},
		function.WithName(RequestAttentionToolName),
		function.WithDescription(meta.Description),
	)
}
