package dir

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// defaultBaseDirName 定义应用数据根目录的默认名称
const defaultBaseDirName = ".lemontea"

// ConfigFileName 定义配置文件名称
const ConfigFileName = "config.json"

// DataBaseFileName 定义数据库文件名称
const DataBaseFileName = "data.db"

// 描述数据目录定位器使用的数据结构
type dataDirLocator struct {
	// 数据目录
	DataDir string `json:"data_dir"`
}

// GetDefaultBaseDir 返回 Lemontea 在用户主目录下使用的固定元数据目录
func GetDefaultBaseDir() (string, error) {
	homeDir, err := os.UserHomeDir() // 主界面目录、处理用户主界面目录时的错误
	if err != nil {
		return "", err
	}
	dataPath := filepath.Join(homeDir, defaultBaseDirName) // 数据路径
	mkdirErr := os.MkdirAll(dataPath, 0o755)               // 创建所需目录时的错误
	if mkdirErr != nil {
		return "", mkdirErr
	}
	return dataPath, nil
}

// GetLocatorFilePath 返回自定义数据目录定位文件的路径
func GetLocatorFilePath() (string, error) {
	baseDir, err := GetDefaultBaseDir() // 基础目录、获取默认基础目录时的错误
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "data_dir.json"), nil
}

// ReadLocatorDataDir 从定位文件读取自定义数据目录，定位文件不存在时返回空字符串
func ReadLocatorDataDir() (string, error) {
	locatorPath, err := GetLocatorFilePath() // 定位器路径、获取定位器文件路径时的错误
	if err != nil {
		return "", err
	}

	bytes, err := os.ReadFile(locatorPath) // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	// 定位器
	var locator dataDirLocator
	decodeErr := json.Unmarshal(bytes, &locator) // 解析序列化内容时的错误
	if decodeErr != nil {
		return "", decodeErr
	}

	return locator.DataDir, nil
}

// WriteLocatorDataDir 将当前自定义数据目录保存到用户主目录下的固定元数据目录
// 接收应用数据目录，返回操作错误
func WriteLocatorDataDir(dataDir string) error {
	locatorPath, err := GetLocatorFilePath() // 定位器路径、获取定位器文件路径时的错误
	if err != nil {
		return err
	}

	content, err := json.Marshal(dataDirLocator{DataDir: dataDir}) // 内容
	if err != nil {
		return err
	}

	return os.WriteFile(locatorPath, content, 0o644)
}

// GetDataDir 返回数据存储目录
func GetDataDir() (string, error) {
	if os.Getenv("LEMONTEA_DATA_DIR") != "" {
		mkdirErr := os.MkdirAll(os.Getenv("LEMONTEA_DATA_DIR"), 0o755) // 创建所需目录时的错误
		if mkdirErr != nil {
			return "", mkdirErr
		}
		return os.Getenv("LEMONTEA_DATA_DIR"), nil
	}

	locatorDir, err := ReadLocatorDataDir() // 定位器目录、读取定位器数据目录时的错误
	if err != nil {
		return "", err
	}
	if locatorDir != "" {
		mkdirErr2 := os.MkdirAll(locatorDir, 0o755) // 创建目录时的错误
		if mkdirErr2 != nil {
			return "", mkdirErr2
		}
		return locatorDir, nil
	}

	dataPath, err := GetDefaultBaseDir() // 数据路径
	if err != nil {
		return "", err
	}
	return dataPath, nil
}

// HomeDir 返回用户主目录
func HomeDir() string {
	homeDir, err := os.UserHomeDir() // 主界面目录、处理用户主界面目录时的错误
	if err != nil {
		return ""
	}
	return homeDir
}

// PluginsSubdirName 定义 data_dir 下统一存放各类扩展的父目录名称
const PluginsSubdirName = "plugins"

// MCPSubdirName 定义扩展根目录下存放 MCP 包的子目录名称
const MCPSubdirName = "mcp"

// PluginSubdirName 定义扩展根目录下存放 Node.js 插件的子目录名称
const PluginSubdirName = "plugin"

// CLISubdirName 定义扩展根目录下安装 CLI 插件的子目录名称
const CLISubdirName = "cli"

// CLIDataSubdirName 定义扩展根目录下持久化 CLI 插件数据的子目录名称
const CLIDataSubdirName = "cli_data"

// ExtensionsRoot 返回各类扩展的统一父目录 {dataDir}/plugins
// 接收应用数据目录，返回文本
func ExtensionsRoot(dataDir string) string {
	return filepath.Join(dataDir, PluginsSubdirName)
}

// MCPRoot 返回 MCP 包的根目录 {dataDir}/plugins/mcp
// 接收应用数据目录，返回文本
func MCPRoot(dataDir string) string {
	return filepath.Join(ExtensionsRoot(dataDir), MCPSubdirName)
}

// PluginRoot 返回 Node.js 插件的根目录 {dataDir}/plugins/plugin
// 接收应用数据目录，返回文本
func PluginRoot(dataDir string) string {
	return filepath.Join(ExtensionsRoot(dataDir), PluginSubdirName)
}

// CLIRoot 返回 CLI 插件的安装根目录 {dataDir}/plugins/cli
// 接收应用数据目录，返回文本
func CLIRoot(dataDir string) string {
	return filepath.Join(ExtensionsRoot(dataDir), CLISubdirName)
}

// CLIDataRoot 返回 CLI 插件的持久数据根目录 {dataDir}/plugins/cli_data
// 接收应用数据目录，返回文本
func CLIDataRoot(dataDir string) string {
	return filepath.Join(ExtensionsRoot(dataDir), CLIDataSubdirName)
}

// LegacyMCPRoot 返回旧版本迁移前使用的 MCP 根目录
// 接收应用数据目录，返回文本
func LegacyMCPRoot(dataDir string) string {
	return filepath.Join(dataDir, MCPSubdirName)
}

// LegacyPluginRoot 返回旧版本迁移前使用的 Node.js 插件根目录
// 接收应用数据目录，返回文本
func LegacyPluginRoot(dataDir string) string {
	return filepath.Join(dataDir, PluginSubdirName)
}

// SkillsSubdirName 定义 data_dir 下存放技能的子目录名称
const SkillsSubdirName = "skills"

// SkillsRoot 返回技能根目录 {dataDir}/skills
// 接收应用数据目录，返回文本
func SkillsRoot(dataDir string) string {
	return filepath.Join(dataDir, SkillsSubdirName)
}
