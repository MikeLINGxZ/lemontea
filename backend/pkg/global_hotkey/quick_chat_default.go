//go:build windows

package global_hotkey

import "golang.design/x/hotkey"

// quickChatModifiers 返回 Windows 上以 Ctrl 为基础的快捷聊天修饰键
func quickChatModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}
}

// 处理快捷聊天修饰键对应令牌，接收令牌，返回修饰键、是否满足条件
func quickChatModifierForToken(token string) (hotkey.Modifier, bool) {
	switch token {
	case "CmdOrCtrl", "Ctrl":
		return hotkey.ModCtrl, true
	case "Cmd":
		return hotkey.ModWin, true
	case "Alt":
		return hotkey.ModAlt, true
	case "Shift":
		return hotkey.ModShift, true
	default:
		return 0, false
	}
}
