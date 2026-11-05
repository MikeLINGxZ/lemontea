package process

import "github.com/wailsapp/wails/v3/pkg/application"

// Process 提供应用进程相关的核心操作
type Process struct {
	// Wails 应用实例
	wailsApp *application.App
}
