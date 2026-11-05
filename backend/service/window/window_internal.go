package window

import (
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/window_id"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// 按主窗口状态生成子窗口所用的选项，接收选项列表，返回Webview窗口选项列表
func (p *Window) childWindowOptions(options application.WebviewWindowOptions) application.WebviewWindowOptions {
	options.InitialPosition = application.WindowCentered
	options.Screen = p.homeWindowScreen()
	if options.Screen == nil && p.wailsApp != nil {
		options.Screen = p.wailsApp.Screen.GetPrimary()
	}
	return options
}

// 在主窗口所在屏幕居中显示指定窗口，接收窗口，无返回值
func (p *Window) showCenteredOnHomeScreen(window application.Window) {
	if window == nil {
		return
	}
	p.centerWindowOnHomeScreen(window)
	window.Show()
	p.centerWindowOnHomeScreen(window)
	window.Focus()
}

// centerWindowOnHomeScreen 优先将指定窗口放置到主窗口所在屏幕
// 接收窗口，无返回值
func (p *Window) centerWindowOnHomeScreen(window application.Window) {
	if window == nil {
		return
	}

	screen := p.homeWindowScreen() // 屏幕
	if screen == nil {
		currentScreen, err := window.GetScreen() // 当前屏幕、获取屏幕时的错误
		if err == nil {
			screen = currentScreen
		}
	}
	if screen == nil && p.wailsApp != nil {
		screen = p.wailsApp.Screen.GetPrimary()
	}
	if screen == nil {
		window.Center()
		return
	}

	window.SetScreen(screen)
}

// homeWindowScreen 返回主窗口所在的屏幕，无法确定时返回 nil
func (p *Window) homeWindowScreen() *application.Screen {
	if p.wailsApp == nil {
		return nil
	}
	homeWindow, ok := p.wailsApp.Window.GetByName(window_id.Home) // 主界面窗口、是否成功取得对应值
	if !ok {
		return nil
	}
	screen, err := homeWindow.GetScreen() // 屏幕、获取屏幕时的错误
	if err != nil {
		return nil
	}
	return screen
}
