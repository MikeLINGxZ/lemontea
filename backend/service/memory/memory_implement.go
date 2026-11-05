package memory

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ServiceStartup 由 Wails 在记忆服务启动时调用
// 接收本次调用的上下文、选项列表，返回操作错误
func (m *Memory) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}
