package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent/tools"
	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
)

// ToolConfirmDecisionInput 描述等待运行时确认决策的工具调用
type ToolConfirmDecisionInput struct {
	// 工具名称
	ToolName string
	// 实参
	Arguments string
	// 用途
	Purpose string
	// 元信息
	Meta tools.ToolMeta
	// 模型名称
	ModelName string
	// 服务基础地址
	BaseURL string
	// API 密钥
	APIKey string
	// 提供商类型
	ProviderType pkgProvider.Type
	// 会话编号
	SessionID uint
}

// ToolConfirmDecision 保存运行时确认决策器返回的结构化结果
type ToolConfirmDecision struct {
	// 需要确认
	RequireConfirm bool
	// 禁止
	Deny bool
	// 原因
	Reason string
	// 来源
	Source string
}

// ToolConfirmDecider 判断单次工具调用是否需要用户确认
type ToolConfirmDecider func(ctx context.Context, input ToolConfirmDecisionInput) (ToolConfirmDecision, error)

// 工具确认决策系统提示词
const toolConfirmDecisionSystemPrompt = `You decide whether a tool call should require user confirmation before execution.

Return strict JSON:
{
  "require_confirm": true|false,
  "reason": "<short reason under 12 words>"
}

Guidelines:
- require_confirm=true for destructive, externally visible, irreversible, costly, privileged, or environment-modifying actions.
- require_confirm=false for read-only inspection, low-risk analysis, and harmless metadata queries.
- If the risk is uncertain, choose true.
- Judge this specific invocation, not just the tool name.`

// 运行默认工具确认策略并为无法判定的结果提供回退，接收调用上下文和请求参数，返回工具确认决策及可能发生的错误
func defaultToolConfirmDecider(ctx context.Context, input ToolConfirmDecisionInput) (ToolConfirmDecision, error) {
	if /* 决策、是否成功取得对应值 */ decision, ok := deterministicToolConfirmDecision(input); ok {
		return decision, nil
	}
	if strings.TrimSpace(input.ModelName) == "" || strings.TrimSpace(input.BaseURL) == "" {
		return heuristicToolConfirmDecision(input), nil
	}

	userPrompt := buildToolConfirmDecisionPrompt(input) // 用户提示词
	// 远程服务响应、处理单次请求完成时的错误
	resp, err := OneshotComplete(ctx, OneshotRequest{
		BaseURL:      input.BaseURL,
		APIKey:       input.APIKey,
		ModelName:    input.ModelName,
		ProviderType: input.ProviderType,
		System:       toolConfirmDecisionSystemPrompt,
		User:         userPrompt,
		MaxTokens:    120,
		Timeout:      20 * time.Second,
	})
	if err != nil {
		fallback := heuristicToolConfirmDecision(input) // 回退
		fallback.Reason = "fallback after decider error"
		return fallback, nil
	}

	// 解析结果
	var parsed struct {
		// 需要确认
		RequireConfirm bool `json:"require_confirm"`
		// 原因
		Reason string `json:"reason"`
	}
	decodeErr := json.Unmarshal([]byte(extractJSONObject(resp.Text)), &parsed) // 解析序列化内容时的错误
	if decodeErr != nil {
		fallback := heuristicToolConfirmDecision(input) // 回退
		fallback.Reason = "fallback after parse error"
		return fallback, nil
	}

	return ToolConfirmDecision{
		RequireConfirm: parsed.RequireConfirm,
		Reason:         strings.TrimSpace(parsed.Reason),
		Source:         "model",
	}, nil
}

// 构造供模型判断工具风险的提示词，接收工具确认决策输入，返回文本
func buildToolConfirmDecisionPrompt(input ToolConfirmDecisionInput) string {
	// 构建器
	var builder strings.Builder
	builder.WriteString("Tool name: ")
	builder.WriteString(input.ToolName)
	builder.WriteString("\n")
	if strings.TrimSpace(input.Purpose) != "" {
		builder.WriteString("Purpose: ")
		builder.WriteString(strings.TrimSpace(input.Purpose))
		builder.WriteString("\n")
	}
	builder.WriteString("Arguments JSON: ")
	builder.WriteString(input.Arguments)
	builder.WriteString("\n")
	builder.WriteString("Static requires_confirm fallback: ")
	builder.WriteString(fmt.Sprintf("%t", input.Meta.RequiresConfirm))
	return builder.String()
}

// 根据工具与参数的可识别特征进行风险判断，接收工具确认决策输入，返回工具确认决策
func heuristicToolConfirmDecision(input ToolConfirmDecisionInput) ToolConfirmDecision {
	if input.Meta.RequiresConfirm {
		return ToolConfirmDecision{
			RequireConfirm: true,
			Reason:         "static risky action",
			Source:         "static",
		}
	}

	// 小写
	lower := strings.ToLower(strings.Join([]string{input.ToolName, input.Purpose, input.Arguments}, " "))
	// 有风险检索词
	riskyTerms := []string{
		"delete", "remove", "destroy", "drop", "send", "post ", "publish", "create",
		"update", "write", "upload", "execute", "shell", "command", "prod", "production",
		"deploy", "restart", "kill", "token", "secret", "config", "permission",
	}
	// 安全检索词
	safeTerms := []string{
		"list", "get", "show", "read", "fetch", "query", "status", "inspect", "describe",
	}

	// 遍历有风险检索词，逐项取得终端
	for _, term := range riskyTerms {
		if strings.Contains(lower, term) {
			return ToolConfirmDecision{
				RequireConfirm: true,
				Reason:         "heuristic risky action",
				Source:         "heuristic",
			}
		}
	}
	// 遍历安全检索词，逐项取得终端
	for _, term := range safeTerms {
		if strings.Contains(lower, term) {
			return ToolConfirmDecision{
				RequireConfirm: false,
				Reason:         "heuristic read only",
				Source:         "heuristic",
			}
		}
	}
	return ToolConfirmDecision{
		RequireConfirm: true,
		Reason:         "heuristic uncertain risk",
		Source:         "heuristic",
	}
}

// deterministicToolConfirmDecision 应用无需模型判断的安全规则
// 接收工具确认决策输入，返回工具确认决策、是否满足条件
func deterministicToolConfirmDecision(input ToolConfirmDecisionInput) (ToolConfirmDecision, bool) {
	if input.ToolName != "shell" {
		return ToolConfirmDecision{}, false
	}
	command := shellCommandFromArgs(input.Arguments) // 命令
	if command == "" {
		return ToolConfirmDecision{RequireConfirm: true, Reason: "missing shell command", Source: "policy"}, true
	}
	normalized := normalizeShellCommand(command) // 规范化值
	if isDeniedShellCommand(normalized) {
		return ToolConfirmDecision{
			Deny:           true,
			RequireConfirm: false,
			Reason:         "destructive shell command",
			Source:         "policy",
		}, true
	}
	if isSafeShellCommand(normalized) {
		return ToolConfirmDecision{
			RequireConfirm: false,
			Reason:         "safe shell inspection",
			Source:         "policy",
		}, true
	}
	if isRiskyShellCommand(normalized) {
		return ToolConfirmDecision{
			RequireConfirm: true,
			Reason:         "file system change",
			Source:         "policy",
		}, true
	}
	return ToolConfirmDecision{}, false
}

// shellCommandFromArgs 从 Shell 工具的参数载荷中提取 command 字段
// 接收参数，返回文本
func shellCommandFromArgs(args string) string {
	// 解析结果
	var parsed struct {
		// 命令
		Command string `json:"command"`
	}
	err := json.Unmarshal([]byte(args), &parsed) // 解析序列化内容时的错误
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Command)
}

// normalizeShellCommand 移除 Shell 包装，使简单规则能稳定匹配
// 接收命令，返回文本
func normalizeShellCommand(command string) string {
	lower := strings.ToLower(strings.TrimSpace(command)) // 小写
	lower = strings.TrimPrefix(lower, "command ")
	return strings.Join(strings.Fields(lower), " ")
}

// isSafeShellCommand 判断命令是否属于日常任务使用的只读命令
// 接收命令，返回是否满足条件
func isSafeShellCommand(command string) bool {
	// 安全精确值
	safeExact := []string{
		"pwd",
	}
	// 遍历安全精确值，逐项取得精确值
	for _, exact := range safeExact {
		if command == exact {
			return true
		}
	}
	// 安全前缀列表
	safePrefixes := []string{
		"ls ",
		"find ",
		"du ",
		"wc ",
		"stat ",
		"file ",
		"mdls ",
	}
	// 遍历安全前缀列表，逐项取得前缀
	for _, prefix := range safePrefixes {
		if command == strings.TrimSpace(prefix) || strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

// isRiskyShellCommand 判断命令是否可能修改文件并需要事先确认
// 接收命令，返回是否满足条件
func isRiskyShellCommand(command string) bool {
	// 有风险前缀列表
	riskyPrefixes := []string{
		"mv ",
		"cp ",
		"mkdir ",
		"touch ",
		"chmod ",
		"chown ",
		"open ",
		"osascript ",
		"zip ",
		"unzip ",
		"tar ",
	}
	// 遍历有风险前缀列表，逐项取得前缀
	for _, prefix := range riskyPrefixes {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

// isDeniedShellCommand 判断命令是否具有明显破坏性并默认拦截
// 接收命令，返回是否满足条件
func isDeniedShellCommand(command string) bool {
	// 已拒绝前缀列表
	deniedPrefixes := []string{
		"rm ",
		"rm -",
		"rmdir ",
		"shred ",
		"mkfs",
		"diskutil erase",
	}
	// 遍历已拒绝前缀列表，逐项取得前缀
	for _, prefix := range deniedPrefixes {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	// 已拒绝片段列表
	deniedFragments := []string{
		"; rm ",
		"&& rm ",
		"|| rm ",
		"| rm ",
		"; rmdir ",
		"&& rmdir ",
		"|| rmdir ",
		"| rmdir ",
	}
	// 遍历已拒绝片段列表，逐项取得片段
	for _, fragment := range deniedFragments {
		if strings.Contains(command, fragment) {
			return true
		}
	}
	return false
}

// 从包含额外文本的响应中定位 JSON 对象，接收尚未解析的原始数据，返回文本
func extractJSONObject(raw string) string {
	trimmed := strings.TrimSpace(raw)      // 已去除空白
	start := strings.Index(trimmed, "{")   // 开始
	end := strings.LastIndex(trimmed, "}") // 结束
	if start >= 0 && end >= start {
		return trimmed[start : end+1]
	}
	return trimmed
}
