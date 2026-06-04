//go:build !darwin && !windows

package global_hotkey

import (
	"context"
	"errors"
)

var errGlobalHotkeyUnsupported = errors.New("global quick chat hotkey is unsupported on this platform")

// StartQuickChat returns an unsupported error on platforms without a safe global hotkey backend.
func StartQuickChat(ctx context.Context, onPressed func()) (func(), error) {
	return nil, errGlobalHotkeyUnsupported
}

// StartQuickChatWithShortcut returns an unsupported error on platforms without a safe global hotkey backend.
func StartQuickChatWithShortcut(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
	return nil, errGlobalHotkeyUnsupported
}

// ProbeQuickChatShortcut returns an unsupported error on platforms without a safe global hotkey backend.
func ProbeQuickChatShortcut(shortcut string) error {
	return errGlobalHotkeyUnsupported
}
