package provider_dto

import (
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
)

// 封装创建模型提供商所需的请求参数
type CreateProviderInput struct {
	ProviderName string             `json:"provider_name"` // 供应商名称
	ProviderType provider.Type      `json:"provider_type"` // 供应商类型
	BaseUrl      string             `json:"base_url"`      // 供应商基础URL
	ApiKey       string             `json:"api_key"`       // 供应商API密钥
	Enable       bool               `json:"enable"`        // 是否启用
	DefaultModel *string            `json:"default_model"` // 默认模型
	Models       []view_model.Model `json:"models"`        // 供应商模型列表
}

// 封装创建模型提供商返回给调用方的数据
type CreateProviderOutput struct {
}
