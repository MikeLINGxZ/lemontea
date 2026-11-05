package config

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/config/config_dto"
)

// 应用版本
var applicationVersion string = "v0.3.0-beta"

// 描述配置使用的数据结构
type Config struct {
	// Wails 应用实例
	wailsApp *application.App
}

// ApplicationVersion 返回应用版本
// 接收本次调用的上下文，返回文本
func (c *Config) ApplicationVersion(ctx context.Context) string {
	return applicationVersion
}

// LanguageList 返回语言列表
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (c *Config) LanguageList(ctx context.Context, input config_dto.LanguageListInput) (*config_dto.LanguageListOutput, error) {
	return &config_dto.LanguageListOutput{
		Languages: []view_model.Language{
			{ID: "zh-CN", Name: "简体中文"},
			{ID: "en", Name: "English"},
		},
	}, nil
}

// RegionList 返回地区列表
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (c *Config) RegionList(ctx context.Context, input config_dto.RegionListInput) (*config_dto.RegionListOutput, error) {
	return &config_dto.RegionListOutput{Regions: []view_model.Region{
		{ID: "CN", Name: "中国", Icon: "🇨🇳"},
		{ID: "US", Name: "United States", Icon: "🇺🇸"},
		{ID: "JP", Name: "日本", Icon: "🇯🇵"},
		{ID: "KR", Name: "대한민국", Icon: "🇰🇷"},
		{ID: "GB", Name: "United Kingdom", Icon: "🇬🇧"},
		{ID: "DE", Name: "Deutschland", Icon: "🇩🇪"},
		{ID: "FR", Name: "France", Icon: "🇫🇷"},
		{ID: "IT", Name: "Italia", Icon: "🇮🇹"},
		{ID: "RU", Name: "Россия", Icon: "🇷🇺"},
		{ID: "CA", Name: "Canada", Icon: "🇨🇦"},
		{ID: "AU", Name: "Australia", Icon: "🇦🇺"},
		{ID: "SG", Name: "Singapore", Icon: "🇸🇬"},
		{ID: "IN", Name: "भारत", Icon: "🇮🇳"},
		{ID: "BR", Name: "Brasil", Icon: "🇧🇷"},
		{ID: "ZA", Name: "South Africa", Icon: "🇿🇦"},
		{ID: "AE", Name: "الإمارات العربية المتحدة", Icon: "🇦🇪"},
		{ID: "HK", Name: "中国香港", Icon: "🇭🇰"},
		{ID: "TW", Name: "中国台灣", Icon: "🇨🇳"},
		{ID: "MO", Name: "中国澳門", Icon: "🇲🇴"},
		{ID: "NL", Name: "Nederland", Icon: "🇳🇱"},
		{ID: "SE", Name: "Sverige", Icon: "🇸🇪"},
		{ID: "CH", Name: "Schweiz", Icon: "🇨🇭"},
		{ID: "ES", Name: "España", Icon: "🇪🇸"},
		{ID: "PT", Name: "Portugal", Icon: "🇵🇹"},
		{ID: "TH", Name: "ประเทศไทย", Icon: "🇹🇭"},
		{ID: "VN", Name: "Việt Nam", Icon: "🇻🇳"},
		{ID: "MY", Name: "Malaysia", Icon: "🇲🇾"},
		{ID: "PH", Name: "Pilipinas", Icon: "🇵🇭"},
		{ID: "ID", Name: "Indonesia", Icon: "🇮🇩"},
		{ID: "NZ", Name: "New Zealand", Icon: "🇳🇿"},
		{ID: "AR", Name: "Argentina", Icon: "🇦🇷"},
		{ID: "MX", Name: "México", Icon: "🇲🇽"},
		{ID: "SA", Name: "المملكة العربية السعودية", Icon: "🇸🇦"},
		{ID: "IL", Name: "ישראל", Icon: "🇮🇱"},
		{ID: "TR", Name: "Türkiye", Icon: "🇹🇷"},
		{ID: "NO", Name: "Norge", Icon: "🇳🇴"},
		{ID: "DK", Name: "Danmark", Icon: "🇩🇰"},
		{ID: "FI", Name: "Suomi", Icon: "🇫🇮"},
		{ID: "PL", Name: "Polska", Icon: "🇵🇱"},
		{ID: "AT", Name: "Österreich", Icon: "🇦🇹"},
		{ID: "BE", Name: "België", Icon: "🇧🇪"},
		{ID: "IE", Name: "Ireland", Icon: "🇮🇪"},
		{ID: "EG", Name: "مصر", Icon: "🇪🇬"},
		{ID: "NG", Name: "Nigeria", Icon: "🇳🇬"},
		{ID: "CL", Name: "Chile", Icon: "🇨🇱"},
		{ID: "CO", Name: "Colombia", Icon: "🇨🇴"},
	}}, nil
}

// SupportedProviderList 返回支持的服务提供商列表
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (c *Config) SupportedProviderList(ctx context.Context, input config_dto.SupportedProviderListInput) (*config_dto.SupportedProviderListOutput, error) {
	return &config_dto.SupportedProviderListOutput{
		SupportedProviders: []view_model.SupportedProvider{
			{
				Type:        provider.Deepseek,
				Icon:        "/providers/deepseek_icon.png",
				Name:        i18n.TCurrent("provider.deepseek.name", nil),
				Description: i18n.TCurrent("provider.deepseek.description", nil),
				BaseURL:     "https://api.deepseek.com/v1",
			},
			{
				Type:        provider.Aliyun,
				Icon:        "/providers/qwen_icon.png",
				Name:        i18n.TCurrent("provider.aliyun.name", nil),
				Description: i18n.TCurrent("provider.aliyun.description", nil),
				BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
			},
			{
				Type:        provider.Ollama,
				Icon:        "/providers/ollama_icon.png",
				Name:        i18n.TCurrent("provider.ollama.name", nil),
				Description: i18n.TCurrent("provider.ollama.description", nil),
				BaseURL:     "http://localhost:11434",
			},
			{
				Type:        provider.OpenAiCompatibility,
				Icon:        "/providers/openai_icon.png",
				Name:        i18n.TCurrent("provider.openai_compatibility.name", nil),
				Description: i18n.TCurrent("provider.openai_compatibility.description", nil),
				BaseURL:     "",
			},
		},
	}, nil
}
