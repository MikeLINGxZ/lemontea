//go:build !windows

package plugin

import (
	"context"
)

// loginFields 保存 CLI 登录会话管理所需的平台专用字段
// 该结构嵌入 Plugin，仅在非 Windows 平台生效
type loginFields struct {
	// loginSessions 按扩展标识记录当前步骤的会话
	// 多步登录流程每进入下一步就会更新对应条目
	loginSessions map[string]loginSessionIface
	// loginCancelled 记录用户取消登录流程的请求，用于跳过剩余步骤
	// 当前会话也会收到 Cancel 调用
	loginCancelled map[string]struct{}
	// startCliLoginCommand 根据 argv 参数向量启动单个登录步骤
	// 实际运行时调用 cliManager.StartLoginCommand，测试时可替换为模拟实现
	startCliLoginCommand func(ctx context.Context, name string, argv []string) (loginSessionIface, error)
}

// initLoginFields 初始化新建 Plugin 的登录会话管理字段
// 接收插件，无返回值
func initLoginFields(p *Plugin) {
	p.loginSessions = map[string]loginSessionIface{}
	p.loginCancelled = map[string]struct{}{}
	p.startCliLoginCommand = func(ctx context.Context, name string, argv []string) (loginSessionIface, error) {
		// 实现开始CLI登录命令回调，接收本次调用的上下文、当前名称、命令参数，返回登录会话Iface及可能发生的错误
		mgr, err := p.resolveCliManager() // 管理器、解析CLI管理器时的错误
		if err != nil {
			return nil, err
		}
		return mgr.StartLoginCommand(ctx, name, argv)
	}
}
