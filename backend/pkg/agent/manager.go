package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
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
	agentpkg "trpc.group/trpc-go/trpc-agent-go/agent"
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

// cachedMCPToolSet 保存同一运行器可跨轮复用的 MCP 工具集
type cachedMCPToolSet struct {
	// path 扩展连接配置的路径
	path string
	// revision 连接配置内容的摘要
	revision [sha256.Size]byte
	// toolSet 当前版本的 MCP 工具集
	toolSet agenttool.ToolSet
	// runs 正在使用当前版本的请求
	runs sync.WaitGroup
}

// mcpToolSetCache 管理单个运行器的 MCP 工具集版本
type mcpToolSetCache struct {
	// mu 串行化版本替换和新请求登记
	mu sync.Mutex
	// items 按扩展 ID 保存可被新请求复用的版本
	items map[string]*cachedMCPToolSet
}

// mcpToolSetSource 保存构建 MCP 工具集所需的扩展和配置快照
type mcpToolSetSource struct {
	// item 扩展条目
	item data_models.ExtensionItem
	// configBytes 已读取的连接配置
	configBytes []byte
}

// buildMCPToolSets 根据已选工具 ID 和配置快照构建工具集，返回可注册的集合
func (m *Manager) buildMCPToolSets(enabledUserTools []string, sources ...mcpToolSetSource) []agenttool.ToolSet {
	if len(enabledUserTools) == 0 {
		return nil
	}

	toolSets := make([]agenttool.ToolSet, 0, len(sources)) // 可注册的 MCP 工具集
	for _, source := range sources {
		toolSet, err := m.mcpManager.BuildToolSetFromConfig(source.item, enabledUserTools, source.configBytes) // 当前扩展的工具集和构建错误
		if err != nil {
			slog.Warn("构建 MCP 工具集失败", "扩展编号", source.item.ID, "错误", err)
			continue
		}
		toolSets = append(toolSets, toolSet)
	}
	return toolSets
}

// buildExtensionToolSets 读取扩展配置并按类型构建工具集
// 接收单轮上下文、已选工具 ID 和当前运行器的 MCP 缓存，返回可注册的工具集
func (m *Manager) buildExtensionToolSets(ctx context.Context, enabledUserTools []string, cache *mcpToolSetCache) []agenttool.ToolSet {
	if len(enabledUserTools) == 0 {
		return nil
	}

	// 在同一把锁内读取并登记版本，避免并发请求按旧配置替换新连接
	cache.mu.Lock()
	config, err := loadAgentConfig() // 扩展配置和读取错误
	if err != nil {
		cache.mu.Unlock()
		slog.WarnContext(ctx, "读取工具扩展配置失败", "错误", err)
		return nil
	}

	var mcpItems, cliItems []data_models.ExtensionItem // 按类型分组的已启用扩展
	for _, item := range config.Extensions {
		if !item.Enabled {
			continue
		}
		switch item.Kind {
		case "mcp":
			mcpItems = append(mcpItems, item)
		case "cli":
			cliItems = append(cliItems, item)
		}
	}

	toolSets := make([]agenttool.ToolSet, 0, len(mcpItems)+len(cliItems)) // 按 MCP、CLI 顺序注册的工具集
	next := make(map[string]*cachedMCPToolSet, len(mcpItems))             // 供后续请求复用的版本
	for _, item := range mcpItems {
		if next[item.ID] != nil {
			continue
		}
		configBytes, err := os.ReadFile(item.ConfigFilePath) // 连接配置内容和读取错误
		if err != nil {
			slog.WarnContext(ctx, "读取 MCP 配置失败", "扩展编号", item.ID, "错误", err)
			continue
		}
		revision := sha256.Sum256(configBytes) // 当前连接配置摘要
		entry := cache.items[item.ID]          // 当前运行器缓存的连接
		if entry != nil && entry.path == item.ConfigFilePath && entry.revision == revision {
			delete(cache.items, item.ID)
		} else {
			sets := m.buildMCPToolSets(enabledUserTools, mcpToolSetSource{item: item, configBytes: configBytes}) // 使用新配置构建的工具集
			if len(sets) == 0 {
				continue
			}
			entry = &cachedMCPToolSet{path: item.ConfigFilePath, revision: revision, toolSet: sets[0]}
		}
		next[item.ID] = entry
		entry.runs.Add(1)
		context.AfterFunc(ctx, entry.runs.Done)
		toolSets = append(toolSets, entry.toolSet)
	}
	retired := cache.items // 配置已变化、停用或删除的旧版本
	cache.items = next
	cache.mu.Unlock()

	for _, entry := range retired {
		// 等待已登记的请求结束，再关闭无法被新请求获取的连接
		go func() {
			entry.runs.Wait()
			err := entry.toolSet.Close() // 旧连接关闭错误
			if err != nil {
				slog.WarnContext(ctx, "关闭 MCP 工具集失败", "工具集", entry.toolSet.Name(), "错误", err)
			}
		}()
	}

	if len(cliItems) == 0 {
		return toolSets
	}

	// CLI 运行时不可用时仍可注册 MCP 工具
	state, err := pkgRuntimeState.LoadPersistedState() // Node 运行时状态和读取错误
	if err != nil {
		slog.WarnContext(ctx, "读取 CLI 运行时状态失败", "错误", err)
		return toolSets
	}
	if state.State != "ready" || state.NodePath == "" || state.NpmPath == "" {
		return toolSets
	}

	mgr := pkgcli.NewManager(mustDataDir(), state.NodePath, state.NpmPath) // CLI 执行管理器
	for _, item := range cliItems {
		manifest, err := pkgcli.LoadManifest(item.ConfigFilePath) // 当前扩展的清单和读取错误
		if err != nil {
			slog.WarnContext(ctx, "读取 CLI 工具清单失败", "扩展编号", item.ID, "错误", err)
			continue
		}
		if manifest.Executable == "" {
			continue
		}
		toolSet, err := pkgcli.BuildToolSet(mgr, item, manifest, enabledUserTools) // 当前扩展的工具集和构建错误
		if err != nil {
			slog.WarnContext(ctx, "构建 CLI 工具集失败", "扩展编号", item.ID, "错误", err)
			continue
		}
		toolSets = append(toolSets, toolSet)
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
// 缓存未命中时创建模型和会话服务，工具集合在每轮运行时组装
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
	// DeepSeek 视觉版 Flash 模型及其兼容别名
	deepSeekVision := false
	if variant == openai.VariantDeepSeek {
		switch strings.ToLower(strings.TrimSpace(modelName)) {
		case "deepseek-flash", "deepseek-v4-flash", "deepseek-v4-flash-vision-exp":
			deepSeekVision = true
		}
	}

	// 创建模型时使用的连接与协议选项
	modelOptions := []openai.Option{
		openai.WithBaseURL(baseURL),
		openai.WithAPIKey(apiKey),
		openai.WithVariant(variant),
	}
	if deepSeekVision {
		// 当前 SDK 的 DeepSeek 变体会丢弃图片，使用兼容协议并保留 DeepSeek 的推理历史处理
		modelOptions = append(modelOptions,
			openai.WithVariant(openai.VariantOpenAI),
			openai.WithReasoningContentBackfill(true),
			openai.WithOptimizeForCache(false),
			openai.WithOmitFileContentParts(true),
		)
	}
	mdl := openai.New(modelName, modelOptions...) // 对话模型

	// 保留本运行器的工具选择，每轮重新读取扩展配置和清单
	selectedToolIDs := slices.Clone(enabledUserTools) // 运行器缓存键对应的工具选择
	sessionSvc := inmemory.NewSessionService()        // 运行器复用的会话服务
	mcpCache := &mcpToolSetCache{}                    // 当前运行器复用的 MCP 连接
	created := runner.NewRunnerWithAgentFactory(
		"lemontea",
		"main",
		// 根据单轮上下文构建 Agent，返回本轮实例和构建错误
		func(ctx context.Context, _ agentpkg.RunOptions) (agentpkg.Agent, error) {
			currentSessionID := sessionID           // 缺少运行上下文时沿用调用方会话编号
			if info, ok := getRunContext(ctx); ok { // 本轮会话信息
				currentSessionID = info.SessionID
			}

			m.mu.RLock()
			builtinTools := m.buildAgentTools(selectedToolIDs, sessionKind, currentSessionID) // 本轮可用的内置工具
			m.mu.RUnlock()
			toolSets := m.buildExtensionToolSets(ctx, selectedToolIDs, mcpCache) // 本轮外部扩展工具集

			return llmagent.New(
				"main",
				llmagent.WithModel(mdl),
				llmagent.WithInstruction(instruction),
				llmagent.WithGenerationConfig(model.GenerationConfig{Stream: true}),
				llmagent.WithTools(builtinTools),
				llmagent.WithToolSets(toolSets),
				llmagent.WithRefreshToolSetsOnRun(true),
				llmagent.WithToolCallbacks(m.newToolCallbacks()),
			), nil
		},
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
