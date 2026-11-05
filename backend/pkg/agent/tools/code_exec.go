package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 代码执行超时
const codeExecTimeout = 30 * time.Second

// 封装处理代码执行所需的请求参数
type codeExecInput struct {
	// 语言
	Language string `json:"language" jsonschema:"description=Programming language (python/javascript/bash),required"`
	// 代码
	Code string `json:"code" jsonschema:"description=Source code to execute,required"`
}

// 封装处理代码执行返回给调用方的数据
type codeExecOutput struct {
	// 标准输出
	Stdout string `json:"stdout"`
	// 标准错误
	Stderr string `json:"stderr"`
	// 退出代码
	ExitCode int `json:"exit_code"`
}

// 执行工具请求中的代码并返回输出，接收调用上下文和请求参数，返回操作结果及可能发生的错误
// 写入文件内容
// 创建带超时限制的子上下文
// 运行当前数据
func codeExecFunc(ctx context.Context, input codeExecInput) (codeExecOutput, error) {
	// 运行器列表
	runners := map[string]struct {
		// 命令
		cmd string
		// 扩展名
		ext string
	}{
		"python":     {cmd: "python3", ext: ".py"},
		"javascript": {cmd: "node", ext: ".js"},
		"bash":       {cmd: "bash", ext: ".sh"},
	}

	runner, ok := runners[input.Language] // runner 对应的数据、是否成功取得对应值
	if !ok {
		return codeExecOutput{}, fmt.Errorf("unsupported language: %s", input.Language)
	}

	// 临时文件
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("lemontea_exec_%d%s", time.Now().UnixNano(), runner.ext))
	writeErr := os.WriteFile(tmpFile, []byte(input.Code), 0o644) // 写入文件内容时的错误
	if writeErr != nil {
		return codeExecOutput{}, fmt.Errorf("write temp file: %w", writeErr)
	}
	defer os.Remove(tmpFile)

	ctx, cancel := context.WithTimeout(ctx, codeExecTimeout) // 用于取消本次操作的回调
	defer cancel()

	cmd := exec.CommandContext(ctx, runner.cmd, tmpFile) // 命令
	// 标准输出、标准错误输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run() // 运行 Wails 应用时的错误
	exitCode := 0    // 退出代码
	if err != nil {
		if /* 退出错误、是否成功取得对应值 */ exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return codeExecOutput{}, fmt.Errorf("exec: %w", err)
		}
	}

	return codeExecOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// 构建代码执行工具，返回Function工具代码执行输入代码执行输出
func NewCodeExecTool() *function.FunctionTool[codeExecInput, codeExecOutput] {
	return function.NewFunctionTool(
		codeExecFunc,
		function.WithName("code_exec"),
		function.WithDescription("Execute code in Python, JavaScript, or Bash and return the output"),
	)
}

// 构造代码执行工具的展示信息，返回工具元信息
func CodeExecMeta() ToolMeta {
	return ToolMeta{
		Name:            "code_exec",
		Description:     "Execute code in Python, JavaScript, or Bash",
		Category:        CategoryUser,
		RequiresConfirm: true,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input codeExecInput
			_ = json.Unmarshal(args, &input)
			preview := input.Code // 预览
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			return fmt.Sprintf("Execute %s code:\n%s", input.Language, preview)
		},
	}
}
