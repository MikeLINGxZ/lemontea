package plugin

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent/tools"
	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto"
	trpcagent "trpc.group/trpc-go/trpc-agent-go/agent"
)

// InstallCliSync 安装 CLI 并上报工具准备阶段，接收上下文、包名、名称和进度回调，返回扩展 JSON 或错误
func (p *Plugin) InstallCliSync(ctx context.Context, npmPackage string, name string, onProgress func(string, string)) (string, error) {
	var sessionID uint // 当前智能安装任务对应的会话编号
	if invocation, ok := trpcagent.InvocationFromContext(ctx); ok && invocation.Session != nil {
		id, err := strconv.ParseUint(invocation.Session.ID, 10, strconv.IntSize) // 会话编号和解析错误
		if err == nil {
			sessionID = uint(id)
		}
	}

	// 将安装和初始化的进度合并到同一条会话记录
	report := /* 发布安装进度，接收阶段、详情和扩展编号，无返回值 */ func(phase, detail, extensionID string) {
		if onProgress != nil {
			onProgress(phase, detail)
		}
		_ = p.ReportCliInstallProgress(ctx, sessionID, tools.CliInstallProgressItem{
			NpmPackage:  npmPackage,
			Name:        name,
			ExtensionID: extensionID,
			Phase:       phase,
			Detail:      detail,
		})
	}
	item, err := p.installCli(ctx, plugin_dto.InstallCliFromNpmInput{NpmPackage: npmPackage, Name: name},
		/* 发布阶段，接收安装阶段，无返回值 */ func(phase string) { report(phase, "", "") })
	if err != nil {
		report("failed", err.Error(), "")
		return "", err
	}
	report("installed", "", item.ID)

	resultBytes, _ := json.Marshal(item) // 扩展 JSON
	return string(resultBytes), nil
}

// ReportCliInstallProgress 实现 tools.CliInstallProgressReporter，报告 CLI 安装进度
// 接收本次调用的上下文、会话编号、CLI安装进度条目，返回操作错误
func (p *Plugin) ReportCliInstallProgress(ctx context.Context, sessionID uint, item tools.CliInstallProgressItem) error {
	_ = ctx
	if p.wailsApp != nil {
		payload := map[string]any{ // 安装进度事件
			"npm_package":  item.NpmPackage,
			"name":         item.Name,
			"extension_id": item.ExtensionID,
			"phase":        item.Phase,
			"detail":       item.Detail,
			"action_url":   item.ActionURL,
			"action_label": item.ActionLabel,
			"expires_at":   item.ExpiresAt,
		}
		if sessionID != 0 {
			payload["session_id"] = sessionID
		}
		p.wailsApp.Event.Emit("cli.install.progress", payload)
	}
	return nil
}

// GenerateCliManifestSync 实现 tools.CliManifestGenerator，同步生成 CLI 插件清单
// 接收本次调用的上下文、扩展编号，返回文本及可能发生的错误
func (p *Plugin) GenerateCliManifestSync(ctx context.Context, extensionID string) (string, error) {
	if p.wailsApp != nil {
		p.wailsApp.Event.Emit("cli.install.progress", map[string]string{
			"id":    extensionID,
			"phase": "generating",
		})
	}
	// 本次处理的输出、处理生成CLI清单时的错误
	out, err := p.GenerateCliManifest(ctx, plugin_dto.GenerateCliManifestInput{ID: extensionID})
	if err != nil {
		if p.wailsApp != nil {
			p.wailsApp.Event.Emit("cli.install.progress", map[string]string{
				"id":    extensionID,
				"phase": "failed",
			})
		}
		return "", err
	}
	if p.wailsApp != nil {
		p.wailsApp.Event.Emit("cli.install.done", map[string]interface{}{
			"extension": out.Extension,
		})
	}
	resultBytes, _ := json.Marshal(out.Extension) // 结果字节
	return string(resultBytes), nil
}

// RunCliCommandSync 实现 tools.CliCommandRunner，同步执行 CLI 命令
// 接收本次调用的上下文、扩展编号、命令参数、输出模式、超时秒、on进度，返回文本及可能发生的错误
func (p *Plugin) RunCliCommandSync(ctx context.Context, extensionID string, argv []string, outputMode string, timeoutSeconds int, onProgress func(result string)) (string, error) {
	name, err := cliNameFromID(extensionID) // 当前名称、处理CLI名称起点编号时的错误
	if err != nil {
		return "", err
	}
	mgr, err := p.resolveCliManager() // 管理器
	if err != nil {
		return "", err
	}

	mode := pkgcli.OutputText // 模式
	switch strings.TrimSpace(outputMode) {
	case "", string(pkgcli.OutputText):
		mode = pkgcli.OutputText
	case string(pkgcli.OutputJSON):
		mode = pkgcli.OutputJSON
	case string(pkgcli.OutputLines):
		mode = pkgcli.OutputLines
	}

	// 结果
	res, err := mgr.RunCommandStreaming(ctx, name, argv, mode, timeoutSeconds, func(progress pkgcli.RunProgress) {
		// 运行命令流式输出时调用的回调，接收进度，无返回值
		if onProgress == nil {
			return
		}
		onProgress(formatCLIProgress(progress))
	})
	if err != nil {
		return "", err
	}
	resultBytes, _ := json.Marshal(res) // 结果字节
	return string(resultBytes), nil
}

// 将 CLI 任务进度整理为前端事件数据，接收进度，返回文本
func formatCLIProgress(progress pkgcli.RunProgress) string {
	// 终端输出
	var terminalOutput string
	switch {
	case progress.Stdout == "" && progress.Stderr == "":
		return ""
	case progress.Stderr == "":
		terminalOutput = progress.Stdout
	case progress.Stdout == "":
		terminalOutput = "[stderr]\n" + progress.Stderr
	default:
		terminalOutput = progress.Stdout + "\n[stderr]\n" + progress.Stderr
	}
	// 载荷
	payload := map[string]any{
		"interactive_terminal": true,
		"terminal_status":      "active",
		"terminal_output":      terminalOutput,
	}
	out, _ := json.Marshal(payload) // 本次处理的输出
	return string(out)
}
