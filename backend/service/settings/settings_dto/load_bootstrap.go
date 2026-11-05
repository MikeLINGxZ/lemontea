package settings_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"

// 封装加载初始化数据所需的请求参数
type LoadBootstrapInput struct {
}

// 封装加载初始化数据返回给调用方的数据
type LoadBootstrapOutput struct {
	// 初始化数据
	Bootstrap view_model.SettingsBootstrap `json:"bootstrap"`
}
