package plugin_dto

// ResetCliDataInput 指定要清理数据目录的 CLI 插件
type ResetCliDataInput struct {
	// 唯一标识
	ID string `json:"id"`
}

// ResetCliDataOutput 是空响应，调用方可重新查询扩展以获取最新状态
type ResetCliDataOutput struct{}
