package global_hotkey

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestQuickChatServiceWaitsForApplicationStartedBeforeRegistration(t *testing.T) {
	startCalls := 0
	var appStarted func()
	service := newQuickChatService(
		func() {},
		func(context.Context, func(), string) (func(), error) {
			startCalls++
			return func() {}, nil
		},
		func(fn func()) func() {
			appStarted = fn
			return func() {}
		},
		func(string, ...any) {},
	)

	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup returned error: %v", err)
	}
	if startCalls != 0 {
		t.Fatalf("hotkey registered synchronously during startup")
	}
	if appStarted == nil {
		t.Fatal("startup did not subscribe to application started event")
	}

	appStarted()

	if startCalls != 1 {
		t.Fatalf("startCalls = %d, want 1 after application started", startCalls)
	}
}

func TestQuickChatServiceShutdownStopsRegisteredHotkey(t *testing.T) {
	stopCalls := 0
	service := newQuickChatService(
		func() {},
		func(context.Context, func(), string) (func(), error) {
			return func() { stopCalls++ }, nil
		},
		func(fn func()) func() {
			fn()
			return func() {}
		},
		func(string, ...any) {},
	)

	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup returned error: %v", err)
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown returned error: %v", err)
	}
	if stopCalls != 1 {
		t.Fatalf("stopCalls = %d, want 1", stopCalls)
	}
}

func TestQuickChatServiceLogsRegistrationError(t *testing.T) {
	var logLine string
	service := newQuickChatService(
		func() {},
		func(context.Context, func(), string) (func(), error) {
			return nil, errors.New("shortcut already taken")
		},
		func(fn func()) func() {
			fn()
			return func() {}
		},
		func(format string, args ...any) {
			logLine = format
		},
	)

	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup returned error: %v", err)
	}
	if !strings.Contains(logLine, "quick chat global hotkey registration failed") {
		t.Fatalf("logLine = %q, want registration failure log", logLine)
	}
}

func TestQuickChatServiceValidateShortcutReturnsAvailableForCurrentShortcut(t *testing.T) {
	var probeCalls int
	service := newQuickChatService(
		func() {},
		func(context.Context, func(), string) (func(), error) {
			return func() {}, nil
		},
		func(fn func()) func() { return func() {} },
		func(string, ...any) {},
	)
	service.setShortcutForTest("CmdOrCtrl+Shift+Space")
	service.probeShortcut = func(shortcut string) error {
		probeCalls++
		return nil
	}

	result := service.ValidateShortcut("CmdOrCtrl+Shift+Space")

	if result.Status != ShortcutValidationAvailable {
		t.Fatalf("status = %q, want available", result.Status)
	}
	if probeCalls != 0 {
		t.Fatalf("probe calls = %d, want 0 for current shortcut", probeCalls)
	}
}

func TestQuickChatServiceValidateShortcutReturnsConflictWhenProbeFails(t *testing.T) {
	service := newQuickChatService(
		func() {},
		func(context.Context, func(), string) (func(), error) {
			return func() {}, nil
		},
		func(fn func()) func() { return func() {} },
		func(string, ...any) {},
	)
	service.setShortcutForTest("CmdOrCtrl+Shift+Space")
	service.probeShortcut = func(shortcut string) error {
		return errors.New("shortcut already taken")
	}

	result := service.ValidateShortcut("CmdOrCtrl+Shift+K")

	if result.Status != ShortcutValidationConflict {
		t.Fatalf("status = %q, want conflict", result.Status)
	}
	if result.Shortcut != "CmdOrCtrl+Shift+K" {
		t.Fatalf("shortcut = %q, want CmdOrCtrl+Shift+K", result.Shortcut)
	}
}

func TestQuickChatServiceUpdateShortcutRestoresPreviousShortcutWhenRegistrationFails(t *testing.T) {
	var started []string
	var stopped []string
	service := newQuickChatService(
		func() {},
		func(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
			started = append(started, shortcut)
			if shortcut == "CmdOrCtrl+Shift+K" {
				return nil, errors.New("shortcut already taken")
			}
			return func() {
				stopped = append(stopped, shortcut)
			}, nil
		},
		func(fn func()) func() { return func() {} },
		func(string, ...any) {},
	)
	service.setShortcutForTest("CmdOrCtrl+Shift+Space")
	service.stop = func() {
		stopped = append(stopped, "CmdOrCtrl+Shift+Space")
	}

	err := service.UpdateShortcut(context.Background(), "CmdOrCtrl+Shift+K")

	if err == nil {
		t.Fatal("expected update conflict")
	}
	if service.CurrentShortcut() != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("current shortcut = %q, want restored default", service.CurrentShortcut())
	}
	if len(started) != 2 || started[0] != "CmdOrCtrl+Shift+K" || started[1] != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("started = %v, want new shortcut then restored default", started)
	}
	if len(stopped) != 1 || stopped[0] != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("stopped = %v, want previous shortcut stopped once", stopped)
	}
}

func TestQuickChatServiceUpdateShortcutRegistersWhenStartedWithoutActiveHotkey(t *testing.T) {
	var started []string
	service := newQuickChatService(
		func() {},
		func(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
			started = append(started, shortcut)
			return func() {}, nil
		},
		func(fn func()) func() { return func() {} },
		func(string, ...any) {},
	)
	service.started = true
	service.setShortcutForTest("CmdOrCtrl+Shift+Space")

	if err := service.UpdateShortcut(context.Background(), "CmdOrCtrl+Shift+K"); err != nil {
		t.Fatalf("UpdateShortcut returned error: %v", err)
	}

	if len(started) != 1 || started[0] != "CmdOrCtrl+Shift+K" {
		t.Fatalf("started = %v, want new shortcut registration", started)
	}
	if service.CurrentShortcut() != "CmdOrCtrl+Shift+K" {
		t.Fatalf("current shortcut = %q, want updated shortcut", service.CurrentShortcut())
	}
}

func TestQuickChatServiceUpdateShortcutUsesServiceLifetimeContext(t *testing.T) {
	service := newQuickChatService(
		func() {},
		func(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return func() {}, nil
		},
		func(fn func()) func() { return func() {} },
		func(string, ...any) {},
	)
	service.started = true
	service.setShortcutForTest("CmdOrCtrl+Shift+Space")
	service.stop = func() {}

	requestCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := service.UpdateShortcut(requestCtx, "CmdOrCtrl+Shift+K"); err != nil {
		t.Fatalf("UpdateShortcut returned error from canceled request context: %v", err)
	}
	if service.CurrentShortcut() != "CmdOrCtrl+Shift+K" {
		t.Fatalf("current shortcut = %q, want updated shortcut", service.CurrentShortcut())
	}
}

func TestQuickChatServiceRetriesCurrentShortcutWhenStartupRegistrationIsStale(t *testing.T) {
	started := make(chan string, 2)
	releaseFirstStart := make(chan struct{})
	var firstStart sync.Once
	service := newQuickChatService(
		func() {},
		func(ctx context.Context, onPressed func(), shortcut string) (func(), error) {
			started <- shortcut
			firstStart.Do(func() {
				<-releaseFirstStart
			})
			return func() {}, nil
		},
		func(fn func()) func() {
			go fn()
			return func() {}
		},
		func(string, ...any) {},
	)

	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup returned error: %v", err)
	}
	if got := <-started; got != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("initial registration = %q, want default", got)
	}
	if err := service.UpdateShortcut(context.Background(), "CmdOrCtrl+Shift+K"); err != nil {
		t.Fatalf("UpdateShortcut returned error: %v", err)
	}
	close(releaseFirstStart)

	select {
	case got := <-started:
		if got != "CmdOrCtrl+Shift+K" {
			t.Fatalf("retry registration = %q, want updated shortcut", got)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for updated shortcut registration")
	}
}
