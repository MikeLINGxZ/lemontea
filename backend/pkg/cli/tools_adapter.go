package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 定义工具Runner的调用契约
type toolRunner interface {
	// 运行工具，接收本次调用的上下文、当前名称、工具名称、按键组织的数据映射，返回运行结果及可能发生的错误
	RunTool(ctx context.Context, name string, toolName string, input map[string]any) (RunResult, error)
}

// 描述CLI工具集合使用的数据结构
type cliToolSet struct {
	// 当前名称
	name string
	// 工具列表
	tools []tool.Tool
}

// Tools 返回当前可用的 CLI 工具
// 接收调用上下文，返回工具列表
func (s cliToolSet) Tools(context.Context) []tool.Tool {
	return s.tools
}

// Close 释放工具集持有的资源
// 返回操作错误
func (s cliToolSet) Close() error {
	return nil
}

// Name 返回工具集的唯一名称
func (s cliToolSet) Name() string {
	return s.name
}

// BuildToolSet 为单个已启用的 CLI 扩展创建智能体 ToolSet
// 接收管理器、扩展条目、清单、已启用的用户工具列表，返回工具集合及可能发生的错误
func BuildToolSet(mgr *Manager, item data_models.ExtensionItem, manifest Manifest, enabledUserTools []string) (tool.ToolSet, error) {
	return buildToolSetFromRunner(mgr, item, manifest, enabledUserTools)
}

// 构造工具集合起点Runner，接收runner 对应的数据、扩展条目、清单、已启用的用户工具列表，返回工具集合及可能发生的错误
func buildToolSetFromRunner(runner toolRunner, item data_models.ExtensionItem, manifest Manifest, enabledUserTools []string) (tool.ToolSet, error) {
	setName := cliToolSetName(item.Name) // 集合名称
	if len(manifest.Tools) == 0 {
		return cliToolSet{name: setName, tools: []tool.Tool{}}, nil
	}

	includeAll := false                                          // 包含全部
	selected := make(map[string]struct{}, len(enabledUserTools)) // 选中项
	// 遍历已启用的用户工具列表，逐项取得唯一标识
	for _, id := range enabledUserTools {
		selected[id] = struct{}{}
		if id == item.ID {
			includeAll = true
		}
	}

	tools := make([]tool.Tool, 0, len(manifest.Tools)) // 工具列表
	runtimeName := cliRuntimeName(item)                // 运行环境名称
	// 遍历工具列表，逐项取得当前
	for _, current := range manifest.Tools {
		if !current.Enabled {
			continue
		}
		toolID := CliToolID(item.Name, current.Name) // 工具编号
		if len(selected) > 0 && !includeAll {
			if /* 是否成功取得对应值 */ _, ok := selected[toolID]; !ok {
				continue
			}
		}

		schema, err := decodeInputSchema(current.InputSchema) // 模式、将清单中的参数模式解析为工具注册所需的数据时的错误
		if err != nil {
			return nil, err
		}

		toolName := current.Name     // 工具名称
		extensionName := runtimeName // 扩展名称
		tools = append(tools, function.NewFunctionTool(
			func(ctx context.Context, input map[string]any) (RunResult, error) {
				// 构建Function工具时调用的回调，接收本次调用的上下文、按键组织的数据映射，返回运行结果及可能发生的错误
				return runner.RunTool(ctx, extensionName, toolName, input)
			},
			function.WithName(current.Name),
			function.WithDescription(current.Description),
			function.WithInputSchema(schema),
		))
	}
	return cliToolSet{name: setName, tools: tools}, nil
}

// 处理CLI运行环境名称，接收扩展条目，返回文本
func cliRuntimeName(item data_models.ExtensionItem) string {
	if /* 基础 */ base := strings.TrimSpace(filepath.Base(item.RootDir)); base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}
	return item.Name
}

// 将清单中的参数模式解析为工具注册所需的数据，接收尚未解析的原始数据，返回模式及可能发生的错误
func decodeInputSchema(raw json.RawMessage) (*tool.Schema, error) {
	if len(raw) == 0 {
		return &tool.Schema{
			Type:       "object",
			Properties: map[string]*tool.Schema{},
		}, nil
	}
	// 模式
	var schema tool.Schema
	err := json.Unmarshal(raw, &schema) // 解析序列化内容时的错误
	if err != nil {
		return nil, fmt.Errorf("decode cli input schema: %w", err)
	}
	if schema.Type == "" {
		schema.Type = "object"
	}
	if schema.Properties == nil {
		schema.Properties = map[string]*tool.Schema{}
	}
	return &schema, nil
}

// 处理CLI工具集合名称，接收当前名称，返回文本
func cliToolSetName(name string) string {
	return sanitizeToolToken("cli_" + name)
}

// CliToolID 生成暴露给聊天智能体的稳定工具 ID
// 接收扩展名称、工具名称，返回文本
func CliToolID(extensionName string, toolName string) string {
	return sanitizeToolToken("cli_" + extensionName + "_" + toolName)
}

// 将工具标识清理为可注册的名称片段，接收文本，返回文本
func sanitizeToolToken(value string) string {
	// 构建器
	var builder strings.Builder
	builder.Grow(len(value))
	lastUnderscore := false // 最后一个下划线
	// 遍历待处理的值，逐项取得读取源
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	name := strings.Trim(builder.String(), "_") // 当前名称
	if name == "" {
		return "cli"
	}
	return name
}
