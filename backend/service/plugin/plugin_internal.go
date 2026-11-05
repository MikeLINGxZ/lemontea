package plugin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// 读取应用数据目录，失败时记录问题并终止当前路径解析，返回文本
func mustDataDir() string {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return dir.HomeDir()
	}
	return dataDir
}

// 加载配置，返回配置及可能发生的错误
// 获取数据目录
// 读取文件内容
// 解析序列化内容
func (p *Plugin) loadConfig() (*data_models.Config, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(dataDir, dir.ConfigFileName) // 配置路径
	bytes, err := os.ReadFile(configPath)                    // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			// 配置
			cfg := &data_models.Config{
				Locale:     "zh-CN",
				Language:   "zh-CN",
				FontSize:   "md",
				DataDir:    dataDir,
				LogLevel:   "info",
				Extensions: []data_models.ExtensionItem{},
			}
			saveErr := p.saveConfig(cfg) // 保存配置时的错误
			if saveErr != nil {
				return nil, saveErr
			}
			return cfg, nil
		}
		return nil, err
	}

	cfg := &data_models.Config{}            // 配置
	decodeErr := json.Unmarshal(bytes, cfg) // 解析序列化内容时的错误
	if decodeErr != nil {
		return nil, decodeErr
	}
	if cfg.Extensions == nil {
		cfg.Extensions = []data_models.ExtensionItem{}
	}
	if cfg.DataDir == "" {
		cfg.DataDir = dataDir
	}
	return cfg, nil
}

// 保存配置，接收配置，返回操作错误
func (p *Plugin) saveConfig(config *data_models.Config) error {
	mkdirErr := os.MkdirAll(config.DataDir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return mkdirErr
	}
	content, err := json.MarshalIndent(config, "", "  ") // 内容、序列化Indent 对应的数据时的错误
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(config.DataDir, dir.ConfigFileName), content, 0o644)
}

// 在配置中按编号查找扩展，接收条目列表、唯一标识，返回扩展条目、是否满足条件
func findExtension(items []data_models.ExtensionItem, id string) (data_models.ExtensionItem, bool) {
	// 遍历条目列表，逐项取得当前条目
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return data_models.ExtensionItem{}, false
}

// 从配置的扩展列表移除指定条目，接收条目列表、唯一标识，返回扩展条目列表
func removeExtension(items []data_models.ExtensionItem, id string) []data_models.ExtensionItem {
	result := make([]data_models.ExtensionItem, 0, len(items)) // 处理make 对应的数据得到的结果
	// 遍历条目列表，逐项取得当前条目
	for _, item := range items {
		if item.ID != id {
			result = append(result, item)
		}
	}
	return result
}

// syncExtensionRuntime 获取指定扩展条目的运行状态和工具列表
// CLI 插件从 manifest.json 读取工具定义，MCP 扩展通过 MCP 管理器发现工具
// 接收本次调用的上下文、扩展条目，返回扩展条目
func (p *Plugin) syncExtensionRuntime(ctx context.Context, item data_models.ExtensionItem) data_models.ExtensionItem {
	if item.Kind == "cli" {
		manifest, err := pkgcli.LoadManifest(item.ConfigFilePath) // 清单、加载 CLI 扩展清单时的错误
		if err != nil || manifest.Executable == "" {
			item.RuntimeStatus = "error"
			item.RuntimeMessage = "manifest missing; reinstall"
			item.Tools = []data_models.ExtensionTool{}
			return item
		}
		item.Tools = projectCliTools(item, manifest)
		if item.Enabled {
			item.RuntimeStatus = "ready"
			item.RuntimeMessage = ""
		} else {
			item.RuntimeStatus = "idle"
			item.RuntimeMessage = ""
		}
		return item
	}
	if item.Kind != "mcp" {
		item.RuntimeStatus = "ready"
		item.RuntimeMessage = ""
		return item
	}
	if !item.Enabled {
		item.Tools = []data_models.ExtensionTool{}
		item.RuntimeStatus = "idle"
		item.RuntimeMessage = ""
		return item
	}

	tools, err := p.mcpManager.DiscoverTools(ctx, item) // 工具列表、处理发现工具列表时的错误
	if err != nil {
		item.Tools = []data_models.ExtensionTool{}
		item.RuntimeStatus = "error"
		item.RuntimeMessage = err.Error()
		return item
	}

	item.Tools = tools
	item.RuntimeStatus = "ready"
	item.RuntimeMessage = ""
	return item
}

// 把 CLI 清单中的工具映射为扩展工具列表，接收扩展条目、清单，返回扩展工具列表
func projectCliTools(item data_models.ExtensionItem, manifest pkgcli.Manifest) []data_models.ExtensionTool {
	result := make([]data_models.ExtensionTool, 0, len(manifest.Tools)) // 处理make 对应的数据得到的结果
	// 遍历工具列表，逐项取得当前
	for _, current := range manifest.Tools {
		result = append(result, data_models.ExtensionTool{
			ToolID:          pkgcli.CliToolID(item.Name, current.Name),
			ServerID:        item.ID,
			Name:            current.Name,
			Description:     current.Description,
			Enabled:         current.Enabled,
			RequiresConfirm: current.RequiresConfirm,
		})
	}
	return result
}
