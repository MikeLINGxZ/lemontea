//go:build !darwin && !windows

package global_hotkey

import (
	"context"
	"errors"
)

// 全局全局快捷键不受支持对应的错误
var errGlobalHotkeyUnsupported = errors.New("global quick chat hotkey is unsupported on this platform")

// StartQuickChat 在缺少安全的全局热键实现的平台上返回不支持错误
// 接收本次调用的上下文、on已按下，返回清理或通知回调及可能发生的错误
func StartQuickChat(ctx context.Context, onPressed func()) (func(), error) {
	return nil, errGlobalHotkeyUnsupported
}

// StartQuickChatWithShortcut 在缺少安全的全局热键实现的平台上返回不支持错误
// 接收本次调用的上下文、on已按下、快捷键，返回清理或通知回调及可能发生的错误
func StartQuickChatWithShortcut(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
	return nil, errGlobalHotkeyUnsupported
}

// ProbeQuickChatShortcut 在缺少安全的全局热键实现的平台上返回不支持错误
// 接收快捷键，返回操作错误
func ProbeQuickChatShortcut(shortcut string) error {
	return errGlobalHotkeyUnsupported
}
