package provider_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"

// 封装处理请求提供商模型列表所需的请求参数
type RequestProviderModelListInput struct {
	BaseUrl string `json:"base_url"` // 供应商基础URL
	ApiKey  string `json:"api_key"`  // 供应商API密钥
}

// 封装处理请求提供商模型列表返回给调用方的数据
type RequestProviderModelListOutput struct {
	// 模型列表
	Models []view_model.Model `json:"models"`
}
