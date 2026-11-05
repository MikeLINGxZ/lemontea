package plugin

import (
	"context"
	"encoding/json"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent/tools"
	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto"
)

// InstallCliSync 实现 tools.CliInstaller，调用 InstallCliFromNpm 执行安装并发送进度事件
// 接收本次调用的上下文、npm包名、当前名称、on进度，返回文本及可能发生的错误
func (p *Plugin) InstallCliSync(ctx context.Context, npmPackage string, name string, onProgress func(string, string)) (string, error) {
	// 发布
	emit := /* 实现发布回调，接收阶段、详情，无返回值 */ func(phase, detail string) {
		if onProgress != nil {
			onProgress(phase, detail)
		}
		if p.wailsApp != nil {
			p.wailsApp.Event.Emit("cli.install.progress", map[string]string{
				"npm_package": npmPackage,
				"name":        name,
				"phase":       phase,
				"detail":      detail,
			})
		}
	}

	emit("downloading", "Fetching "+npmPackage)
	// 本次处理的输出、处理安装CLI起点npm时的错误
	out, err := p.InstallCliFromNpm(ctx, plugin_dto.InstallCliFromNpmInput{
		NpmPackage: npmPackage,
		Name:       name,
	})
	if err != nil {
		emit("failed", err.Error())
		return "", err
	}
	emit("installed", "Package installed to "+out.Extension.RootDir)

	resultBytes, _ := json.Marshal(out.Extension) // 结果字节
	return string(resultBytes), nil
}

// ReportCliInstallProgress 实现 tools.CliInstallProgressReporter，报告 CLI 安装进度
// 接收本次调用的上下文、会话编号、CLI安装进度条目，返回操作错误
func (p *Plugin) ReportCliInstallProgress(ctx context.Context, sessionID uint, item tools.CliInstallProgressItem) error {
	_ = ctx
	if p.wailsApp != nil {
		p.wailsApp.Event.Emit("cli.install.progress", map[string]any{
			"session_id":   sessionID,
			"npm_package":  item.NpmPackage,
			"name":         item.Name,
			"extension_id": item.ExtensionID,
			"phase":        item.Phase,
			"detail":       item.Detail,
			"action_url":   item.ActionURL,
			"action_label": item.ActionLabel,
			"expires_at":   item.ExpiresAt,
		})
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
