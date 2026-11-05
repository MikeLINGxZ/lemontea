//go:build darwin || windows

package global_hotkey

import (
	"context"
	"fmt"
	"sync"

	"golang.design/x/hotkey"
)

// 定义全局快捷键句柄的调用契约
type hotkeyHandle interface {
	// 注册全局快捷键句柄，返回操作错误
	Register() error
	// 注销全局快捷键句柄，返回操作错误
	Unregister() error
	// 处理按键按下，返回事件通道
	Keydown() <-chan hotkey.Event
}

// 定义全局快捷键工厂回调的参数与返回值
type hotkeyFactory func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle

// StartQuickChat 注册用于切换快捷聊天窗口的系统级全局快捷键
// 接收本次调用的上下文、on已按下，返回清理或通知回调及可能发生的错误
func StartQuickChat(ctx context.Context, onPressed func()) (func(), error) {
	return StartQuickChatWithShortcut(ctx, onPressed, DefaultQuickChatShortcut())
}

// StartQuickChatWithShortcut 注册自定义的系统级快捷聊天快捷键
// 接收本次调用的上下文、on已按下、快捷键，返回清理或通知回调及可能发生的错误
func StartQuickChatWithShortcut(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
	return startQuickChatWithShortcut(ctx, onPressed, shortcut, func(mods []hotkey.Modifier, key hotkey.Key) hotkeyHandle {
		// 启动快捷聊天带有快捷键时调用的回调，接收mods 对应的数据、键，返回全局快捷键句柄
		return hotkey.New(mods, key)
	})
}

// ProbeQuickChatShortcut 通过临时注册快捷键检查是否存在冲突
// 接收快捷键，返回操作错误
func ProbeQuickChatShortcut(shortcut string) error {
	spec, err := hotkeySpecFromShortcut(shortcut) // 规格、处理全局快捷键规格起点快捷键时的错误
	if err != nil {
		return err
	}
	hk := hotkey.New(spec.modifiers, spec.key) // 快捷键
	registerErr := hk.Register()               // 注册当前数据时的错误
	if registerErr != nil {
		return registerErr
	}
	return hk.Unregister()
}

// startQuickChat 注入热键工厂，以便测试默认全局快捷键的注册
// 接收本次调用的上下文、on已按下、工厂，返回清理或通知回调及可能发生的错误
func startQuickChat(ctx context.Context, onPressed func(), factory hotkeyFactory) (func(), error) {
	return startQuickChatWithShortcut(ctx, onPressed, DefaultQuickChatShortcut(), factory)
}

// startQuickChatWithShortcut 注入热键工厂，以便测试全局快捷键的注册
// 接收本次调用的上下文、on已按下、快捷键、工厂，返回清理或通知回调及可能发生的错误
// 注册当前数据
// 创建可取消的子上下文
func startQuickChatWithShortcut(ctx context.Context, onPressed func(), shortcut string, factory hotkeyFactory) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}

	spec, err := hotkeySpecFromShortcut(shortcut) // 规格、处理全局快捷键规格起点快捷键时的错误
	if err != nil {
		return nil, err
	}
	hk := factory(spec.modifiers, spec.key) // 快捷键
	registerErr := hk.Register()            // 注册当前数据时的错误
	if registerErr != nil {
		return nil, registerErr
	}

	listenCtx, cancel := context.WithCancel(ctx) // 监听上下文、用于取消本次操作的回调
	done := make(chan struct{})                  // 完成

	go /* 处理按键按下if是否成功returnifon已按下nilon已按下时调用的回调，无返回值 */ func() {
		defer close(done)
		for {
			select {
			case <-listenCtx.Done():
				return
			case /* 是否成功取得对应值 */ _, ok := <-hk.Keydown():
				if !ok {
					return
				}
				if onPressed != nil {
					onPressed()
				}
			}
		}
	}()

	// 停止单次执行锁
	var stopOnce sync.Once
	// 停止
	stop := /* 实现停止回调，无返回值 */ func() {
		stopOnce.Do( /* 发送请求并读取响应时调用的回调，无返回值 */ func() {
			cancel()
			_ = hk.Unregister()
			<-done
		})
	}

	return stop, nil
}

// 描述快捷聊天全局快捷键规格使用的数据结构
type quickChatHotkeySpec struct {
	// 修饰键
	modifiers []hotkey.Modifier
	// 键
	key hotkey.Key
}

// 处理全局快捷键规格起点快捷键，接收快捷键，返回快捷聊天全局快捷键规格及可能发生的错误
func hotkeySpecFromShortcut(shortcut string) (quickChatHotkeySpec, error) {
	parsed, err := ParseQuickChatShortcut(shortcut) // 解析结果、解析快捷聊天快捷键时的错误
	if err != nil {
		return quickChatHotkeySpec{}, err
	}

	modifiers := make([]hotkey.Modifier, 0, len(parsed.Modifiers)) // 修饰键
	// 遍历修饰键，逐项取得修饰键令牌
	for _, modifierToken := range parsed.Modifiers {
		modifier, ok := quickChatModifierForToken(modifierToken) // 修饰键、是否成功取得对应值
		if !ok {
			return quickChatHotkeySpec{}, fmt.Errorf("unsupported shortcut modifier %q", modifierToken)
		}
		modifiers = append(modifiers, modifier)
	}

	key, ok := quickChatKeyForToken(parsed.Key) // 键、是否成功取得对应值
	if !ok {
		return quickChatHotkeySpec{}, fmt.Errorf("unsupported shortcut key %q", parsed.Key)
	}
	return quickChatHotkeySpec{modifiers: modifiers, key: key}, nil
}

// 把快捷键令牌转换为平台热键库的键值，返回键值及是否支持
// 先查找通用键值映射，未命中时再查找平台扩展键
func quickChatKeyForToken(token string) (hotkey.Key, bool) {
	// 键集合
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
	if /* 键、是否成功取得对应值 */ key, ok := keys[token]; ok {
		return key, true
	}
	return quickChatExtraKeyForToken(token)
}
