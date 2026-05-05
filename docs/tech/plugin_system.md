# 插件系统技术实现梳理

本文基于当前仓库实现整理，目标是说明 Lemon Tea 现有插件系统在后端、前端、LLM 工具链和插件示例中的真实落地方式，而不是需求稿中的理想形态。

## 1. 总览

当前插件系统是一套“宿主拉起 Node.js 子进程 + 基于 `stdin/stdout` 的轻量 RPC + 前端 iframe 视图桥”的扩展机制。

核心职责分层如下：

- `backend/pkg/plugins/types.go`
  定义 manifest、能力模型、运行时状态和 hook 载荷。
- `backend/pkg/plugins/manager.go`
  插件管理器，负责运行时安装、启停、状态持久化、RPC 调用、hook 分发、视图文档装载。
- `backend/pkg/plugins/rpc.go`
  宿主与插件进程之间的行分隔 JSON-RPC 风格通信实现。
- `backend/pkg/plugins/credentials.go`
  插件凭据存储，使用本地 AES-GCM 加密。
- `backend/service/plugins.go`
  Wails Service 暴露层，给前端提供插件管理、设置、视图、工具调用接口。
- `backend/pkg/llm_provider/tools/plugin_tool.go`
  把插件声明的 tool 包装成 LLM 可调用工具。
- `frontend/src/components/plugin/PluginViewFrame.tsx`
  插件视图承载容器，负责 iframe、消息桥和宿主 API 映射。
- `examples/email_plugin`
  当前最完整的参考实现，覆盖 manifest、tool、view、settings、hook、host credential 调用。

## 2. 数据模型

### 2.1 Manifest

插件根目录必须包含 `plugin.json`。宿主读取后会映射到 `plugins.Manifest`：

- 基础字段：`id`、`name`、`version`、`plugin_api_version`、`type`、`main`
- 可选元数据：`description`、`author`、`minHostVersion`
- 扩展声明：`permissions`、`views`、`settingsView`、`capabilities`

当前插件类型常量定义在 `backend/pkg/plugins/types.go`：

- `agent_plugin`
- `general_plugin`

### 2.2 能力模型

`Capabilities` 当前包含：

- `useTools`
- `viewTools`
- `agents`
- `views`
- `hooks`

其中：

- `useTools` 面向动作型调用
- `viewTools` 面向侧边面板/视图渲染
- `views` / `settingsView` 描述 HTML 入口
- `hooks` 目前用于 LLM 前后置钩子

### 2.3 当前实现与设计稿的差异

当前代码中的能力约束以 `validateCapabilities()` 为准：

- `agent_plugin` 不能注册 `useTools` / `viewTools`
- `general_plugin` 不能注册 `agents`

这点和 `docs/public/plugin_dev_doc.md` 里“`agent_plugin` 也可提供工具”的说法不一致。现阶段应以代码行为为准。

## 3. 宿主侧目录结构与状态持久化

`plugins.NewManager()` 启动时会通过 `utils.GetDataPath()` 计算插件相关目录，并自动创建：

- `<data>/plugin_runtime`
  插件 Node.js 运行时目录
- `<data>/plugins/installed`
  已安装插件目录
- `<data>/plugins/state/plugins.json`
  插件状态持久化文件
- `<data>/plugins/logs`
  插件日志目录

每个插件安装后会被复制到：

- `<data>/plugins/installed/<plugin_id>`

同时在插件目录下创建：

- `<installPath>/data`

它是通过环境变量 `LEMONTEA_PLUGIN_DATA_DIR` 传给插件进程的插件私有数据目录。

状态模型保存在 `PluginRecord` 中，关键字段包括：

- `Manifest`
- `InstallPath`
- `DataPath`
- `Enabled`
- `Status`
- `LastError`
- `Runtime`
- `InstalledAt`
- `LastStartedAt`
- `LastStoppedAt`
- `RestartCount`
- `RuntimeHealthy`

## 4. Node.js 运行时管理

插件并不直接依赖系统 Node，而是走宿主管理的独立运行时。

### 4.1 版本与位置

当前固定版本常量在 `backend/pkg/plugins/manager.go`：

- `pluginNodeVersion = "v22.22.2"`

宿主会在 `plugin_runtime` 下寻找 `node` 可执行文件，兼容 macOS / Linux / Windows 的多个候选路径。

### 4.2 下载与安装

`Manager.DownloadRuntime()` 负责：

1. 解析当前平台对应的下载地址
2. 下载压缩包到 `plugin_runtime`
3. 解压到临时目录
4. 将解压根目录重命名为 `plugin_runtime/node`
5. 再次执行 `node --version` 做最终可用性校验

`RuntimeStatus` 不仅表示“是否可用”，还附带下载过程状态：

- `Downloading`
- `Progress`
- `DownloadedBytes`
- `TotalBytes`
- `Phase`
- `DownloadURL`
- `Error`

因此前端可以直接据此做安装引导和进度展示。

## 5. 安装、启用、禁用、删除

### 5.1 安装

`InstallFromFolder()` 的流程是：

1. 读取 `plugin.json`
2. 执行 manifest 校验
3. 将整个目录复制到宿主安装目录
4. 创建插件私有 `data/` 目录
5. 初始化 `PluginRecord`
6. 将记录写入 `plugins.json`

安装阶段并不会启动插件进程。当前默认行为由 `backend/service/plugins.go` 的 `AddPluginFromFolder()` 负责补上一层“安装后立即启用”，因此用户从设置页新添加插件后会直接进入启用状态。

### 5.2 启用

`Enable()` 会调用 `startProcess()`：

1. 检查 Node runtime
2. 检查 `manifest.main` 指向的入口文件
3. 用 `node <main>` 启动插件子进程
4. 通过 `stdin/stdout` 建立 RPC 客户端
5. 调用插件侧 `initialize`
6. 把插件返回的运行时能力与 manifest 能力合并
7. 再次执行能力合法性校验
8. 更新状态为 `enabled`

启用成功后，插件处于常驻运行状态，记录保存在 `Manager.processes` 中。

### 5.3 自动恢复

`Service.ServiceStartup()` 初始化 `plugins.Manager` 后，会调用 `StartEnabled(ctx)`，把上次处于启用状态的插件自动拉起。

### 5.4 禁用与删除

- `Disable()` 会先 `stopProcess()`，再写回 `disabled`
- `Delete()` 会先禁用，再删除安装目录和状态记录

## 6. 插件进程模型

### 6.1 启动方式

宿主通过 `exec.CommandContext(node, mainPath)` 启动插件，注入两个关键环境变量：

- `LEMONTEA_PLUGIN_ID`
- `LEMONTEA_PLUGIN_DATA_DIR`

标准错误输出会被重定向到：

- `<data>/plugins/logs/<plugin_id>.log`

这意味着：

- 插件协议数据必须走 `stdout`
- 调试日志应优先写 `stderr`

### 6.2 常驻进程与临时进程

当前存在两类运行方式：

- 常驻进程：用于已启用插件的正常工作流
- 临时进程：当插件未常驻，但宿主需要调用 `get_settings` / `save_settings` / `test_connection` 等 RPC 时，会通过 `startTransientProcess()` 短暂拉起，结束后执行 `shutdown`

这个设计保证了“插件设置页”和“连接测试”不强依赖插件必须先启用。

## 7. RPC 通信机制

### 7.1 协议形式

`backend/pkg/plugins/rpc.go` 实现的是一种按行读取 JSON 消息的轻量 RPC：

- 每条消息一行 JSON
- 请求字段：`id`、`protocolVersion`、`method`、`params`
- 响应字段：`id`、`protocolVersion`、`result` 或 `error`

不是标准 JSON-RPC 2.0，但风格接近。

### 7.2 宿主到插件的方法

当前宿主会调用的插件方法主要包括：

- `initialize`
- `call_use_tool`
- `call_view_tool`
- `get_settings`
- `save_settings`
- `test_connection`
- `before_llm_send`
- `after_llm_send`
- `shutdown`

### 7.3 插件到宿主的方法

当前宿主只开放了凭据相关 Host RPC：

- `get_credential`
- `set_credential`
- `delete_credential`

处理入口在 `pluginHostRPCHandler()`。

## 8. 能力注册与工具接入

### 8.1 Manifest 能力与运行时能力合并

插件的最终运行时能力来自两部分：

- manifest 静态声明
- `initialize` 返回的 `capabilities`

`mergeRuntimeCapabilities()` 会把两者合并后去重。

因此插件既可以纯静态声明能力，也可以在初始化阶段动态返回能力。

### 8.2 工具别名生成

插件 tool 不直接暴露原始 `toolId` 给 LLM，而是通过 `backend/pkg/llm_provider/tools/plugin_tool.go` 包装为别名：

- `plugin_<plugin>_tool_<tool>`
- `plugin_<plugin>_view_tool_<tool>`

同时还兼容一类旧 alias 形态，避免模型猜错名字时无法命中 view tool。

### 8.3 工具聚合模型

在设置与会话工具选择层，插件以一个聚合项出现：

- `plugin:<plugin_id>`

当用户选择该聚合项时，`resolveSelectedTools()` 会把该插件下的全部 `useTools` / `viewTools` 展开注入给模型。

### 8.4 调用链

插件工具的完整调用路径是：

1. 前端或 LLM 选择插件聚合项
2. 宿主通过 `NewPluginTools()` 生成可调用工具
3. 模型调用某个 alias
4. `PluginTool.InvokableRun()` 回调到 `Manager.CallTool()`
5. 宿主将调用转成插件 RPC：
   - `call_use_tool`
   - `call_view_tool`
6. 插件返回 `content`
7. 宿主按字符串或 JSON 序列化结果返回给上层

### 8.5 工具确认

`PluginTool` 实现了 `RequireConfirmation()`，值来自 manifest 里的 `requireConfirmation`。  
这说明插件工具已经接入宿主统一的工具确认语义，至少在模型工具层面具备“高风险工具要求确认”的能力。

## 9. Hook 接入点

当前已经落地两类 hook：

- `before_llm_send`
- `after_llm_send`

### 9.1 before

在 `backend/service/chat.go` 中，宿主会在组装好 `schemaMessages` 后执行：

- `plugins.RunBeforeLLMSend()`

插件可以返回新的消息列表，宿主会用返回值覆盖原消息流。

这意味着插件已经有能力在 LLM 发送前做：

- 追加上下文
- 改写用户消息
- 注入系统化辅助信息

### 9.2 after

在 `backend/service/chat_completion_runner.go` 的任务终态逻辑中，宿主会异步触发：

- `plugins.RunAfterLLMSend()`

插件能拿到：

- `chat_uuid`
- `message_uuid`
- `finish_reason`
- `finish_error`
- `assistant_text`

适合做审计、同步、归档、二次写入等后置逻辑。

## 10. 视图系统实现

### 10.1 视图入口

插件可声明两类视图入口：

- `views`
- `settingsView`

入口 `entry` 支持：

- 本地 HTML 文件
- `http://` / `https://`
- 以 `/` 开头的宿主内路由

### 10.2 当前主路径：内嵌 HTML 文档

实际前端渲染主路径不是直接打开 URL，而是：

1. 前端调用 `Service.GetPluginViewDocument()`
2. 后端读取本地 HTML
3. 对资源做内联处理
4. 前端将文档塞入 `iframe srcDoc`

因此本地视图更像“由宿主读取并嵌入的静态页面”。

### 10.3 前端宿主桥

`PluginViewFrame.tsx` 是插件视图和宿主之间的桥：

- 负责加载 `srcDoc`
- 等待插件页发送 `lemontea-plugin-view:ready`
- 回发 `lemontea-plugin-view:init`
- 处理插件页发来的 `lemontea-plugin-view:request`
- 把请求映射为宿主 API

当前桥接的宿主方法包括：

- `get_context`
- `call_tool`
- `get_settings`
- `save_settings`
- `test_connection`
- `update_view`
- `open_view`
- `compose_message`

### 10.4 视图位置语义

当前 `HostContext.location` 只有两个位置值：

- `settings_window`
- `chat_side_panel`

同一个插件视图可以据此决定自己的行为。

### 10.5 插件前端 SDK

`examples/email_plugin/src/views/sdk.js` 展示了建议写法。它本质上就是对 `postMessage` 协议做了一层轻封装，暴露：

- `whenReady()`
- `getContext()`
- `callTool()`
- `getSettings()`
- `saveSettings()`
- `testConnection()`
- `updateView()`
- `openView()`
- `composeMessage()`

这说明当前插件前端 SDK 还没有沉淀成宿主统一依赖包，而是以示例代码形式存在。

## 11. 凭据系统

### 11.1 实现方式

插件凭据由宿主持有，不直接交给插件自己落盘。当前实现位于 `backend/pkg/plugins/credentials.go`：

- 对称密钥保存到 `credentials.key`
- 凭据内容保存到 `credentials.json.enc`
- 使用 AES-GCM 加密

### 11.2 命名空间

宿主会把凭据键标准化成：

- `<pluginID>:<scope>:<key>`

因此不同插件之间天然隔离。

### 11.3 当前能力边界

目前宿主只提供“按插件命名空间读写字符串秘密”的能力，还没有看到：

- 系统钥匙串集成
- 权限审批弹窗
- 更细粒度 secret policy

所以它更像“本地加密文件凭据库”，而不是操作系统级安全存储。

## 12. 前后端 Service 接口

`backend/service/plugins.go` 已经把插件系统暴露为完整的前端服务接口，主要包括：

- `SelectPluginFolder`
- `AddPluginFromFolder`
- `ListPlugins`
- `GetPlugin`
- `SetPluginEnabled`
- `DeletePlugin`
- `GetPluginRuntimeStatus`
- `DownloadPluginRuntime`
- `OpenPluginSettingsWindow`
- `GetPluginViewURL`
- `GetPluginViewDocument`
- `GetPluginSettings`
- `SavePluginSettings`
- `TestPluginConnection`
- `CallPluginTool`
- `CallPluginToolDirect`

前端 `frontend/src/services/pluginService.ts` 只是对这些 Wails bindings 的薄封装。

另外，插件设置页和会话页通过事件：

- `settings:plugins:changed`

来做刷新同步。

## 13. 示例插件：Email

`examples/email_plugin` 是理解当前插件系统最有价值的样例。

它展示了这些能力组合：

- `general_plugin`
- 多个 `useTools`
- 一个 `viewTool`
- 两个普通视图 + 一个设置页
- `before_llm_send` / `after_llm_send`
- 宿主凭据读写

插件主进程的 `handle()` 方法完整实现了宿主要求的 RPC 面：

- `initialize`
- `call_use_tool`
- `call_view_tool`
- `get_settings`
- `save_settings`
- `test_connection`
- `before_llm_send`
- `after_llm_send`
- `shutdown`

这基本上可以视为“当前插件协议最小可用参考实现”。

## 14. 当前实现边界与判断

结合代码，当前插件系统已经具备以下成熟能力：

- 插件安装、启停、删除和状态持久化
- 独立 Node runtime 下载与管理
- 基于子进程的插件隔离
- 静态 + 动态能力注册
- LLM 工具接入
- chat side panel / settings window 视图桥
- 宿主持有凭据
- LLM 前后置 hook

但也有几处需要明确的边界：

- `permissions` 当前主要是声明与展示元数据，未看到基于该字段的强制执行框架
- 插件沙箱仍然较弱，本质上插件是宿主拉起的本地 Node 进程
- Host RPC 目前能力面很窄，主要只有 credential
- `agent_plugin` 的真实能力边界以代码校验为准，和开发文档存在不一致
- 前端插件 SDK 还没有正式抽成共享包

## 15. 建议的阅读顺序

如果后续要继续演进插件系统，建议按下面顺序读代码：

1. `backend/pkg/plugins/types.go`
2. `backend/pkg/plugins/manager.go`
3. `backend/pkg/plugins/rpc.go`
4. `backend/service/plugins.go`
5. `backend/pkg/llm_provider/tools/plugin_tool.go`
6. `frontend/src/components/plugin/PluginViewFrame.tsx`
7. `examples/email_plugin/plugin.json`
8. `examples/email_plugin/src/main.js`
9. `examples/email_plugin/src/views/sdk.js`

这样能先理解宿主管理器，再理解模型接入，最后看插件端如何配合。
