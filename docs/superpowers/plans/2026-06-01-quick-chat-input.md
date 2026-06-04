# Quick Chat Input Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert quick chat into a compact frameless input popup with `新建 | 继续` switching and Escape close behavior.

**Architecture:** Window chrome is controlled in `DefaultQuickChat`. The React quick chat surface owns the compact input UI, finds the target session, and reuses `useChatStore.sendMessage` plus the default provider/model/tool lookup used by `ChatInput`.

**Tech Stack:** Go, Wails v3, React, Vitest, Testing Library, Zustand, Tailwind CSS.

---

### Task 1: Window Chrome

**Files:**
- Modify: `backend/pkg/window_options/window_options.go`
- Test: `backend/pkg/window_options/quick_chat_test.go`

- [ ] Add a failing Go test that expects `DefaultQuickChat` to be frameless, buttonless, compact, and `HideOnEscape`.
- [ ] Run `go test ./backend/pkg/window_options` and confirm it fails on the new expectations.
- [ ] Set `Frameless`, hidden button states, `HideOnEscape`, and compact dimensions in `DefaultQuickChat`.
- [ ] Re-run `go test ./backend/pkg/window_options` and confirm it passes.

### Task 2: Quick Input UI And Submit

**Files:**
- Modify: `frontend/src/components/quick-chat/QuickChatApp.tsx`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh-CN.ts`
- Test: `frontend/src/__tests__/quickChatApp.test.tsx`

- [ ] Add failing React tests for focused input, `New | Continue` switching, Escape close, empty Enter select, and typed Enter send.
- [ ] Run the targeted Vitest command and confirm the new tests fail.
- [ ] Replace the chooser layout with a compact input row and segmented mode switch.
- [ ] Load default provider/model and enabled user tools, then call `sendMessage` after selecting or creating the target session when the input has content.
- [ ] Re-run the targeted Vitest command and confirm it passes.

### Task 3: Regression Check

**Files:**
- Existing test suite only.

- [ ] Run `go test ./backend/pkg/window_options ./backend/service/window`.
- [ ] Run the quick chat Vitest target.
- [ ] Review `git diff` to ensure unrelated `build/linux/desktop` is untouched.
