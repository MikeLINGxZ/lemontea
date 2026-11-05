package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// 封装切换扩展所需的请求参数
type ToggleExtensionInput struct {
	// 唯一标识
	ID string `json:"id"`
	// 启用状态
	Enabled bool `json:"enabled"`
}

// 封装切换扩展返回给调用方的数据
type ToggleExtensionOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
}
