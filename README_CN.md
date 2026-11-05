# Lemontea Desktop

[English](README.md) | 简体中文

Lemontea Desktop 是一款面向桌面场景的跨平台 AI 客户端，适合希望在本机拥有一个可配置、可扩展、可观察 AI 助手的用户。它把聊天、模型配置、工具执行、插件、Skills 和长期记忆放进同一个本地工作台，让 AI 使用不再停留在单一输入框。

## 产品愿景

Lemontea 希望成为个人桌面 AI 工作台。它帮助用户连接不同模型供应商，管理对话和上下文，调用本地工具执行任务，并通过可复用能力持续扩展助手。

产品重点围绕三个方向：

- **可配置**：用户可以选择供应商、模型、语言、显示偏好、插件、Skills 和记忆行为。
- **可观察**：对话、工具调用、审批、错误和后台任务都应该清晰可见、可理解。
- **可扩展**：助手能力可以通过 CLI 工具、插件、MCP 集成和 Markdown Skills 不断增强。

## 目标用户

- 希望用桌面 AI 助手完成写作、翻译、总结、代码辅助和资料整理的个人用户。
- 需要本地工具、终端能力、文件附件、自定义模型接口和执行轨迹的开发者。
- 希望沉淀可复用技能、接入插件、使用长期记忆和结构化工作流的 AI 重度用户。
- 偏好本地优先掌控 AI 客户端配置和对话数据的用户。

## 核心体验

### 对话工作台

Lemontea 提供聚焦的聊天界面，支持流式响应、Markdown 渲染、代码高亮、推理消息展示、文件附件和图片预览。

用户可以创建、重命名、收藏、删除和回看会话。应用也可以自动生成会话标题，让长期任务和历史记录更容易浏览。

### 模型供应商配置

首次启动引导会帮助用户连接第一个模型供应商。内置供应商预设包括 DeepSeek、阿里云百炼 / 通义千问兼容接口、Ollama，以及任意 OpenAI 兼容 API。

完成配置后，用户可以在设置中管理供应商：编辑 API Key 和 Base URL、拉取模型列表、添加自定义模型、启用或禁用供应商，并设置默认项。

### 带人工控制的工具执行

助手可以调用日期、时间、文件处理、命令执行、二维码生成、网页搜索和网页抓取等内置工具。风险更高的操作会进入确认流程，用户可以允许、拒绝，或给出自定义反馈后再继续执行。

### 插件、CLI 工具、MCP 与 Skills

Lemontea 从设计上支持扩展：

- 插件和 CLI 工具用于接入外部能力。
- MCP 集成用于连接兼容的工具服务。
- Skills 用 Markdown 文件和 YAML frontmatter 封装可复用指令。
- 内置和用户自定义 Skills 可以指导助手完成重复工作流。

这让 Lemontea 既适合日常 AI 助手场景，也适合更专业的个人工作流程。

### 长期记忆

记忆系统帮助 Lemontea 在多轮会话之外保留高价值信息。用户可以创建、更新、遗忘、恢复和查看记忆，也可以在设置中明确管理记忆行为。

它的目标不是保存每一句聊天记录，而是保留能改善未来交互的稳定上下文。

### 设置中心

设置中心围绕真实产品能力组织，包括通用偏好、模型供应商、插件与工具、Skills、记忆和关于信息。语言和字体大小都可以在运行时切换。

## 典型场景

- 配置本地或远程模型供应商，并开始桌面 AI 对话。
- 在聊天中附加文件或图片，请助手分析、改写或辅助实现。
- 让助手在审批后执行本地命令，同时保留清晰的执行过程。
- 安装或配置插件工具，服务特定工作流。
- 为团队规范、写作风格、报告模板或操作手册创建可复用 Skills。
- 把稳定偏好和项目上下文保存为长期记忆，供后续对话使用。

## 平台支持

- macOS
- Windows
- Linux
- 实验性的 iOS / Android 构建脚手架

## 技术栈

- 桌面框架：Wails v3.1.0
- 后端：Go 1.26+
- 前端：React 19.3、TypeScript 7.0、Vite 8.3、Tailwind CSS 4.3
- 状态管理：Zustand
- 数据存储：SQLite / GORM
- 测试：Go test、Vitest、Testing Library
- 构建工具：Wails 3 CLI

## 环境要求

请先安装：

- Go 1.26 或更高版本
- Node.js 与 npm
- Wails v3.1.0 CLI（`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`）

如需构建可在 macOS 12 上运行的版本，请使用 Go 1.26。使用 Go 1.27 构建的版本需要 macOS 13 或更高版本。
Tailwind CSS 4 要求 macOS 的 WebView 使用 Safari 16.4 或更高版本对应的 WebKit。

可用以下命令确认环境：

```bash
go version
npm --version
wails3 version
```

## 快速开始

安装前端依赖：

```bash
cd frontend
npm install
cd ..
```

启动开发模式：

```bash
wails3 dev -config ./build/config.yml
```

如需指定前端开发服务器端口：

```bash
wails3 dev -config ./build/config.yml -port 9246
```

## 常用命令

```bash
# 开发模式
wails3 dev -config ./build/config.yml

# 构建当前平台桌面应用
wails3 build

# 打包当前平台产物
wails3 package

# 生成前端 Wails 绑定
wails3 generate bindings -clean=true -ts
```

## 测试

后端测试：

```bash
go test ./...
```

前端测试：

```bash
cd frontend
npx vitest run
```

## 项目结构

```text
.
├── backend/                 # Go 后端服务、基础包、存储和模型
├── frontend/                # React + Vite 前端
├── build/                   # Wails 构建配置和各平台资源
├── docs/                    # 产品、技术设计和实施计划
├── main.go                  # 应用入口与服务注册
└── go.mod                   # Go module 配置
```

## 相关文档

- 产品背景：[docs/dev/00.background.md](docs/dev/00.background.md)
- 通用开发规则：[docs/dev/00.rules.md](docs/dev/00.rules.md)
- 初始化引导：[docs/dev/11.welcome.md](docs/dev/11.welcome.md)
- 插件与 CLI：[docs/dev/13.plugin_cli.md](docs/dev/13.plugin_cli.md)
- 记忆系统：[docs/dev/17.memory_core.md](docs/dev/17.memory_core.md)

## 许可证

MIT
