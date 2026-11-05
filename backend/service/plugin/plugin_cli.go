package plugin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	agentpkg "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent"
	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto"
)

// InstallCliFromNpm 将已发布的 npm 包安装为 CLI 插件，并把插件条目保存到 config.json
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) InstallCliFromNpm(ctx context.Context, input plugin_dto.InstallCliFromNpmInput) (*plugin_dto.InstallCliFromNpmOutput, error) {
	name := pickInstallName(input.Name, input.NpmPackage) // 当前名称
	mgr, err := p.resolveCliManager()                     // 管理器、解析CLI管理器时的错误
	if err != nil {
		return nil, err
	}
	// 结果
	res, err := mgr.InstallFromNpm(ctx, pkgcli.InstallParams{NpmPackage: input.NpmPackage, Name: name})
	if err != nil {
		return nil, ierror.Error(ierror.ErrCliInstallFailed, err)
	}
	item, err := p.persistCliExtension(res, true) // 当前条目
	if err != nil {
		return nil, err
	}
	return &plugin_dto.InstallCliFromNpmOutput{Extension: item}, nil
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

// persistCliExtension 将 cli.InstallResult 转换为 ExtensionItem 并写入 config.json
// enabled 指定新条目的 Enabled 初始值
// 接收结果、是否启用，返回扩展条目及可能发生的错误
func (p *Plugin) persistCliExtension(res pkgcli.InstallResult, enabled bool) (data_models.ExtensionItem, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	// 当前条目
	item := data_models.ExtensionItem{
		ID:             "cli:" + res.Name,
		Name:           res.Name,
		Description:    res.Description,
		Author:         res.Author,
		Version:        res.Version,
		Kind:           "cli",
		Enabled:        enabled,
		RuntimeStatus:  "ready",
		RuntimeMessage: "",
		RootDir:        res.InstallDir,
		SourceDir:      "",
		ConfigFilePath: res.ManifestPath,
		Tools:          []data_models.ExtensionTool{},
	}
	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return data_models.ExtensionItem{}, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}
	return item, nil
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
	pkgMeta, err := pkgcli.ReadPackageJSON(item.RootDir) // 包名元信息
	if err != nil {
		return nil, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	if existingManifest.Executable == "" {
		executable, err := pkgcli.SelectExecutable(item.RootDir, pkgMeta) // 可执行文件、处理选择可执行文件时的错误
		if err != nil {
			return nil, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
		}
		existingManifest.Executable = executable
	}
	runtimeState, err := p.loadPersistedRuntime() // 运行环境状态
	if err != nil {
		return nil, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	cliName, err := cliNameFromID(item.ID) // CLI名称
	if err != nil {
		return nil, err
	}
	pluginBin := filepath.Join(item.RootDir, "node_modules", ".bin") // 插件可执行目录
	runtimeBin := ""                                                 // 运行环境可执行目录
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
	helpText, err := p.probeCliHelp(ctx, existingManifest.Executable, env) // 帮助文本
	if err != nil {
		return nil, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	modelCfg, err := p.resolveDefaultChatModel() // 模型配置
	if err != nil {
		return nil, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}
	// 下一项清单
	nextManifest, err := p.generateManifest(ctx, pkgcli.GenerateParams{
		HelpText:    helpText,
		PackageName: pkgMeta.Name,
		PackageMeta: pkgMeta,
		Executable:  existingManifest.Executable,
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
		return nil, ierror.Error(ierror.ErrCliManifestGenerateFailed, err)
	}

	// 重新生成清单时保留用户自定义的登录配置
	// 模型往往无法识别多步登录流程，例如 lark-cli 需要依次运行 config init 和 auth login
	// 只要旧清单设置了任一登录字段，就原样保留两个字段
	// 这样可以避免覆盖用户手动配置的登录步骤
	if len(existingManifest.LoginSteps) > 0 || len(existingManifest.LoginCommand) > 0 {
		nextManifest.LoginSteps = existingManifest.LoginSteps
		nextManifest.LoginCommand = existingManifest.LoginCommand
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
