# Hello SDK Plugin

Minimal Lemon Tea plugin template powered by the official SDK in `packages/plugin-sdk`.

It demonstrates:

- `definePlugin()` for a Node plugin entrypoint
- `ctx.storage.jsonStore()` for simple plugin settings
- `useTools` and `viewTools`
- the browser view SDK via `window.LemonTeaPluginView`

## Build

```text
npm run build
```

## Install

1. Open `Settings -> Plugins`.
2. Click `Add`.
3. Select `examples/hello_sdk_plugin`.
4. Enable the plugin.

## Example prompts

```text
Call Hello SDK and greet me.
```

```text
Use Hello SDK to show a hello message in the side panel.
```
