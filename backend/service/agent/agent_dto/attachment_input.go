package agent_dto

// AttachmentInput 包含前端发送消息时携带的附件元数据
// Name 和 Mime 可留空，后端根据 Path 推断缺失值后写入存储和 model.Message
type AttachmentInput struct {
	// 路径
	Path string `json:"path"`
	// 名称
	Name string `json:"name,omitempty"`
	// MIME类型
	Mime string `json:"mime,omitempty"`
}
