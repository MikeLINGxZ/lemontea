package plugin_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// InstallCliFromNpmInput 包含从已发布的 npm 包安装 CLI 插件所需的参数
type InstallCliFromNpmInput struct {
	// npm包名
	NpmPackage string `json:"npm_package"`
	// 名称
	Name string `json:"name"`
}

// InstallCliFromNpmOutput 返回安装成功后保存的 ExtensionItem
type InstallCliFromNpmOutput struct {
	// 扩展
	Extension data_models.ExtensionItem `json:"extension"`
}
