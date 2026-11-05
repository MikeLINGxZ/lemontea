package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
)

// 默认单次请求超时
const defaultOneshotTimeout = 60 * time.Second

// OneshotRequest 描述一次非流式模型请求
type OneshotRequest struct {
	// 服务基础地址
	BaseURL string
	// API 密钥
	APIKey string
	// 模型名称
	ModelName string
	// 提供商类型
	ProviderType pkgProvider.Type
	// 系统
	System string
	// 用户
	User string
	// 上限Token
	MaxTokens int
	// 超时
	Timeout time.Duration
}

// OneshotResponse 包含一次模型回复和基本用量数据
type OneshotResponse struct {
	// 文本
	Text string
	// 提示词Token
	PromptTokens int
	// 完成状态Token
	CompletionTokens int
}

// 发起一次不绑定工具的非流式模型请求，接收上下文和模型参数，返回模型文本、用量及可能发生的错误
// 先按调用配置选择模型和超时，再合并响应事件，遇到模型错误或取消时提前结束
func OneshotComplete(ctx context.Context, req OneshotRequest) (OneshotResponse, error) {
	timeout := req.Timeout // 本次操作的最长等待时长
	if timeout <= 0 {
		timeout = defaultOneshotTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout) // 调用上下文、用于取消本次操作的回调
	defer cancel()

	// 模型
	mdl := openai.New(
		req.ModelName,
		openai.WithBaseURL(req.BaseURL),
		openai.WithAPIKey(req.APIKey),
		openai.WithVariant(inferOpenAIVariant(req.ProviderType)),
	)

	messages := make([]model.Message, 0, 2) // 消息列表
	if strings.TrimSpace(req.System) != "" {
		messages = append(messages, model.NewSystemMessage(req.System))
	}
	messages = append(messages, model.NewUserMessage(req.User))

	// 请求
	request := &model.Request{
		Messages: messages,
		GenerationConfig: model.GenerationConfig{
			Stream: false,
		},
	}
	if req.MaxTokens > 0 {
		request.MaxTokens = &req.MaxTokens
	}

	responseChan, err := mdl.GenerateContent(callCtx, request) // 响应通道、处理生成内容时的错误
	if err != nil {
		if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			return OneshotResponse{}, callCtx.Err()
		}
		return OneshotResponse{}, err
	}

	// 构建器
	var builder strings.Builder
	// 用量
	var usage *model.Usage
	// 遍历响应通道，逐项取得响应
	for response := range responseChan {
		if response == nil {
			continue
		}
		if response.Error != nil {
			return OneshotResponse{}, fmt.Errorf("oneshot llm error: %s", response.Error.Message)
		}
		if response.Usage != nil {
			usage = response.Usage
		}
		// 遍历Choices 对应的数据，逐项取得选择
		for _, choice := range response.Choices {
			if choice.Message.Content != "" {
				builder.WriteString(choice.Message.Content)
				continue
			}
			if choice.Delta.Content != "" {
				builder.WriteString(choice.Delta.Content)
			}
		}
	}
	if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
		return OneshotResponse{}, callCtx.Err()
	}

	// 本次处理的输出
	out := OneshotResponse{
		Text: strings.TrimSpace(builder.String()),
	}
	if usage != nil {
		out.PromptTokens = usage.PromptTokens
		out.CompletionTokens = usage.CompletionTokens
	}
	return out, nil
}
