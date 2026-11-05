# Lemontea Desktop

English | [简体中文](README_CN.md)

Lemontea Desktop is a cross-platform AI desktop client for people who want a capable assistant on their own machine. It brings chat, model configuration, tool execution, plugins, skills, and long-term memory into one local workspace, so daily AI work can move beyond a single prompt box.

## Product Vision

Lemontea is designed to be a personal AI workbench for desktop workflows. It helps users connect different model providers, keep conversations and context organized, run tasks with local tools, and extend the assistant with reusable capabilities.

The product focuses on three ideas:

- **Configurable**: users can choose providers, models, language, display preferences, plugins, skills, and memory behavior.
- **Observable**: conversations, tool calls, approvals, errors, and background tasks should be visible and understandable.
- **Extensible**: the assistant can grow through CLI tools, plugins, MCP integrations, and Markdown-based skills.

## Who It Is For

- Individual users who want a desktop AI assistant for writing, translation, summarization, coding help, and research.
- Developers who need local tools, terminal access, file attachments, custom model endpoints, and inspectable execution traces.
- AI power users who want reusable skills, plugin integrations, long-term memory, and more structured workflows.
- Users who prefer local-first control over their AI client configuration and conversation data.

## Core Experience

### Chat Workspace

Lemontea provides a focused chat interface with streaming responses, Markdown rendering, code highlighting, reasoning message display, file attachments, image preview, and conversation management.

Users can create, rename, favorite, delete, and revisit sessions. The app can also generate titles automatically, keeping long-running work easier to scan later.

### Model Provider Setup

The first-run onboarding flow guides users through connecting their first model provider. Built-in provider presets include DeepSeek, Alibaba Cloud Bailian / Qwen-compatible endpoints, Ollama, and any OpenAI-compatible API.

After setup, users can manage providers in Settings: edit API keys and base URLs, fetch model lists, add custom models, enable or disable providers, and choose defaults.

### Tools With Human Control

The assistant can call built-in tools such as date, time, file handling, command execution, QR code generation, web search, and web fetch. Riskier actions go through a confirmation flow, so users can allow, reject, or respond with custom guidance before execution continues.

### Plugins, CLI Tools, MCP, and Skills

Lemontea is built to be extended:

- Plugins and CLI tools add external capabilities.
- MCP integrations connect compatible tool servers.
- Skills package reusable instructions as Markdown files with YAML frontmatter.
- Built-in and user-created skills can guide the assistant through repeated workflows.

This makes the app suitable for both everyday assistant use and specialized work processes.

### Long-Term Memory

The memory system helps Lemontea preserve high-value information across conversations. Users can create, update, forget, restore, and inspect memories, while settings allow memory behavior to be managed explicitly.

The goal is not to store every message, but to retain durable context that improves future interactions.

### Settings Center

Settings are organized around real product surfaces: general preferences, model providers, plugins and tools, skills, memory, and about information. Language and font size can be changed at runtime.

## Typical Use Cases

- Configure a local or remote model provider and start a desktop AI conversation.
- Attach files or images to a chat and ask for analysis, rewriting, or implementation help.
- Let the assistant run approved local commands while keeping the execution visible.
- Install or configure plugin tools for specific workflows.
- Create reusable skills for team conventions, writing styles, reports, or operational playbooks.
- Save stable preferences and project context into memory for future conversations.

## Platform Support

- macOS
- Windows
- Linux
- Experimental iOS and Android build scaffolding

## Tech Stack

- Desktop framework: Wails v3.1.0
- Backend: Go 1.26+
- Frontend: React 19.3, TypeScript 7.0, Vite 8.3, Tailwind CSS 4.3
- State management: Zustand
- Storage: SQLite / GORM
- Testing: Go test, Vitest, Testing Library
- Build tooling: Wails 3 CLI

## Requirements

Install the following tools first:

- Go 1.26 or later
- Node.js and npm
- Wails v3.1.0 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`)

Use Go 1.26 to build for macOS 12. Builds with Go 1.27 require macOS 13 or later.
The macOS WebView needs the WebKit version included with Safari 16.4 or later for Tailwind CSS 4.

Check your environment:

```bash
go version
npm --version
wails3 version
```

## Quick Start

Install frontend dependencies:

```bash
cd frontend
npm install
cd ..
```

Start development mode:

```bash
wails3 dev -config ./build/config.yml
```

To choose a specific frontend dev server port:

```bash
wails3 dev -config ./build/config.yml -port 9246
```

## Common Commands

```bash
# Development mode
wails3 dev -config ./build/config.yml

# Build the desktop app for the current platform
wails3 build

# Package production artifacts for the current platform
wails3 package

# Generate frontend Wails bindings
wails3 generate bindings -clean=true -ts
```

## Testing

Backend tests:

```bash
go test ./...
```

Frontend tests:

```bash
cd frontend
npx vitest run
```

## Project Structure

```text
.
├── backend/                 # Go backend services, packages, storage, and models
├── frontend/                # React + Vite frontend
├── build/                   # Wails build config and platform assets
├── docs/                    # Product notes, technical designs, and plans
├── main.go                  # App entry point and service registration
└── go.mod                   # Go module config
```

## Documentation

- Product background: [docs/dev/00.background.md](docs/dev/00.background.md)
- Development rules: [docs/dev/00.rules.md](docs/dev/00.rules.md)
- Onboarding: [docs/dev/11.welcome.md](docs/dev/11.welcome.md)
- Plugins and CLI: [docs/dev/13.plugin_cli.md](docs/dev/13.plugin_cli.md)
- Memory system: [docs/dev/17.memory_core.md](docs/dev/17.memory_core.md)

## License

MIT
