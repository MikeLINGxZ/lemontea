package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 封装处理文件已读所需的请求参数
type fileReadInput struct {
	// 路径
	Path string `json:"path" jsonschema:"description=Absolute file path to read,required"`
}

// 封装处理文件已读返回给调用方的数据
type fileReadOutput struct {
	// 内容
	Content string `json:"content"`
	// 尺寸
	Size int64 `json:"size"`
}

// 读取指定路径的文件并返回内容，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func fileReadFunc(ctx context.Context, input fileReadInput) (fileReadOutput, error) {
	content, err := os.ReadFile(input.Path) // 内容、读取文件内容时的错误
	if err != nil {
		return fileReadOutput{}, fmt.Errorf("read file: %w", err)
	}
	info, _ := os.Stat(input.Path) // 信息
	// 尺寸
	var size int64
	if info != nil {
		size = info.Size()
	}
	return fileReadOutput{Content: string(content), Size: size}, nil
}

// 构建文件已读工具，返回Function工具文件已读输入文件已读输出
func NewFileReadTool() *function.FunctionTool[fileReadInput, fileReadOutput] {
	return function.NewFunctionTool(
		fileReadFunc,
		function.WithName("file_read"),
		function.WithDescription("Read the content of a file at the given path"),
	)
}

// 封装处理文件写入所需的请求参数
type fileWriteInput struct {
	// 路径
	Path string `json:"path" jsonschema:"description=Absolute file path to write,required"`
	// 内容
	Content string `json:"content" jsonschema:"description=Content to write,required"`
}

// 封装处理文件写入返回给调用方的数据
type fileWriteOutput struct {
	// 字节已写入数量
	BytesWritten int `json:"bytes_written"`
}

// 将工具输入写入指定文件，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func fileWriteFunc(ctx context.Context, input fileWriteInput) (fileWriteOutput, error) {
	mkdirErr := os.MkdirAll(filepath.Dir(input.Path), 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return fileWriteOutput{}, fmt.Errorf("create dir: %w", mkdirErr)
	}
	writeErr := os.WriteFile(input.Path, []byte(input.Content), 0o644) // 写入文件内容时的错误
	if writeErr != nil {
		return fileWriteOutput{}, fmt.Errorf("write file: %w", writeErr)
	}
	return fileWriteOutput{BytesWritten: len(input.Content)}, nil
}

// 构建文件写入工具，返回Function工具文件写入输入文件写入输出
func NewFileWriteTool() *function.FunctionTool[fileWriteInput, fileWriteOutput] {
	return function.NewFunctionTool(
		fileWriteFunc,
		function.WithName("file_write"),
		function.WithDescription("Write content to a file at the given path"),
	)
}

// 构造文件已读工具的展示信息，返回工具元信息
func FileReadMeta() ToolMeta {
	return ToolMeta{
		Name:            "file_read",
		Description:     "Read file content",
		Category:        CategoryBuiltin,
		RequiresConfirm: true,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input fileReadInput
			_ = json.Unmarshal(args, &input)
			return fmt.Sprintf("Read file: %s", input.Path)
		},
	}
}

// 构造文件写入工具的展示信息，返回工具元信息
func FileWriteMeta() ToolMeta {
	return ToolMeta{
		Name:            "file_write",
		Description:     "Write content to file",
		Category:        CategoryBuiltin,
		RequiresConfirm: true,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input fileWriteInput
			_ = json.Unmarshal(args, &input)
			return fmt.Sprintf("Write to file: %s (%d bytes)", input.Path, len(input.Content))
		},
	}
}
