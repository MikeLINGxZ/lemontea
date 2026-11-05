package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	agenttools "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent/tools"
	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	pkgmcp "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/mcp"
	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
	pkgRuntimeState "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/runtime_state"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

// Plugin 管理导入的 MCP 工具及其他插件条目
type Plugin struct {
	// MCP管理器
	mcpManager *pkgmcp.Manager
	// CLI管理器
	cliManager *pkgcli.Manager
	// CLI 管理器初始化互斥锁
	cliManagerMu sync.Mutex
	// 安装任务登记和扩展配置提交的互斥锁
	cliInstallMu sync.Mutex
	// 按安装名称登记正在运行的 CLI 安装任务
	cliInstalling map[string]struct{}
	// 处理探测CLI帮助，接收本次调用的上下文、可执行文件、环境变量，返回文本及可能发生的错误
	probeCliHelp func(ctx context.Context, executable string, env []string) (string, error)
	// 处理生成清单，接收本次调用的上下文、参数，返回清单及可能发生的错误
	generateManifest func(ctx context.Context, params pkgcli.GenerateParams) (pkgcli.Manifest, error)
	// 解析默认聊天模型，返回默认聊天模型及可能发生的错误
	resolveDefaultChatModel func() (defaultChatModel, error)
	// 加载持久化运行环境，返回状态快照及可能发生的错误
	loadPersistedRuntime func() (pkgRuntimeState.StateSnapshot, error)
	// Wails 应用实例
	wailsApp *application.App
}

// NewPlugin 使用当前应用的数据目录创建插件服务
// 返回插件
func NewPlugin() *Plugin {
	return &Plugin{
		mcpManager:              pkgmcp.NewManager(mustDataDir()),
		cliInstalling:           make(map[string]struct{}),
		probeCliHelp:            pkgcli.ProbeHelp,
		generateManifest:        pkgcli.Generate,
		resolveDefaultChatModel: loadDefaultChatModel,
		loadPersistedRuntime:    pkgRuntimeState.LoadPersistedState,
	}
}

// resolveCliManager 首次调用时读取运行时 state.json 中的 Node.js 和 npm 路径，并据此创建 CLI 管理器
// 运行时尚未下载或就绪时返回错误
func (p *Plugin) resolveCliManager() (*pkgcli.Manager, error) {
	p.cliManagerMu.Lock()
	defer p.cliManagerMu.Unlock()
	if p.cliManager != nil {
		return p.cliManager, nil
	}
	state, err := p.loadPersistedRuntime() // 状态、加载持久化运行环境时的错误
	if err != nil {
		return nil, err
	}
	if state.State != "ready" || state.NodePath == "" || state.NpmPath == "" {
		return nil, ierror.Error(ierror.ErrCliRuntimeUnavailable, errors.New("plugin runtime not ready"))
	}
	p.cliManager = pkgcli.NewManager(mustDataDir(), state.NodePath, state.NpmPath)
	return p.cliManager, nil
}

// 描述默认聊天模型使用的数据结构
type defaultChatModel struct {
	// 服务基础地址
	BaseURL string
	// API 密钥
	APIKey string
	// 模型名称
	ModelName string
	// 提供商类型
	ProviderType pkgProvider.Type
}

// loadDefaultChatModel 读取已保存的默认提供商及其选定的默认聊天模型
// 返回默认聊天模型及可能发生的错误
// 构建存储
// 列出提供商列表
func loadDefaultChatModel() (defaultChatModel, error) {
	stor, err := storage.NewStorage() // 存储、构建存储时的错误
	if err != nil {
		return defaultChatModel{}, err
	}
	providers, err := stor.ListProviders() // 提供商列表
	if err != nil {
		return defaultChatModel{}, err
	}
	defaultProviderID := readDefaultProviderIDLocal() // 默认提供商编号
	if defaultProviderID == 0 {
		return defaultChatModel{}, os.ErrNotExist
	}
	// 遍历提供商列表，逐项取得提供商
	for _, provider := range providers {
		if provider.ID != defaultProviderID {
			continue
		}
		defaultModel, err := stor.GetDefaultModel(provider.ID) // 默认模型、获取默认模型时的错误
		if err != nil {
			return defaultChatModel{}, err
		}
		if defaultModel == nil {
			return defaultChatModel{}, os.ErrNotExist
		}
		models, err := stor.ListModelsForProvider(provider.ID) // 模型列表
		if err != nil {
			return defaultChatModel{}, err
		}
		// 遍历模型列表，逐项取得当前
		for _, current := range models {
			if current.ID == defaultModel.ModelId {
				return defaultChatModel{
					BaseURL:      provider.BaseUrl,
					APIKey:       provider.ApiKey,
					ModelName:    current.Model,
					ProviderType: provider.ProviderType,
				}, nil
			}
		}
		return defaultChatModel{}, os.ErrNotExist
	}
	return defaultChatModel{}, os.ErrNotExist
}

// 从本地配置读取默认提供商编号，返回无符号整数
func readDefaultProviderIDLocal() uint {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return 0
	}
	bytes, err := os.ReadFile(filepath.Join(dataDir, dir.ConfigFileName)) // 原始字节内容
	if err != nil {
		return 0
	}
	// 配置
	var cfg data_models.Config
	decodeErr := json.Unmarshal(bytes, &cfg) // 解析序列化内容时的错误
	if decodeErr != nil {
		return 0
	}
	return cfg.DefaultProviderID
}

// ListExtensions 返回配置中已保存的所有插件和 MCP 扩展条目
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) ListExtensions(ctx context.Context, input plugin_dto.ListExtensionsInput) (*plugin_dto.ListExtensionsOutput, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	changed := false                                                      // 是否有变化
	items := make([]data_models.ExtensionItem, 0, len(config.Extensions)) // 条目列表
	// 遍历扩展列表，逐项取得当前条目
	for _, item := range config.Extensions {
		next := item // 下一项
		if (item.Kind == "mcp" || item.Kind == "cli") && (len(item.Tools) == 0 || item.RuntimeStatus == "") {
			next = p.syncExtensionRuntime(ctx, item)
			changed = changed || next.RuntimeStatus != item.RuntimeStatus || next.RuntimeMessage != item.RuntimeMessage || len(next.Tools) != len(item.Tools)
		}
		items = append(items, next)
	}
	if changed {
		config.Extensions = items
		saveConfigErr := p.saveConfig(config) // 保存配置时的错误
		if saveConfigErr != nil {
			return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
		}
	}
	return &plugin_dto.ListExtensionsOutput{Extensions: items}, nil
}

// ImportExtension 将本地扩展目录导入应用数据目录，并在配置中保存扩展条目
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) ImportExtension(ctx context.Context, input plugin_dto.ImportExtensionInput) (*plugin_dto.ImportExtensionOutput, error) {
	if input.Kind != "mcp" && input.Kind != "plugin" {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, os.ErrInvalid)
	}

	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	// 当前条目
	var item data_models.ExtensionItem
	if input.Kind == "plugin" {
		item, err = p.mcpManager.ImportPlugin(ctx, pkgmcp.ImportInput{
			SourceDir: input.Path,
			Enabled:   true,
		})
	} else {
		item, err = p.mcpManager.ImportMCP(ctx, pkgmcp.ImportInput{
			SourceDir: input.Path,
			Enabled:   true,
		})
	}
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}

	item = p.syncExtensionRuntime(ctx, item)

	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}

	return &plugin_dto.ImportExtensionOutput{Extension: item}, nil
}

// GetExtensionDetail 返回指定扩展及其可编辑的配置文件内容
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) GetExtensionDetail(ctx context.Context, input plugin_dto.GetExtensionDetailInput) (*plugin_dto.GetExtensionDetailOutput, error) {
	_ = ctx
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, os.ErrNotExist)
	}
	if item.ConfigFilePath == "" {
		return &plugin_dto.GetExtensionDetailOutput{
			Extension:  item,
			ConfigText: "",
		}, nil
	}
	bytes, err := os.ReadFile(item.ConfigFilePath) // 原始字节内容
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &plugin_dto.GetExtensionDetailOutput{Extension: item, ConfigText: ""}, nil
		}
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, err)
	}
	return &plugin_dto.GetExtensionDetailOutput{
		Extension:  item,
		ConfigText: string(bytes),
	}, nil
}

// ToggleExtension 更新扩展的启用状态，并在启用时重新发现工具
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) ToggleExtension(ctx context.Context, input plugin_dto.ToggleExtensionInput) (*plugin_dto.ToggleExtensionOutput, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, os.ErrNotExist)
	}
	item.Enabled = input.Enabled
	item = p.syncExtensionRuntime(ctx, item)
	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}
	return &plugin_dto.ToggleExtensionOutput{Extension: item}, nil
}

// ReloadExtension 根据指定扩展当前的磁盘配置重新发现工具
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) ReloadExtension(ctx context.Context, input plugin_dto.ReloadExtensionInput) (*plugin_dto.ReloadExtensionOutput, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, os.ErrNotExist)
	}
	item = p.syncExtensionRuntime(ctx, item)
	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}
	return &plugin_dto.ReloadExtensionOutput{Extension: item}, nil
}

// DeleteExtension 从配置中移除指定扩展，并删除其安装目录
// CLI 插件的 plugins/cli_data/<name>/ 目录会保留，其中包含登录状态和清单
// 可通过 ResetCliData 单独清理该目录
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) DeleteExtension(ctx context.Context, input plugin_dto.DeleteExtensionInput) (*plugin_dto.DeleteExtensionOutput, error) {
	_ = ctx
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, os.ErrNotExist)
	}
	removeErr := os.RemoveAll(item.RootDir) // 移除指定路径下的数据时的错误
	if removeErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, removeErr)
	}
	config.Extensions = removeExtension(config.Extensions, item.ID)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}
	return &plugin_dto.DeleteExtensionOutput{}, nil
}

// SaveExtensionConfig 写入编辑后的扩展配置，并重新发现工具
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) SaveExtensionConfig(ctx context.Context, input plugin_dto.SaveExtensionConfigInput) (*plugin_dto.SaveExtensionConfigOutput, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, os.ErrNotExist)
	}
	if item.Kind == "cli" {
		// 本次处理的输出、更新CLI清单时的错误
		out, err := p.UpdateCliManifest(ctx, plugin_dto.UpdateCliManifestInput{ID: input.ID, ManifestText: input.ConfigText})
		if err != nil {
			return nil, err
		}
		return &plugin_dto.SaveExtensionConfigOutput{Extension: out.Extension}, nil
	}
	if item.ConfigFilePath == "" {
		return &plugin_dto.SaveExtensionConfigOutput{Extension: item}, nil
	}
	writeErr := os.WriteFile(item.ConfigFilePath, []byte(input.ConfigText), 0o644) // 写入文件内容时的错误
	if writeErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, writeErr)
	}
	item = p.syncExtensionRuntime(ctx, item)
	config.Extensions = append(removeExtension(config.Extensions, item.ID), item)
	saveConfigErr := p.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}
	return &plugin_dto.SaveExtensionConfigOutput{Extension: item}, nil
}

// ListAvailableTools 返回内置工具及当前配置中可发现的全部已启用扩展工具组
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) ListAvailableTools(ctx context.Context, input plugin_dto.ListAvailableToolsInput) (*plugin_dto.ListAvailableToolsOutput, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	result := make([]plugin_dto.ToolItem, 0) // 处理make 对应的数据得到的结果
	// 遍历内置工具元数据列表，逐项取得元信息
	for _, meta := range builtinToolMetas() {
		result = append(result, plugin_dto.ToolItem{
			ID:          meta.Name,
			Name:        meta.Name,
			Description: meta.Description,
			Category:    meta.Category,
		})
	}

	// 遍历扩展列表，逐项取得当前条目
	for _, item := range config.Extensions {
		if !item.Enabled || (item.Kind != "mcp" && item.Kind != "cli") {
			continue
		}
		resolved := item // 已处理
		if len(resolved.Tools) == 0 {
			resolved = p.syncExtensionRuntime(ctx, item)
		}
		if len(resolved.Tools) == 0 {
			continue
		}
		result = append(result, plugin_dto.ToolItem{
			ID:          resolved.ID,
			Name:        resolved.Name,
			Description: resolved.Description,
			Category:    resolved.Kind,
		})
	}

	return &plugin_dto.ListAvailableToolsOutput{Tools: result}, nil
}

// 返回内置工具的标识和展示信息，返回工具元信息列表
func builtinToolMetas() []agenttools.ToolMeta {
	return []agenttools.ToolMeta{
		agenttools.DateTimeMeta(),
		agenttools.ShellMeta(),
		agenttools.BuildQuestionTool(),
		agenttools.WebFetchMeta(),
		agenttools.BuildWebSearchTool(),
		agenttools.BuildTodoWriteTool(),
		{
			Name:        agenttools.SkillToolName,
			Description: "Load a task template or workflow instruction for repeatable user tasks",
			Category:    agenttools.CategoryBuiltin,
		},
	}
}
