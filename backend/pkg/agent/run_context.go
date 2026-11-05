package agent

import (
	"context"

	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
)

// 描述运行上下文键使用的数据结构
type runContextKey struct{}

// 描述运行上下文使用的数据结构
type runContext struct {
	// 会话编号
	SessionID uint
	// 模型名称
	ModelName string
	// 服务基础地址
	BaseURL string
	// API 密钥
	APIKey string
	// 提供商类型
	ProviderType pkgProvider.Type
}

// 描述工具确认使用的数据结构
type toolConfirmation struct {
	// 已同意
	Approved bool
	// 消息
	Message string
	// 操作
	Action string
}

// 描述工具确认上下文键使用的数据结构
type toolConfirmationContextKey struct{}

// 处理带有运行上下文，接收本次调用的上下文、信息，返回调用上下文
func withRunContext(ctx context.Context, info runContext) context.Context {
	return context.WithValue(ctx, runContextKey{}, info)
}

// 获取运行上下文，接收本次调用的上下文，返回运行上下文、是否满足条件
func getRunContext(ctx context.Context) (runContext, bool) {
	info, ok := ctx.Value(runContextKey{}).(runContext) // 信息、是否成功取得对应值
	return info, ok
}

// 处理带有工具确认，接收本次调用的上下文、确认，返回调用上下文
func withToolConfirmation(ctx context.Context, confirmation toolConfirmation) context.Context {
	return context.WithValue(ctx, toolConfirmationContextKey{}, confirmation)
}

// 获取工具确认，接收本次调用的上下文，返回工具确认、是否满足条件
func getToolConfirmation(ctx context.Context) (toolConfirmation, bool) {
	info, ok := ctx.Value(toolConfirmationContextKey{}).(toolConfirmation) // 信息、是否成功取得对应值
	return info, ok
}
