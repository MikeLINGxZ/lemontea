package window_options

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/window_id"
)

// TestDefaultQuickChat verifies the quick chat popup uses the dedicated route and compact floating size.
func TestDefaultQuickChat(t *testing.T) {
	opts := DefaultQuickChat()

	if opts.Name != window_id.QuickChat {
		t.Fatalf("Name = %q, want %q", opts.Name, window_id.QuickChat)
	}
	if opts.URL != "/?entry=quick_chat" {
		t.Fatalf("URL = %q, want quick chat entry", opts.URL)
	}
	if opts.Width != 720 || opts.Height != 54 {
		t.Fatalf("size = %dx%d, want 720x54", opts.Width, opts.Height)
	}
	if opts.MinWidth != 560 || opts.MinHeight != 54 {
		t.Fatalf("min size = %dx%d, want 560x54", opts.MinWidth, opts.MinHeight)
	}
	if !opts.HideOnFocusLost {
		t.Fatalf("HideOnFocusLost = false, want true so the popup hides when focus leaves")
	}
	if !opts.Hidden {
		t.Fatalf("Hidden = false, want true so the popup can be preloaded without flashing on startup")
	}
	if opts.Frameless {
		t.Fatalf("Frameless = true, want false so the hidden titlebar configuration is used")
	}
	if opts.Mac.InvisibleTitleBarHeight != 54 {
		t.Fatalf("Mac.InvisibleTitleBarHeight = %d, want 54", opts.Mac.InvisibleTitleBarHeight)
	}
	if opts.HideOnEscape {
		t.Fatalf("HideOnEscape = true, want false so Esc uses CloseQuickChat instead of hiding")
	}
	if opts.MinimiseButtonState != application.ButtonHidden ||
		opts.MaximiseButtonState != application.ButtonHidden ||
		opts.CloseButtonState != application.ButtonHidden {
		t.Fatalf("button states = %v/%v/%v, want all hidden", opts.MinimiseButtonState, opts.MaximiseButtonState, opts.CloseButtonState)
	}
}

// TestQuickChatShortcut verifies the app-level shortcut stays discoverable and stable.
func TestQuickChatShortcut(t *testing.T) {
	if QuickChatShortcut() != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("shortcut = %q, want CmdOrCtrl+Shift+Space", QuickChatShortcut())
	}
}
