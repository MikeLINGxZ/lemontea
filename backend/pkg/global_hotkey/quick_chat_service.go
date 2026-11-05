package global_hotkey

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// 定义快捷聊天启动器回调的参数与返回值
type quickChatStarter func(context.Context, func(), string) (func(), error)

// 定义快捷聊天探测回调的参数与返回值
type quickChatProbe func(string) error

// 定义应用已启动订阅者回调的参数与返回值
type applicationStartedSubscriber func(func()) func()

// 定义日志记录器函数回调的参数与返回值
type loggerFunc func(string, ...any)

// QuickChatService 等待 Wails 启动平台应用后注册全局快捷聊天快捷键，
// 确保调用原生热键 API 时 AppKit 已运行
type QuickChatService struct {
	// 处理已按下，无返回值
	onPressed func()
	// 开始
	start quickChatStarter
	// 探测快捷键
	probeShortcut quickChatProbe
	// 订阅应用已启动
	subscribeAppStarted applicationStartedSubscriber
	// 日志回调
	logf loggerFunc

	// 保护共享状态的互斥锁
	mu sync.Mutex
	// 停止快捷聊天服务，无返回值
	stop func()
	// 处理解除订阅，无返回值
	unsubscribe func()
	// 快捷键
	shortcut string
	// 已启动
	started bool
	// 正在注册
	registering bool
}

// 构建快捷聊天服务，接收on已按下，返回快捷聊天服务
func NewQuickChatService(onPressed func()) *QuickChatService {
	return NewQuickChatServiceWithShortcut(onPressed, DefaultQuickChatShortcut())
}

// NewQuickChatServiceWithShortcut 使用指定的初始快捷键创建快捷聊天服务
// 接收on已按下、快捷键，返回快捷聊天服务
func NewQuickChatServiceWithShortcut(onPressed func(), shortcut string) *QuickChatService {
	normalized, err := NormalizeQuickChatShortcut(shortcut) // 规范化值、规范化快捷聊天快捷键时的错误
	if err != nil {
		normalized = DefaultQuickChatShortcut()
	}
	return newQuickChatService(
		onPressed,
		StartQuickChatWithShortcut,
		subscribeApplicationStarted,
		log.Printf,
		normalized,
	)
}

// 订阅平台应用已启动事件并在就绪时注册快捷键，接收fn 对应的清理或通知回调，返回清理或通知回调
func subscribeApplicationStarted(fn func()) func() {
	app := application.Get() // Wails 应用实例
	// 解绑函数
	var unsubscribers []func()
	// 订阅
	subscribe := /* 实现订阅回调，接收事件类型，无返回值 */ func(eventType events.ApplicationEventType) {
		unsubscribers = append(unsubscribers, app.Event.OnApplicationEvent(eventType, func(*application.ApplicationEvent) {
			// 处理应用事件时调用的回调，接收应用事件，无返回值
			fn()
		}))
	}

	subscribe(events.Common.ApplicationStarted)
	switch runtime.GOOS {
	case "darwin":
		subscribe(events.Mac.ApplicationDidFinishLaunching)
	case "windows":
		subscribe(events.Windows.ApplicationStarted)
	case "linux":
		subscribe(events.Linux.ApplicationStartup)
	}

	return /* 执行当前步骤的回调，无返回值 */ func() {
		// 遍历解绑函数，逐项取得解除订阅
		for _, unsubscribe := range unsubscribers {
			if unsubscribe != nil {
				unsubscribe()
			}
		}
	}
}

// 构建快捷聊天服务，接收on已按下、开始、订阅应用已启动、日志回调、快捷键，返回快捷聊天服务
func newQuickChatService(
	onPressed func(),
	start quickChatStarter,
	subscribeAppStarted applicationStartedSubscriber,
	logf loggerFunc,
	shortcut ...string,
) *QuickChatService {
	if subscribeAppStarted == nil {
		subscribeAppStarted = /* 实现订阅应用已启动回调，接收fn 对应的清理或通知回调，返回清理或通知回调 */ func(fn func()) func() {
			fn()
			return /* 执行当前步骤的回调，无返回值 */ func() {}
		}
	}
	if logf == nil {
		logf = /* 实现日志回调回调，接收文本、待检查的动态数据，无返回值 */ func(string, ...any) {}
	}
	initialShortcut := DefaultQuickChatShortcut() // 初始快捷键
	if len(shortcut) > 0 {
		normalized, err := NormalizeQuickChatShortcut(shortcut[0]) // 规范化值、规范化快捷聊天快捷键时的错误
		if err == nil {
			initialShortcut = normalized
		}
	}
	return &QuickChatService{
		onPressed:           onPressed,
		start:               start,
		probeShortcut:       ProbeQuickChatShortcut,
		subscribeAppStarted: subscribeAppStarted,
		logf:                logf,
		shortcut:            initialShortcut,
	}
}

// 在 Wails 服务启动时保存运行环境并初始化当前服务，接收本次调用的上下文、服务选项列表，返回操作错误
func (s *QuickChatService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	s.started = true
	needsSubscribe := s.unsubscribe == nil // 需要订阅
	s.mu.Unlock()

	if !needsSubscribe {
		return nil
	}

	// 解除订阅
	unsubscribe := s.subscribeAppStarted( /* 订阅应用已启动时调用的回调，无返回值 */ func() {
		s.logf("quick chat global hotkey startup event received")
		s.register(ctx)
	})
	go s.registerAfterFallbackDelay(ctx, 2*time.Second)

	s.mu.Lock()
	if s.started && s.unsubscribe == nil {
		s.unsubscribe = unsubscribe
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if unsubscribe != nil {
		unsubscribe()
	}
	return nil
}

// 平台未发送启动事件时延迟尝试快捷键注册，接收本次调用的上下文、延时，无返回值
func (s *QuickChatService) registerAfterFallbackDelay(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay) // 定时器
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		s.logf("quick chat global hotkey fallback registration attempt")
		s.register(ctx)
	}
}

// 在 Wails 服务关闭时释放当前服务持有的资源，返回操作错误
func (s *QuickChatService) ServiceShutdown() error {
	s.mu.Lock()
	s.started = false
	stop := s.stop // 停止
	s.stop = nil
	unsubscribe := s.unsubscribe // 解除订阅
	s.unsubscribe = nil
	s.mu.Unlock()

	if unsubscribe != nil {
		unsubscribe()
	}
	if stop != nil {
		stop()
	}
	return nil
}

// 注册快捷聊天服务，接收本次调用的上下文，无返回值
func (s *QuickChatService) register(ctx context.Context) {
	s.mu.Lock()
	if !s.started || s.stop != nil || s.registering {
		s.mu.Unlock()
		return
	}
	s.registering = true
	shortcut := s.shortcut // 快捷键
	s.mu.Unlock()

	stop, err := s.start(ctx, s.onPressed, shortcut) // 停止、启动当前数据时的错误
	s.mu.Lock()
	s.registering = false
	if err != nil {
		shouldRetry := s.started && s.stop == nil && s.shortcut != shortcut // 是否需重试
		if s.started {
			s.logf("quick chat global hotkey registration failed: %v", err)
		}
		s.mu.Unlock()
		if shouldRetry {
			go s.register(context.Background())
		}
		return
	}
	if !s.started || s.stop != nil {
		s.mu.Unlock()
		stop()
		return
	}
	if s.shortcut != shortcut {
		s.mu.Unlock()
		stop()
		go s.register(context.Background())
		return
	}
	s.stop = stop
	s.mu.Unlock()
	s.logf("quick chat global hotkey registered")
}

// CurrentShortcut 返回服务当前管理的规范化快捷聊天快捷键
func (s *QuickChatService) CurrentShortcut() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shortcut == "" {
		return DefaultQuickChatShortcut()
	}
	return s.shortcut
}

// ValidateShortcut 检查快捷键是否可用，保留当前注册状态
// 接收快捷键，返回快捷键校验结果
func (s *QuickChatService) ValidateShortcut(shortcut string) ShortcutValidationResult {
	normalized, err := NormalizeQuickChatShortcut(shortcut) // 规范化值、规范化快捷聊天快捷键时的错误
	if err != nil {
		return ShortcutValidationResult{Status: ShortcutValidationInvalid, Message: err.Error()}
	}
	if normalized == s.CurrentShortcut() {
		return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationAvailable}
	}
	if s.probeShortcut == nil {
		return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationAvailable}
	}
	probeShortcutErr := s.probeShortcut(normalized) // 处理探测快捷键时的错误
	if probeShortcutErr != nil {
		return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationConflict, Message: probeShortcutErr.Error()}
	}
	return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationAvailable}
}

// UpdateShortcut 立即切换当前注册的全局快捷聊天快捷键
// 接收本次调用的上下文、快捷键，返回操作错误
func (s *QuickChatService) UpdateShortcut(ctx context.Context, shortcut string) error {
	normalized, err := NormalizeQuickChatShortcut(shortcut) // 规范化值、规范化快捷聊天快捷键时的错误
	if err != nil {
		return err
	}
	startCtx := context.Background() // 开始上下文
	if ctx != nil {
		startCtx = context.WithoutCancel(ctx)
	}

	s.mu.Lock()
	previousShortcut := s.shortcut // 先前快捷键
	if previousShortcut == "" {
		previousShortcut = DefaultQuickChatShortcut()
	}
	if normalized == previousShortcut {
		s.shortcut = normalized
		s.mu.Unlock()
		return nil
	}
	previousStop := s.stop                        // 先前停止
	wasRegistered := previousStop != nil          // 曾经已注册
	shouldRegister := s.started && !s.registering // 是否需注册
	s.stop = nil
	s.shortcut = normalized
	s.mu.Unlock()

	if previousStop != nil {
		previousStop()
	}
	if !wasRegistered && !shouldRegister {
		return nil
	}

	newStop, err := s.start(startCtx, s.onPressed, normalized) // 新停止
	if err != nil {
		// 恢复停止
		var restoreStop func()
		// 恢复错误
		var restoreErr error
		if wasRegistered {
			restoreStop, restoreErr = s.start(startCtx, s.onPressed, previousShortcut)
		}
		s.mu.Lock()
		s.shortcut = previousShortcut
		s.stop = restoreStop
		s.mu.Unlock()
		if wasRegistered && restoreErr != nil {
			s.logf("quick chat global hotkey restore failed: %v", restoreErr)
		}
		return err
	}

	s.mu.Lock()
	s.stop = newStop
	s.mu.Unlock()
	return nil
}

// 设置快捷键对应测试，接收快捷键，无返回值
func (s *QuickChatService) setShortcutForTest(shortcut string) {
	normalized, err := NormalizeQuickChatShortcut(shortcut) // 规范化值、规范化快捷聊天快捷键时的错误
	if err != nil {
		normalized = DefaultQuickChatShortcut()
	}
	s.mu.Lock()
	s.shortcut = normalized
	s.mu.Unlock()
}
