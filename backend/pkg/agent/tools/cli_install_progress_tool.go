package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// CliInstallProgressReporter 上报智能 CLI 配置任务的安装和初始化进度
type CliInstallProgressReporter interface {
	// 处理报告CLI安装进度，接收本次调用的上下文、会话编号、CLI安装进度条目，返回操作错误
	ReportCliInstallProgress(ctx context.Context, sessionID uint, item CliInstallProgressItem) error
}

// CliInstallProgressItem 表示提供给界面的通用进度数据
type CliInstallProgressItem struct {
	// npm包名
	NpmPackage string `json:"npm_package,omitempty"`
	// 名称
	Name string `json:"name,omitempty"`
	// 扩展编号
	ExtensionID string `json:"extension_id,omitempty"`
	// 阶段
	Phase string `json:"phase"`
	// 详情
	Detail string `json:"detail,omitempty"`
	// 操作地址
	ActionURL string `json:"action_url,omitempty"`
	// 操作标签
	ActionLabel string `json:"action_label,omitempty"`
	// 过期时间
	ExpiresAt string `json:"expires_at,omitempty"`
}

// 报告CLI安装进度工具名称
const ReportCliInstallProgressToolName = "ReportCliInstallProgress"

// 封装处理报告CLI安装进度所需的请求参数
type reportCliInstallProgressInput struct {
	// npm包名
	NpmPackage string `json:"npm_package" jsonschema:"description=installed npm package name when known"`
	// 名称
	Name string `json:"name" jsonschema:"description=CLI display name when known"`
	// 扩展编号
	ExtensionID string `json:"extension_id" jsonschema:"description=installed extension id when known"`
	// 阶段
	Phase string `json:"phase" jsonschema:"description=current phase such as downloading|installed|generating|initializing|waiting_auth|verifying|done|failed,required"`
	// 详情
	Detail string `json:"detail" jsonschema:"description=short user-facing detail for the current phase"`
	// 操作地址
	ActionURL string `json:"action_url" jsonschema:"description=optional URL the user should open or click"`
	// 操作标签
	ActionLabel string `json:"action_label" jsonschema:"description=optional short label for the URL action"`
	// 过期时间
	ExpiresAt string `json:"expires_at" jsonschema:"description=optional expiration timestamp or user-facing expiry note"`
}

// 构造报告CLI安装进度工具，返回工具元信息
func BuildReportCliInstallProgressTool() ToolMeta {
	return ToolMeta{
		Name:        ReportCliInstallProgressToolName,
		Description: "Report smart CLI install or initialization progress to the UI. Use this at key checkpoints such as install started, initializing, waiting for auth, verifying, done, or failed.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input reportCliInstallProgressInput
			_ = json.Unmarshal(args, &input)
			phase := strings.TrimSpace(input.Phase) // 阶段
			if phase == "" {
				phase = "updating progress"
			}
			if input.Name != "" {
				return "Report CLI progress: " + input.Name + " (" + phase + ")"
			}
			if input.NpmPackage != "" {
				return "Report CLI progress: " + input.NpmPackage + " (" + phase + ")"
			}
			return "Report CLI progress: " + phase
		},
	}
}

// 调用报告CLI安装进度，接收本次调用的上下文、上报器、会话编号、参数，返回文本及可能发生的错误
func InvokeReportCliInstallProgress(ctx context.Context, reporter CliInstallProgressReporter, sessionID uint, args json.RawMessage) (string, error) {
	// 请求参数
	var input reportCliInstallProgressInput
	decodeErr := json.Unmarshal(args, &input) // 解析序列化内容时的错误
	if decodeErr != nil {
		return "", decodeErr
	}
	if strings.TrimSpace(input.Phase) == "" {
		return "", errors.New("phase is required")
	}
	// 当前条目
	item := CliInstallProgressItem{
		NpmPackage:  strings.TrimSpace(input.NpmPackage),
		Name:        strings.TrimSpace(input.Name),
		ExtensionID: strings.TrimSpace(input.ExtensionID),
		Phase:       strings.TrimSpace(input.Phase),
		Detail:      strings.TrimSpace(input.Detail),
		ActionURL:   strings.TrimSpace(input.ActionURL),
		ActionLabel: strings.TrimSpace(input.ActionLabel),
		ExpiresAt:   strings.TrimSpace(input.ExpiresAt),
	}
	// 处理报告CLI安装进度时的错误
	reportCliInstallProgressErr := reporter.ReportCliInstallProgress(ctx, sessionID, item)
	if reportCliInstallProgressErr != nil {
		return "", reportCliInstallProgressErr
	}
	return "cli install progress updated", nil
}

// 构建报告CLI安装进度工具，接收上报器、会话编号，返回Function工具报告CLI安装进度输入string
func NewReportCliInstallProgressTool(reporter CliInstallProgressReporter, sessionID uint) *function.FunctionTool[reportCliInstallProgressInput, string] {
	meta := BuildReportCliInstallProgressTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input reportCliInstallProgressInput) (string, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回文本及可能发生的错误
			payload, err := json.Marshal(input) // 载荷、将数据序列化时的错误
			if err != nil {
				return "", err
			}
			return InvokeReportCliInstallProgress(ctx, reporter, sessionID, payload)
		},
		function.WithName(ReportCliInstallProgressToolName),
		function.WithDescription(meta.Description),
	)
}
