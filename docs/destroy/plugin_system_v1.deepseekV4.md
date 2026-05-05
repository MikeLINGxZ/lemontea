# 插件系统 V1 需求文档

## 1. 背景与目标

Lemon Tea Desktop 需要提供一套基于 Node.js 的插件系统，使第三方能力可以以独立进程接入主程序，并在不影响主程序稳定性的前提下扩展聊天、工具调用、Agent、View 渲染、后台任务和通知能力。

V1 的目标是建立可运行、可管理、可隔离、可演进的插件基础设施：

- 主程序内置 Node.js 运行环境，插件统一运行在该环境之上。
- 插件以独立进程运行，插件崩溃、超时、卡死不得导致主程序崩溃。
- 插件能力通过 Manifest 声明和运行时注册协议暴露给主程序。
- 插件分为 `agent_plugin` 和 `general_plugin` 两类，二者能力边界清晰。
- 插件可提供 View、Tool（use_tool / view_tool）、Agent、LLM Hook、后台任务、通知等能力。
- 设置页提供插件管理入口，聊天输入框展示启用插件提供的能力。

---

## 2. 术语定义

| 术语 | 含义 |
| --- | --- |
| Host | Lemon Tea Desktop 主程序，负责插件发现、安装、启动、通信、权限控制和 UI 集成。 |
| Plugin | 运行在 Node.js 环境中的插件包。 |
| Plugin Runtime | 主程序内置的 Node.js 运行时及插件执行所需基础文件，位于 `<GetDataPath()>/plugin_runtime`。 |
| Plugin Process | 每个插件独立启动的 Node.js 子进程。 |
| Manifest | 插件声明文件 `plugin.json`，描述插件元信息、类型、入口、权限和静态能力。 |
| Capability | 插件向 Host 暴露的能力，包括 Tool、View、Agent、Hook、后台任务等。 |
| use_tool | 与现有大模型工具完全一致的调用形态，被 LLM 直接调用并返回结构化结果。 |
| view_tool | 被 LLM 调用后可在聊天界面展示插件 View 的工具类型。 |
| View | 插件提供的可渲染 UI 界面，可用于插件设置页、聊天侧边栏或工具结果展示。 |
| A2A | Agent-to-Agent 调用协议。`agent_plugin` 内部 Agent 通过该协议与主程序 Agent 系统协作。 |
| agent_plugin | 仅注册 Agent 和 View 的插件类型，不暴露 Tool。 |
| general_plugin | 仅注册 Tool（use_tool / view_tool）和 View 的插件类型，不注册 Agent。 |

---

## 3. 插件类型

插件必须在 Manifest 中声明唯一插件类型。V1 支持两类插件：`agent_plugin` 与 `general_plugin`。

### 3.1 agent_plugin

`agent_plugin` 用于扩展 Agent 能力。

定义：
- 仅允许注册 Agent 和 View。
- 不允许向主程序暴露普通 Tool（包括 `use_tool` 和 `view_tool`），仅暴露 Agent。
- 使用 A2A 协议与主程序交互。
- 可提供插件设置 View。

插件内部 Agent 能力范围：
- 可以使用插件自身携带的工具。
- 可以使用主程序允许开放的系统内置工具。
- 插件内部工具不暴露到主程序工具选择。

数据面展示要求：
- 主程序对外只展示该插件注册的 Agent（在聊天输入框工具选择中以插件名称作为聚合项、标记"插件"标签出现）。
- 在聊天详情页中，插件 Agent 执行任务时仅显示"该 Agent 正在执行"及最终结果或错误，不显示类似内置 Agent 的详细过程追踪（如 classify、plan、dispatch、agent_run、tool_call、synthesize、review 等步骤）。
- 插件 Agent 的内部推理、内部工具调用、内部多 Agent 过程不进入主程序现有执行追踪。

### 3.2 general_plugin

`general_plugin` 用于扩展通用工具和界面能力。

定义：
- 仅允许注册 Tool 和 View。
- 不允许注册 Agent。

注册能力：
- 可注册 `use_tool`，用于让 LLM 获取数据或执行动作。
- 可注册 `view_tool`，用于根据上下文或 `use_tool` 结果渲染界面。
- 可提供插件设置 View。

---

## 4. 目录与运行时

### 4.1 Node.js 运行时目录

主程序需要内置 Node.js 环境。运行时目录位于 `backend/utils/dir.go:8` 中 `GetDataPath()` 返回的目录下的 `plugin_runtime` 子目录：

```text
<GetDataPath()>/plugin_runtime
```

当前 `GetDataPath()` 默认返回 `~/.lemon_tea`，因此默认运行时目录为 `~/.lemon_tea/plugin_runtime`。

要求：
- 首次启动或插件系统首次初始化时，主程序需要确保 `plugin_runtime` 目录存在。
- `plugin_runtime` 下应包含当前平台可执行的 Node.js 运行环境。
- 主程序应能校验 Node.js 运行时是否存在、是否可执行、版本是否满足插件系统最低要求。
- Node.js 运行时缺失或损坏时，插件系统应进入不可用状态，并在设置页展示明确错误，不影响主程序其他功能。

### 4.2 插件安装目录结构

V1 建议使用如下目录结构：

```text
<GetDataPath()>/plugins
  installed/
    <plugin_id>/
      plugin.json        (Manifest 文件)
      package.json        (Node.js 依赖声明)
      dist/               (插件构建产物，含 main.js 入口)
      assets/             (静态资源)
      data/               (插件私有数据目录)
  logs/
    <plugin_id>.log       (插件运行日志)
  state/
    plugins.json           (插件持久化状态)
```

要求：
- `installed/<plugin_id>` 保存插件包完整内容。
- `data/` 为插件私有数据目录，插件只能默认访问自己的目录。
- `logs/` 保存插件运行日志，单个文件需要限制大小并滚动。
- `state/plugins.json` 或等价持久化存储保存插件安装、启用、禁用、版本和权限状态。
- 插件访问自身数据目录之外的目录必须经过 Host 权限控制。

---

## 5. 插件生命周期

### 5.1 安装

用户在插件设置页点击"添加"按钮后，主程序应允许选择插件包或插件目录。

安装流程要求：
1. 校验 Manifest 文件是否存在且 JSON 格式合法。
2. 校验必填字段完整性：`id`、`name`、`version`、`type`、`main`、`minHostVersion`。
3. 校验插件 ID 全局唯一、插件类型已知且合法。
4. 校验插件类型与声明能力匹配：
   - `agent_plugin` 不得声明 Tool 能力。
   - `general_plugin` 不得声明 Agent 能力。
5. 校验入口文件存在性。
6. 为插件创建私有 `data/` 目录和日志文件。
7. 安装成功后插件默认处于禁用状态。

以下 Manifest 必须被拒绝并展示原因：
- 缺少必填字段。
- 插件类型未知或非法。
- `agent_plugin` 声明普通 Tool。
- `general_plugin` 声明 Agent。
- 入口文件不存在。
- 插件 ID 与已安装插件冲突且非升级流程。

### 5.2 启用

用户启用插件时：
1. Host 启动该插件独立进程。
2. Host 与插件建立通信通道并完成握手（默认超时 15 秒）。
3. Host 拉取或接收插件的运行时能力注册。
4. Host 将插件能力合并到：
   - 工具选择列表（聊天输入框）
   - Agent 注册表
   - Hook 链（before_llm_send / after_llm_send）
   - View 注册表
5. 启用状态持久化到 `state/plugins.json`。

启用失败处理：
- 插件状态应保持禁用或标记为启动失败。
- 设置页展示失败原因。
- 不应影响其他启用插件和主程序核心功能。

### 5.3 禁用

用户禁用插件时：
1. Host 从工具选择、Agent 注册、Hook 链、View 注册表中移除该插件全部能力。
2. Host 通知插件执行关闭流程。
3. Host 在超时（默认 5 秒）后强制终止插件进程。
4. 禁用状态持久化。
5. 正在执行的插件 Tool、Agent、后台任务应被取消，并向调用方返回明确的取消错误。

### 5.4 删除

用户删除插件时：
1. 若插件正在启用，必须先执行禁用流程（停止进程、移除能力、取消进行中的任务）。
2. 删除插件安装目录 `installed/<plugin_id>/`。
3. 删除插件能力注册记录。
4. 删除插件状态记录。
5. 插件私有数据（`data/`）默认随插件删除。V1 默认删除，不保留。

### 5.5 异常恢复

要求：
- 插件进程崩溃时，Host 应标记插件异常并清理已注册运行时能力。
- V1 支持有限次数自动重启：默认最多 1 次；再次失败则保持异常状态，需要用户手动处理。
- 插件卡死或 RPC 超时不得阻塞主程序 UI 和聊天流程。
- 插件异常需要写入日志，并在设置页插件详情中展示最近错误信息。
- 异常期间主程序聊天、设置、模型调用等核心能力需保持可用。

---

## 6. Manifest 规范

插件根目录必须包含 `plugin.json`。

### 6.1 Manifest 结构

```json
{
  "id": "com.example.mail",
  "name": "Mail",
  "version": "1.0.0",
  "description": "邮件插件，提供邮件搜索、阅读和管理能力",
  "type": "general_plugin",
  "main": "dist/main.js",
  "minHostVersion": "1.0.0",
  "author": "Example",
  "permissions": [
    "model.call",
    "notification.send",
    "background.task"
  ],
  "views": [
    {
      "id": "mail_list",
      "name": "邮件列表",
      "entry": "dist/views/mail-list.html"
    },
    {
      "id": "mail_detail",
      "name": "邮件详情",
      "entry": "dist/views/mail-detail.html"
    }
  ],
  "settingsView": {
    "id": "settings",
    "entry": "dist/views/settings.html"
  }
}
```

### 6.2 字段定义

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 全局唯一标识，建议反向域名格式；安装后不可变更。 |
| `name` | string | 是 | 插件展示名称，用于设置页列表、工具选择聚合项和标签。 |
| `version` | string | 是 | 语义化版本号。 |
| `description` | string | 是 | 插件描述文本。 |
| `type` | string | 是 | 必须为 `"agent_plugin"` 或 `"general_plugin"`。 |
| `main` | string | 是 | Node.js 入口文件路径，相对于插件根目录。 |
| `minHostVersion` | string | 是 | 插件要求的最低 Host 版本。 |
| `author` | string | 否 | 插件作者名称。 |
| `permissions` | string[] | 否 | 插件申请的 Host API 权限列表。 |
| `views` | object[] | 否 | 插件静态声明的 View 列表，允许为空数组。 |
| `settingsView` | object | 否 | 插件设置页面声明；若不存在，设置页中该插件的详情区域不显示"设置"按钮。 |

### 6.3 View 对象结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | View 唯一标识，在插件内唯一。Host 内部组合为 `plugin:<plugin_id>:view:<view_id>`。 |
| `name` | string | View 显示名称。 |
| `entry` | string | View HTML 入口文件路径，相对于插件根目录。 |

---

## 7. 进程模型与隔离

### 7.1 独立进程

核心原则：
- 每个启用插件运行在独立 Node.js 子进程中。
- 不同插件不得共享同一个进程，确保插件间完全隔离。
- 插件进程由 Host 统一管理生命周期（创建、启动、监控、停止、强制终止）。
- 插件进程退出、崩溃、卡死时，Host 必须隔离影响并释放资源，不能影响主程序。

### 7.2 通信协议

Host 与插件之间采用 JSON-RPC 风格协议。

协议要求：
- 每个请求包含 `id`、`method`、`params`、`protocolVersion`。
- 每个响应包含 `id`、`result` 或 `error`。
- 错误对象包含 `code`、`message`、`data`。
- 支持请求超时机制。
- 支持取消正在执行的请求。
- 支持 Host 主动推送事件到插件。
- 支持插件主动调用 Host API。

请求示例：
```json
{
  "id": "req_001",
  "protocolVersion": "1.0",
  "method": "register_use_tool",
  "params": {
    "id": "search_mail",
    "name": "搜索邮件",
    "description": "根据关键词和日期范围搜索邮件",
    "inputSchema": {
      "type": "object",
      "properties": {
        "query": { "type": "string", "description": "搜索关键词" },
        "since": { "type": "string", "description": "起始日期" }
      },
      "required": ["query"]
    }
  }
}
```

### 7.3 超时限制

| 操作 | 默认超时 | 说明 |
| --- | --- | --- |
| 插件启动握手 | 15 秒 | 超时视为启动失败 |
| 插件正常关闭等待 | 5 秒 | 超时强制终止进程 |
| 单次 Tool 调用 | 2 分钟 | `use_tool` / `view_tool` 执行上限 |
| LLM Hook 调用 | 10 秒 | `before_llm_send` / `after_llm_send` |
| Agent 任务执行 | 5 分钟 | 插件 Agent 单次任务最大时长 |
| RPC 请求 | 30 秒 | 通用 RPC 请求默认超时 |

---

## 8. Host API 能力

插件只能调用已授权的 Host API。Host API 必须校验插件身份、插件状态、权限和请求参数。

### 8.1 LLM 消息发送前 Hook（`before_llm_send`）

触发时机：
- 用户消息组装完成后。
- 实际调用 LLM 模型之前。

能力接口：
- 接收当前聊天上下文摘要（对话历史摘要、当前用户消息）。
- 接收即将发送给模型的消息列表。
- 可返回修改后的消息、附加系统提示或元数据。

要求：
- 多个插件的 `before_llm_send` Hook 按启用顺序或明确优先级顺序依次执行。
- 单个 Hook 超时或失败时，不应阻断主流程；默认记录错误并跳过该 Hook，继续后续 Hook 和 LLM 调用。
- V1 默认不允许插件阻断消息发送。
- 需要权限 `hook.llm.before_send`。

### 8.2 LLM 消息发送后 Hook（`after_llm_send`）

触发时机：
- LLM 模型响应完成后（含流式结束）。
- 助手消息持久化后。

能力接口：
- 接收用户消息、模型响应内容、工具调用结果摘要和任务状态。
- 可返回附加元数据、通知请求或后台任务请求。

要求：
- Hook 执行失败不得影响用户看到模型响应。
- Hook 执行结果需要记录到日志或可观测事件中，便于排查。
- 需要权限 `hook.llm.after_send`。

### 8.3 插件调用模型（`call_model`）

能力：
- 插件可请求 Host 使用当前已配置模型或指定模型进行一次 LLM 调用。
- Host 负责复用现有模型供应商配置、密钥管理和调用链。

要求：
- 插件不得直接读取模型供应商密钥，所有模型调用必须通过 Host 代理流转。
- 调用必须受权限 `model.call` 控制。
- 调用纳入主程序的速率限制、并发控制、取消和错误处理策略。
- Host 应记录插件发起的模型调用来源，便于追踪和审计。

### 8.4 后台任务（`register_background_task` / `schedule_task`）

能力：
- 插件可注册后台任务。
- 后台任务可按执行策略分类：一次性延迟执行、固定间隔、事件触发。

要求：
- 后台任务必须绑定插件 ID。
- 插件禁用或删除后，其所有后台任务必须停止并清理。
- 后台任务执行失败应记录日志，不弹出无关错误打扰用户。
- 需要限制并发数，避免插件后台任务占满资源。
- 需要权限 `background.task`。

### 8.5 通知程序（`notify`）

能力：
- 插件可请求 Host 展示系统通知。
- 通知可包含：标题、正文、通知等级（info / warning / error）、操作按钮和相关联的 View ID。

要求：
- 调用必须受权限 `notification.send` 控制。
- Host 需要限制通知频率，避免插件刷屏（例如每分钟最多 5 条）。
- 通知内容需要标记来源插件名称。
- 用户点击通知时可触发关联 View 或跳转。

---

## 9. Tool、View 与 Agent 能力设计

### 9.1 use_tool

`use_tool` 与现有大模型工具完全一致，用于被 LLM 直接调用并返回结构化结果。

定义要求：
- 必须包含 `id`、`name`、`description`、`inputSchema`。
- 返回结果必须可序列化（JSON）。
- Tool ID 在 Host 内部使用命名空间避免冲突：`plugin:<plugin_id>:tool:<tool_id>`。
- 接入现有工具调用追踪和审批体系：
  - Tool 如果声明需要用户确认，则调用前进入审批流程。
  - Tool 调用记录写入追踪日志。
  - Tool 调用结果在消息中展示（如有必要，使用内联 Tool Card）。
- Tool 运行失败时，返回结构化错误，不将插件进程异常直接泄漏到模型上下文。

### 9.2 view_tool

`view_tool` 是一种特殊工具类型，被 LLM 调用后可以在聊天界面展示插件提供的 View。

典型工作流（邮件插件示例）：

```
用户: "帮我看看最新的邮件列表"
  1. LLM 发现邮件插件注册了 use_tool: "搜索邮件", view_tool: "显示邮件列表"
  2. LLM 调用 use_tool "搜索邮件" 获取邮件列表数据
  3. LLM 发现可以使用 view_tool "显示邮件列表" 渲染结果
  4. LLM 调用 view_tool，传入步骤 2 获取的邮件列表数据
  5. 聊天界面右侧面板展示插件提供的邮件列表 View
```

定义要求：
- `view_tool` 必须绑定一个已注册的 View（在 Manifest 中声明）。
- `view_tool` 输入可包含数据本体或 Host 可解析的数据引用。
- `view_tool` 返回值应包含：View ID、展示区域（V1 至少支持聊天详情页右侧面板）、初始数据和可选标题。
- View 渲染失败时，聊天区展示可理解的错误状态。
- Tool ID 使用命名空间 `plugin:<plugin_id>:view_tool:<tool_id>`。

### 9.3 View

View 是插件提供的 UI 页面或 UI 片段，通过 HTML 入口文件加载渲染。

要求：
- View 与 Host 通信必须走受控桥接 API，不允许直接访问 Host 内部对象。
- View 标题或边界区域需要标记来源插件名称。
- View 关闭时，Host 需要通知插件释放相关资源。
- 支持的展示位置：
  - **聊天侧边栏**：View 作为 `view_tool` 调用结果，展示在聊天详情页右侧面板。
  - **独立窗口**：插件设置 View，在新建窗口中打开。

### 9.4 Agent

插件 Agent 仅由 `agent_plugin` 类型注册。

定义要求：
- Agent 必须包含 `id`、`name`、`description`、能力说明和输入约束。
- Agent ID 在 Host 内部使用命名空间避免冲突：`plugin:<plugin_id>:agent:<agent_id>`。
- Agent 调用使用 A2A 协议。

Agent 能力范围：
- 插件 Agent 可以使用插件自身的内部工具（不暴露到主程序）。
- 插件 Agent 可以使用 Host 明确开放的系统内置工具（需要权限 `host.tool.builtin`）。

聊天界面展示要求：
- 插件 Agent 执行任务时，聊天详情页仅显示：
  - "Agent `<agent_name>` 正在执行"（执行中状态）
  - 执行完成的最终结果或错误信息
- 不显示内置 Agent 的详细过程追踪（如 classify、plan、dispatch、agent_run、tool_call、synthesize、review、retry 等 TraceNode 步骤）。
- 插件 Agent 内部推理、内部工具调用、内部多 Agent 协作过程不进入主程序现有 ExecutionTracePanel。
- 执行完成后，结果以普通助手消息形式展示。

---

## 10. 设置页需求

### 10.1 入口

设置页面需要新增"插件"设置页，样式和布局参考现有设置页面（`frontend/src/pages/settings/index.tsx`）。

要求：
- 设置菜单新增"插件"入口项（在现有菜单项中增加 `plugin` key）。
- 桌面端采用左右布局（参考现有 `skills`、`agents`、`provider` 等设置页的 `desktopLayout` + `listColumn` / `editorColumn` 模式）。
- 移动端按现有设置页模式适配：列表与详情切换（参考 `showEditorOnMobile` 模式）。
- 多窗口支持：添加插件使用独立窗口（参考 `OpenAddAgentWindow` 模式）。

### 10.2 插件列表（左侧栏）

左侧为插件列表 Card。

展示要求：
- 右上角显示"添加"按钮（`PlusOutlined` 图标，参考 SkillSettingsPage 的添加按钮样式）。
- 列表项展示：插件名称、插件类型标签（`agent_plugin` / `general_plugin`）、版本号、启用/禁用状态指示。
- 支持点击选中插件项，选中后在右侧展示详情。
- 插件异常时（崩溃、启动失败），列表项需要有错误状态提示（如红色状态点或警告图标）。
- 列表项头像或图标可使用插件默认图标或首字母头像。

### 10.3 插件详情（右侧栏）

右侧为插件详情 Card。

内容展示要求：
- 插件基本信息：
  - 插件名称（Title 级别展示）
  - 插件 ID
  - 版本号
  - 插件类型（类型标签）
  - 描述文本
  - 作者信息
- 当前状态：
  - 已启用（绿色标签）
  - 已禁用（灰色标签）
  - 启动失败 / 运行异常（红色标签，附错误原因）
- 已注册能力列表（启用状态下展示）：
  - Tool 列表（`use_tool` 名称和 `view_tool` 名称，区分显示）
  - View 列表
  - Agent 列表（仅 `agent_plugin`）
  - Hook 注册信息
- 权限列表（以 Tag 列表展示）
- 最近错误摘要或日志片段（异常时展示）

操作按钮区域要求：
- 右上角操作区包含以下按钮（Ant Design `Space` 组件，参考 SkillSettingsPage 的 `editorActions` 区域）：
  1. **启用 / 禁用按钮**：切换插件启用状态
     - 当前已启用 → 显示"禁用"按钮
     - 当前已禁用 → 显示"启用"按钮（Primary 风格）
     - 操作中显示 loading 状态
  2. **设置按钮**：仅当插件 Manifest 中声明了 `settingsView` 时显示
     - 点击在新建窗口中打开插件设置 View
  3. **删除按钮**：始终显示，danger 风格
     - 点击弹出二次确认 Modal
     - 确认后执行删除流程

### 10.4 插件设置窗口

点击插件详情中的"设置"按钮时：
- Host 创建新窗口。
- 新窗口加载插件提供的设置 View 的 HTML 入口文件。
- 窗口标题包含插件名称（例如 `{plugin_name} - 设置`）。
- 设置 View 关闭时释放关联资源和通信通道。
- 插件未声明 `settingsView` 时，详情区域不显示"设置"按钮。

---

## 11. 聊天输入框工具选择需求

### 11.1 插件能力聚合展示

当插件启用时，其提供的 Tool 或 Agent 需要合并到聊天输入框的工具选择中。

展示要求：
- 每个启用插件在工具选择下拉菜单中展示为**一个**聚合菜单项（而非分散的多个单独工具项）。
- 聚合项使用以下组合标识：
  - **名称为插件名称**（取自 Manifest 的 `name` 字段）。
  - **标签标记为"插件"**（使用 Ant Design `Tag` 组件，蓝色或紫色风格）。
- 聚合项的图标或视觉标识需要区分插件项与内置工具项。

### 11.2 悬停内容

鼠标悬停聚合项时，右侧弹出悬浮面板展示该插件提供的详细能力。

对 `general_plugin`：
- 展示注册的 `use_tool` 列表（名称 + 描述）。
- 展示注册的 `view_tool` 列表（名称 + 描述 + 关联 View）。
- 不展示 Agent 相关内容。

对 `agent_plugin`：
- 展示注册的 Agent 列表（名称 + 描述）。
- 不展示插件内部工具（内部工具不暴露到主程序）。

### 11.3 选择行为

- 选择（勾选）插件聚合项 → 表示当前聊天启用该插件暴露给 LLM 的全部公开能力。
- 取消选择插件聚合项 → 表示当前聊天不使用该插件公开能力。
- 全局禁用插件后，该插件聚合项从工具选择中移除。

### 11.4 兼容性要求

- 内置工具（CurrentDate、CurrentTime、Block、FileTool、ShellTool、LoadSkillTool）保持现有展示和选择逻辑不变。
- MCP 工具保持现有展示和选择逻辑不变。
- 插件聚合项使用稳定 ID：`plugin:<plugin_id>`，与现有 `selectedToolIds` 模型兼容。
- 插件聚合项与现有内置工具、MCP 工具在列表中分开区域展示（或使用分隔线），确保用户能清晰区分。

---

## 12. 安全与权限模型

### 12.1 权限列表

V1 权限定义：

| 权限 | 含义 | 风险等级 |
| --- | --- | --- |
| `model.call` | 允许插件通过 Host 调用 LLM 模型。 | 中 |
| `notification.send` | 允许插件发送系统通知。 | 低 |
| `background.task` | 允许插件注册后台任务。 | 中 |
| `filesystem.plugin_data` | 允许插件读写自身的 `data/` 目录。 | 低 |
| `host.tool.builtin` | 允许 `agent_plugin` 的 Agent 使用 Host 开放的内置工具。 | 高 |
| `hook.llm.before_send` | 允许注册发送前 Hook，可读取和修改即将发送的消息。 | 高 |
| `hook.llm.after_send` | 允许注册发送后 Hook，可读取模型响应。 | 中 |

### 12.2 安全要求

- 插件只能使用 Manifest 中声明且用户安装时确认的权限。
- 敏感权限（`host.tool.builtin`、`hook.llm.before_send`）在启用插件时需要向用户明确展示并获取确认。
- 插件不得读取主程序密钥、供应商 API Key、其他插件的数据。
- Host API 必须基于插件 ID 进行鉴权和隔离。
- 插件 View 不得直接执行 Host 内部未授权操作，必须通过受控桥接 API 通信。
- 插件运行环境需要沙箱化，限制文件系统访问范围（仅限于自身 `data/` 目录）。

---

## 13. 稳定性、日志与错误处理

### 13.1 稳定性要求

- 插件任何异常不得导致主程序崩溃。
- 插件 RPC 调用必须有超时。
- 插件启动、停止、Tool 调用、Hook 调用、Agent 调用、后台任务执行都需要可取消。
- 插件系统不可用时（Node.js 运行时缺失等），主程序聊天、设置、模型调用等核心能力仍需可用。
- 多个插件同时运行时，单个插件异常不影响其他插件。

### 13.2 日志要求

- 每个插件独立日志文件，位于 `<GetDataPath()>/plugins/logs/<plugin_id>.log`。
- 每个日志文件限制大小（如 10 MB），支持滚动保存最近 3 个文件。
- Host 记录插件生命周期事件（安装、启用、禁用、删除、崩溃、重启）。
- Tool、Agent、Hook、后台任务失败时记录插件 ID、能力 ID、错误码和错误信息。
- 设置页插件详情中展示最近错误摘要。

### 13.3 错误处理策略

| 错误类型 | 处理策略 |
| --- | --- |
| 插件进程崩溃 | 标记异常，清理运行时能力，自动重启（最多 1 次） |
| Tool 调用超时 | 向 LLM 返回超时错误摘要 |
| Tool 调用异常 | 记录日志，向 LLM 返回结构化错误（不泄漏插件进程内部异常） |
| Hook 超时/失败 | 记录日志，跳过该 Hook，继续后续流程 |
| Agent 任务失败 | 在聊天详情页展示"Agent 执行失败"及错误摘要 |
| 后台任务失败 | 记录日志，不打扰用户，可重试 |
| 通知发送失败 | 无声降级，记录日志 |

---

## 14. 架构集成点

### 14.1 后端集成

以下后端模块需要支持插件能力集成：

| 模块 | 文件 | 集成内容 |
| --- | --- | --- |
| 工具注册表 | `backend/pkg/llm_provider/tools/common.go` | 支持注册/注销插件 Tool |
| Agent 注册表 | `backend/pkg/llm_provider/agents/agent_registry.go` | 支持注册/注销插件 Agent |
| 完成流程 | `backend/service/chat_completion_runner.go` | 集成 `before_llm_send` / `after_llm_send` Hook |
| 编排流程 | `backend/service/chat_orchestration.go` | 支持插件 Agent 的任务分发和 A2A 调用 |
| 工具审批 | `backend/pkg/tool_approval/runtime.go` | 插件 Tool 可接入审批流程 |
| 模型调用 | `backend/pkg/llm_provider/provider.go` | 支持插件通过 Host 调用模型 |
| 服务入口 | `backend/service/service.go` | 插件系统初始化、生命周期管理 |
| 设置 API | `backend/service/settings.go` | 新增插件管理窗口方法 |

### 14.2 前端集成

以下前端模块需要支持插件能力展示和交互：

| 模块 | 文件 | 集成内容 |
| --- | --- | --- |
| 设置入口 | `frontend/src/pages/settings/index.tsx` | 新增"插件"菜单项 |
| 插件设置页 | `frontend/src/pages/settings/plugins/index.tsx` | 新建插件管理页面 |
| 聊天输入 | `frontend/src/components/chat/input/index.tsx` | 工具选择中新增插件聚合项 |
| 聊天消息展示 | `frontend/src/components/chat/message/index.tsx` | Agent 执行状态简化展示 |
| 内联工具卡片 | `frontend/src/components/chat/message/inline_tool_card/index.tsx` | 支持插件 view_tool 触发的 View 渲染 |
| 侧边面板 | `frontend/src/pages/home/chat/index.tsx` | 支持聊天详情页右侧展示插件 View |

---

## 15. 兼容与扩展

V1 为后续版本扩展预留空间：

- Manifest 包含 `minHostVersion`，未来可加入 `maxHostVersion` 或能力版本。
- RPC 协议包含 `protocolVersion`，支持向后兼容升级。
- 能力注册需支持版本字段。
- Tool、Agent、View ID 采用插件命名空间（`plugin:<plugin_id>:<capability_type>:<id>`），避免未来跨插件冲突。
- 权限模型应允许后续增加网络访问、完整文件系统、剪贴板、系统命令等权限。
- 插件间通信、插件依赖等能力仅预留命名但不实现。

---

## 16. 测试与验收标准

### 16.1 运行时与目录

- [ ] 首次启动插件系统时，自动创建 `<GetDataPath()>/plugin_runtime` 目录。
- [ ] Node.js 运行时存在且可执行时，插件系统状态为可用。
- [ ] Node.js 运行时缺失或损坏时，设置页显示明确错误提示，主程序其他功能不受影响。

### 16.2 插件管理

- [ ] 可以安装合法插件，安装成功后默认禁用。
- [ ] 非法 Manifest 被拒绝并展示具体原因。
- [ ] 可以启用、禁用、删除插件。
- [ ] 插件启用/禁用/删除状态在应用重启后保持一致。
- [ ] 删除插件后，其能力不再出现在工具选择、Agent 注册、Hook 链和 View 注册表中。

### 16.3 隔离与稳定性

- [ ] 单个插件进程崩溃不影响主程序。
- [ ] 单个插件 RPC 卡死不阻塞聊天输入、设置页和其他插件。
- [ ] 多个插件同时启用时，进程、日志、能力和状态相互隔离。
- [ ] 插件启动失败时，其他启用插件仍可正常运行。

### 16.4 general_plugin

- [ ] `general_plugin` 可以注册 `use_tool` 和 `view_tool`。
- [ ] LLM 可以调用 `use_tool` 并获得结构化返回结果。
- [ ] LLM 可以调用 `view_tool`，并在聊天详情页右侧渲染插件 View。
- [ ] `general_plugin` 尝试注册 Agent 时被 Host 拒绝。

### 16.5 agent_plugin

- [ ] `agent_plugin` 可以注册 Agent 和 View。
- [ ] `agent_plugin` 尝试向主程序暴露 Tool 时被 Host 拒绝。
- [ ] 聊天输入框工具选择中仅展示插件聚合项（显示插件 Agent），不展示内部工具。
- [ ] 插件 Agent 执行时，聊天详情页仅显示粗粒度执行状态（"正在执行" / "执行完成" / "执行失败"），不显示详细过程追踪。
- [ ] 插件 Agent 可以按权限使用 Host 开放的内置工具。

### 16.6 Hook、后台任务与通知

- [ ] `before_llm_send` Hook 按顺序执行，Hook 失败时被跳过并记录日志。
- [ ] `after_llm_send` Hook 失败时不影响用户看到模型响应。
- [ ] 插件可以在授权后通过 Host 调用 LLM 模型，且无法直接读取模型密钥。
- [ ] 插件可以注册后台任务，插件禁用/删除后后台任务被停止。
- [ ] 插件可以在授权后发送通知，通知标记来源插件，频率受限。

### 16.7 UI 验收

- [ ] 设置页菜单存在"插件"入口。
- [ ] 插件设置页桌面端为左右布局（左侧列表 + 右侧详情），移动端支持列表/详情切换。
- [ ] 插件列表右上角有"添加"按钮。
- [ ] 插件详情右上角有启用/禁用、设置（条件显示）、删除按钮。
- [ ] 未声明 `settingsView` 的插件不显示"设置"按钮。
- [ ] 点击"设置"按钮在新窗口打开插件设置 View，窗口标题包含插件名称。
- [ ] 聊天输入框工具选择中，启用插件以插件名称聚合项展示，旁边标记"插件"标签。
- [ ] 鼠标悬停插件聚合项时，右侧展示该插件提供的 Tool（`general_plugin`）或 Agent（`agent_plugin`）。
- [ ] `agent_plugin` 的悬停面板中不展示插件内部工具。

---

## 17. V1 默认策略

- 插件安装后默认禁用状态。
- 插件删除时默认删除插件私有数据，不保留。
- Hook 调用失败默认跳过，不阻断 LLM 请求流程。
- `before_llm_send` Hook 默认不允许阻断消息发送。
- 插件启动失败默认最多自动重试 1 次，再次失败保持异常状态。
- 插件 Agent 内部过程和内部工具调用不进入主程序详细执行追踪。
- 插件聚合项在工具选择中的勾选仅影响当前聊天，不改变插件全局启用状态。
- 通知频率限制为每分钟每插件最多 5 条。
