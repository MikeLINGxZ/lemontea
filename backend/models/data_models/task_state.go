package data_models

// TaskState 保存自动任务会话可用于恢复的键值状态
type TaskState struct {
	// ORM模型
	OrmModel
	// 会话编号
	SessionID uint `gorm:"index:idx_task_state_session_key,unique" json:"session_id"`
	// 键
	Key string `gorm:"type:varchar(120);index:idx_task_state_session_key,unique" json:"key"`
	// 值
	Value string `gorm:"type:text" json:"value"`
}
