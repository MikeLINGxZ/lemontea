package agent_dto

// 封装按分页参数加载指定会话的历史消息所需的请求参数
type LoadSessionMessagesInput struct {
	// 会话编号
	SessionID uint `json:"session_id"`
	// 偏移
	Offset int `json:"offset"`
	// 上限
	Limit int `json:"limit"`
}

// 描述消息条目使用的数据结构
type MessageItem struct {
	// 唯一标识
	ID uint `json:"id"`
	// 会话编号
	SessionID uint `json:"session_id"`
	// 父级编号
	ParentID *uint `json:"parent_id"`
	// 角色
	Role string `json:"role"`
	// 内容类型
	ContentType string `json:"content_type"`
	// 内容
	Content string `json:"content"`
	// 模型名称
	ModelName string `json:"model_name"`
	// 智能体名称
	AgentName string `json:"agent_name"`
	// Token输入
	TokensIn int `json:"tokens_in"`
	// Token输出
	TokensOut int `json:"tokens_out"`
	// 附加数据
	Extra string `json:"extra"`
	// 附件列表
	Attachments string `json:"attachments"`
	// 创建
	Created string `json:"created"`
}

// 封装按分页参数加载指定会话的历史消息返回给调用方的数据
type LoadSessionMessagesOutput struct {
	// 消息列表
	Messages []MessageItem `json:"messages"`
	// 总量
	Total int64 `json:"total"`
	// 是否有更多
	HasMore bool `json:"has_more"`
}
