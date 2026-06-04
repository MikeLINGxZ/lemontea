package window_options

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/window_id"
)

// DefaultHome returns the default WebviewWindowOptions used by the application home window.
func DefaultHome() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:           window_id.Home,
		Title:          "lemontea",
		EnableFileDrop: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 48,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Width:            1580,
		Height:           980,
		MinWidth:         550,
		MinHeight:        550,
	}
}

// DefaultOnboarding returns the WebviewWindowOptions for the first-launch onboarding window.
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

// DefaultQuickChat returns the floating WebviewWindowOptions for the quick chat chooser.
func DefaultQuickChat() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
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
	}
}

// QuickChatShortcut returns the app-level accelerator used to toggle the quick chat popup.
func QuickChatShortcut() string {
	return "CmdOrCtrl+Shift+Space"
}
