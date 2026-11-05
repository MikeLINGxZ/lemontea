package plugin_dto

// LoginCliInput 指定要启动登录会话的 CLI 插件
type LoginCliInput struct {
	// 唯一标识
	ID string `json:"id"`
}

// LoginCliOutput 是空响应，登录输出和结束状态分别通过 cli.login.output 和 cli.login.done 事件返回
type LoginCliOutput struct{}

// SendLoginStdinInput 携带写入运行中登录会话标准输入的原始字节
type SendLoginStdinInput struct {
	// 唯一标识
	ID string `json:"id"`
	// 数据
	Data string `json:"data"`
}

// CancelLoginCliInput 指定要终止的登录会话
type CancelLoginCliInput struct {
	// 唯一标识
	ID string `json:"id"`
}

// ResizeLoginCliInput 携带运行中登录会话的 PTY 新尺寸
type ResizeLoginCliInput struct {
	// 唯一标识
	ID string `json:"id"`
	// 行数
	Rows uint16 `json:"rows"`
	// 列数
	Cols uint16 `json:"cols"`
}
