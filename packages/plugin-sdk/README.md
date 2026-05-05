# Lemon Tea Plugin SDK

官方仓内插件 SDK，目标是简化 Lemon Tea Node 插件开发。

当前提供两部分：

- `runtime.js`
  Node 插件主进程 SDK，封装 `stdin/stdout` RPC、host credential 调用、settings/hook/tool 分发、插件数据目录 helper。
- `browser.js`
  插件视图 SDK，封装 `postMessage`、`ready/init` 握手和 view request API。

## 主进程示例

```js
const { definePlugin } = require('./runtime');

definePlugin({
  useTools: {
    hello_world(args) {
      return {
        ok: true,
        message: `Hello ${String(args.name || 'world')}`,
      };
    },
  },
  settings: {
    get(ctx) {
      return ctx.storage.jsonStore('config.json', {}).read() || {};
    },
  },
}).start();
```

## 视图示例

```html
<script src="./sdk.js"></script>
<script>
  const sdk = window.LemonTeaPluginView;
  sdk.whenReady().then(async () => {
    const context = await sdk.getContext();
    console.log(context);
  });
</script>
```
