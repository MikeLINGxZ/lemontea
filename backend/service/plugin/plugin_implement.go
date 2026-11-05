package plugin

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ServiceStartup 在服务注册后保存 Wails 应用句柄
// 接收本次调用的上下文、选项列表，返回操作错误
func (p *Plugin) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	_ = ctx
	_ = options
	p.wailsApp = application.Get()
	return nil
}
