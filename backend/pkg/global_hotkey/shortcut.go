package global_hotkey

import (
	"fmt"
	"strconv"
	"strings"
)

// 默认快捷聊天快捷键
const defaultQuickChatShortcut = "CmdOrCtrl+Shift+Space"

// 修饰键顺序
var modifierOrder = []string{"CmdOrCtrl", "Cmd", "Ctrl", "Alt", "Shift"}

// 修饰键别名列表
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

// 键别名列表
var keyAliases = map[string]string{
	"space":        "Space",
	"return":       "Return",
	"enter":        "Return",
	"escape":       "Escape",
	"esc":          "Escape",
	"delete":       "Delete",
	"backspace":    "Delete",
	"tab":          "Tab",
	"left":         "Left",
	"arrowleft":    "Left",
	"right":        "Right",
	"arrowright":   "Right",
	"up":           "Up",
	"arrowup":      "Up",
	"down":         "Down",
	"arrowdown":    "Down",
	"home":         "Home",
	"end":          "End",
	"pageup":       "PageUp",
	"pagedown":     "PageDown",
	"insert":       "Insert",
	"ins":          "Insert",
	"backquote":    "Backquote",
	"grave":        "Backquote",
	"`":            "Backquote",
	"minus":        "Minus",
	"-":            "Minus",
	"equal":        "Equal",
	"=":            "Equal",
	"bracketleft":  "BracketLeft",
	"leftbracket":  "BracketLeft",
	"[":            "BracketLeft",
	"bracketright": "BracketRight",
	"rightbracket": "BracketRight",
	"]":            "BracketRight",
	"backslash":    "Backslash",
	"\\":           "Backslash",
	"semicolon":    "Semicolon",
	";":            "Semicolon",
	"quote":        "Quote",
	"apostrophe":   "Quote",
	"'":            "Quote",
	"comma":        "Comma",
	",":            "Comma",
	"period":       "Period",
	".":            "Period",
	"slash":        "Slash",
	"/":            "Slash",
}

// ShortcutSpec 是用户快捷键经过规范化后的表示
type ShortcutSpec struct {
	// 快捷键
	Accelerator string
	// 修饰键
	Modifiers []string
	// 键
	Key string
}

// ShortcutValidationStatus 表示快捷键的实时可用状态
type ShortcutValidationStatus string

const (
	// 快捷键校验可用项
	ShortcutValidationAvailable ShortcutValidationStatus = "available"
	// 快捷键校验冲突
	ShortcutValidationConflict ShortcutValidationStatus = "conflict"
	// 快捷键校验无效
	ShortcutValidationInvalid ShortcutValidationStatus = "invalid"
)

// ShortcutValidationResult 保存快捷键校验请求的结果
type ShortcutValidationResult struct {
	// 快捷键
	Shortcut string
	// 当前状态
	Status ShortcutValidationStatus
	// 消息
	Message string
}

// DefaultQuickChatShortcut 返回配置缺失时使用的默认快捷键表示
func DefaultQuickChatShortcut() string {
	return defaultQuickChatShortcut
}

// NormalizeQuickChatShortcut 将用户输入的快捷键字符串转换为统一格式
// 接收文本，返回文本及可能发生的错误
func NormalizeQuickChatShortcut(input string) (string, error) {
	parsed, err := ParseQuickChatShortcut(input) // 解析结果、解析快捷聊天快捷键时的错误
	if err != nil {
		return "", err
	}
	return parsed.Accelerator, nil
}

// ParseQuickChatShortcut 解析受支持的快捷聊天快捷键字符串
// 接收文本，返回快捷键规格及可能发生的错误
func ParseQuickChatShortcut(input string) (ShortcutSpec, error) {
	trimmed := strings.TrimSpace(input) // 已去除空白
	if trimmed == "" {
		trimmed = DefaultQuickChatShortcut()
	}

	parts := strings.Split(trimmed, "+") // 组成部分
	modifiers := map[string]bool{}       // 修饰键
	key := ""                            // 键
	// 遍历组成部分，逐项取得原始值片段
	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart) // 片段
		if part == "" {
			continue
		}
		normalizedPart := strings.ToLower(strings.ReplaceAll(part, " ", "")) // 规范化值片段
		if /* 修饰键、是否成功取得对应值 */ modifier, ok := modifierAliases[normalizedPart]; ok {
			modifiers[modifier] = true
			continue
		}
		if key != "" {
			return ShortcutSpec{}, fmt.Errorf("shortcut must contain exactly one non-modifier key")
		}
		normalizedKey, err := normalizeShortcutKey(normalizedPart) // 规范化值键、将快捷键按键名称归一化为统一标识时的错误
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

	orderedModifiers := make([]string, 0, len(modifiers)) // 已排序修饰键
	// 遍历修饰键顺序，逐项取得候选值
	for _, candidate := range modifierOrder {
		if modifiers[candidate] {
			orderedModifiers = append(orderedModifiers, candidate)
		}
	}
	tokens := append([]string{}, orderedModifiers...) // tokens 对应的数据
	tokens = append(tokens, key)
	return ShortcutSpec{
		Accelerator: strings.Join(tokens, "+"),
		Modifiers:   orderedModifiers,
		Key:         key,
	}, nil
}

// 将快捷键按键名称归一化为统一标识，接收键，返回文本及可能发生的错误
func normalizeShortcutKey(key string) (string, error) {
	if len(key) == 1 {
		ch := key[0] // 字符
		if ch >= 'a' && ch <= 'z' {
			return strings.ToUpper(key), nil
		}
		if ch >= '0' && ch <= '9' {
			return key, nil
		}
	}
	if /* 别名、是否成功取得对应值 */ alias, ok := keyAliases[key]; ok {
		return alias, nil
	}
	if strings.HasPrefix(key, "f") {
		number, err := strconv.Atoi(strings.TrimPrefix(key, "f")) // 数值、将文本转换为整数时的错误
		if err == nil && number >= 1 && number <= 20 {
			return "F" + strconv.Itoa(number), nil
		}
	}
	return "", fmt.Errorf("unsupported shortcut key %q", key)
}
