package settings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/global_hotkey"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/version"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings/settings_dto"
)

// QuickChatShortcutController 校验并应用快捷聊天快捷键
type QuickChatShortcutController interface {
	// 处理当前快捷键，返回文本
	CurrentShortcut() string
	// 校验快捷键，接收快捷键，返回快捷键校验结果
	ValidateShortcut(shortcut string) global_hotkey.ShortcutValidationResult
	// 更新快捷键，接收本次调用的上下文、快捷键，返回操作错误
	UpdateShortcut(ctx context.Context, shortcut string) error
}

// Dependencies 声明设置服务可选的运行时依赖
type Dependencies struct {
	// 快捷聊天快捷键列表
	QuickChatShortcuts QuickChatShortcutController
}

// Settings 管理配置持久化和涉及文件系统的设置操作
type Settings struct {
	// 快捷聊天快捷键列表
	quickChatShortcuts QuickChatShortcutController
}

// NewSettings 使用可选的运行时依赖创建设置服务
// 接收deps 对应的数据，返回设置
func NewSettings(deps Dependencies) *Settings {
	return &Settings{quickChatShortcuts: deps.QuickChatShortcuts}
}

// LoadQuickChatShortcutForStartup 读取应用启动时需要的已保存快捷聊天快捷键
// 返回文本及可能发生的错误
func LoadQuickChatShortcutForStartup() (string, error) {
	config, err := (&Settings{}).loadConfig() // 配置、本次操作的错误
	if err != nil {
		return "", err
	}
	return config.QuickChatShortcut, nil
}

// LoadBootstrap 读取设置页初始化所需的配置文件和提供商列表
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Settings) LoadBootstrap(ctx context.Context, input settings_dto.LoadBootstrapInput) (*settings_dto.LoadBootstrapOutput, error) {
	config, err := s.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	return &settings_dto.LoadBootstrapOutput{
		Bootstrap: view_model.SettingsBootstrap{
			Locale:            config.Locale,
			Language:          config.Language,
			FontSize:          config.FontSize,
			DataDir:           config.DataDir,
			LogLevel:          config.LogLevel,
			QuickChatShortcut: config.QuickChatShortcut,
			DefaultProviderID: config.DefaultProviderID,
			Version:           version.ApplicationVersion,
		},
	}, nil
}

// ValidateShortcutSettings 检查所给的快捷聊天快捷键是否可用
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Settings) ValidateShortcutSettings(ctx context.Context, input settings_dto.ValidateShortcutSettingsInput) (*settings_dto.ValidateShortcutSettingsOutput, error) {
	result := s.validateShortcut(input.QuickChatShortcut) // 解析快捷键并检查注册可用性得到的结果
	return &settings_dto.ValidateShortcutSettingsOutput{
		QuickChatShortcut: result.Shortcut,
		Status:            string(result.Status),
		Message:           result.Message,
	}, nil
}

// SaveShortcutSettings 保存快捷聊天快捷键并立即应用
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Settings) SaveShortcutSettings(ctx context.Context, input settings_dto.SaveShortcutSettingsInput) (*settings_dto.SaveShortcutSettingsOutput, error) {
	result := s.validateShortcut(input.QuickChatShortcut) // 解析快捷键并检查注册可用性得到的结果
	if result.Status != global_hotkey.ShortcutValidationAvailable {
		return &settings_dto.SaveShortcutSettingsOutput{
			QuickChatShortcut: result.Shortcut,
			Status:            string(result.Status),
			Message:           result.Message,
		}, nil
	}

	config, err := s.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	previousShortcut := config.QuickChatShortcut // 先前快捷键
	if s.quickChatShortcuts != nil {
		updateErr := s.quickChatShortcuts.UpdateShortcut(ctx, result.Shortcut) // 更新快捷键时的错误
		if updateErr != nil {
			return &settings_dto.SaveShortcutSettingsOutput{
				QuickChatShortcut: result.Shortcut,
				Status:            string(global_hotkey.ShortcutValidationConflict),
				Message:           updateErr.Error(),
			}, nil
		}
	}

	config.QuickChatShortcut = result.Shortcut
	saveConfigToDirErr := s.saveConfigToDir(config.DataDir, config) // 保存配置终点目录时的错误
	if saveConfigToDirErr != nil {
		if s.quickChatShortcuts != nil {
			_ = s.quickChatShortcuts.UpdateShortcut(ctx, previousShortcut)
		}
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigToDirErr)
	}

	return &settings_dto.SaveShortcutSettingsOutput{
		QuickChatShortcut: result.Shortcut,
		Status:            string(global_hotkey.ShortcutValidationAvailable),
	}, nil
}

// ApplyFileSettings 将配置和数据库文件迁移到新的数据目录，并更新目录定位文件
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
// 获取数据目录
// 创建所需目录
// 写入定位器数据目录
func (s *Settings) ApplyFileSettings(ctx context.Context, input settings_dto.ApplyFileSettingsInput) (*settings_dto.ApplyFileSettingsOutput, error) {
	sourceDir, err := dir.GetDataDir() // 来源目录、获取数据目录时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	if input.TargetDir == "" {
		return nil, ierror.Error(ierror.ErrSettingsTargetDir, fmt.Errorf("target data dir is required"))
	}
	mkdirErr := os.MkdirAll(input.TargetDir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsCreateDir, mkdirErr)
	}
	mkdirErr2 := os.MkdirAll(filepath.Join(input.TargetDir, "logs"), 0o755) // 创建目录时的错误
	if mkdirErr2 != nil {
		return nil, ierror.Error(ierror.ErrSettingsCreateDir, mkdirErr2)
	}

	config, err := s.loadConfig() // 配置
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	config.DataDir = input.TargetDir

	// 复制文件时的错误
	copyFileErr := copyFile(filepath.Join(sourceDir, dir.DataBaseFileName), filepath.Join(input.TargetDir, dir.DataBaseFileName))
	if copyFileErr != nil && !os.IsNotExist(copyFileErr) {
		return nil, ierror.Error(ierror.ErrSettingsCopyFile, copyFileErr)
	}
	saveConfigToDirErr := s.saveConfigToDir(input.TargetDir, config) // 保存配置终点目录时的错误
	if saveConfigToDirErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigToDirErr)
	}
	writeErr := dir.WriteLocatorDataDir(input.TargetDir) // 写入定位器数据目录时的错误
	if writeErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsWriteLocator, writeErr)
	}

	return &settings_dto.ApplyFileSettingsOutput{}, nil
}

// SaveLocaleSettings 更新配置文件中的地区和语言设置
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Settings) SaveLocaleSettings(ctx context.Context, input settings_dto.SaveLocaleSettingsInput) (*settings_dto.SaveLocaleSettingsOutput, error) {
	config, err := s.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	config.Locale = input.Locale
	config.Language = input.Language

	err = s.saveConfigToDir(config.DataDir, config)
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}

	return &settings_dto.SaveLocaleSettingsOutput{}, nil
}

// SaveDisplaySettings 更新已保存的应用字号偏好
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Settings) SaveDisplaySettings(ctx context.Context, input settings_dto.SaveDisplaySettingsInput) (*settings_dto.SaveDisplaySettingsOutput, error) {
	config, err := s.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	config.FontSize = input.FontSize

	err = s.saveConfigToDir(config.DataDir, config)
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}

	return &settings_dto.SaveDisplaySettingsOutput{}, nil
}

// 解析快捷键并检查注册可用性，接收快捷键，返回快捷键校验结果
func (s *Settings) validateShortcut(shortcut string) global_hotkey.ShortcutValidationResult {
	if s.quickChatShortcuts != nil {
		return s.quickChatShortcuts.ValidateShortcut(shortcut)
	}
	normalized, err := global_hotkey.NormalizeQuickChatShortcut(shortcut) // 规范化值、规范化快捷聊天快捷键时的错误
	if err != nil {
		return global_hotkey.ShortcutValidationResult{Status: global_hotkey.ShortcutValidationInvalid, Message: err.Error()}
	}
	return global_hotkey.ShortcutValidationResult{Shortcut: normalized, Status: global_hotkey.ShortcutValidationAvailable}
}
