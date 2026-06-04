package global_hotkey

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultQuickChatShortcut = "CmdOrCtrl+Shift+Space"

var modifierOrder = []string{"CmdOrCtrl", "Cmd", "Ctrl", "Alt", "Shift"}

var modifierAliases = map[string]string{
	"cmdorctrl":        "CmdOrCtrl",
	"commandorcontrol": "CmdOrCtrl",
	"commandorctrl":    "CmdOrCtrl",
	"cmd":              "Cmd",
	"command":          "Cmd",
	"meta":             "Cmd",
	"ctrl":             "Ctrl",
	"control":          "Ctrl",
	"alt":              "Alt",
	"option":           "Alt",
	"shift":            "Shift",
}

var keyAliases = map[string]string{
	"space":      "Space",
	"return":     "Return",
	"enter":      "Return",
	"escape":     "Escape",
	"esc":        "Escape",
	"delete":     "Delete",
	"backspace":  "Delete",
	"tab":        "Tab",
	"left":       "Left",
	"arrowleft":  "Left",
	"right":      "Right",
	"arrowright": "Right",
	"up":         "Up",
	"arrowup":    "Up",
	"down":       "Down",
	"arrowdown":  "Down",
}

// ShortcutSpec is the normalized representation of a user-facing accelerator.
type ShortcutSpec struct {
	Accelerator string
	Modifiers   []string
	Key         string
}

// ShortcutValidationStatus describes realtime shortcut availability.
type ShortcutValidationStatus string

const (
	ShortcutValidationAvailable ShortcutValidationStatus = "available"
	ShortcutValidationConflict  ShortcutValidationStatus = "conflict"
	ShortcutValidationInvalid   ShortcutValidationStatus = "invalid"
)

// ShortcutValidationResult is returned by shortcut validation requests.
type ShortcutValidationResult struct {
	Shortcut string
	Status   ShortcutValidationStatus
	Message  string
}

// DefaultQuickChatShortcut returns the persisted accelerator used when config is missing.
func DefaultQuickChatShortcut() string {
	return defaultQuickChatShortcut
}

// NormalizeQuickChatShortcut converts a user shortcut string into stable accelerator syntax.
func NormalizeQuickChatShortcut(input string) (string, error) {
	parsed, err := ParseQuickChatShortcut(input)
	if err != nil {
		return "", err
	}
	return parsed.Accelerator, nil
}

// ParseQuickChatShortcut parses supported quick chat accelerator strings.
func ParseQuickChatShortcut(input string) (ShortcutSpec, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		trimmed = DefaultQuickChatShortcut()
	}

	parts := strings.Split(trimmed, "+")
	modifiers := map[string]bool{}
	key := ""
	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		normalizedPart := strings.ToLower(strings.ReplaceAll(part, " ", ""))
		if modifier, ok := modifierAliases[normalizedPart]; ok {
			modifiers[modifier] = true
			continue
		}
		if key != "" {
			return ShortcutSpec{}, fmt.Errorf("shortcut must contain exactly one non-modifier key")
		}
		normalizedKey, err := normalizeShortcutKey(normalizedPart)
		if err != nil {
			return ShortcutSpec{}, err
		}
		key = normalizedKey
	}

	if key == "" {
		return ShortcutSpec{}, fmt.Errorf("shortcut must include a non-modifier key")
	}
	if len(modifiers) == 0 {
		return ShortcutSpec{}, fmt.Errorf("shortcut must include at least one modifier")
	}
	if modifiers["CmdOrCtrl"] && (modifiers["Cmd"] || modifiers["Ctrl"]) {
		return ShortcutSpec{}, fmt.Errorf("CmdOrCtrl cannot be combined with Cmd or Ctrl")
	}

	orderedModifiers := make([]string, 0, len(modifiers))
	for _, candidate := range modifierOrder {
		if modifiers[candidate] {
			orderedModifiers = append(orderedModifiers, candidate)
		}
	}
	tokens := append([]string{}, orderedModifiers...)
	tokens = append(tokens, key)
	return ShortcutSpec{
		Accelerator: strings.Join(tokens, "+"),
		Modifiers:   orderedModifiers,
		Key:         key,
	}, nil
}

func normalizeShortcutKey(key string) (string, error) {
	if len(key) == 1 {
		ch := key[0]
		if ch >= 'a' && ch <= 'z' {
			return strings.ToUpper(key), nil
		}
		if ch >= '0' && ch <= '9' {
			return key, nil
		}
	}
	if alias, ok := keyAliases[key]; ok {
		return alias, nil
	}
	if strings.HasPrefix(key, "f") {
		number, err := strconv.Atoi(strings.TrimPrefix(key, "f"))
		if err == nil && number >= 1 && number <= 20 {
			return "F" + strconv.Itoa(number), nil
		}
	}
	return "", fmt.Errorf("unsupported shortcut key %q", key)
}
