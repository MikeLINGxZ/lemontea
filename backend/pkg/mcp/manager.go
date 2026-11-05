package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	agenttool "trpc.group/trpc-go/trpc-agent-go/tool"
	toolmcp "trpc.group/trpc-go/trpc-agent-go/tool/mcp"
)

// ImportInput 描述导入本地 MCP 目录的请求
type ImportInput struct {
	// 来源目录
	SourceDir string
	// 启用状态
	Enabled bool
}

// Manager 负责导入 MCP 包，并提供运行时辅助操作
type Manager struct {
	// 应用数据目录
	dataDir string
}

// NewManager 创建使用指定应用数据目录的 MCP 管理器
// 接收应用数据目录，返回管理器
func NewManager(dataDir string) *Manager {
	return &Manager{dataDir: dataDir}
}

// ImportMCP 将本地 MCP 包复制到应用数据目录并返回已保存的元数据
// 接收调用上下文和请求参数，返回扩展条目及可能发生的错误
func (m *Manager) ImportMCP(ctx context.Context, input ImportInput) (data_models.ExtensionItem, error) {
	_ = ctx

	manifest, err := loadManifestOptional(input.SourceDir) // 清单、加载清单可选时的错误
	if err != nil {
		return data_models.ExtensionItem{}, err
	}
	if manifest.ConfigFile == "" {
		manifest.ConfigFile = "mcp.json"
	}
	if manifest.Name == "" {
		manifest.Name = filepath.Base(input.SourceDir)
	}

	versionDir := strings.TrimSpace(manifest.Version) // 版本目录
	if versionDir == "" {
		versionDir = "default"
	}

	targetDir := filepath.Join(dir.MCPRoot(m.dataDir), manifest.Name, versionDir) // 目标目录
	copyDirectoryErr := copyDirectory(input.SourceDir, targetDir)                 // 复制目录时的错误
	if copyDirectoryErr != nil {
		return data_models.ExtensionItem{}, copyDirectoryErr
	}
	writeManifestErr := writeManifest(targetDir, manifest) // 写入清单时的错误
	if writeManifestErr != nil {
		return data_models.ExtensionItem{}, writeManifestErr
	}

	configPath := filepath.Join(targetDir, manifest.ConfigFile) // 配置路径
	_, loadServerConfigErr := loadServerConfig(configPath)      // 加载服务配置时的错误
	if loadServerConfigErr != nil {
		return data_models.ExtensionItem{}, loadServerConfigErr
	}

	return data_models.ExtensionItem{
		ID:             buildExtensionID("mcp", manifest.Name, versionDir),
		Name:           manifest.Name,
		Description:    manifest.Description,
		Author:         manifest.Author,
		Version:        manifest.Version,
		Kind:           "mcp",
		Enabled:        input.Enabled,
		RuntimeStatus:  "idle",
		RuntimeMessage: "",
		RootDir:        targetDir,
		SourceDir:      input.SourceDir,
		ConfigFilePath: configPath,
		Tools:          []data_models.ExtensionTool{},
	}, nil
}

// ImportPlugin 将本地插件包复制到应用数据目录并返回已保存的元数据
// 接收调用上下文和请求参数，返回扩展条目及可能发生的错误
func (m *Manager) ImportPlugin(ctx context.Context, input ImportInput) (data_models.ExtensionItem, error) {
	_ = ctx

	manifest, err := loadManifestOptional(input.SourceDir) // 清单、加载清单可选时的错误
	if err != nil {
		return data_models.ExtensionItem{}, err
	}

	versionDir := strings.TrimSpace(manifest.Version) // 版本目录
	if versionDir == "" {
		versionDir = "default"
	}

	targetDir := filepath.Join(dir.PluginRoot(m.dataDir), manifest.Name, versionDir) // 目标目录
	copyDirectoryErr := copyDirectory(input.SourceDir, targetDir)                    // 复制目录时的错误
	if copyDirectoryErr != nil {
		return data_models.ExtensionItem{}, copyDirectoryErr
	}

	return data_models.ExtensionItem{
		ID:             buildExtensionID("plugin", manifest.Name, versionDir),
		Name:           manifest.Name,
		Description:    manifest.Description,
		Author:         manifest.Author,
		Version:        manifest.Version,
		Kind:           "plugin",
		Enabled:        input.Enabled,
		RuntimeStatus:  "idle",
		RuntimeMessage: "",
		RootDir:        targetDir,
		SourceDir:      input.SourceDir,
		ConfigFilePath: "",
		Tools:          []data_models.ExtensionTool{},
	}, nil
}

// 构造扩展编号，接收类别标识、当前名称、版本，返回文本
func buildExtensionID(kind string, name string, version string) string {
	return fmt.Sprintf("%s:%s:%s", kind, name, version)
}

// 处理MCP工具集合名称，接收扩展编号，返回文本
func mcpToolSetName(extensionID string) string {
	// 构建器
	var builder strings.Builder
	builder.Grow(len(extensionID))

	lastUnderscore := false // 最后一个下划线
	// 遍历扩展编号，逐项取得读取源
	for _, r := range extensionID {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
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
		return "mcp"
	}
	return name
}

// 处理MCP工具名称前缀，接收扩展条目，返回文本
func mcpToolNamePrefix(item data_models.ExtensionItem) string {
	return mcpToolSetName(item.ID) + "_"
}

// DiscoverTools 初始化一个 MCP 扩展，并返回暴露给智能体的带前缀工具 ID
// 接收本次调用的上下文、扩展条目，返回扩展工具列表及可能发生的错误
// 构造工具集合
// 结束时关闭当前资源
func (m *Manager) DiscoverTools(ctx context.Context, item data_models.ExtensionItem) ([]data_models.ExtensionTool, error) {
	toolSet, err := m.BuildToolSet(item, nil) // 工具集合、构造工具集合时的错误
	if err != nil {
		return nil, err
	}

	// 结束时关闭当前资源
	defer toolSet.Close()

	if /* 初始化工具集合、是否成功取得对应值 */ initToolSet, ok := toolSet.(*toolmcp.ToolSet); ok {
		initErr := initToolSet.Init(ctx) // 初始化当前数据时的错误
		if initErr != nil {
			return nil, initErr
		}
	}

	tools := toolSet.Tools(ctx)                                // 工具列表
	result := make([]data_models.ExtensionTool, 0, len(tools)) // 处理make 对应的数据得到的结果
	safePrefix := mcpToolNamePrefix(item)                      // 安全前缀
	legacyPrefix := item.ID + "_"                              // 旧版前缀
	// 遍历工具列表，逐项取得当前
	for _, current := range tools {
		decl := current.Declaration() // 声明
		name := decl.Name             // 当前名称
		switch {
		case strings.HasPrefix(name, safePrefix):
			name = strings.TrimPrefix(name, safePrefix)
		case strings.HasPrefix(name, legacyPrefix):
			name = strings.TrimPrefix(name, legacyPrefix)
		}
		result = append(result, data_models.ExtensionTool{
			ToolID:          decl.Name,
			ServerID:        item.ID,
			Name:            name,
			Description:     decl.Description,
			Enabled:         true,
			RequiresConfirm: false,
		})
	}
	return result, nil
}

// BuildToolSet 为指定扩展创建 MCP 工具集，可按指定工具 ID 筛选
// 接收扩展条目、启用状态工具索引Ds，返回工具集合及可能发生的错误
func (m *Manager) BuildToolSet(item data_models.ExtensionItem, enabledToolIDs []string) (agenttool.ToolSet, error) {
	configBytes, err := os.ReadFile(item.ConfigFilePath) // 连接配置内容和读取错误
	if err != nil {
		return nil, err
	}
	return m.BuildToolSetFromConfig(item, enabledToolIDs, configBytes)
}

// BuildToolSetFromConfig 使用已读取的连接配置构建 MCP 工具集
// 接收扩展条目、已选工具 ID 和配置内容，返回工具集及可能发生的错误
func (m *Manager) BuildToolSetFromConfig(item data_models.ExtensionItem, enabledToolIDs []string, configBytes []byte) (agenttool.ToolSet, error) {
	config, err := parseServerConfig(configBytes) // 服务配置和解析错误
	if err != nil {
		return nil, err
	}

	safeToolSetName := mcpToolSetName(item.ID)                            // 安全工具集合名称
	options := []toolmcp.ToolSetOption{toolmcp.WithName(safeToolSetName)} // 选项列表
	if len(enabledToolIDs) > 0 {
		// 遍历启用状态工具索引Ds，逐项取得工具编号
		for _, toolID := range enabledToolIDs {
			if toolID == item.ID {
				return toolmcp.NewMCPToolSet(config.toToolSetConfig(), options...), nil
			}
		}

		rawNames := make([]string, 0, len(enabledToolIDs)) // 原始值名称列表
		safePrefix := safeToolSetName + "_"                // 安全前缀
		legacyPrefix := item.ID + "_"                      // 旧版前缀
		// 遍历启用状态工具索引Ds，逐项取得工具编号
		for _, toolID := range enabledToolIDs {
			switch {
			case strings.HasPrefix(toolID, safePrefix):
				rawNames = append(rawNames, strings.TrimPrefix(toolID, safePrefix))
			case strings.HasPrefix(toolID, legacyPrefix):
				rawNames = append(rawNames, strings.TrimPrefix(toolID, legacyPrefix))
			}
		}
		if len(rawNames) == 0 {
			return emptyToolSet{name: item.ID}, nil
		}
		options = append(options, toolmcp.WithToolFilterFunc(agenttool.NewIncludeToolNamesFilter(rawNames...)))
	}

	return toolmcp.NewMCPToolSet(config.toToolSetConfig(), options...), nil
}

// 描述空工具集合使用的数据结构
type emptyToolSet struct {
	// 当前名称
	name string
}

// 处理工具列表，接收调用上下文，返回工具列表
func (e emptyToolSet) Tools(context.Context) []agenttool.Tool {
	return []agenttool.Tool{}
}

// 关闭当前资源，返回操作错误
func (e emptyToolSet) Close() error {
	return nil
}

// 处理名称，返回文本
func (e emptyToolSet) Name() string {
	return e.name
}
