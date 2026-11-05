package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// 封装处理导入扩展所需的请求参数
type ImportExtensionInput struct {
	// 类别
	Kind string `json:"kind"`
	// 路径
	Path string `json:"path"`
}

// 封装处理导入扩展返回给调用方的数据
type ImportExtensionOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
}
