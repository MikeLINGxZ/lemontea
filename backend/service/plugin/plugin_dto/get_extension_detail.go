package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// 封装获取扩展详情所需的请求参数
type GetExtensionDetailInput struct {
	// 唯一标识
	ID string `json:"id"`
}

// 封装获取扩展详情返回给调用方的数据
type GetExtensionDetailOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
	// 配置文本
	ConfigText string `json:"config_text"`
}
