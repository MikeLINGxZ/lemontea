package view_model

// 描述提供商模型使用的数据结构
type ProviderModel struct {
	// 提供商
	Provider Provider `json:"provider"`
	// 模型列表
	Models []Model `json:"models"`
}
