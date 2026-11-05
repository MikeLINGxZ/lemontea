package onboarding_dto

import (
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
)

// SaveProviderAndMarkInitializedInput 复用 provider_dto.CreateProviderInput 的字段，用于首次启动引导
type SaveProviderAndMarkInitializedInput struct {
	// 提供商名称
	ProviderName string `json:"provider_name"`
	// 提供商类型
	ProviderType provider.Type `json:"provider_type"`
	// 服务基础地址
	BaseUrl string `json:"base_url"`
	// API 密钥
	ApiKey string `json:"api_key"`
	// 启用
	Enable bool `json:"enable"`
	// 默认模型
	DefaultModel *string `json:"default_model"`
	// 模型列表
	Models []view_model.Model `json:"models"`
}

// SaveProviderAndMarkInitializedOutput 是初始化成功后的空响应
type SaveProviderAndMarkInitializedOutput struct{}
