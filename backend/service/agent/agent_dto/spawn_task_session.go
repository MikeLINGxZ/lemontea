package agent_dto

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"

// SpawnTaskSessionInput 描述自动化任务会话的初始提示词
type SpawnTaskSessionInput struct {
	// 标题
	Title string `json:"title"`
	// 系统提示词
	SystemPrompt string `json:"system_prompt"`
	// 用户消息
	UserMessage string `json:"user_message"`
	// 技能名称
	SkillName string `json:"skill_name"`
	// 服务基础地址
	BaseURL string `json:"base_url"`
	// API 密钥
	ApiKey string `json:"api_key"`
	// 模型名称
	ModelName string `json:"model_name"`
	// 提供商类型
	ProviderType provider.Type `json:"provider_type"`
}

// SpawnTaskSessionOutput 返回新建任务会话的标识
type SpawnTaskSessionOutput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 标题
	Title string `json:"title"`
}
