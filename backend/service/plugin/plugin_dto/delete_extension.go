package plugin_dto

// 封装删除扩展所需的请求参数
type DeleteExtensionInput struct {
	// 唯一标识
	ID string `json:"id"`
}

// 封装删除扩展返回给调用方的数据
type DeleteExtensionOutput struct{}
