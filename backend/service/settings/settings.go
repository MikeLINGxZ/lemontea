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

// QuickChatShortcutController validates and applies quick chat shortcuts.
type QuickChatShortcutController interface {
	CurrentShortcut() string
	ValidateShortcut(shortcut string) global_hotkey.ShortcutValidationResult
	UpdateShortcut(ctx context.Context, shortcut string) error
}

// Dependencies contains optional runtime integrations for Settings.
type Dependencies struct {
	QuickChatShortcuts QuickChatShortcutController
}

// Settings manages config persistence and file-system level settings actions.
type Settings struct {
	quickChatShortcuts QuickChatShortcutController
}

// NewSettings creates a Settings service with optional runtime dependencies.
func NewSettings(deps Dependencies) *Settings {
	return &Settings{quickChatShortcuts: deps.QuickChatShortcuts}
}

// LoadQuickChatShortcutForStartup reads the persisted quick chat shortcut for app startup.
func LoadQuickChatShortcutForStartup() (string, error) {
	config, err := (&Settings{}).loadConfig()
	if err != nil {
		return "", err
	}
	return config.QuickChatShortcut, nil
}

// LoadBootstrap loads the config file and provider list required by the settings frontend entry.
func (s *Settings) LoadBootstrap(ctx context.Context, input settings_dto.LoadBootstrapInput) (*settings_dto.LoadBootstrapOutput, error) {
	config, err := s.loadConfig()
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

// ValidateShortcutSettings checks whether a quick chat shortcut can be used.
func (s *Settings) ValidateShortcutSettings(ctx context.Context, input settings_dto.ValidateShortcutSettingsInput) (*settings_dto.ValidateShortcutSettingsOutput, error) {
	result := s.validateShortcut(input.QuickChatShortcut)
	return &settings_dto.ValidateShortcutSettingsOutput{
		QuickChatShortcut: result.Shortcut,
		Status:            string(result.Status),
		Message:           result.Message,
	}, nil
}

// SaveShortcutSettings persists and immediately applies the quick chat shortcut.
func (s *Settings) SaveShortcutSettings(ctx context.Context, input settings_dto.SaveShortcutSettingsInput) (*settings_dto.SaveShortcutSettingsOutput, error) {
	result := s.validateShortcut(input.QuickChatShortcut)
	if result.Status != global_hotkey.ShortcutValidationAvailable {
		return &settings_dto.SaveShortcutSettingsOutput{
			QuickChatShortcut: result.Shortcut,
			Status:            string(result.Status),
			Message:           result.Message,
		}, nil
	}

	config, err := s.loadConfig()
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	previousShortcut := config.QuickChatShortcut
	if s.quickChatShortcuts != nil {
		if err := s.quickChatShortcuts.UpdateShortcut(ctx, result.Shortcut); err != nil {
			return &settings_dto.SaveShortcutSettingsOutput{
				QuickChatShortcut: result.Shortcut,
				Status:            string(global_hotkey.ShortcutValidationConflict),
				Message:           err.Error(),
			}, nil
		}
	}

	config.QuickChatShortcut = result.Shortcut
	if err := s.saveConfigToDir(config.DataDir, config); err != nil {
		if s.quickChatShortcuts != nil {
			_ = s.quickChatShortcuts.UpdateShortcut(ctx, previousShortcut)
		}
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}

	return &settings_dto.SaveShortcutSettingsOutput{
		QuickChatShortcut: result.Shortcut,
		Status:            string(global_hotkey.ShortcutValidationAvailable),
	}, nil
}

// ApplyFileSettings migrates config and database artifacts into a new data directory and updates the locator file.
func (s *Settings) ApplyFileSettings(ctx context.Context, input settings_dto.ApplyFileSettingsInput) (*settings_dto.ApplyFileSettingsOutput, error) {
	sourceDir, err := dir.GetDataDir()
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	if input.TargetDir == "" {
		return nil, ierror.Error(ierror.ErrSettingsTargetDir, fmt.Errorf("target data dir is required"))
	}
	if err := os.MkdirAll(input.TargetDir, 0o755); err != nil {
		return nil, ierror.Error(ierror.ErrSettingsCreateDir, err)
	}
	if err := os.MkdirAll(filepath.Join(input.TargetDir, "logs"), 0o755); err != nil {
		return nil, ierror.Error(ierror.ErrSettingsCreateDir, err)
	}

	config, err := s.loadConfig()
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	config.DataDir = input.TargetDir

	if err := copyFile(filepath.Join(sourceDir, dir.DataBaseFileName), filepath.Join(input.TargetDir, dir.DataBaseFileName)); err != nil && !os.IsNotExist(err) {
		return nil, ierror.Error(ierror.ErrSettingsCopyFile, err)
	}
	if err := s.saveConfigToDir(input.TargetDir, config); err != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, err)
	}
	if err := dir.WriteLocatorDataDir(input.TargetDir); err != nil {
		return nil, ierror.Error(ierror.ErrSettingsWriteLocator, err)
	}

	return &settings_dto.ApplyFileSettingsOutput{}, nil
}

// SaveLocaleSettings updates locale and language in the persisted config file.
func (s *Settings) SaveLocaleSettings(ctx context.Context, input settings_dto.SaveLocaleSettingsInput) (*settings_dto.SaveLocaleSettingsOutput, error) {
	config, err := s.loadConfig()
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

// SaveDisplaySettings updates the persisted application font-size preference.
func (s *Settings) SaveDisplaySettings(ctx context.Context, input settings_dto.SaveDisplaySettingsInput) (*settings_dto.SaveDisplaySettingsOutput, error) {
	config, err := s.loadConfig()
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

func (s *Settings) validateShortcut(shortcut string) global_hotkey.ShortcutValidationResult {
	if s.quickChatShortcuts != nil {
		return s.quickChatShortcuts.ValidateShortcut(shortcut)
	}
	normalized, err := global_hotkey.NormalizeQuickChatShortcut(shortcut)
	if err != nil {
		return global_hotkey.ShortcutValidationResult{Status: global_hotkey.ShortcutValidationInvalid, Message: err.Error()}
	}
	return global_hotkey.ShortcutValidationResult{Shortcut: normalized, Status: global_hotkey.ShortcutValidationAvailable}
}
