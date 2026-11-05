package notification

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ServiceStartup 保存 Wails 应用句柄，供通知服务发送全局事件
// 接收本次调用的上下文、选项列表，返回操作错误
func (n *Notification) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	n.wailsApp = application.Get()
	return nil
}
