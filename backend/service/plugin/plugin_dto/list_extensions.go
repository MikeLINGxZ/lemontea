package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// 封装列出扩展列表所需的请求参数
type ListExtensionsInput struct{}

// 封装列出扩展列表返回给调用方的数据
type ListExtensionsOutput struct {
	// 扩展列表
	Extensions []data_models.ExtensionItem `json:"extensions"`
}
