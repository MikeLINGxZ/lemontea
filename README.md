# Lemon Tea Desktop

English | [简体中文](./README_CN.md)

Lemon Tea Desktop is a cross-platform AI desktop client built with Wails v3, Go, React, and TypeScript. It focuses on chat, tool calling, multi-agent workflow execution, and local desktop integration.

<p align="center"><img src="docs/imgs/app_home.png" alt="Lemon Tea Desktop Home" width="80%" height="auto" /></p>

## Features

### Chat

- Multi-turn conversations with streaming output and manual stop.
- Conversation management: create, rename, delete, and favorite chats.
- Automatic chat title generation.
- File attachments in chat input with local image preview.
- Markdown rendering with syntax highlighting.
- Reasoning message display.

### Model Providers

- DeepSeek
- Alibaba Cloud Bailian / Qwen compatible endpoint
- OpenRouter
- Ollama
- Any OpenAI-compatible API

### Model Management

- Load model lists from provider endpoints.
- Set provider default model.
- Add and remove custom models per provider.
- Remember a local default chat model.

### Agent System

- **Multi-agent architecture** with 7 system agents: Main, Workflow, Planner, Worker, Synthesizer, Reviewer, and custom user-defined agents.
- **Dual-path execution**: direct answer for simple queries, workflow orchestration for complex tasks, with smart routing and user override.
- **Sub-agent delegation**: main agent can delegate to custom agents with their own tools and skills.
- **Custom agents**: create agents with custom prompts, tool bindings, and skill bindings.

### Workflow Orchestration

- 7-stage pipeline: Classify → Plan → Dispatch → Synthesize → Review → Retry → Finalize.
- Task dependency graph with topological sorting and parallel batch execution.
- Review feedback-driven retry logic (max 2 retries).
- Task recovery for interrupted running tasks after restart.

### Execution Tracing

- Real-time trace visualization for plan steps, tool calls, stage transitions, and elapsed time.
- Nested step tracking with status indicators (pending, running, awaiting approval, done, error, skipped).
- Collapsible detail blocks for inputs, outputs, and approval decisions.

### Tool System

- Tool calling based on CloudWeGo Eino / ADK.
- Built-in tools: current date, current time, block/wait, file operations, shell command execution, workflow escalation.
- **Tool approval system**: real-time approval UI with Allow / Reject / Custom decisions, timeout handling, and approval history.

### MCP Tool Integration

- Import MCP servers from a local folder.
- Enable/disable MCP tools.
- Remove imported MCP tools.
- MCP server process management.

### Memory System

- Automatic memory encoding from conversations, driven by LLM decisions.
- Multi-phase lifecycle: encoding → consolidation → forgetting → contradiction detection.
- Hybrid search combining keyword and semantic retrieval.
- Configurable embedding model engine.

### Skill System

- Create, edit, and delete custom skills (Markdown with YAML frontmatter).
- Skill tagging for organization.
- Bind skills to custom agents.
- Import skills from a local folder.

### Prompt Management

- View built-in and user prompt files.
- Edit and save prompt files.
- Reset prompt files to defaults.

### Internationalization

- Full-stack i18n with Chinese (zh_CN) and English (en_US).
- Language-specific model prompts.
- Dynamic language switching.

### Settings

- Provider configuration: API keys, base URLs, enable/disable.
- Agent management: view system agents, create/edit custom agents.
- Skill management: browse, create, edit, delete.
- Prompt management: view, edit, reset.
- Memory settings: toggle memory, configure embedding engine.
- General: font size, language.
- Experimental features lab.
- First-launch onboarding wizard.

### Cross-Platform

- macOS
- Windows
- Linux
- Server mode with Docker support
- Experimental iOS / Android build scaffolding

## Tech Stack

- Backend: Go
- Desktop shell: Wails v3
- Frontend: React 19 + TypeScript + Vite
- UI: Ant Design + Ant Design X
- Rich editor: TipTap
- State management: Zustand
- Agent / tool orchestration: CloudWeGo Eino
- Local storage: SQLite via GORM

## Quick Start

### 1. Clone

```bash
git clone <repo>
cd lemon_tea_desktop
```

### 2. Install dependencies

Install Wails v3:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 doctor
```

Install frontend dependencies:

```bash
cd frontend
npm install
cd ..
```

### 3. Run development mode

Using Wails directly:

```bash
wails3 dev -config ./build/config.yml
```

Or with dev:

```bash
wails3 dev
```

## Project Structure

```text
.
├── backend/          Go services, storage, provider adapters, agent workflow logic
│   ├── agents/       Memory system (encoding, search, lifecycle)
│   ├── models/       Data models, view models, wrapper models
│   ├── pkg/          LLM providers, agents, tools, skills, i18n, task execution
│   ├── service/      Core service layer (chat, orchestration, MCP, memory)
│   ├── storage/      GORM/SQLite persistence
│   └── utils/        Event system, error handling
├── frontend/         React UI, chat pages, settings pages, components
├── build/            Wails build and packaging configuration
├── docs/             README assets
└── main.go           Desktop app entry
```

## License

MIT
