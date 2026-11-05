package view_model

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"

// 描述语言使用的数据结构
type Language struct {
	// 唯一标识
	ID string `json:"id"`
	// 名称
	Name string `json:"name"`
}

// 描述区域使用的数据结构
type Region struct {
	// 图标
	Icon string `json:"icon"`
	// 唯一标识
	ID string `json:"id"`
	// 名称
	Name string `json:"name"`
}

// 描述受支持提供商使用的数据结构
type SupportedProvider struct {
	// 类型
	Type provider.Type `json:"type"`
	// 图标
	Icon string `json:"icon"`
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 服务基础地址
	BaseURL string `json:"base_url"`
}
