package view_model

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"

// Provider 是返回给前端的供应商列表项和详情数据
type Provider struct {
	// 唯一标识
	ID uint `json:"id"`
	// 图标
	Icon string `json:"icon"`
	// 类型
	Type provider.Type `json:"provider_type"`
	// 名称
	Name string `json:"provider_name"`
	// 服务基础地址
	BaseURL string `json:"base_url"`
	// 启用状态
	Enabled bool `json:"enabled"`
	// API 密钥
	ApiKey    string `json:"api_key"`
	IsDefault bool   `json:"is_default"` // 是否是默认供应商
}
