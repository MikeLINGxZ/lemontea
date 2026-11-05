package tools

import (
	"context"
	"encoding/json"
	"errors"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// CliManifestGenerator 是由插件服务实现的 CLI 清单生成接口
type CliManifestGenerator interface {
	// 处理生成CLI清单同步，接收本次调用的上下文、扩展编号，返回扩展JSON、操作错误
	GenerateCliManifestSync(ctx context.Context, extensionID string) (extensionJSON string, err error)
}

// 生成CLI清单工具名称
const GenerateCliManifestToolName = "GenerateCliManifest"

// 构造生成CLI清单工具，返回工具元信息
func BuildGenerateCliManifestTool() ToolMeta {
	return ToolMeta{
		Name:        GenerateCliManifestToolName,
		Description: "Generate (or regenerate) the manifest for an installed CLI plugin. Call with {\"id\":\"cli:<name>\"}. Returns the updated extension item JSON.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 解析结果
			var parsed struct {
				// 唯一标识
				ID string `json:"id"`
			}
			_ = json.Unmarshal(args, &parsed)
			return "Generating manifest for: " + parsed.ID
		},
	}
}

// 封装处理生成CLI清单所需的请求参数
type generateCliManifestInput struct {
	// 唯一标识
	ID string `json:"id" jsonschema:"description=extension ID in the form cli:<name>,required"`
}

// 调用生成CLI清单，接收本次调用的上下文、生成器、参数，返回文本及可能发生的错误
func InvokeGenerateCliManifest(ctx context.Context, generator CliManifestGenerator, args json.RawMessage) (string, error) {
	// 解析结果
	var parsed generateCliManifestInput
	err := json.Unmarshal(args, &parsed) // 解析序列化内容时的错误
	if err != nil {
		return "", err
	}
	if parsed.ID == "" {
		return "", errors.New("id is required")
	}
	return generator.GenerateCliManifestSync(ctx, parsed.ID)
}

// NewGenerateCliManifestTool 创建生成 CLI 插件清单的函数工具
// 接收生成器，返回Function工具生成CLI清单输入string
func NewGenerateCliManifestTool(generator CliManifestGenerator) *function.FunctionTool[generateCliManifestInput, string] {
	meta := BuildGenerateCliManifestTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input generateCliManifestInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, _ := json.Marshal(input) // 载荷
			return InvokeGenerateCliManifest(ctx, generator, payload)
		},
		function.WithName(GenerateCliManifestToolName),
		function.WithDescription(meta.Description),
	)
}
