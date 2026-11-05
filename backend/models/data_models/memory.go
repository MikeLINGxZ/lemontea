package data_models

import "time"

// Memory 保存可注入上下文或检索的简要长期记忆
type Memory struct {
	// ORM模型
	OrmModel
	// 摘要
	Summary string `gorm:"type:varchar(255);index" json:"summary"`
	// 内容
	Content string `gorm:"type:text" json:"content"`
	// 类型
	Type string `gorm:"type:varchar(32);index" json:"type"`
	// 目标
	Target string `gorm:"type:varchar(32);index" json:"target"`
	// 来源
	Source string `gorm:"type:varchar(32);index" json:"source"`
	// 字符数量
	CharCount int `json:"char_count"`
	// 向量编号
	EmbeddingID *uint `gorm:"index" json:"embedding_id"`
	// 是否已遗忘
	IsForgotten bool `gorm:"index" json:"is_forgotten"`
	// 回忆数量
	RecallCount int `json:"recall_count"`
	// 最后一个已检索时间
	LastRecalledAt *time.Time `json:"last_recalled_at"`
	// 最后一个已使用时间
	LastUsedAt *time.Time `json:"last_used_at"`
	// 重要度
	Importance int `gorm:"index" json:"importance"`
	// 置信度
	Confidence int `gorm:"index" json:"confidence"`
	// 已置顶
	Pinned bool `gorm:"index" json:"pinned"`
}

// MemoryEmbedding 保存记忆可选的向量数据
type MemoryEmbedding struct {
	// ORM模型
	OrmModel
	// 记忆编号
	MemoryID uint `gorm:"index" json:"memory_id"`
	// 向量
	Vector string `gorm:"type:text" json:"vector"`
	// 模型名称
	ModelName string `gorm:"type:varchar(255)" json:"model_name"`
	// 维数
	Dimensions int `json:"dimensions"`
	// 模式版本
	SchemaVersion int `json:"schema_version"`
}
