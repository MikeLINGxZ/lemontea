# Quick Chat Shortcut Settings Design

## Goal

Add a second-level "快捷键" page under Settings -> General Settings so users can configure the quick chat shortcut. The UI must report shortcut conflicts as soon as the user records a shortcut, and saving must immediately apply the new global hotkey.

## Architecture

Persist one quick chat shortcut string in `config.json` as `quick_chat_shortcut`. Existing configs default to `CmdOrCtrl+Shift+Space`, preserving current behavior.

The backend owns shortcut parsing, normalization, conflict detection, and runtime registration. The frontend records keyboard input, sends the normalized accelerator string to the backend for validation, displays the returned status, and disables saving while the shortcut is invalid or conflicting.

## Behavior

- The General settings submenu gains a "快捷键" item.
- The page shows the current quick chat shortcut and one recorder button/input.
- Recording a shortcut updates a draft and immediately calls backend validation.
- Validation returns:
  - `available` when the shortcut is valid and can be registered, or when it matches the currently persisted quick chat shortcut.
  - `conflict` when the OS refuses temporary registration.
  - `invalid` when parsing fails or the user records an unsupported key combination.
- Save repeats validation, writes `quick_chat_shortcut`, and asks the quick chat hotkey service to re-register immediately.
- If applying the new hotkey fails, the backend restores the previous active hotkey and returns an error so the frontend keeps the draft unsaved.

## Boundaries

This change only makes the quick chat shortcut configurable. It does not add a general shortcut registry, multiple app shortcuts, shortcut import/export, or per-window keybinding editing.

## Testing

Backend tests cover default migration, shortcut parsing, conflict status, save persistence, and runtime re-registration fallback. Frontend tests cover the new submenu/page, realtime validation state, disabled save on conflicts, and store hydration/dirty state.
