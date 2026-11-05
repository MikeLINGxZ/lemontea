package terminal

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"
)

// ServiceStartup 将终端管理器事件转发为 Wails 运行时事件
// 接收本次调用的上下文、选项列表，返回操作错误
func (t *Terminal) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	_ = ctx
	_ = options
	t.wailsApp = application.Get()
	t.manager.SetOutputEmitter( /* 设置输出事件发布器时调用的回调，接收事件，无返回值 */ func(event pkgterminal.OutputEvent) {
		if t.wailsApp != nil {
			t.wailsApp.Event.Emit(eventTerminalOutput, event)
		}
	})
	t.manager.SetStatusEmitter( /* 设置状态事件发布器时调用的回调，接收事件，无返回值 */ func(event pkgterminal.StatusEvent) {
		if t.wailsApp != nil {
			t.wailsApp.Event.Emit(eventTerminalExited, event)
		}
	})
	return nil
}
