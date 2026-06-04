//go:build darwin || windows

package global_hotkey

import (
	"context"
	"fmt"
	"sync"

	"golang.design/x/hotkey"
)

type hotkeyHandle interface {
	Register() error
	Unregister() error
	Keydown() <-chan hotkey.Event
}

type hotkeyFactory func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle

// StartQuickChat registers the OS-level global shortcut for toggling quick chat.
func StartQuickChat(ctx context.Context, onPressed func()) (func(), error) {
	return StartQuickChatWithShortcut(ctx, onPressed, DefaultQuickChatShortcut())
}

// StartQuickChatWithShortcut registers a custom OS-level quick chat shortcut.
func StartQuickChatWithShortcut(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
	return startQuickChatWithShortcut(ctx, onPressed, shortcut, func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle {
		return hotkey.New(mods, key)
	})
}

// ProbeQuickChatShortcut temporarily registers a shortcut to detect conflicts.
func ProbeQuickChatShortcut(shortcut string) error {
	spec, err := hotkeySpecFromShortcut(shortcut)
	if err != nil {
		return err
	}
	hk := hotkey.New(spec.modifiers, spec.key)
	if err := hk.Register(); err != nil {
		return err
	}
	return hk.Unregister()
}

// startQuickChat wires a hotkey factory for testable default global shortcut registration.
func startQuickChat(ctx context.Context, onPressed func(), factory hotkeyFactory) (func(), error) {
	return startQuickChatWithShortcut(ctx, onPressed, DefaultQuickChatShortcut(), factory)
}

// startQuickChatWithShortcut wires a hotkey factory for testable global shortcut registration.
func startQuickChatWithShortcut(ctx context.Context, onPressed func(), shortcut string, factory hotkeyFactory) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}

	spec, err := hotkeySpecFromShortcut(shortcut)
	if err != nil {
		return nil, err
	}
	hk := factory(spec.modifiers, spec.key)
	if err := hk.Register(); err != nil {
		return nil, err
	}

	listenCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-listenCtx.Done():
				return
			case _, ok := <-hk.Keydown():
				if !ok {
					return
				}
				if onPressed != nil {
					onPressed()
				}
			}
		}
	}()

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			cancel()
			_ = hk.Unregister()
			<-done
		})
	}

	return stop, nil
}

type quickChatHotkeySpec struct {
	modifiers []hotkey.Modifier
	key       hotkey.Key
}

func hotkeySpecFromShortcut(shortcut string) (quickChatHotkeySpec, error) {
	parsed, err := ParseQuickChatShortcut(shortcut)
	if err != nil {
		return quickChatHotkeySpec{}, err
	}

	modifiers := make([]hotkey.Modifier, 0, len(parsed.Modifiers))
	for _, modifierToken := range parsed.Modifiers {
		modifier, ok := quickChatModifierForToken(modifierToken)
		if !ok {
			return quickChatHotkeySpec{}, fmt.Errorf("unsupported shortcut modifier %q", modifierToken)
		}
		modifiers = append(modifiers, modifier)
	}

	key, ok := quickChatKeyForToken(parsed.Key)
	if !ok {
		return quickChatHotkeySpec{}, fmt.Errorf("unsupported shortcut key %q", parsed.Key)
	}
	return quickChatHotkeySpec{modifiers: modifiers, key: key}, nil
}

func quickChatKeyForToken(token string) (hotkey.Key, bool) {
	keys := map[string]hotkey.Key{
		"Space":  hotkey.KeySpace,
		"0":      hotkey.Key0,
		"1":      hotkey.Key1,
		"2":      hotkey.Key2,
		"3":      hotkey.Key3,
		"4":      hotkey.Key4,
		"5":      hotkey.Key5,
		"6":      hotkey.Key6,
		"7":      hotkey.Key7,
		"8":      hotkey.Key8,
		"9":      hotkey.Key9,
		"A":      hotkey.KeyA,
		"B":      hotkey.KeyB,
		"C":      hotkey.KeyC,
		"D":      hotkey.KeyD,
		"E":      hotkey.KeyE,
		"F":      hotkey.KeyF,
		"G":      hotkey.KeyG,
		"H":      hotkey.KeyH,
		"I":      hotkey.KeyI,
		"J":      hotkey.KeyJ,
		"K":      hotkey.KeyK,
		"L":      hotkey.KeyL,
		"M":      hotkey.KeyM,
		"N":      hotkey.KeyN,
		"O":      hotkey.KeyO,
		"P":      hotkey.KeyP,
		"Q":      hotkey.KeyQ,
		"R":      hotkey.KeyR,
		"S":      hotkey.KeyS,
		"T":      hotkey.KeyT,
		"U":      hotkey.KeyU,
		"V":      hotkey.KeyV,
		"W":      hotkey.KeyW,
		"X":      hotkey.KeyX,
		"Y":      hotkey.KeyY,
		"Z":      hotkey.KeyZ,
		"Return": hotkey.KeyReturn,
		"Escape": hotkey.KeyEscape,
		"Delete": hotkey.KeyDelete,
		"Tab":    hotkey.KeyTab,
		"Left":   hotkey.KeyLeft,
		"Right":  hotkey.KeyRight,
		"Up":     hotkey.KeyUp,
		"Down":   hotkey.KeyDown,
		"F1":     hotkey.KeyF1,
		"F2":     hotkey.KeyF2,
		"F3":     hotkey.KeyF3,
		"F4":     hotkey.KeyF4,
		"F5":     hotkey.KeyF5,
		"F6":     hotkey.KeyF6,
		"F7":     hotkey.KeyF7,
		"F8":     hotkey.KeyF8,
		"F9":     hotkey.KeyF9,
		"F10":    hotkey.KeyF10,
		"F11":    hotkey.KeyF11,
		"F12":    hotkey.KeyF12,
		"F13":    hotkey.KeyF13,
		"F14":    hotkey.KeyF14,
		"F15":    hotkey.KeyF15,
		"F16":    hotkey.KeyF16,
		"F17":    hotkey.KeyF17,
		"F18":    hotkey.KeyF18,
		"F19":    hotkey.KeyF19,
		"F20":    hotkey.KeyF20,
	}
	key, ok := keys[token]
	return key, ok
}
