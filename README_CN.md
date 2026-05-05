# Lemon Tea Desktop

[English](./README.md) | 简体中文

Lemon Tea Desktop 是一个基于 Wails v3、Go、React 和 TypeScript 构建的跨平台 AI 桌面客户端，当前重点覆盖聊天、工具调用、多 Agent 工作流执行，以及桌面端本地能力集成。

<p align="center"><img src="docs/imgs/app_home.png" alt="Lemon Tea Desktop Home" width="80%" height="auto" /></p>

## 功能特性

### 聊天

- 支持多轮对话、流式输出与手动停止生成。
- 支持会话创建、重命名、删除、收藏。
- 支持自动生成会话标题。
- 支持在聊天输入区附加本地文件，并对图片生成预览。
- 支持 Markdown 渲染与代码高亮。
- 支持推理过程消息展示。

### 模型供应商

- DeepSeek
- 阿里云百炼 / 通义千问兼容接口
- OpenRouter
- Ollama
- 任意 OpenAI 兼容接口

### 模型管理

- 从供应商接口拉取模型列表。
- 设置供应商默认模型。
- 为供应商添加或删除自定义模型。
- 在本地记忆默认聊天模型。

### Agent 系统

- **多 Agent 架构**：内置 7 种系统 Agent（Main、Workflow、Planner、Worker、Synthesizer、Reviewer）及用户自定义 Agent。
- **双路径执行**：简单问题走直接回答，复杂任务走工作流编排，支持智能路由与用户手动切换。
- **子 Agent 委派**：Main Agent 可将任务委派给拥有独立工具和技能的自定义 Agent。
- **自定义 Agent**：支持创建自定义 Agent，配置专属提示词、工具绑定和技能绑定。

### 工作流编排

- 7 阶段流水线：分类 → 规划 → 调度 → 汇总 → 审核 → 重试 → 输出。
- 任务依赖图，支持拓扑排序与并行批次执行。
- 基于审核反馈的重试逻辑（最多 2 次重试）。
- 支持程序重启后的运行中任务恢复。

### 执行轨迹

- 实时轨迹可视化，展示规划步骤、工具调用、阶段流转和耗时。
- 支持嵌套步骤追踪，含状态指示（待执行、运行中、等待审批、完成、异常、已跳过）。
- 可折叠的详情面板，展示输入、输出和审批决策。

### 工具系统

- 基于 CloudWeGo Eino / ADK 的工具调用能力。
- 内置工具：获取当前日期、获取当前时间、阻塞/等待、文件操作、Shell 命令执行、工作流升级。
- **工具审批系统**：实时审批界面，支持允许 / 拒绝 / 自定义决策，超时处理与审批历史记录。

### MCP 工具接入

- 从本地目录导入 MCP Server。
- 启用/禁用 MCP 工具。
- 删除已导入的 MCP 工具。
- MCP Server 进程管理。

### 记忆系统

- 由 LLM 自主决定从对话中提取和编码记忆。
- 多阶段生命周期管理：编码 → 整合 → 遗忘 → 矛盾检测。
- 混合检索，结合关键词搜索与语义搜索。
- 支持配置嵌入模型引擎。

### 技能系统

- 支持创建、编辑、删除自定义技能（Markdown + YAML frontmatter 格式）。
- 支持技能标签分类管理。
- 支持将技能绑定到自定义 Agent。
- 支持从本地目录批量导入技能。

### 提示词管理

- 浏览内置和用户提示词文件。
- 在线编辑并保存提示词。
- 一键恢复默认提示词。

### 国际化

- 全栈 i18n 支持，覆盖中文（zh_CN）和英文（en_US）。
- 模型提示词多语言适配。
- 支持动态切换语言。

### 设置

- 供应商配置：API Key、接口地址、启用/禁用。
- Agent 管理：查看系统 Agent、创建/编辑自定义 Agent。
- 技能管理：浏览、创建、编辑、删除。
- 提示词管理：查看、编辑、重置。
- 记忆设置：开关记忆功能、配置嵌入引擎。
- 通用设置：字体大小、语言。
- 实验室功能。
- 首次启动引导向导。

### 跨平台

- macOS
- Windows
- Linux
- 服务器模式（支持 Docker 部署）
- 实验性的 iOS / Android 构建脚手架

## 技术栈

- 后端：Go
- 桌面框架：Wails v3
- 前端：React 19 + TypeScript + Vite
- UI：Ant Design + Ant Design X
- 富文本编辑：TipTap
- 状态管理：Zustand
- Agent / Tool 编排：CloudWeGo Eino
- 本地存储：SQLite + GORM

## 快速开始

### 1. 克隆项目

```bash
git clone <repo>
cd lemon_tea_desktop
```

### 2. 安装依赖

先安装 Wails v3：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 doctor
```

再安装前端依赖：

```bash
cd frontend
npm install
cd ..
```

### 3. 启动开发模式

直接使用 Wails：

```bash
wails3 dev -config ./build/config.yml
```

或使用 dev：

```bash
wails3 dev
```

## 项目结构

```text
.
├── backend/          Go 服务、存储层、模型供应商适配、Agent 工作流逻辑
│   ├── agents/       记忆系统（编码、检索、生命周期）
│   ├── models/       数据模型、视图模型、包装模型
│   ├── pkg/          LLM 供应商、Agent、工具、技能、国际化、任务执行
│   ├── service/      核心服务层（聊天、编排、MCP、记忆）
│   ├── storage/      GORM/SQLite 持久化
│   └── utils/        事件系统、错误处理
├── frontend/         React 界面、聊天页、设置页、通用组件
├── build/            Wails 构建与打包配置
├── docs/             README 资源文件
└── main.go           桌面应用入口
```

## 许可证

MIT
