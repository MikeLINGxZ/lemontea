package provider_dto

// 封装删除提供商所需的请求参数
type DeleteProviderInput struct {
	// 提供商编号
	ProviderId int64 `json:"provider_id"`
}

// 封装删除提供商返回给调用方的数据
type DeleteProviderOutput struct {
}
