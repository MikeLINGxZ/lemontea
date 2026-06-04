//go:build windows

package global_hotkey

import "golang.design/x/hotkey"

func quickChatModifiersMatch(mods []hotkey.Modifier) bool {
	return len(mods) == 2 && mods[0] == hotkey.ModCtrl && mods[1] == hotkey.ModShift
}
