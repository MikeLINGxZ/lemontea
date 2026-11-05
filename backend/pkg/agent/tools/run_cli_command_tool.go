package tools

import (
	"context"
	"encoding/json"
	"errors"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// CliCommandRunner 是由插件服务实现的 CLI 命令执行接口
type CliCommandRunner interface {
	// 运行CLI命令同步，接收本次调用的上下文、扩展编号、命令参数、输出模式、超时秒、on进度，返回结果JSON、操作错误
	RunCliCommandSync(ctx context.Context, extensionID string, argv []string, outputMode string, timeoutSeconds int, onProgress func(result string)) (resultJSON string, err error)
}

// 定义CLI命令进度事件发布器的调用契约
type CliCommandProgressEmitter interface {
	// 将工具执行结果发布给对应会话，接收会话编号、工具名称、本次操作的结果，无返回值
	EmitToolResult(sessionID uint, toolName, result string)
}

// 运行CLI命令工具名称
const RunCliCommandToolName = "RunCliCommand"

// 默认运行CLI命令超时秒
const defaultRunCliCommandTimeoutSeconds = 600

// 构造运行CLI命令工具，返回工具元信息
func BuildRunCliCommandTool() ToolMeta {
	return ToolMeta{
		Name:        RunCliCommandToolName,
		Description: "Run an arbitrary command against one installed CLI plugin using Lemontea's bundled runtime and isolated env. Call with {\"id\":\"cli:<name>\",\"argv\":[...],\"output_mode\":\"json|text|lines\",\"timeout_seconds\":600}. Returns the CLI run result JSON and may stream intermediate output while the command is running.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 解析结果
			var parsed struct {
				// 唯一标识
				ID string `json:"id"`
				// 命令参数
				Argv []string `json:"argv"`
			}
			_ = json.Unmarshal(args, &parsed)
			if len(parsed.Argv) == 0 {
				return "Running CLI command for: " + parsed.ID
			}
			return "Running CLI command for: " + parsed.ID + " " + parsed.Argv[0]
		},
	}
}

// 封装运行CLI命令所需的请求参数
type runCliCommandInput struct {
	// 唯一标识
	ID string `json:"id" jsonschema:"description=extension ID in the form cli:<name>,required"`
	// 命令参数
	Argv []string `json:"argv" jsonschema:"description=argv to run against the CLI executable,required"`
	// 输出模式
	OutputMode string `json:"output_mode,omitempty" jsonschema:"description=one of json text lines"`
	// 超时秒
	TimeoutSeconds int `json:"timeout_seconds,omitempty" jsonschema:"description=timeout in seconds"`
}

// 调用运行CLI命令，接收本次调用的上下文、runner 对应的数据、参数、on进度，返回文本及可能发生的错误
func InvokeRunCliCommand(ctx context.Context, runner CliCommandRunner, args json.RawMessage, onProgress func(result string)) (string, error) {
	// 解析结果
	var parsed runCliCommandInput
	err := json.Unmarshal(args, &parsed) // 解析序列化内容时的错误
	if err != nil {
		return "", err
	}
	if parsed.ID == "" {
		return "", errors.New("id is required")
	}
	if len(parsed.Argv) == 0 {
		return "", errors.New("argv is required")
	}
	if parsed.TimeoutSeconds <= 0 {
		parsed.TimeoutSeconds = defaultRunCliCommandTimeoutSeconds
	}
	return runner.RunCliCommandSync(ctx, parsed.ID, parsed.Argv, parsed.OutputMode, parsed.TimeoutSeconds, onProgress)
}

// 构建运行CLI命令工具，接收runner 对应的数据、事件发布器、会话编号，返回Function工具运行CLI命令输入string
func NewRunCliCommandTool(runner CliCommandRunner, emitter CliCommandProgressEmitter, sessionID uint) *function.FunctionTool[runCliCommandInput, string] {
	meta := BuildRunCliCommandTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input runCliCommandInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, _ := json.Marshal(input) // 载荷
			return InvokeRunCliCommand(ctx, runner, payload /* 调用运行CLI命令时调用的回调，接收本次操作的结果，无返回值 */, func(result string) {
				if emitter != nil && sessionID != 0 {
					emitter.EmitToolResult(sessionID, RunCliCommandToolName, result)
				}
			})
		},
		function.WithName(RunCliCommandToolName),
		function.WithDescription(meta.Description),
	)
}
