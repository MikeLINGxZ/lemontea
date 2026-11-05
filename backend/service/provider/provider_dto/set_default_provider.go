package provider_dto

// 封装设置默认所需的请求参数
type SetDefaultInput struct {
	ProviderId int64  `json:"provider_id"` // 供应商ID
	ModelId    *int64 `json:"model_id"`    // 默认模型id
}

// 封装设置默认返回给调用方的数据
type SetDefaultOutput struct {
}
