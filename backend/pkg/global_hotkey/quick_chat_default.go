//go:build windows

package global_hotkey

import "golang.design/x/hotkey"

// quickChatModifiers returns the Windows control-based quick chat shortcut modifiers.
func quickChatModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}
}

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
