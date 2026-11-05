//go:build windows

package global_hotkey

import "golang.design/x/hotkey"

// 处理快捷聊天附加数据键对应令牌，接收令牌，返回键、是否满足条件
func quickChatExtraKeyForToken(token string) (hotkey.Key, bool) {
	// 键集合
	keys := map[string]hotkey.Key{
		"Backquote":    hotkey.Key(0xC0),
		"Minus":        hotkey.Key(0xBD),
		"Equal":        hotkey.Key(0xBB),
		"BracketLeft":  hotkey.Key(0xDB),
		"BracketRight": hotkey.Key(0xDD),
		"Backslash":    hotkey.Key(0xDC),
		"Semicolon":    hotkey.Key(0xBA),
		"Quote":        hotkey.Key(0xDE),
		"Comma":        hotkey.Key(0xBC),
		"Period":       hotkey.Key(0xBE),
		"Slash":        hotkey.Key(0xBF),
		"Home":         hotkey.Key(0x24),
		"End":          hotkey.Key(0x23),
		"PageUp":       hotkey.Key(0x21),
		"PageDown":     hotkey.Key(0x22),
		"Insert":       hotkey.Key(0x2D),
	}
	key, ok := keys[token] // 键、是否成功取得对应值
	return key, ok
}
