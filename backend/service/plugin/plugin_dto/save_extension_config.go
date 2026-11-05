package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// 封装保存扩展配置所需的请求参数
type SaveExtensionConfigInput struct {
	// 唯一标识
	ID string `json:"id"`
	// 配置文本
	ConfigText string `json:"config_text"`
}

// 封装保存扩展配置返回给调用方的数据
type SaveExtensionConfigOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
}
