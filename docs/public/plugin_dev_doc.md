# Lemon Tea 插件开发指南

本文档面向希望为 Lemon Tea 开发插件的开发者，目标是帮助你从零开始做出一个可安装、可启用、可调用、可展示、可配置的插件。

如果你把 Lemon Tea 插件理解成“一套由宿主拉起的独立程序，并通过标准协议向宿主暴露工具和视图”，那你已经抓住了核心。

## 0. 推荐方式：优先使用官方 SDK

仓库现在提供了一个官方、仓内的插件 SDK：

- `packages/plugin-sdk/runtime.js`
- `packages/plugin-sdk/browser.js`

推荐开发路径：

1. 参考 `examples/hello_sdk_plugin` 起一个最小插件
2. 参考 `examples/email_plugin` 看完整 settings / tool / hook / view 组合
3. 只在需要理解底层协议时，再继续阅读后面的原始 RPC 章节

SDK 已经封装了这些最容易重复出错的部分：

- 插件主进程 `stdin/stdout` RPC
- `initialize` / `shutdown` / tool / settings / hook 分发
- host credential 调用
- 插件 `dataDir` JSON 存储 helper
- 插件视图 `postMessage` 桥

如果你只是要做一个业务插件，推荐直接用 SDK，不要从手写协议开始。

## 1. 什么是 Lemon Tea 插件

Lemon Tea 插件是一个带 `plugin.json` 的目录。宿主会：

1. 读取插件的 manifest
2. 启动插件入口程序
3. 通过 `stdin/stdout` 与插件通信
4. 调用插件暴露的工具、视图、设置页和钩子

插件适合做这几类扩展：

- 接入外部服务，如邮件、日历、知识库、工单系统
- 暴露工具能力，如搜索、发送、同步、创建、更新
- 提供侧边面板视图，如列表、详情、状态页
- 提供插件设置页
- 在模型请求前后执行 hook

## 2. 插件能力模型

当前最常见的插件类型有两类：

- `general_plugin`
  适合绝大多数业务插件，支持工具、视图、设置页和 hook。
- `agent_plugin`
  适合需要 Agent 能力的插件，也可以同时提供工具和视图。

一个插件可以声明这些能力：

- `useTools`
  动作型工具，供宿主或模型调用。
- `viewTools`
  展示型工具，返回右侧面板需要渲染的数据。
- `views`
  视图资源定义。
- `settingsView`
  插件设置页入口。
- `hooks`
  模型请求前后执行的钩子。
- `agents`
  Agent 相关能力声明。

对外开发时，最推荐的起步组合是：

- 一个 `general_plugin`
- 一个 `useTool`
- 一个 `viewTool`
- 一个 `settingsView`

## 3. 最小目录结构

推荐从下面这个结构开始：

```text
my_plugin/
  plugin.json
  package.json
  src/
    main.js
  dist/
    main.js
    views/
      index.html
  scripts/
    build.js
  README.md
```

关键约定：

- `plugin.json` 必须存在
- `main` 指向的文件必须可执行
- 入口程序通常使用 Node.js
- `dist/` 放构建产物，适合发布
- `src/` 放源码，适合开发

如果你希望本地开发更轻量，可以让 `dist/main.js` 只做一层薄 bootstrap，再去加载 `src/main.js`。

## 4. 快速开始

建议第一次开发按下面顺序来：

1. 创建插件目录和 `plugin.json`
2. 写一个最小 `initialize` 响应
3. 写一个最小 `useTool`
4. 写一个最小 `viewTool`
5. 在宿主中安装并启用插件
6. 确认工具能被调用、视图能被打开
7. 最后再补设置页、凭据、hook 和更复杂的业务逻辑

这个顺序的好处是能先打通协议和宿主链路，再逐步增加能力，排错成本最低。

如果你采用 SDK，这个流程可以进一步简化成：

1. 复制 `examples/hello_sdk_plugin`
2. 修改 `plugin.json`
3. 在 `src/main.js` 里用 `definePlugin()` 注册工具、设置和 hook
4. 在 `src/views/*.html` 里通过 `window.LemonTeaPluginView` 调宿主能力
5. 运行 `npm run build`
6. 在宿主中安装并启用插件

## 5. Manifest 说明

插件通过 `plugin.json` 描述自己的身份、入口和能力。一个典型示例如下：

```json
{
  "id": "com.example.demo",
  "name": "Demo Plugin",
  "version": "1.0.0",
  "plugin_api_version": 1,
  "description": "Example Lemon Tea plugin.",
  "type": "general_plugin",
  "main": "dist/main.js",
  "minHostVersion": "1.0.0",
  "author": "Example Team",
  "permissions": [
    "filesystem.plugin_data"
  ],
  "views": [
    {
      "id": "demo_list",
      "name": "Demo List",
      "entry": "dist/views/index.html"
    }
  ],
  "settingsView": {
    "id": "settings",
    "name": "Demo Settings",
    "entry": "/?entry=form_plugin_demo"
  },
  "capabilities": {
    "useTools": [
      {
        "id": "search_demo",
        "name": "Search Demo",
        "description": "Search demo data.",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {
              "type": "string",
              "description": "Search keyword."
            }
          },
          "required": ["query"]
        }
      }
    ],
    "viewTools": [
      {
        "id": "show_demo_list",
        "name": "Show Demo List",
        "description": "Render demo results in the side panel.",
        "viewId": "demo_list",
        "inputSchema": {
          "type": "object",
          "properties": {
            "title": {
              "type": "string"
            },
            "result": {
              "type": "object"
            }
          }
        }
      }
    ],
    "hooks": [
      "before_llm_send",
      "after_llm_send"
    ]
  }
}
```

`plugin_api_version` 用于声明插件所遵循的插件系统版本。当前 Lemon Tea 插件系统版本为 `1`，因此现阶段插件清单应填写 `1`。后续当宿主支持多个插件系统版本时，会根据这个字段走不同的兼容处理分支。

### 5.1 关键字段

#### `id`

- 必须全局唯一
- 推荐使用反向域名风格，如 `com.example.demo`
- 一旦发布，尽量不要变更

#### `name`

- 面向最终用户显示
- 建议简洁、可读、可识别

#### `version`

- 推荐使用语义化版本，如 `1.0.0`
- 对外发布时，任何不兼容改动都应升主版本号

#### `type`

当前常用值：

- `general_plugin`
- `agent_plugin`

#### `main`

- 插件进程入口文件
- 路径相对于插件目录
- 宿主会直接执行它

#### `permissions`

建议只声明实际需要的权限。公开发布时，权限收敛会直接影响用户信任感和审核通过率。

常见权限示例：

- `filesystem.plugin_data`
- `network.imap`
- `network.smtp`
- `credential.mail_account`
- `hook.llm.before_send`
- `hook.llm.after_send`

#### `inputSchema`

- 使用 JSON Schema 风格描述输入参数
- 宿主可据此生成参数说明和表单能力
- 应尽量写清字段含义、类型和必填项

## 6. 通信协议

如果你使用官方 SDK，本章的大部分细节都已经被封装了。保留本章主要是为了：

- 理解宿主和插件之间的真实协议
- 调试复杂问题
- 为非 Node 运行时预研自定义实现

宿主和插件之间通过标准输入输出传输 JSON 消息。你可以把它理解成“基于 `stdio` 的轻量 JSON-RPC 风格协议”。

每条消息一行 JSON。

### 6.1 宿主请求示例

```json
{
  "id": "host-1",
  "protocolVersion": "1.0",
  "method": "call_use_tool",
  "params": {
    "toolId": "search_demo",
    "args": {
      "query": "hello"
    }
  }
}
```

### 6.2 成功响应示例

```json
{
  "id": "host-1",
  "protocolVersion": "1.0",
  "result": {
    "content": {
      "ok": true,
      "items": []
    }
  }
}
```

### 6.3 失败响应示例

```json
{
  "id": "host-1",
  "protocolVersion": "1.0",
  "error": {
    "code": "PLUGIN_ERROR",
    "message": "query is required"
  }
}
```

### 6.4 建议实现的方法

宿主通常会调用这些方法：

- `initialize`
- `call_use_tool`
- `call_view_tool`
- `get_settings`
- `save_settings`
- `test_connection`
- `before_llm_send`
- `after_llm_send`
- `shutdown`

不是每个插件都需要完整实现全部业务逻辑，但建议把这些入口都显式处理掉，并返回稳定结果。

## 7. SDK 最小入口示例

现在推荐的最小写法如下：

```js
const { definePlugin } = require('../dist/sdk/runtime');

let settingsStore = null;

definePlugin({
  onInitialize(ctx) {
    settingsStore = ctx.storage.jsonStore('config.json', { displayName: 'Lemon Tea' });
  },
  useTools: {
    hello_world(args) {
      const name = String(args.name || 'world');
      return {
        ok: true,
        message: `Hello ${name}`,
      };
    },
  },
  viewTools: {
    show_hello(args) {
      return {
        viewId: 'hello_view',
        region: 'chat_side_panel',
        title: 'Hello',
        data: {
          message: String(args.message || 'Hello Lemon Tea'),
        },
      };
    },
  },
  settings: {
    get() {
      return settingsStore.read() || {};
    },
    save(config) {
      settingsStore.write(config || {});
      return settingsStore.read() || {};
    },
  },
}).start();
```

配套视图页可以直接使用浏览器 SDK：

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

完整可运行模板见：

- `examples/hello_sdk_plugin`
- `examples/email_plugin`

## 8. 原始协议最小插件入口示例

下面是一个足够小、又能跑通基础链路的 Node.js 插件示例：

```js
const readline = require('node:readline');

const protocolVersion = '1.0';
let pluginId = '';
let dataDir = '';

function write(message) {
  process.stdout.write(`${JSON.stringify(message)}\n`);
}

async function handle(method, params = {}) {
  switch (method) {
    case 'initialize':
      pluginId = params.pluginId || '';
      dataDir = params.dataDir || '';
      return {
        capabilities: {
          useTools: params.manifest?.capabilities?.useTools || [],
          viewTools: params.manifest?.capabilities?.viewTools || [],
          views: params.manifest?.views || [],
          hooks: params.manifest?.capabilities?.hooks || [],
        },
      };

    case 'call_use_tool':
      if (params.toolId === 'search_demo') {
        const query = params.args?.query || '';
        if (!query) throw new Error('query is required');
        return {
          content: {
            ok: true,
            items: [{ id: '1', title: `Result for ${query}` }],
          },
        };
      }
      throw new Error(`unknown use tool: ${params.toolId}`);

    case 'call_view_tool':
      if (params.toolId === 'show_demo_list') {
        return {
          content: {
            viewId: 'demo_list',
            region: 'chat_side_panel',
            title: 'Demo Results',
            data: {
              result: params.args?.result || { items: [] },
            },
          },
        };
      }
      throw new Error(`unknown view tool: ${params.toolId}`);

    case 'get_settings':
      return { settings: {} };

    case 'save_settings':
      return { ok: true };

    case 'test_connection':
      return { ok: true };

    case 'before_llm_send':
      return { messages: params.messages || [] };

    case 'after_llm_send':
      return { ok: true };

    case 'shutdown':
      return { ok: true };

    default:
      throw new Error(`unknown method: ${method}`);
  }
}

const rl = readline.createInterface({ input: process.stdin });
rl.on('line', async (line) => {
  if (!line.trim()) return;

  let request;
  try {
    request = JSON.parse(line);
    const result = await handle(request.method, request.params || {});
    write({
      id: request.id,
      protocolVersion,
      result,
    });
  } catch (error) {
    write({
      id: request?.id || null,
      protocolVersion,
      error: {
        code: 'PLUGIN_ERROR',
        message: error instanceof Error ? error.message : String(error),
      },
    });
  }
});
```

这个版本已经具备了：

- 初始化
- 一个工具
- 一个侧边视图
- 基础设置接口
- 基础 hook

## 8. 如何设计 `useTools`

`useTools` 适合承载真正的业务动作，比如：

- 搜索
- 发送
- 创建
- 更新
- 同步
- 批量处理

推荐约定：

- 用 `toolId` 做分发
- 用 `args` 传结构化参数
- 返回 JSON 可序列化对象
- 对外暴露的字段尽量稳定

一个简单返回示例：

```js
return {
  content: {
    ok: true,
    count: items.length,
    items,
  },
};
```

设计建议：

- 返回结构尽量扁平、清晰
- 列表数据统一使用 `items`
- 总数、游标、分页状态建议单独命名
- 不要把调试字符串当作正式协议

## 9. 如何设计 `viewTools`

`viewTools` 用于在宿主右侧面板展示内容。插件负责返回视图数据，宿主负责统一承载 tab、加载态、刷新、关闭等交互。

一个典型返回如下：

```js
return {
  content: {
    viewId: 'demo_list',
    region: 'chat_side_panel',
    title: 'Demo · Results',
    data: {
      result,
    },
  },
};
```

字段说明：

- `viewId`
  当前视图标识，应与 manifest 中的 `views[].id` 对应
- `region`
  当前推荐使用 `chat_side_panel`
- `title`
  宿主显示的标签标题
- `data`
  视图真实数据

设计建议：

- 把 `viewTool` 当作展示协议，不要把宿主 UI 状态硬编码进插件
- 同一类列表与详情尽量使用稳定的数据结构
- 分页、刷新、游标等信息应明确返回

## 10. 设置与凭据

大多数插件最终都需要配置，例如：

- API Key
- 域名
- 用户名
- 邮箱账号
- 端口
- 默认项目或空间

推荐区分两类数据：

- 普通配置
  可放在宿主分配的 `dataDir` 下
- 敏感凭据
  应通过宿主提供的凭据接口管理，不要直接明文落盘

插件通常需要处理：

- `get_settings`
- `save_settings`
- `test_connection`

建议：

- `save_settings` 只负责保存
- `test_connection` 只负责联通性检测
- 测试连接不要隐式覆盖配置

## 11. 宿主级能力调用

除了宿主调插件，插件也可以向宿主请求少量宿主级能力。最常见的是凭据读写。

常见宿主方法示例：

- `get_credential`
- `set_credential`
- `delete_credential`

推荐使用场景：

- 保存密码
- 保存 Token
- 删除失效凭据

不推荐使用场景：

- 传递大块业务数据
- 承担插件主逻辑
- 替代正常工具返回

## 12. Hook 设计建议

当前最常见的 hook 有两类：

- `before_llm_send`
- `after_llm_send`

适合用途：

- 追加轻量上下文
- 对消息做轻量清洗
- 记录日志
- 做状态同步

不适合用途：

- 长时间阻塞任务
- 重型网络处理
- 与用户交互强耦合的流程

设计原则：

- 保持轻量
- 返回结构稳定
- 允许失败后安全跳过

## 13. 错误处理建议

公开插件的体验很大程度上取决于错误是否可理解。

推荐：

```js
throw new Error('message uid is required');
```

不推荐：

```js
throw new Error('bad request');
```

建议遵循这些规则：

- 错误信息说明“缺什么、错在哪、下一步做什么”
- 业务错误与系统错误尽量区分
- 对外字段保持一致，便于宿主和前端做软处理

## 14. 分页、刷新与真实列表

如果你的插件要展示“像真实应用一样可持续浏览的列表”，建议一开始就把下面几件事设计进去：

- `items`
  当前页数据
- `hasMore`
  是否还有下一页
- `nextCursor`
  下一页游标
- `refresh`
  从最新数据重新拉取
- `loadMore`
  基于游标继续加载

不要把列表能力只做成“一次性返回 10 条结果”。如果插件面向真实业务，列表几乎一定会需要：

- 连续翻页
- 刷新最新内容
- 局部状态更新
- 详情与列表之间的联动

## 15. 调试建议

### 15.1 先跑最小链路

第一次调试只验证这三件事：

- `initialize` 成功
- `call_use_tool` 成功
- `call_view_tool` 成功

### 15.2 多返回结构化数据

推荐：

```js
return {
  content: {
    ok: true,
    input: args,
    debug: { step: 'search_done' },
  },
};
```

比起拼接字符串，结构化返回更容易被宿主消费，也更容易排查问题。

### 15.3 谨慎处理流式 SDK

如果插件依赖 IMAP、数据库流式游标、长连接 SDK 或其他流式 API，要认真阅读其并发约束。

典型问题包括：

- 在流式读取过程中再次发命令
- 一个连接上混用并发请求
- 未正确释放锁或会话

这类问题常见表现不是直接崩溃，而是“一直 loading”。

### 15.4 保留稳定日志

建议日志中至少带上：

- 方法名
- 工具 ID
- 关键参数摘要
- 外部服务响应摘要
- 错误原因

但要避免把敏感信息直接打进日志。

## 16. 安装与测试流程

开发中的常见流程：

1. 准备插件目录
2. 编写 `plugin.json`
3. 构建产物
4. 在 Lemon Tea 中添加插件目录
5. 启用插件
6. 打开设置页并完成配置
7. 调用工具进行测试
8. 打开侧边视图验证显示效果

建议每次发布前至少覆盖：

- 全新安装
- 已安装后升级
- 配置为空
- 配置错误
- 外部服务返回异常
- 分页到底
- 详情打开失败

## 17. 对外发布建议

如果这份插件要给团队外部或社区使用，建议补齐这些内容：

- `README.md`
  说明用途、安装方式、权限、配置步骤、已知限制
- 版本日志
  说明新增能力与兼容性变化
- 截图或录屏
  展示列表、详情和设置页效果
- 最小示例配置
  降低首次上手门槛
- 故障排查章节
  说明常见错误及处理方式

同时建议做到：

- manifest 字段稳定
- 输出协议稳定
- 权限尽量最小化
- 错误提示可读
- 升级时尽量保持兼容

## 18. 发布前检查清单

- `plugin.json` 字段完整
- `id` 全局唯一
- `main` 可执行
- 构建产物齐全
- `initialize` 正常
- 至少一个 `useTool` 可调用
- 至少一个 `viewTool` 可打开
- 如有设置页，设置可保存
- 敏感信息不明文落盘
- 权限声明与实际能力一致
- 错误信息可读
- 文档可独立指导安装和使用

## 19. 推荐开发路径

如果你要开发第一个正式插件，推荐按这个顺序推进：

1. 先做最小工具链路
2. 再做侧边视图
3. 再做设置页
4. 再接入凭据
5. 最后补 hook、分页、状态同步和细节体验

这个顺序最稳，因为每一步都可以单独验证，不容易把协议问题、UI 问题和业务问题混在一起。

## 20. 参考实现

如果你正在基于当前仓库开发，可以继续参考这些文件：

- 插件类型定义：
  [backend/pkg/plugins/types.go](/Users/linhuafeng/Work/lemon_tea_desktop/backend/pkg/plugins/types.go)
- 插件服务入口：
  [backend/service/plugins.go](/Users/linhuafeng/Work/lemon_tea_desktop/backend/service/plugins.go)
- RPC 通道实现：
  [backend/pkg/plugins/rpc.go](/Users/linhuafeng/Work/lemon_tea_desktop/backend/pkg/plugins/rpc.go)
- 示例插件 manifest：
  [examples/email_plugin/plugin.json](/Users/linhuafeng/Work/lemon_tea_desktop/examples/email_plugin/plugin.json)
- 示例插件实现：
  [examples/email_plugin/src/main.js](/Users/linhuafeng/Work/lemon_tea_desktop/examples/email_plugin/src/main.js)

如果你只是单独阅读这份文档，也可以完全按本文档给出的最小示例直接开始。

## 21. 一句话总结

开发 Lemon Tea 插件时，可以把它理解成：

- 一个带 manifest 的独立程序
- 一组由宿主调度的工具和视图
- 一套基于 `stdio` 的结构化通信协议
- 一种由插件返回数据、由宿主统一承载体验的扩展方式

先把最小链路跑通，再逐步把它做成真正可发布的产品级插件。
