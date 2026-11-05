package provider_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"

// 封装处理提供商并模型列表所需的请求参数
type ProviderAndModelListInput struct {
}

// 封装处理提供商并模型列表返回给调用方的数据
type ProviderAndModelListOutput struct {
	// 提供商模型列表
	ProviderModels []view_model.ProviderModel `json:"provider_models"`
}
