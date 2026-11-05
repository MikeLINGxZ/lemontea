package tools

import (
	"context"
	"encoding/json"
	"errors"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// CliInstaller 由插件服务实现，用接口隔离两者以避免循环导入
type CliInstaller interface {
	// 处理安装CLI同步，接收本次调用的上下文、npm包名、当前名称、on进度，返回安装结果JSON、操作错误
	InstallCliSync(ctx context.Context, npmPackage string, name string, onProgress func(phase string, detail string)) (installResultJSON string, err error)
}

// 安装CLI工具名称
const InstallCliToolName = "InstallCli"

// 构造安装CLI工具，返回工具元信息
func BuildInstallCliTool() ToolMeta {
	return ToolMeta{
		Name:        InstallCliToolName,
		Description: "Install a CLI plugin from npm. Call with {\"npm_package\":\"@scope/pkg\",\"name\":\"my-cli\"}. Returns the installed extension item JSON.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 解析结果
			var parsed struct {
				// npm包名
				NpmPackage string `json:"npm_package"`
			}
			_ = json.Unmarshal(args, &parsed)
			return "Installing CLI: " + parsed.NpmPackage
		},
	}
}

// 封装处理安装CLI所需的请求参数
type installCliInput struct {
	// npm包名
	NpmPackage string `json:"npm_package" jsonschema:"description=npm package name to install,required"`
	// 名称
	Name string `json:"name" jsonschema:"description=short kebab-case name for the installed plugin"`
}

// 调用安装CLI，接收本次调用的上下文、安装器、参数，返回文本及可能发生的错误
func InvokeInstallCli(ctx context.Context, installer CliInstaller, args json.RawMessage) (string, error) {
	// 解析结果
	var parsed installCliInput
	err := json.Unmarshal(args, &parsed) // 解析序列化内容时的错误
	if err != nil {
		return "", err
	}
	if parsed.NpmPackage == "" {
		return "", errors.New("npm_package is required")
	}
	return installer.InstallCliSync(ctx, parsed.NpmPackage, parsed.Name, nil)
}

// NewInstallCliTool 创建从 npm 安装 CLI 插件的函数工具
// 接收安装器，返回Function工具安装CLI输入string
func NewInstallCliTool(installer CliInstaller) *function.FunctionTool[installCliInput, string] {
	meta := BuildInstallCliTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input installCliInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, _ := json.Marshal(input) // 载荷
			return InvokeInstallCli(ctx, installer, payload)
		},
		function.WithName(InstallCliToolName),
		function.WithDescription(meta.Description),
	)
}
