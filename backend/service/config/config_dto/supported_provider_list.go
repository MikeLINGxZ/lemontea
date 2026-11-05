package config_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"

// 封装处理受支持提供商列表所需的请求参数
type SupportedProviderListInput struct {
}

// 封装处理受支持提供商列表返回给调用方的数据
type SupportedProviderListOutput struct {
	// 受支持提供商列表
	SupportedProviders []view_model.SupportedProvider `json:"supported_providers"`
}
