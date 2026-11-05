package agent_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"

// 封装启动指定会话的模型请求并将处理结果返回给前端所需的请求参数
type SendMessageInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 内容
	Content string `json:"content"`
	// 系统提示词
	SystemPrompt string `json:"system_prompt,omitempty"`
	// 技能名称
	SkillName string `json:"skill_name,omitempty"`
	// 服务基础地址
	BaseURL string `json:"base_url"`
	// API 密钥
	ApiKey string `json:"api_key"`
	// 模型名称
	ModelName string `json:"model_name"`
	// 提供商类型
	ProviderType provider.Type `json:"provider_type"`
	// 已启用的用户工具列表
	EnabledUserTools []string `json:"enabled_user_tools"`
	// 附件列表
	Attachments []AttachmentInput `json:"attachments,omitempty"`
}

// 封装启动指定会话的模型请求并将处理结果返回给前端返回给调用方的数据
type SendMessageOutput struct {
}
