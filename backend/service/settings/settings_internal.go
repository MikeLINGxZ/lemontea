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

// loadConfig reads the current config file and returns default values when it does not exist yet.
func (s *Settings) loadConfig() (*data_models.Config, error) {
	dataDir, err := dir.GetDataDir()
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	configPath := filepath.Join(dataDir, dir.ConfigFileName)
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config := defaultConfig(dataDir)
			if err := s.saveConfigToDir(dataDir, config); err != nil {
				return nil, ierror.Error(ierror.ErrSettingsSaveConfig, err)
			}
			return config, nil
		}
		return nil, ierror.Error(ierror.ErrSettingsReadConfig, err)
	}

	config := defaultConfig(dataDir)
	if err := json.Unmarshal(bytes, config); err != nil {
		return nil, ierror.Error(ierror.ErrSettingsParseConfig, err)
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

// saveConfigToDir writes a config file into the provided directory.
func (s *Settings) saveConfigToDir(targetDir string, config *data_models.Config) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return ierror.Error(ierror.ErrSettingsCreateDir, err)
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}

	return ierror.Error(ierror.ErrSettingsSaveConfig, os.WriteFile(filepath.Join(targetDir, dir.ConfigFileName), content, 0o644))
}

// defaultConfig builds the baseline config values used for first-time startup.
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

// copyFile copies one file to another path, creating the target file with standard app permissions.
func copyFile(sourcePath string, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return ierror.Error(ierror.ErrSettingsCopyFile, err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return ierror.Error(ierror.ErrSettingsCopyFile, err)
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	return ierror.Error(ierror.ErrSettingsCopyFile, err)
}
