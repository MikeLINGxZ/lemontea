//go:build windows

package plugin

// loginFields 是 Windows 平台的空占位结构
// 交互式 CLI 登录依赖 PTY，plugin_login.go 通过 //go:build !windows 将该实现限定在其他平台
// Windows 运行时无需访问 CLI 登录会话字段
type loginFields struct{}

// initLoginFields 在 Windows 平台直接返回
// 接收插件，无返回值
func initLoginFields(p *Plugin) {}
