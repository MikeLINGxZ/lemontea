# Quick Chat Inline Answer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep quick chat answers inside the quick chat popup instead of jumping to the main window.

**Architecture:** `QuickChatApp` owns an active quick session id and submits through the existing chat store. `ChatMessages` gains a `quick` variant for compact rendering. The quick window expands via Wails runtime `Window.SetSize` after the first prompt.

**Tech Stack:** React, Zustand, Wails runtime, Vitest, Go window options tests.

---

### Task 1: Lock submit behavior

**Files:**
- Modify: `frontend/src/__tests__/quickChatApp.test.tsx`

- [ ] Add tests proving typed Enter sends but does not call `Window.SelectQuickChat`.
- [ ] Add tests proving empty Enter does not create/select a session.
- [ ] Add tests proving the quick chat renders a local message region after submit.
- [ ] Run `npx vitest run src/__tests__/quickChatApp.test.tsx` and confirm failures.

### Task 2: Inline answer implementation

**Files:**
- Modify: `frontend/src/components/quick-chat/QuickChatApp.tsx`
- Modify: `frontend/src/components/chat/ChatMessages.tsx`

- [ ] Add `variant?: 'default' | 'quick'` to `ChatMessages`.
- [ ] Render compact spacing and no welcome screen for the quick variant.
- [ ] Track `activeSessionId` in `QuickChatApp`.
- [ ] Remove `Window.SelectQuickChat` from submit.
- [ ] Expand the Wails window with `Window.SetSize(720, 520)` once quick chat has an active session.
- [ ] Render `<ChatMessages sessionId={activeSessionId} variant="quick" />` above the input row.

### Task 3: Verify

**Files:**
- Existing tests only.

- [ ] Run `npx vitest run src/__tests__/quickChatApp.test.tsx`.
- [ ] Run `npm run build:dev`.
- [ ] Run `go test ./backend/pkg/window_options`.
