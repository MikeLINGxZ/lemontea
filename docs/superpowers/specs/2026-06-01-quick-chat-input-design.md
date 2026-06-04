# Quick Chat Input Design

## Goal

Refine quick chat from a chooser window into a compact frameless input popup. The popup opens from the existing global shortcut, lets the user choose `新建 | 继续`, accepts a message immediately, closes with `Esc`, and hands the selected session back to the main window.

## Behavior

- The quick chat window is frameless and has no native title bar controls.
- The window is small and input-focused, with the message field focused on open.
- The left side of the input contains a segmented mode switch rendered as `新建 | 继续`.
- `继续` targets the latest user conversation when one exists.
- `新建` creates a fresh conversation before sending.
- Pressing `Enter` with a non-empty message sends that message using the same default provider/model/tool settings as the normal chat input, selects the target session in the main window, and closes the popup.
- Pressing `Enter` with an empty message only opens/selects the target session.
- Pressing `Esc` closes the popup.
- If no history exists, `继续` is disabled and the mode defaults to `新建`.

## Architecture

- Keep the existing `Window.SelectQuickChat` bridge and extend its payload with an optional initial message only if needed by the main window event path.
- Prefer sending from the quick chat frontend through `useChatStore.sendMessage` so the normal optimistic update, streaming, title generation, and error handling path remains shared.
- Load default provider/model and enabled user tools the same way `ChatInput` does, but only enough for quick submit.
- Keep window-level behavior in `backend/pkg/window_options/DefaultQuickChat`.

## Tests

- Go unit tests assert the quick chat window is frameless, hides title bar controls, uses compact input dimensions, and handles Escape at the window level.
- React tests assert the input is focused, the segmented `新建 | 继续` controls are present, Escape closes, empty Enter opens/selects, and non-empty Enter sends through the chat store.
