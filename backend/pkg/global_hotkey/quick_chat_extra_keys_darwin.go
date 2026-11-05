//go:build darwin

package global_hotkey

import "golang.design/x/hotkey"

// 处理快捷聊天附加数据键对应令牌，接收令牌，返回键、是否满足条件
func quickChatExtraKeyForToken(token string) (hotkey.Key, bool) {
	// 键集合
	keys := map[string]hotkey.Key{
		"Backquote":    hotkey.Key(50),
		"Minus":        hotkey.Key(27),
		"Equal":        hotkey.Key(24),
		"BracketLeft":  hotkey.Key(33),
		"BracketRight": hotkey.Key(30),
		"Backslash":    hotkey.Key(42),
		"Semicolon":    hotkey.Key(41),
		"Quote":        hotkey.Key(39),
		"Comma":        hotkey.Key(43),
		"Period":       hotkey.Key(47),
		"Slash":        hotkey.Key(44),
		"Home":         hotkey.Key(115),
		"End":          hotkey.Key(119),
		"PageUp":       hotkey.Key(116),
		"PageDown":     hotkey.Key(121),
		"Insert":       hotkey.Key(114),
	}
	key, ok := keys[token] // 键、是否成功取得对应值
	return key, ok
}
