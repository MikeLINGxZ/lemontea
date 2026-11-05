//go:build !darwin

package window

import "context"

// SetNativeTheme 保留跨平台绑定入口，接收调用上下文和主题偏好，无返回值
func (p *Window) SetNativeTheme(_ context.Context, _ string) {}
