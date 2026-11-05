package provider_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"

// 封装列出提供商列表所需的请求参数
type ListProvidersInput struct {
}

// 封装列出提供商列表返回给调用方的数据
type ListProvidersOutput struct {
	// 提供商列表
	Providers []ProviderWrapper `json:"providers"`
}

// 描述提供商包装结构使用的数据结构
type ProviderWrapper struct {
	// 提供商
	view_model.Provider `json:"providers"`
	// 模型列表
	Models []view_model.Model `json:"models"`
}
