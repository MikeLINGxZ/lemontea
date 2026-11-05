package window_options

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/window_id"
)

// DefaultHome 返回应用主窗口使用的默认 WebviewWindowOptions
func DefaultHome() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:           window_id.Home,
		Title:          "lemontea",
		EnableFileDrop: true,
		Mac: application.MacWindow{
			// 主窗口由前端标题栏声明拖动区域，避免原生拖动区域拦截双击
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInsetUnified,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Width:            1580,
		Height:           980,
		MinWidth:         550,
		MinHeight:        550,
	}
}

// DefaultOnboarding 返回首次启动引导窗口的 WebviewWindowOptions
func DefaultOnboarding() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:           window_id.Onboarding,
		Title:          "lemontea",
		EnableFileDrop: false,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 48,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/?entry=onboarding",
		Width:            1140,
		Height:           676,
		MinWidth:         720,
		MinHeight:        560,
	}
}

// DefaultQuickChat 返回快捷聊天选择器的浮动窗口配置 WebviewWindowOptions
func DefaultQuickChat() application.WebviewWindowOptions {
	return WithHiddenTitleBar(application.WebviewWindowOptions{
		Name:                window_id.QuickChat,
		Title:               "lemontea",
		MinimiseButtonState: application.ButtonHidden,
		MaximiseButtonState: application.ButtonHidden,
		CloseButtonState:    application.ButtonHidden,
		Hidden:              true,
		HideOnFocusLost:     true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 54,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
		},

		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/?entry=quick_chat",
		Width:            720,
		Height:           54,
		MinWidth:         560,
		MinHeight:        54,
	})
}

// WithHiddenTitleBar 应用当前平台的隐藏标题栏配置
// 接收选项列表，返回Webview窗口选项列表
func WithHiddenTitleBar(options application.WebviewWindowOptions) application.WebviewWindowOptions {
	if runtime.GOOS == "windows" {
		options.Frameless = true
		options.Windows.DisableFramelessWindowDecorations = false
	}
	return options
}

// QuickChatShortcut 返回用于切换快捷聊天弹窗的应用级快捷键
func QuickChatShortcut() string {
	return "CmdOrCtrl+Shift+Space"
}
