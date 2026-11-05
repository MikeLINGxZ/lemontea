package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// UpdateCliManifestInput 携带指定 CLI 插件编辑后的清单 JSON 文本
type UpdateCliManifestInput struct {
	// 唯一标识
	ID string `json:"id"`
	// 清单文本
	ManifestText string `json:"manifest_text"`
}

// UpdateCliManifestOutput 返回清单校验并保存后的 ExtensionItem
type UpdateCliManifestOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
}
