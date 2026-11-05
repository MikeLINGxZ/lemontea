package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// 封装处理重新加载扩展所需的请求参数
type ReloadExtensionInput struct {
	// 唯一标识
	ID string `json:"id"`
}

// 封装处理重新加载扩展返回给调用方的数据
type ReloadExtensionOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
}
