package agent_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"

// 封装处理生成标题所需的请求参数
type GenerateTitleInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 服务基础地址
	BaseURL string `json:"base_url"`
	// API 密钥
	ApiKey string `json:"api_key"`
	// 模型名称
	ModelName string `json:"model_name"`
	// 提供商类型
	ProviderType provider.Type `json:"provider_type"`
}

// 封装处理生成标题返回给调用方的数据
type GenerateTitleOutput struct {
	// 标题
	Title string `json:"title"`
}
