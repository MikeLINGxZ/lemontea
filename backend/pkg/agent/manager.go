package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent/tools"
	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/event_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/prompt_id"
	pkgMCP "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/mcp"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/prompt"
	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
	pkgRuntimeState "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/runtime_state"
	pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	agenttool "trpc.group/trpc-go/trpc-agent-go/tool"
	toolpkg "trpc.group/trpc-go/trpc-agent-go/tool"
)

// 默认应用主入口智能体提示词
const defaultMainAgentPrompt = `You are Lemontea, a helpful AI assistant. You can help users with various tasks including answering questions, writing, coding, and more. Be concise, accurate, and helpful. When using tools, explain what you are doing and why.`

// 默认生成器聊天标题提示词
const defaultGenChatTitlePrompt = `Generate a concise title under 20 characters for this conversation. Return only the title.`

// Manager 管理主智能体的运行器、提示词和工具注册
type Manager struct {
	// 保护共享状态的互斥锁
	mu sync.RWMutex
	// 存储接口
	istorage *storage.Storage
	// Wails 应用实例
	wailsApp *application.App
	// 工具注册表
	toolRegistry *tools.Registry
	// 流式输出管理器
	streamManager *StreamManager
	// 运行器列表
	runners map[string]runner.Runner
	// MCP管理器
	mcpManager *pkgMCP.Manager
	// 技能提供商
	skillProvider tools.SkillProvider
	// 注意提醒
	attention tools.AttentionRequester
	// 技能创建器
	skillCreator tools.SkillCreator
	// CLI安装器
	cliInstaller tools.CliInstaller
	// CLI安装进度
	cliInstallProgress tools.CliInstallProgressReporter
	// CLI清单生成器
	cliManifestGenerator tools.CliManifestGenerator
	// CLI命令Runner
	cliCommandRunner tools.CliCommandRunner
	// 终端Runner
	terminalRunner *pkgterminal.Manager
	// 工具确认决策器
	toolConfirmDecider ToolConfirmDecider
	// 记忆编码并发限流
	memoryEncodeSem chan struct{}
}

// 描述工具结果事件发布器使用的数据结构
type toolResultEmitter struct {
	// Wails 应用实例
	app *application.App
}

// 将工具执行结果发布给对应会话，接收会话编号、工具名称、本次操作的结果，无返回值
func (e toolResultEmitter) EmitToolResult(sessionID uint, toolName, result string) {
	if e.app == nil {
		return
	}
	e.app.Event.Emit(event_id.AgentStreamToolResult, map[string]any{
		"sessionId": sessionID,
		"toolName":  toolName,
		"result":    result,
	})
}

// NewManager 创建管理器并注册主智能体的默认提示词和工具
// 接收存储接口，返回管理器
// 注册当前数据
// 构建注册表
// 注册当前数据
func NewManager(istorage *storage.Storage) *Manager {
	prompt.Register(prompt_id.MainAgent, defaultMainAgentPrompt)
	prompt.Register(prompt_id.GenChatTitle, defaultGenChatTitlePrompt)

	registry := tools.NewRegistry() // 注册表

	// 注册当前数据
	registry.Register(tools.DateTimeMeta())
	registry.Register(tools.FileReadMeta())
	registry.Register(tools.FileWriteMeta())
	registry.Register(tools.ShellMeta())
	registry.Register(tools.BuildQuestionTool())
	registry.Register(tools.WebFetchMeta())
	registry.Register(tools.BuildWebSearchTool())
	registry.Register(tools.CodeExecMeta())
	registry.Register(tools.BuildQRCodeTool())
	registry.Register(tools.BuildInteractiveTerminalTool())
	registry.Register(tools.BuildRequestAttentionTool())
	registry.Register(tools.BuildProposeSkillTool())
	registry.Register(tools.BuildInstallCliTool())
	registry.Register(tools.BuildReportCliInstallProgressTool())
	registry.Register(tools.BuildGenerateCliManifestTool())
	registry.Register(tools.BuildRunCliCommandTool())
	registry.Register(tools.BuildSaveTaskStateTool())
	registry.Register(tools.BuildLoadTaskStateTool())
	registry.Register(tools.BuildTodoWriteTool())
	// 注册静态 Skill 工具元数据供发现使用
	// buildAgentTools 每轮都根据当前提供者生成动态描述
	registry.Register(tools.ToolMeta{
		Name:        tools.SkillToolName,
		Description: "Load a skill's instructions by name",
		Category:    tools.CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 解析结果
			var parsed struct {
				// 名称
				Name string `json:"name"`
			}
			_ = json.Unmarshal(args, &parsed)
			return "Loading skill: " + parsed.Name
		},
	})

	return &Manager{
		istorage:           istorage,
		toolRegistry:       registry,
		streamManager:      NewStreamManager(),
		runners:            make(map[string]runner.Runner),
		mcpManager:         pkgMCP.NewManager(mustDataDir()),
		toolConfirmDecider: defaultToolConfirmDecider,
		memoryEncodeSem:    make(chan struct{}, 2),
	}
}

// SetApp 在服务启动时保存 Wails 应用引用
// 接收Wails 应用实例，无返回值
func (m *Manager) SetApp(app *application.App) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wailsApp = app
}

// ToolRegistry 返回已注册的工具元数据注册表
func (m *Manager) ToolRegistry() *tools.Registry {
	return m.toolRegistry
}

// Streams 返回当前流管理器
func (m *Manager) Streams() *StreamManager {
	return m.streamManager
}

// Storage 返回底层存储
func (m *Manager) Storage() *storage.Storage {
	return m.istorage
}

// App 返回 Wails 应用引用
func (m *Manager) App() *application.App {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.wailsApp
}

// SetSkillProvider 设置 Skill 元工具使用的技能提供者
// 必须在处理聊天消息前调用
// 接收提供商，无返回值
func (m *Manager) SetSkillProvider(provider tools.SkillProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skillProvider = provider
}

// SkillProvider 返回当前技能提供者，未配置时返回 nil
func (m *Manager) SkillProvider() tools.SkillProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skillProvider
}

// SetAttentionRequester 设置通过通知请求用户关注的组件
// 接收请求器，无返回值
func (m *Manager) SetAttentionRequester(requester tools.AttentionRequester) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attention = requester
}

// SetSkillCreator 设置 ProposeSkill 用于保存技能提案的组件
// 接收创建器，无返回值
func (m *Manager) SetSkillCreator(creator tools.SkillCreator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skillCreator = creator
}

// SetCliInstaller 设置 InstallCli 工具使用的 CLI 安装器
// 接收循环索引，无返回值
func (m *Manager) SetCliInstaller(i tools.CliInstaller) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cliInstaller = i
}

// SetCliInstallProgressReporter 设置任务安装流程使用的进度上报器
// 接收读取源，无返回值
func (m *Manager) SetCliInstallProgressReporter(r tools.CliInstallProgressReporter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cliInstallProgress = r
}

// SetCliManifestGenerator 设置 GenerateCliManifest 工具使用的清单生成器
// 接收g 对应的数据，无返回值
func (m *Manager) SetCliManifestGenerator(g tools.CliManifestGenerator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cliManifestGenerator = g
}

// SetCliCommandRunner 设置 RunCliCommand 工具使用的命令运行器
// 接收读取源，无返回值
func (m *Manager) SetCliCommandRunner(r tools.CliCommandRunner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cliCommandRunner = r
}

// 设置终端Runner，接收读取源，无返回值
func (m *Manager) SetTerminalRunner(r *pkgterminal.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminalRunner = r
}

// SetToolConfirmDecider 替换工具执行前的运行时确认决策器
// 接收决策器，无返回值
func (m *Manager) SetToolConfirmDecider(decider ToolConfirmDecider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolConfirmDecider = decider
}

// 构造智能体可使用的工具，接收已启用工具、会话类别和会话编号，返回工具列表
// 按内置工具标识注册对应实现，并按任务会话和当前依赖限制可用工具
// 技能元工具仅在已有技能提供者时创建，工具说明从当前可用技能生成
func (m *Manager) buildAgentTools(enabledUserTools []string, sessionKind string, sessionID uint) []toolpkg.Tool {
	metas := m.toolRegistry.EnabledTools(enabledUserTools) // 元数据列表
	agentTools := make([]toolpkg.Tool, 0, len(metas))      // 智能体工具列表

	// 遍历元数据列表，逐项取得元信息
	for _, meta := range metas {
		switch meta.Name {
		case "datetime":
			agentTools = append(agentTools, tools.NewDateTimeTool())
		case "file_read":
			agentTools = append(agentTools, tools.NewFileReadTool())
		case "file_write":
			agentTools = append(agentTools, tools.NewFileWriteTool())
		case "shell":
			if m.terminalRunner != nil {
				agentTools = append(agentTools, tools.NewShellTool(toolResultEmitter{app: m.wailsApp}, sessionID, m.terminalRunner))
			} else {
				agentTools = append(agentTools, tools.NewShellTool(toolResultEmitter{app: m.wailsApp}, sessionID))
			}
		case tools.QuestionToolName:
			agentTools = append(agentTools, tools.NewQuestionTool())
		case "web_fetch":
			agentTools = append(agentTools, tools.NewWebFetchTool())
		case tools.WebSearchToolName:
			agentTools = append(agentTools, tools.NewWebSearchTool())
		case "code_exec":
			agentTools = append(agentTools, tools.NewCodeExecTool())
		case tools.QRCodeToolName:
			agentTools = append(agentTools, tools.NewQRCodeTool())
		case tools.InteractiveTerminalToolName:
			agentTools = append(agentTools, tools.NewInteractiveTerminalTool())
		case tools.SkillToolName:
			// 仅在配置技能提供者时加入 Skill 元工具
			// 工具描述根据当前启用的技能动态生成
			//
			// buildAgentTools 始终由持有 m.mu 的调用方调用
			// 因此这里可以直接读取 m.skillProvider
			if m.skillProvider != nil {
				agentTools = append(agentTools, tools.NewSkillTool(m.skillProvider))
			}
		case tools.RequestAttentionToolName:
			if sessionKind == "task" && m.attention != nil {
				agentTools = append(agentTools, tools.NewRequestAttentionTool(m.attention, sessionID))
			}
		case tools.ProposeSkillToolName:
			if sessionKind == "task" && m.attention != nil && m.skillCreator != nil {
				agentTools = append(agentTools, tools.NewProposeSkillTool(m.attention, m.skillCreator, sessionID))
			}
		case tools.SaveTaskStateToolName:
			if sessionKind == "task" {
				agentTools = append(agentTools, tools.NewSaveTaskStateTool(m.istorage, sessionID))
			}
		case tools.LoadTaskStateToolName:
			if sessionKind == "task" {
				agentTools = append(agentTools, tools.NewLoadTaskStateTool(m.istorage, sessionID))
			}
		case tools.TodoWriteToolName:
			agentTools = append(agentTools, tools.NewTodoWriteTool())
		case tools.InstallCliToolName:
			if m.cliInstaller != nil {
				agentTools = append(agentTools, tools.NewInstallCliTool(m.cliInstaller))
			}
		case tools.ReportCliInstallProgressToolName:
			if sessionKind == "task" && m.cliInstallProgress != nil {
				agentTools = append(agentTools, tools.NewReportCliInstallProgressTool(m.cliInstallProgress, sessionID))
			}
		case tools.GenerateCliManifestToolName:
			if m.cliManifestGenerator != nil {
				agentTools = append(agentTools, tools.NewGenerateCliManifestTool(m.cliManifestGenerator))
			}
		case tools.RunCliCommandToolName:
			if m.cliCommandRunner != nil {
				agentTools = append(agentTools, tools.NewRunCliCommandTool(m.cliCommandRunner, toolResultEmitter{app: m.wailsApp}, sessionID))
			}
		}
	}

	return agentTools
}

// 按 MCP 服务分组构建可注册的工具集，接收已启用的用户工具列表，返回工具集合列表
func (m *Manager) buildMCPToolSets(enabledUserTools []string) []agenttool.ToolSet {
	if len(enabledUserTools) == 0 {
		return nil
	}

	config, err := loadAgentConfig() // 配置、加载智能体配置时的错误
	if err != nil || len(config.Extensions) == 0 {
		return nil
	}

	toolSets := make([]agenttool.ToolSet, 0) // 工具集合
	// 遍历扩展列表，逐项取得当前条目
	for _, item := range config.Extensions {
		if item.Kind != "mcp" || !item.Enabled {
			continue
		}
		toolSet, buildErr := m.mcpManager.BuildToolSet(item, enabledUserTools) // 工具集合、构造工具集合时的错误
		if buildErr != nil {
			continue
		}
		if toolSet.Name() != "" {
			toolSets = append(toolSets, toolSet)
		}
	}
	return toolSets
}

// 为已启用的 CLI 扩展构建工具集合，接收已启用的用户工具列表，返回工具集合列表
func (m *Manager) buildCliToolSets(enabledUserTools []string) []agenttool.ToolSet {
	if len(enabledUserTools) == 0 {
		return nil
	}

	config, err := loadAgentConfig() // 配置、加载智能体配置时的错误
	if err != nil || len(config.Extensions) == 0 {
		return nil
	}
	state, err := pkgRuntimeState.LoadPersistedState() // 状态
	if err != nil || state.State != "ready" || state.NodePath == "" || state.NpmPath == "" {
		return nil
	}
	mgr := pkgcli.NewManager(mustDataDir(), state.NodePath, state.NpmPath) // 管理器

	toolSets := make([]agenttool.ToolSet, 0) // 工具集合
	// 遍历扩展列表，逐项取得当前条目
	for _, item := range config.Extensions {
		if item.Kind != "cli" || !item.Enabled {
			continue
		}
		manifest, loadErr := pkgcli.LoadManifest(item.ConfigFilePath) // 清单、加载 CLI 扩展清单时的错误
		if loadErr != nil || manifest.Executable == "" {
			continue
		}
		toolSet, buildErr := pkgcli.BuildToolSet(mgr, item, manifest, enabledUserTools) // 工具集合、构造工具集合时的错误
		if buildErr != nil {
			continue
		}
		if toolSet.Name() != "" {
			toolSets = append(toolSets, toolSet)
		}
	}
	return toolSets
}

// inferOpenAIVariant 将持久化的提供方类型映射为对应的 OpenAI 变体
// 接收提供商类型，返回变体
func inferOpenAIVariant(providerType pkgProvider.Type) openai.Variant {
	switch providerType {
	case pkgProvider.Aliyun:
		return openai.VariantQwen
	case pkgProvider.Deepseek:
		return openai.VariantDeepSeek
	default:
		return openai.VariantOpenAI
	}
}

// 根据提供商、模型、工具和会话配置获取运行器，返回运行器及可能发生的错误
// 优先复用缓存中匹配的运行器
// 缓存未命中时组装模型、会话服务和工具集合，并将新运行器写入缓存
func (m *Manager) GetOrCreateRunner(
	baseURL string,
	apiKey string,
	modelName string,
	providerType pkgProvider.Type,
	enabledUserTools []string,
	sessionKind string,
	sessionID uint,
) (runner.Runner, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := runnerCacheKey(baseURL, modelName, providerType, enabledUserTools, sessionKind) // 键
	if /* 已存在、是否成功取得对应值 */ existing, ok := m.runners[key]; ok {
		return existing, nil
	}
	legacyKey := fmt.Sprintf("%s/%s", baseURL, modelName) // 旧版键
	if len(enabledUserTools) == 0 {
		if /* 已存在、是否成功取得对应值 */ existing, ok := m.runners[legacyKey]; ok {
			return existing, nil
		}
	}

	instruction, err := prompt.Load(prompt_id.MainAgent) // 指令、加载当前数据时的错误
	if err != nil {
		instruction = defaultMainAgentPrompt
	}

	variant := inferOpenAIVariant(providerType) // 变体

	// 模型
	mdl := openai.New(
		modelName,
		openai.WithBaseURL(baseURL),
		openai.WithAPIKey(apiKey),
		openai.WithVariant(variant),
	)

	// ag 对应的数据
	ag := llmagent.New(
		"main",
		llmagent.WithModel(mdl),
		llmagent.WithInstruction(instruction),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			Stream: true,
		}),
		llmagent.WithTools(m.buildAgentTools(enabledUserTools, sessionKind, sessionID)),
		llmagent.WithToolSets(append(m.buildMCPToolSets(enabledUserTools), m.buildCliToolSets(enabledUserTools)...)),
		llmagent.WithRefreshToolSetsOnRun(true),
		llmagent.WithToolCallbacks(m.newToolCallbacks()),
	)

	sessionSvc := inmemory.NewSessionService() // 会话服务
	// 创建
	created := runner.NewRunner(
		"lemontea",
		ag,
		runner.WithSessionService(sessionSvc),
	)

	m.runners[key] = created
	if len(enabledUserTools) == 0 {
		m.runners[legacyKey] = created
	}
	return created, nil
}

// 根据提供商与模型配置生成运行器缓存键，接收服务基础地址、模型名称、提供商类型、已启用的用户工具列表、会话类别，返回文本
func runnerCacheKey(baseURL string, modelName string, providerType pkgProvider.Type, enabledUserTools []string, sessionKind string) string {
	names := append([]string(nil), enabledUserTools...) // 名称列表
	slices.Sort(names)
	return fmt.Sprintf("%s|%s|%s|%s|%s", baseURL, modelName, providerType, strings.Join(names, ","), sessionKind)
}

// 加载智能体配置，返回配置及可能发生的错误
func loadAgentConfig() (*data_models.Config, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return nil, err
	}
	bytes, err := os.ReadFile(filepath.Join(dataDir, dir.ConfigFileName)) // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			return &data_models.Config{
				Extensions: []data_models.ExtensionItem{},
				Memory:     data_models.MemoryConfig{Enabled: true},
			}, nil
		}
		return nil, err
	}

	config := &data_models.Config{}            // 配置
	decodeErr := json.Unmarshal(bytes, config) // 解析序列化内容时的错误
	if decodeErr != nil {
		return nil, decodeErr
	}
	if !strings.Contains(string(bytes), `"memory"`) {
		config.Memory.Enabled = true
	}
	if config.Extensions == nil {
		config.Extensions = []data_models.ExtensionItem{}
	}
	return config, nil
}

// 读取应用数据目录，失败时记录问题并终止当前路径解析，返回文本
func mustDataDir() string {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return dir.HomeDir()
	}
	return dataDir
}

// NewChatHandler 创建绑定到当前管理器的聊天处理器
// 返回聊天处理器
func (m *Manager) NewChatHandler() *ChatHandler {
	return NewChatHandler(m)
}
