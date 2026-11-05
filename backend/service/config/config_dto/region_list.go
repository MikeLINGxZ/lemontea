package config_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"

// 封装处理区域列表所需的请求参数
type RegionListInput struct {
}

// 封装处理区域列表返回给调用方的数据
type RegionListOutput struct {
	// 地区列表
	Regions []view_model.Region `json:"regions"`
}
