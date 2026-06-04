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

type quickChatStarter func(context.Context, func(), string) (func(), error)
type quickChatProbe func(string) error
type applicationStartedSubscriber func(func()) func()
type loggerFunc func(string, ...any)

// QuickChatService registers the global quick chat shortcut after Wails has
// started its platform application, so native hotkey APIs run after AppKit is live.
type QuickChatService struct {
	onPressed           func()
	start               quickChatStarter
	probeShortcut       quickChatProbe
	subscribeAppStarted applicationStartedSubscriber
	logf                loggerFunc

	mu          sync.Mutex
	stop        func()
	unsubscribe func()
	shortcut    string
	started     bool
	registering bool
}

func NewQuickChatService(onPressed func()) *QuickChatService {
	return NewQuickChatServiceWithShortcut(onPressed, DefaultQuickChatShortcut())
}

// NewQuickChatServiceWithShortcut creates a quick chat service with an initial shortcut.
func NewQuickChatServiceWithShortcut(onPressed func(), shortcut string) *QuickChatService {
	normalized, err := NormalizeQuickChatShortcut(shortcut)
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

func subscribeApplicationStarted(fn func()) func() {
	app := application.Get()
	var unsubscribers []func()
	subscribe := func(eventType events.ApplicationEventType) {
		unsubscribers = append(unsubscribers, app.Event.OnApplicationEvent(eventType, func(*application.ApplicationEvent) {
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

	return func() {
		for _, unsubscribe := range unsubscribers {
			if unsubscribe != nil {
				unsubscribe()
			}
		}
	}
}

func newQuickChatService(
	onPressed func(),
	start quickChatStarter,
	subscribeAppStarted applicationStartedSubscriber,
	logf loggerFunc,
	shortcut ...string,
) *QuickChatService {
	if subscribeAppStarted == nil {
		subscribeAppStarted = func(fn func()) func() {
			fn()
			return func() {}
		}
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	initialShortcut := DefaultQuickChatShortcut()
	if len(shortcut) > 0 {
		if normalized, err := NormalizeQuickChatShortcut(shortcut[0]); err == nil {
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

func (s *QuickChatService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	s.started = true
	needsSubscribe := s.unsubscribe == nil
	s.mu.Unlock()

	if !needsSubscribe {
		return nil
	}

	unsubscribe := s.subscribeAppStarted(func() {
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

func (s *QuickChatService) registerAfterFallbackDelay(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		s.logf("quick chat global hotkey fallback registration attempt")
		s.register(ctx)
	}
}

func (s *QuickChatService) ServiceShutdown() error {
	s.mu.Lock()
	s.started = false
	stop := s.stop
	s.stop = nil
	unsubscribe := s.unsubscribe
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

func (s *QuickChatService) register(ctx context.Context) {
	s.mu.Lock()
	if !s.started || s.stop != nil || s.registering {
		s.mu.Unlock()
		return
	}
	s.registering = true
	shortcut := s.shortcut
	s.mu.Unlock()

	stop, err := s.start(ctx, s.onPressed, shortcut)
	s.mu.Lock()
	s.registering = false
	if err != nil {
		shouldRetry := s.started && s.stop == nil && s.shortcut != shortcut
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

// CurrentShortcut returns the normalized quick chat shortcut currently owned by the service.
func (s *QuickChatService) CurrentShortcut() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shortcut == "" {
		return DefaultQuickChatShortcut()
	}
	return s.shortcut
}

// ValidateShortcut reports whether a shortcut is usable without changing the active registration.
func (s *QuickChatService) ValidateShortcut(shortcut string) ShortcutValidationResult {
	normalized, err := NormalizeQuickChatShortcut(shortcut)
	if err != nil {
		return ShortcutValidationResult{Status: ShortcutValidationInvalid, Message: err.Error()}
	}
	if normalized == s.CurrentShortcut() {
		return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationAvailable}
	}
	if s.probeShortcut == nil {
		return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationAvailable}
	}
	if err := s.probeShortcut(normalized); err != nil {
		return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationConflict, Message: err.Error()}
	}
	return ShortcutValidationResult{Shortcut: normalized, Status: ShortcutValidationAvailable}
}

// UpdateShortcut immediately switches the active quick chat global shortcut.
func (s *QuickChatService) UpdateShortcut(ctx context.Context, shortcut string) error {
	normalized, err := NormalizeQuickChatShortcut(shortcut)
	if err != nil {
		return err
	}
	startCtx := context.Background()
	if ctx != nil {
		startCtx = context.WithoutCancel(ctx)
	}

	s.mu.Lock()
	previousShortcut := s.shortcut
	if previousShortcut == "" {
		previousShortcut = DefaultQuickChatShortcut()
	}
	if normalized == previousShortcut {
		s.shortcut = normalized
		s.mu.Unlock()
		return nil
	}
	previousStop := s.stop
	wasRegistered := previousStop != nil
	shouldRegister := s.started && !s.registering
	s.stop = nil
	s.shortcut = normalized
	s.mu.Unlock()

	if previousStop != nil {
		previousStop()
	}
	if !wasRegistered && !shouldRegister {
		return nil
	}

	newStop, err := s.start(startCtx, s.onPressed, normalized)
	if err != nil {
		var restoreStop func()
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

func (s *QuickChatService) setShortcutForTest(shortcut string) {
	normalized, err := NormalizeQuickChatShortcut(shortcut)
	if err != nil {
		normalized = DefaultQuickChatShortcut()
	}
	s.mu.Lock()
	s.shortcut = normalized
	s.mu.Unlock()
}
