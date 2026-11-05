package memory_dto

// 封装更新记忆所需的请求参数
type UpdateMemoryInput struct {
	// 唯一标识
	ID uint `json:"id"`
	// 摘要
	Summary string `json:"summary"`
	// 内容
	Content string `json:"content"`
	// 类型
	Type string `json:"type"`
	// 目标
	Target string `json:"target"`
	// 来源
	Source string `json:"source"`
	// 重要度
	Importance int `json:"importance"`
	// 置信度
	Confidence int `json:"confidence"`
	// 已置顶
	Pinned bool `json:"pinned"`
}

// 封装更新记忆返回给调用方的数据
type UpdateMemoryOutput struct {
	// 记忆
	Memory MemoryItem `json:"memory"`
}
