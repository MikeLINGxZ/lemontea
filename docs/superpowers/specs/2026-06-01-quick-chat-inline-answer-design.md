# Quick Chat Inline Answer Design

## Goal

Quick chat should answer in place. Submitting a prompt from the quick chat window must not select a session in the main window or focus the main window.

## Behavior

- The quick chat window starts as the compact input bar.
- Pressing `Enter` with an empty input does nothing.
- Pressing `Enter` with text resolves the target session:
  - `继续` uses the latest user conversation.
  - `新建` creates a new session.
- The prompt is sent through the existing `useChatStore.sendMessage` path.
- The quick chat window expands to an inline conversation panel and renders the answer in place.
- The result area reuses the full chat message renderer with a quick-chat-specific compact layout.
- The input remains at the bottom for follow-up questions in the same quick chat session.
- `Esc` closes only the quick chat window.

## Implementation Notes

- Do not call `Window.SelectQuickChat` after submit.
- Add a `variant="quick"` option to `ChatMessages` for tighter spacing and no welcome screen.
- Use Wails runtime `Window.SetSize(720, 520)` when the first submitted session becomes active.
- Keep the existing draggable/no-drag styling on the quick chat shell and controls.
