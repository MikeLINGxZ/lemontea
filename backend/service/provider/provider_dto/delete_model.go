package provider_dto

// 封装删除模型所需的请求参数
type DeleteModelInput struct {
	// 模型编号
	ModelId int64 `json:"model_id"`
}

// 封装删除模型返回给调用方的数据
type DeleteModelOutput struct{}
