package memory_dto

// MemoryItem 表示返回给前端的长期记忆
type MemoryItem struct {
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
	// 字符数量
	CharCount int `json:"char_count"`
	// 向量编号
	EmbeddingID *uint `json:"embedding_id"`
	// 是否已遗忘
	IsForgotten bool `json:"is_forgotten"`
	// 回忆数量
	RecallCount int `json:"recall_count"`
	// 最后一个已检索时间
	LastRecalledAt *string `json:"last_recalled_at"`
	// 最后一个已使用时间
	LastUsedAt *string `json:"last_used_at"`
	// 重要度
	Importance int `json:"importance"`
	// 置信度
	Confidence int `json:"confidence"`
	// 已置顶
	Pinned bool `json:"pinned"`
	// 创建
	Created string `json:"created"`
	// 更新
	Updated string `json:"updated"`
}

// MemoryStatsItem 包含长期记忆的汇总计数
type MemoryStatsItem struct {
	// 总量
	Total int64 `json:"total"`
	// 当前
	Active int64 `json:"active"`
	// 已遗忘
	Forgotten int64 `json:"forgotten"`
	// 按目标
	ByTarget map[string]int64 `json:"by_target"`
	// 按类型
	ByType map[string]int64 `json:"by_type"`
}
