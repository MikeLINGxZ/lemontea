package service

import (
	"context"
	"os"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	llmtools "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/tools"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils"
)

// GetSupportProviders 获取支持的供应商列表
func (s *Service) GetSupportProviders() ([]view_models.SupportProvider, error) {
	return []view_models.SupportProvider{
		{
			ProviderType:      data_models.ProviderTypeDeepseek,
			Icon:              "/providers/deepseek_icon.png",
			Name:              i18n.TCurrent("provider.deepseek.name", nil),
			BaseUrl:           "https://api.deepseek.com/v1",
			FileUploadBaseUrl: nil,
			Description:       i18n.TCurrent("provider.deepseek.description", nil),
		}, {
			ProviderType:      data_models.ProviderTypeAliyuns,
			Icon:              "/providers/qwen_icon.png",
			Name:              i18n.TCurrent("provider.aliyuns.name", nil),
			BaseUrl:           "https://dashscope.aliyuncs.com/compatible-mode/v1",
			FileUploadBaseUrl: utils.StringPointer("https://dashscope.aliyuncs.com/api/v1/uploads"),
			Description:       i18n.TCurrent("provider.aliyuns.description", nil),
		}, {
			ProviderType:      data_models.ProviderTypeOpenrouter,
			Icon:              "/providers/openrouter_icon.png",
			Name:              i18n.TCurrent("provider.openrouter.name", nil),
			BaseUrl:           "https://openrouter.ai/api/v1",
			FileUploadBaseUrl: nil,
			Description:       i18n.TCurrent("provider.openrouter.description", nil),
		}, {
			ProviderType:      data_models.ProviderTypeOllama,
			Icon:              "/providers/ollama_icon.png",
			Name:              i18n.TCurrent("provider.ollama.name", nil),
			BaseUrl:           "http://localhost:11434",
			FileUploadBaseUrl: nil,
			Description:       i18n.TCurrent("provider.ollama.description", nil),
		}, {
			ProviderType:      data_models.ProviderTypeOther,
			Icon:              "/providers/openai_icon.png",
			Name:              i18n.TCurrent("provider.other.name", nil),
			BaseUrl:           "",
			FileUploadBaseUrl: utils.StringPointer(""),
			Description:       i18n.TCurrent("provider.other.description", nil),
		},
	}, nil
}

func (s *Service) GetTools() []view_models.Tool {
	var res []view_models.Tool
	toolsInfo := llmtools.ToolRouter.GetBuiltinToolsInfo()
	for _, item := range toolsInfo {
		res = append(res, view_models.Tool{
			Id:          item.Id(),
			Name:        item.Name(),
			Description: item.Description(),
			SourceType:  toolSourceBuiltin,
			Enabled:     true,
			IsDeletable: false,
		})
	}

	customServers, err := s.storage.ListCustomMCPServers(context.Background())
	if err != nil {
		return res
	}
	for _, server := range customServers {
		if _, statErr := os.Stat(server.ConfigPath); statErr != nil {
			continue
		}
		res = append(res, s.customServerToViewTool(server))
	}
	if s.plugins != nil {
		for _, plugin := range s.plugins.List() {
			if !plugin.Enabled {
				continue
			}
			res = append(res, view_models.Tool{
				Id:          llmtools.PluginAggregateID(plugin.ID),
				Name:        plugin.Name,
				Description: plugin.Description,
				SourceType:  toolSourcePlugin,
				Enabled:     true,
				IsDeletable: false,
				PluginType:  plugin.Type,
				UseTools:    plugin.UseTools,
				ViewTools:   plugin.ViewTools,
				Agents:      plugin.Agents,
			})
		}
	}

	return res
}
