//go:build darwin

package global_hotkey

import "golang.design/x/hotkey"

// quickChatModifiers returns the macOS command-based quick chat shortcut modifiers.
func quickChatModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}
}

func quickChatModifierForToken(token string) (hotkey.Modifier, bool) {
	switch token {
	case "CmdOrCtrl", "Cmd":
		return hotkey.ModCmd, true
	case "Ctrl":
		return hotkey.ModCtrl, true
	case "Alt":
		return hotkey.ModOption, true
	case "Shift":
		return hotkey.ModShift, true
	default:
		return 0, false
	}
}
