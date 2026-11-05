package config_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"

// 封装处理语言列表所需的请求参数
type LanguageListInput struct {
}

// 封装处理语言列表返回给调用方的数据
type LanguageListOutput struct {
	// 语言列表
	Languages []view_model.Language `json:"languages"`
}
