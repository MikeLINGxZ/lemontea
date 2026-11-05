package config

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// 在 Wails 服务启动时保存运行环境并初始化当前服务，接收本次调用的上下文、选项列表，返回操作错误
func (c *Config) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	c.wailsApp = application.Get()
	return nil
}
