package data_models

import "time"

// Terminal 保存一次通过 PTY 执行命令的元数据
type Terminal struct {
	// ORM模型
	OrmModel
	// 终端编号
	TerminalID string `gorm:"type:varchar(80);uniqueIndex" json:"terminal_id"`
	// 会话编号
	SessionID uint `gorm:"index" json:"session_id"`
	// 消息编号
	MessageID *uint `gorm:"index" json:"message_id"`
	// 工具调用编号
	ToolCallID string `gorm:"type:varchar(255);index" json:"tool_call_id"`
	// 标题
	Title string `gorm:"type:varchar(255)" json:"title"`
	// 命令
	Command string `gorm:"type:text" json:"command"`
	// 参数
	Args string `gorm:"type:text" json:"args"`
	// 工作目录
	Cwd string `gorm:"type:text" json:"cwd"`
	// 当前状态
	Status string `gorm:"type:varchar(32);index" json:"status"`
	// 可见
	Visible bool `gorm:"type:bool;default:0;index" json:"visible"`
	// 进程号
	PID int `json:"pid"`
	// 退出代码
	ExitCode *int `json:"exit_code"`
	// 当前游标
	CurrentCursor int64 `json:"current_cursor"`
	// 已启动时间
	StartedAt time.Time `json:"started_at"`
	// 已结束时间
	EndedAt *time.Time `json:"ended_at"`
}

// TerminalOutputChunk 保存终端输出中只追加的一段数据
type TerminalOutputChunk struct {
	// ORM模型
	OrmModel
	// 终端编号
	TerminalID string `gorm:"type:varchar(80);index" json:"terminal_id"`
	// 序号
	Seq int64 `gorm:"index" json:"seq"`
	// 游标开始
	CursorStart int64 `gorm:"index" json:"cursor_start"`
	// 游标结束
	CursorEnd int64 `gorm:"index" json:"cursor_end"`
	// 数据
	Data string `gorm:"type:text" json:"data"`
}
