package global_hotkey

import "testing"

func TestNormalizeQuickChatShortcutReturnsDefaultForBlank(t *testing.T) {
	got, err := NormalizeQuickChatShortcut("")
	if err != nil {
		t.Fatalf("NormalizeQuickChatShortcut returned error: %v", err)
	}
	if got != DefaultQuickChatShortcut() {
		t.Fatalf("shortcut = %q, want %q", got, DefaultQuickChatShortcut())
	}
}

func TestNormalizeQuickChatShortcutOrdersModifiers(t *testing.T) {
	got, err := NormalizeQuickChatShortcut("space + shift + cmdorctrl")
	if err != nil {
		t.Fatalf("NormalizeQuickChatShortcut returned error: %v", err)
	}
	if got != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("shortcut = %q, want CmdOrCtrl+Shift+Space", got)
	}
}

func TestNormalizeQuickChatShortcutRejectsModifierOnly(t *testing.T) {
	if _, err := NormalizeQuickChatShortcut("CmdOrCtrl+Shift"); err == nil {
		t.Fatal("expected modifier-only shortcut to fail")
	}
}

func TestNormalizeQuickChatShortcutRejectsUnsupportedKeys(t *testing.T) {
	if _, err := NormalizeQuickChatShortcut("CmdOrCtrl+Shift+AudioVolumeUp"); err == nil {
		t.Fatal("expected unsupported key to fail")
	}
}

func TestParseQuickChatShortcut(t *testing.T) {
	parsed, err := ParseQuickChatShortcut("Ctrl+Alt+K")
	if err != nil {
		t.Fatalf("ParseQuickChatShortcut returned error: %v", err)
	}
	if parsed.Accelerator != "Ctrl+Alt+K" {
		t.Fatalf("Accelerator = %q, want Ctrl+Alt+K", parsed.Accelerator)
	}
	if parsed.Key != "K" {
		t.Fatalf("Key = %q, want K", parsed.Key)
	}
	if len(parsed.Modifiers) != 2 || parsed.Modifiers[0] != "Ctrl" || parsed.Modifiers[1] != "Alt" {
		t.Fatalf("Modifiers = %v, want [Ctrl Alt]", parsed.Modifiers)
	}
}
