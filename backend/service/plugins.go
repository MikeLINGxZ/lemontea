package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	llmtools "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/tools"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils/ierror"
)

const EventSettingsPluginsChanged = "settings:plugins:changed"

func (s *Service) SelectPluginFolder() (string, error) {
	path, err := s.app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(i18n.TCurrent("app.dialog.select_plugin_folder", nil)).
		PromptForSingleSelection()
	if err != nil {
		return "", ierror.NewError(err)
	}
	return path, nil
}

func (s *Service) AddPluginFromFolder(path string) (*plugins.Summary, error) {
	if s.plugins == nil {
		return nil, ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	summary, err := s.plugins.InstallFromFolder(path)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	defer s.emitPluginsChanged()
	if err := s.plugins.Enable(context.Background(), summary.ID); err != nil {
		return nil, ierror.NewError(err)
	}
	updated, ok := s.plugins.Get(summary.ID)
	if !ok {
		return nil, ierror.NewError(fmt.Errorf("plugin %s not found after install", summary.ID))
	}
	return updated, nil
}

func (s *Service) ListPlugins() []plugins.Summary {
	if s.plugins == nil {
		return []plugins.Summary{}
	}
	return s.plugins.List()
}

func (s *Service) GetPluginRuntimeStatus(ctx context.Context) plugins.RuntimeStatus {
	if s.plugins == nil {
		return plugins.RuntimeStatus{
			Available: false,
			Error:     "plugin manager is not available",
		}
	}
	return s.plugins.RuntimeStatus(ctx)
}

func (s *Service) DownloadPluginRuntime(ctx context.Context) (*plugins.RuntimeStatus, error) {
	if s.plugins == nil {
		return nil, ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	status, err := s.plugins.DownloadRuntime(ctx)
	if err != nil {
		return &status, ierror.NewError(err)
	}
	s.emitPluginsChanged()
	return &status, nil
}

func (s *Service) GetPlugin(id string) (*plugins.Summary, error) {
	if s.plugins == nil {
		return nil, ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	summary, ok := s.plugins.Get(id)
	if !ok {
		return nil, ierror.NewError(fmt.Errorf("plugin %s not found", id))
	}
	return summary, nil
}

func (s *Service) SetPluginEnabled(id string, enabled bool) error {
	if s.plugins == nil {
		return ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	var err error
	if enabled {
		err = s.plugins.Enable(context.Background(), id)
	} else {
		err = s.plugins.Disable(id)
	}
	if err != nil {
		return ierror.NewError(err)
	}
	s.emitPluginsChanged()
	return nil
}

func (s *Service) DeletePlugin(id string) error {
	if s.plugins == nil {
		return ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	if err := s.plugins.Delete(id); err != nil {
		return ierror.NewError(err)
	}
	s.emitPluginsChanged()
	return nil
}

func (s *Service) OpenPluginSettingsWindow(id string) error {
	if s.plugins == nil {
		return ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	summary, ok := s.plugins.Get(id)
	if !ok {
		return ierror.NewError(fmt.Errorf("plugin %s not found", id))
	}
	if !summary.HasSettings {
		return ierror.NewError(fmt.Errorf("plugin %s does not provide settings", id))
	}

	name := "window_plugin_settings_" + id
	title := summary.Name + " " + i18n.TCurrent("app.window.plugin_settings_title", nil)
	url := fmt.Sprintf("/?entry=plugin_view&id=%s&view=settings", id)
	if existing, ok := s.app.Window.GetByName(name); ok {
		existing.SetURL(url)
		existing.Focus()
		existing.Show()
		return nil
	}
	window := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  name,
		Title: title,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              url,
		Width:            900,
		Height:           720,
		MinWidth:         520,
		MinHeight:        520,
	})
	window.Focus()
	s.centerWindowOnHomeScreen(window)
	window.Show()
	return nil
}

func (s *Service) GetPluginViewURL(id string, viewID string) (string, error) {
	if s.plugins == nil {
		return "", ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	url, err := s.plugins.ViewURL(id, viewID)
	if err != nil {
		return "", ierror.NewError(err)
	}
	return url, nil
}

func (s *Service) GetPluginViewDocument(id string, viewID string) (string, error) {
	if s.plugins == nil {
		return "", ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	content, err := s.plugins.ViewDocument(id, viewID)
	if err != nil {
		return "", ierror.NewError(err)
	}
	return content, nil
}

func (s *Service) GetPluginSettings(ctx context.Context, id string) (map[string]interface{}, error) {
	if s.plugins == nil {
		return nil, ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	config, err := s.plugins.GetSettings(ctx, id)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	return config, nil
}

func (s *Service) SavePluginSettings(ctx context.Context, id string, config map[string]interface{}) (map[string]interface{}, error) {
	if s.plugins == nil {
		return nil, ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	saved, err := s.plugins.SaveSettings(ctx, id, config)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	return saved, nil
}

func (s *Service) TestPluginConnection(ctx context.Context, id string, protocol string, config map[string]interface{}) (map[string]interface{}, error) {
	if s.plugins == nil {
		return nil, ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	result, err := s.plugins.TestConnection(ctx, id, protocol, config)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	return result, nil
}

func (s *Service) CallPluginTool(ctx context.Context, aliasID string, args string) (string, error) {
	if s.plugins == nil {
		return "", ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	for _, summary := range s.plugins.List() {
		if !summary.Enabled {
			continue
		}
		caps := pluginsSummaryToCapabilities(summary)
		for _, item := range llmtools.NewPluginTools(summary.ID, summary.Name, caps, s.plugins) {
			pluginTool, ok := item.(*llmtools.PluginTool)
			if !ok || pluginTool.Id() != aliasID {
				continue
			}
			result, err := s.plugins.CallTool(ctx, pluginTool.PluginID, pluginTool.Kind, pluginTool.ToolID, args)
			if err != nil {
				return "", ierror.NewError(err)
			}
			return result, nil
		}
	}
	return "", ierror.NewError(fmt.Errorf("plugin tool %s not found", aliasID))
}

func (s *Service) CallPluginToolDirect(ctx context.Context, pluginID string, kind string, toolID string, args string) (string, error) {
	if s.plugins == nil {
		return "", ierror.NewError(fmt.Errorf("plugin manager is not available"))
	}
	for _, summary := range s.plugins.List() {
		if !summary.Enabled || summary.ID != pluginID {
			continue
		}
		result, err := s.plugins.CallTool(ctx, pluginID, kind, toolID, args)
		if err != nil {
			raw, marshalErr := json.Marshal(map[string]interface{}{
				"ok":       false,
				"error":    err.Error(),
				"plugin":   pluginID,
				"toolId":   toolID,
				"toolKind": kind,
			})
			if marshalErr != nil {
				return "", ierror.NewError(err)
			}
			return string(raw), nil
		}
		return result, nil
	}
	return "", ierror.NewError(fmt.Errorf("plugin %s is not enabled or not found", pluginID))
}

func (s *Service) emitPluginsChanged() {
	if s.app != nil {
		s.app.Event.Emit(EventSettingsPluginsChanged, map[string]string{"type": "changed"})
	}
}
