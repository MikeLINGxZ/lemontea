package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	agentpkg "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent"
	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto"
)

// InstallCliFromNpm 安装并注册可调用的 CLI 工具，接收调用上下文和安装参数，返回扩展条目或错误
func (p *Plugin) InstallCliFromNpm(ctx context.Context, input plugin_dto.InstallCliFromNpmInput) (*plugin_dto.InstallCliFromNpmOutput, error) {
	item, err := p.installCli(ctx, input /* 发布设置页安装阶段，接收阶段，无返回值 */, func(phase string) {
		if p.wailsApp != nil {
			p.wailsApp.Event.Emit("cli.install.progress", map[string]string{
				"request_id":  "npm:" + input.NpmPackage,
				"npm_package": input.NpmPackage,
				"name":        input.Name,
				"phase":       phase,
			})
		}
	})
	if err != nil {
		return nil, err
	}
	return &plugin_dto.InstallCliFromNpmOutput{Extension: item}, nil
}

// installCli 依次安装包、准备工具并登记扩展
// 接收上下文、安装参数和阶段回调，返回已经登记的扩展及可能发生的错误
func (p *Plugin) installCli(ctx context.Context, input plugin_dto.InstallCliFromNpmInput, reportPhase func(string)) (data_models.ExtensionItem, error) {
	name := pickInstallName(input.Name, input.NpmPackage) // 安装目录使用的名称
	p.cliInstallMu.Lock()
	_, installing := p.cliInstalling[name] // 同一安装目录是否已有任务
	if !installing {
		p.cliInstalling[name] = struct{}{}
	}
	p.cliInstallMu.Unlock()
	if installing {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrCliInstallFailed, errors.New("an installation is already running for "+name))
	}
	defer func() {
		// 释放安装目录，允许失败后重试
		p.cliInstallMu.Lock()
		delete(p.cliInstalling, name)
		p.cliInstallMu.Unlock()
	}()

	reportPhase("downloading")
	mgr, err := p.resolveCliManager() // CLI 管理器及初始化错误
	if err != nil {
		return data_models.ExtensionItem{}, err
	}
	res, err := mgr.InstallFromNpm(ctx, pkgcli.InstallParams{NpmPackage: input.NpmPackage, Name: name}) // 安装结果和错误
	if err != nil {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrCliInstallFailed, err)
	}

	reportPhase("generating")
	manifest, err := pkgcli.LoadManifest(res.ManifestPath) // 包安装时生成的清单和读取错误
	if err != nil {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrCliManifestInvalid, err)
	}
	if len(manifest.Tools) == 0 {
		generateCtx, cancel := context.WithTimeout(ctx, 45*time.Second)                           // 自动生成的等待上限和取消函数
		generated, generateErr := p.generateCliTools(generateCtx, res.InstallDir, name, manifest) // 模型生成的清单和错误
		cancel()
		if generateErr == nil && slices.ContainsFunc(generated.Tools /* 检查工具是否启用，接收工具条目，返回启用状态 */, func(tool pkgcli.Tool) bool { return tool.Enabled }) {
			// 验证生成的参数定义能被实际工具适配器加载
			_, generateErr = pkgcli.BuildToolSet(mgr, data_models.ExtensionItem{Name: name}, generated, nil)
			if generateErr == nil {
				manifest.Tools = generated.Tools
			}
		}
		if len(manifest.Tools) == 0 {
			// 模型不可用或没有生成可用工具时，提供直接传参的命令工具
			manifest.Tools = []pkgcli.Tool{{
				Name:            "run",
				Description:     "Run " + manifest.Name + " with the supplied command-line arguments.",
				InputSchema:     json.RawMessage(`{"type":"object","properties":{"argv":{"type":"array","items":{"type":"string"},"description":"Command arguments in order, without the executable name"}},"required":["argv"]}`),
				ArgvTemplate:    []string{pkgcli.ArgvListPlaceholder},
				OutputMode:      pkgcli.OutputText,
				RequiresConfirm: true,
				Enabled:         true,
			}}
		}
	}

	if ctx.Err() != nil {
		return data_models.ExtensionItem{}, ctx.Err()
	}
	reportPhase("registering")
	err = pkgcli.SaveManifest(res.ManifestPath, manifest)
	if err != nil {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrCliManifestSaveFailed, err)
	}

	// 并行安装完成时逐个更新扩展列表
	p.cliInstallMu.Lock()
	defer p.cliInstallMu.Unlock()
	config, err := p.loadConfig() // 最新扩展配置和读取错误
	if err != nil {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item := data_models.ExtensionItem{
		ID:             "cli:" + res.Name,
		Name:           res.Name,
		Description:    res.Description,
		Author:         res.Author,
		Version:        res.Version,
		Kind:           "cli",
		Enabled:        true,
		RuntimeStatus:  "ready",
		RootDir:        res.InstallDir,
		ConfigFilePath: res.ManifestPath,
	}
	item.Tools = projectCliTools(item, manifest)
	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	err = p.saveConfig(config)
	if err != nil {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}
	return item, nil
}

// pickInstallName 优先返回用户指定的名称，未指定时返回 npm 包名
// 接收provided 对应的文本、回退，返回文本
func pickInstallName(provided string, fallback string) string {
	if /* 当前值 */ v := strings.TrimSpace(provided); v != "" {
		return sanitizeName(v)
	}
	return sanitizeName(fallback)
}

// sanitizeName 移除目录名中不允许的字符，仅保留字符集 [a-zA-Z0-9_.-] 中的字符
// 接收文本，返回文本
func sanitizeName(s string) string {
	// 数据字节
	var b strings.Builder
	// 遍历当前值，逐项取得读取源
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '.':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "cli-plugin"
	}
	return b.String()
}

// ResetCliData 删除指定 CLI 插件的 cli_data/<name> 子目录，其中包含登录状态、清单和 XDG 目录
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) ResetCliData(ctx context.Context, input plugin_dto.ResetCliDataInput) (*plugin_dto.ResetCliDataOutput, error) {
	name, err := cliNameFromID(input.ID) // 当前名称、处理CLI名称起点编号时的错误
	if err != nil {
		return nil, err
	}
	mgr, err := p.resolveCliManager() // 管理器
	if err != nil {
		return nil, err
	}
	resetDataErr := mgr.ResetData(ctx, name) // 重置数据时的错误
	if resetDataErr != nil {
		return nil, ierror.Error(ierror.ErrCliResetDataFailed, resetDataErr)
	}
	return &plugin_dto.ResetCliDataOutput{}, nil
}

// UpdateCliManifest 校验提交的 JSON，通过 cli.SaveManifest 写入清单并更新已保存的 ExtensionItem
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
// 解析序列化内容
// 校验插件
// 保存 CLI 扩展清单
func (p *Plugin) UpdateCliManifest(ctx context.Context, input plugin_dto.UpdateCliManifestInput) (*plugin_dto.UpdateCliManifestOutput, error) {
	_ = ctx
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok || item.Kind != "cli" {
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, os.ErrNotExist)
	}
	// 清单
	var manifest pkgcli.Manifest
	decodeErr := json.Unmarshal([]byte(input.ManifestText), &manifest) // 解析序列化内容时的错误
	if decodeErr != nil {
		return nil, ierror.Error(ierror.ErrCliManifestInvalid, decodeErr)
	}
	validateErr := pkgcli.Validate(manifest) // 校验当前数据时的错误
	if validateErr != nil {
		return nil, ierror.Error(ierror.ErrCliManifestInvalid, validateErr)
	}
	saveErr := pkgcli.SaveManifest(item.ConfigFilePath, manifest) // 保存 CLI 扩展清单时的错误
	if saveErr != nil {
		return nil, ierror.Error(ierror.ErrCliManifestSaveFailed, saveErr)
	}
	// 将清单的名称、版本和描述同步到已保存的条目，使列表展示与清单一致
	item.Name = manifest.Name
	item.Version = manifest.Version
	item.Description = manifest.Description
	item.Tools = projectCliTools(item, manifest)
	if item.Enabled {
		item.RuntimeStatus = "ready"
	} else {
		item.RuntimeStatus = "idle"
	}
	item.RuntimeMessage = ""
	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}
	return &plugin_dto.UpdateCliManifestOutput{Extension: item}, nil
}

// 为指定 CLI 扩展生成并保存工具清单，接收调用上下文和扩展标识，返回生成结果或错误
// 先加载现有清单并收集命令帮助，再用默认聊天模型生成清单
// 将生成结果保存到扩展的清单路径并更新应用配置
func (p *Plugin) GenerateCliManifest(ctx context.Context, input plugin_dto.GenerateCliManifestInput) (*plugin_dto.GenerateCliManifestOutput, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok || item.Kind != "cli" {
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, os.ErrNotExist)
	}

	existingManifest, err := pkgcli.LoadManifest(item.ConfigFilePath) // 已存在清单
	if err != nil {
		return nil, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	cliName, err := cliNameFromID(item.ID) // 安装目录名和解析错误
	if err != nil {
		return nil, err
	}
	nextManifest, err := p.generateCliTools(ctx, item.RootDir, cliName, existingManifest) // 新清单和生成错误
	if err != nil {
		return nil, err
	}

	saveErr := pkgcli.SaveManifest(item.ConfigFilePath, nextManifest) // 保存 CLI 扩展清单时的错误
	if saveErr != nil {
		return nil, ierror.Error(ierror.ErrCliManifestSaveFailed, saveErr)
	}

	item.Name = nextManifest.Name
	item.Version = nextManifest.Version
	item.Description = nextManifest.Description
	item.Tools = projectCliTools(item, nextManifest)
	if item.Enabled {
		item.RuntimeStatus = "ready"
	} else {
		item.RuntimeStatus = "idle"
	}
	item.RuntimeMessage = ""
	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}
	return &plugin_dto.GenerateCliManifestOutput{Extension: item}, nil
}

// generateCliTools 根据 CLI 帮助和默认模型生成工具清单
// 接收上下文、安装目录、插件名称及现有清单，返回生成结果或错误
func (p *Plugin) generateCliTools(ctx context.Context, installDir, cliName string, manifest pkgcli.Manifest) (pkgcli.Manifest, error) {
	modelCfg, err := p.resolveDefaultChatModel() // 模型配置
	if err != nil {
		return pkgcli.Manifest{}, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	pkgMeta, err := pkgcli.ReadPackageJSON(installDir, manifest.Name) // 包名元信息
	if err != nil {
		return pkgcli.Manifest{}, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	if manifest.Executable == "" {
		executable, err := pkgcli.SelectExecutable(installDir, pkgMeta) // 可执行文件、处理选择可执行文件时的错误
		if err != nil {
			return pkgcli.Manifest{}, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
		}
		manifest.Executable = executable
	}
	runtimeState, err := p.loadPersistedRuntime() // 运行环境状态
	if err != nil {
		return pkgcli.Manifest{}, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	pluginBin := filepath.Join(installDir, "node_modules", ".bin") // 插件可执行目录
	runtimeBin := ""                                               // 运行环境可执行目录
	if runtimeState.NodePath != "" {
		runtimeBin = filepath.Dir(runtimeState.NodePath)
	}
	// 环境变量
	env := pkgcli.BuildEnv(pkgcli.EnvParams{
		Isolation:  pkgcli.IsolationShared,
		CLIDataDir: filepath.Join(mustDataDir(), "plugins", "cli_data", cliName),
		RuntimeBin: runtimeBin,
		PluginBin:  pluginBin,
		ParentEnv:  os.Environ(),
	})
	helpText, err := p.probeCliHelp(ctx, manifest.Executable, env) // 帮助文本
	if err != nil {
		return pkgcli.Manifest{}, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	// 下一项清单
	nextManifest, err := p.generateManifest(ctx, pkgcli.GenerateParams{
		HelpText:    helpText,
		PackageName: pkgMeta.Name,
		PackageMeta: pkgMeta,
		Executable:  manifest.Executable,
		Caller: /* 执行当前步骤的回调，接收本次调用的上下文、系统、用户，返回文本及可能发生的错误 */ func(ctx context.Context, system, user string) (string, error) {
			// 远程服务响应、处理单次请求完成时的错误
			resp, err := agentpkg.OneshotComplete(ctx, agentpkg.OneshotRequest{
				BaseURL:      modelCfg.BaseURL,
				APIKey:       modelCfg.APIKey,
				ModelName:    modelCfg.ModelName,
				ProviderType: modelCfg.ProviderType,
				System:       system,
				User:         user,
				Timeout:      3 * time.Minute,
			})
			if err != nil {
				return "", err
			}
			return resp.Text, nil
		},
	})
	if err != nil {
		return pkgcli.Manifest{}, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}

	return nextManifest, nil
}

// cliNameFromID 从 "cli:<name>" 中提取 "<name>"，标识不属于 CLI 插件时返回错误
// 接收唯一标识，返回文本及可能发生的错误
func cliNameFromID(id string) (string, error) {
	// 前缀
	const prefix = "cli:"
	if len(id) <= len(prefix) || id[:len(prefix)] != prefix {
		return "", ierror.Error(ierror.ErrSettingsReadConfig, os.ErrInvalid)
	}
	return id[len(prefix):], nil
}
