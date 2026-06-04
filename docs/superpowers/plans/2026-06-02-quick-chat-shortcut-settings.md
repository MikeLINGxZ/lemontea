# Quick Chat Shortcut Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a configurable quick chat shortcut page with realtime conflict detection and immediate runtime application.

**Architecture:** Store a single `quick_chat_shortcut` accelerator in config. Backend validates and applies shortcuts through the existing global hotkey service; frontend records keyboard input and renders backend validation status.

**Tech Stack:** Go, Wails service bindings, `golang.design/x/hotkey`, React, Zustand, Vitest, Testing Library.

---

### Task 1: Backend Shortcut Model And Parser

**Files:**
- Modify: `backend/models/data_models/config.go`
- Modify: `backend/models/view_model/settings.go`
- Create: `backend/pkg/global_hotkey/shortcut.go`
- Test: `backend/pkg/global_hotkey/shortcut_test.go`

- [ ] Write tests for default shortcut normalization, modifier ordering, unsupported keys, and empty shortcut errors.
- [ ] Implement `DefaultQuickChatShortcut`, `NormalizeQuickChatShortcut`, and `ParseQuickChatShortcut`.
- [ ] Add `QuickChatShortcut string json:"quick_chat_shortcut"` to config and bootstrap view model.
- [ ] Run `go test ./backend/pkg/global_hotkey`.

### Task 2: Backend Validation, Save, And Runtime Apply

**Files:**
- Modify: `backend/pkg/global_hotkey/quick_chat.go`
- Modify: `backend/pkg/global_hotkey/quick_chat_service.go`
- Modify: `backend/service/settings/settings.go`
- Modify: `backend/service/settings/settings_internal.go`
- Create: `backend/service/settings/settings_dto/save_shortcut_settings.go`
- Test: `backend/pkg/global_hotkey/quick_chat_service_test.go`
- Test: `backend/service/settings/settings_test.go`

- [ ] Write failing tests for realtime validation statuses and save persistence.
- [ ] Extend quick chat service with current shortcut tracking, validation, and update/re-register support.
- [ ] Add `ValidateShortcutSettings` and `SaveShortcutSettings` service methods.
- [ ] Wire `main.go` so settings service receives the quick chat shortcut controller.
- [ ] Run `go test ./backend/pkg/global_hotkey ./backend/service/settings`.

### Task 3: Frontend Store, Types, Bindings, And UI

**Files:**
- Modify: `frontend/src/types/settings.ts`
- Modify: `frontend/src/store/settingsStore.ts`
- Modify: `frontend/src/components/settings/SettingsApp.tsx`
- Modify: `frontend/src/components/settings/general/GeneralSettingsPanel.tsx`
- Create: `frontend/src/components/settings/general/ShortcutSettingsView.tsx`
- Modify or create generated binding shims under `frontend/bindings/.../backend/service/settings`
- Modify: `frontend/src/i18n/locales/zh-CN.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/__tests__/settingsStore.test.ts`
- Test: `frontend/src/__tests__/generalSettings.test.tsx`

- [ ] Write failing frontend tests for shortcut hydration, realtime conflict display, and disabled save.
- [ ] Add shortcut draft/dirty/validation state to the settings store.
- [ ] Add the "快捷键" submenu item and render the shortcut settings view.
- [ ] Implement a recorder button that captures keyboard shortcuts and calls backend validation.
- [ ] Save through `Settings.SaveShortcutSettings` and update store state on success.
- [ ] Run targeted Vitest tests.

### Task 4: Verification

**Files:**
- All modified files.

- [ ] Run backend targeted tests.
- [ ] Run frontend targeted tests.
- [ ] Run broader `go test ./...` if targeted backend tests pass.
- [ ] Inspect `git diff --stat` and confirm only scoped files changed.
