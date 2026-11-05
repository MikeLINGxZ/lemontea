package window

import (
	"context"

	"github.com/ebitengine/purego/objc"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SetNativeTheme 将调用窗口的原生标题栏与应用主题同步
// 接收调用上下文和主题偏好，自动模式恢复系统外观，无返回值
func (p *Window) SetNativeTheme(ctx context.Context, theme string) {
	window, ok := ctx.Value(application.WindowKey).(application.Window) // 发起调用的窗口及是否存在
	if !ok {
		return
	}

	// 原生外观名称为空时清除窗口的外观覆盖
	var name application.MacAppearanceType
	switch theme {
	case "light":
		name = application.NSAppearanceNameAqua
	case "dark":
		name = application.NSAppearanceNameDarkAqua
	case "auto":
	default:
		return
	}

	// 在 AppKit 主线程设置外观，避免操作已关闭的窗口
	application.InvokeSync(func() {
		handle := window.NativeWindow() // 窗口的原生句柄
		if handle == nil {
			return
		}
		var appearance objc.ID // 原生外观对象，零值表示跟随系统
		if name != "" {
			value := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), string(name)) // 原生外观名称
			appearance = objc.ID(objc.GetClass("NSAppearance")).Send(objc.RegisterName("appearanceNamed:"), value)
		}
		objc.ID(uintptr(handle)).Send(objc.RegisterName("setAppearance:"), appearance)
	})
}
