package settings

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/global_hotkey"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
)

// loadConfig 读取当前配置文件，文件尚不存在时返回默认值
// 获取数据目录
// 读取文件内容
// 解析序列化内容
func (s *Settings) loadConfig() (*data_models.Config, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	configPath := filepath.Join(dataDir, dir.ConfigFileName) // 配置路径
	bytes, err := os.ReadFile(configPath)                    // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			config := defaultConfig(dataDir)                         // 配置
			saveConfigToDirErr := s.saveConfigToDir(dataDir, config) // 保存配置终点目录时的错误
			if saveConfigToDirErr != nil {
				return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigToDirErr)
			}
			return config, nil
		}
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, err)
	}

	config := defaultConfig(dataDir)           // 配置
	decodeErr := json.Unmarshal(bytes, config) // 解析序列化内容时的错误
	if decodeErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsParseConfig, decodeErr)
	}
	if !strings.Contains(string(bytes), `"memory"`) {
		config.Memory.Enabled = true
	}
	if config.DataDir == "" {
		config.DataDir = dataDir
	}
	if config.FontSize == "" {
		config.FontSize = "md"
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.Locale == "" {
		config.Locale = "zh-CN"
	}
	if config.Language == "" {
		config.Language = "zh-CN"
	}
	if config.QuickChatShortcut == "" {
		config.QuickChatShortcut = global_hotkey.DefaultQuickChatShortcut()
	}
	return config, nil
}

// saveConfigToDir 将配置文件写入指定目录
// 接收目标目录、配置，返回操作错误
func (s *Settings) saveConfigToDir(targetDir string, config *data_models.Config) error {
	mkdirErr := os.MkdirAll(targetDir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return ierror.Error(ierror.ErrSettingsCreateDir, mkdirErr)
	}

	content, err := json.MarshalIndent(config, "", "  ") // 内容、序列化Indent 对应的数据时的错误
	if err != nil {
		return ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}

	return ierror.Error(ierror.ErrSettingsSaveConfig, os.WriteFile(filepath.Join(targetDir, dir.ConfigFileName), content, 0o644))
}

// defaultConfig 构造首次启动时使用的默认配置
// 接收应用数据目录，返回配置
func defaultConfig(dataDir string) *data_models.Config {
	return &data_models.Config{
		Locale:            "zh-CN",
		Language:          "zh-CN",
		FontSize:          "md",
		DataDir:           dataDir,
		LogLevel:          "info",
		QuickChatShortcut: global_hotkey.DefaultQuickChatShortcut(),
		Memory: data_models.MemoryConfig{
			Enabled: true,
		},
	}
}

// copyFile 将源文件复制到目标路径，以应用的常规权限创建目标文件
// 接收来源路径、目标路径，返回操作错误
func copyFile(sourcePath string, targetPath string) error {
	sourceFile, err := os.Open(sourcePath) // 来源文件、打开当前数据时的错误
	if err != nil {
		return ierror.Error(ierror.ErrSettingsCopyFile, err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644) // 目标文件
	if err != nil {
		return ierror.Error(ierror.ErrSettingsCopyFile, err)
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	return ierror.Error(ierror.ErrSettingsCopyFile, err)
}
