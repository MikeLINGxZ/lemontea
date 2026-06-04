//go:build darwin || windows

package global_hotkey

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"golang.design/x/hotkey"
)

type fakeHotkey struct {
	keydown      chan hotkey.Event
	registered   atomic.Bool
	unregistered atomic.Bool
	registerErr  error
}

func newFakeHotkey() *fakeHotkey {
	return &fakeHotkey{keydown: make(chan hotkey.Event, 1)}
}

func (f *fakeHotkey) Register() error {
	if f.registerErr != nil {
		return f.registerErr
	}
	f.registered.Store(true)
	return nil
}

func (f *fakeHotkey) Unregister() error {
	f.unregistered.Store(true)
	return nil
}

func (f *fakeHotkey) Keydown() <-chan hotkey.Event {
	return f.keydown
}

// TestStartQuickChatRegistersSystemHotkey verifies the quick chat shortcut is registered as a global hotkey.
func TestStartQuickChatRegistersSystemHotkey(t *testing.T) {
	fake := newFakeHotkey()
	var gotMods []hotkey.Modifier
	var gotKey hotkey.Key

	stop, err := startQuickChat(context.Background(), func() {}, func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle {
		gotMods = append([]hotkey.Modifier(nil), mods...)
		gotKey = key
		return fake
	})
	if err != nil {
		t.Fatalf("startQuickChat returned error: %v", err)
	}
	defer stop()

	if !fake.registered.Load() {
		t.Fatal("hotkey was not registered")
	}
	if gotKey != hotkey.KeySpace {
		t.Fatalf("key = %v, want KeySpace", gotKey)
	}
	if !quickChatModifiersMatch(gotMods) {
		t.Fatalf("mods = %v, want quick chat modifiers", gotMods)
	}
}

// TestStartQuickChatInvokesCallbackOnKeydown verifies the registered global shortcut toggles quick chat.
func TestStartQuickChatInvokesCallbackOnKeydown(t *testing.T) {
	fake := newFakeHotkey()
	called := make(chan struct{}, 1)

	stop, err := startQuickChat(context.Background(), func() {
		called <- struct{}{}
	}, func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle {
		return fake
	})
	if err != nil {
		t.Fatalf("startQuickChat returned error: %v", err)
	}
	defer stop()

	fake.keydown <- hotkey.Event{}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("callback was not called after keydown")
	}
}

// TestStartQuickChatStopsRegistration verifies shutdown unregisters the global shortcut.
func TestStartQuickChatStopsRegistration(t *testing.T) {
	fake := newFakeHotkey()

	stop, err := startQuickChat(context.Background(), func() {}, func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle {
		return fake
	})
	if err != nil {
		t.Fatalf("startQuickChat returned error: %v", err)
	}

	stop()

	if !fake.unregistered.Load() {
		t.Fatal("hotkey was not unregistered")
	}
}

// TestStartQuickChatReturnsRegisterError verifies registration conflicts are surfaced to startup logging.
func TestStartQuickChatReturnsRegisterError(t *testing.T) {
	fake := newFakeHotkey()
	fake.registerErr = errors.New("shortcut already taken")

	stop, err := startQuickChat(context.Background(), func() {}, func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle {
		return fake
	})
	if err == nil {
		if stop != nil {
			stop()
		}
		t.Fatal("expected registration error")
	}
}
