package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// GenerateCliManifestInput 指定要重新生成清单的 CLI 插件
type GenerateCliManifestInput struct {
	// 唯一标识
	ID string `json:"id"`
}

// GenerateCliManifestOutput 返回清单生成成功后的扩展条目
type GenerateCliManifestOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
}
