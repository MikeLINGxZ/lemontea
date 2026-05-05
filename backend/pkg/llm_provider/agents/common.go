package agents

// AgentType 表示 Agent 类型
type AgentType string

const (
	AgentTypeSystem AgentType = "system" // 系统内置 Agent
	AgentTypeCustom AgentType = "custom" // 用户自定义 Agent（未来扩展）
)

// AgentRole 表示 Agent 在工作流中的功能角色
type AgentRole string

const (
	AgentRoleMain        AgentRole = "main"
	AgentRoleWorkflow    AgentRole = "workflow"
	AgentRolePlanner     AgentRole = "planner"
	AgentRoleWorker      AgentRole = "worker"
	AgentRoleReviewer    AgentRole = "reviewer"
	AgentRoleSynthesizer AgentRole = "synthesizer"
)

// AgentPromptMeta 描述一个 Agent 提示词文件的元数据。
type AgentPromptMeta struct {
	FileName    string // 文件名，如 "system.main_agent.md"
	Title       string // 显示标题
	Description string // 描述
	IsSystem    bool   // 系统提示词 vs 用户模板
}

// IAgent 定义 Agent 的元数据接口。
// 该接口描述 Agent 的身份、提示词归属等静态信息，
// 不负责创建 adk.Agent 运行时实例。
type IAgent interface {
	Name() string                      // 唯一标识，如 "main_agent"
	Desc() string                      // 人类可读的描述
	Prompt() string                    // 当前主系统提示词内容（从 Agent 目录加载）
	Type() AgentType                   // system / custom
	Role() AgentRole                   // 功能角色
	PromptNames() []string             // 关联的提示词文件名列表
	PromptMetas() []AgentPromptMeta    // 提示词元数据列表
	DefaultPrompts() map[string]string // promptFileName → 默认内容
}

// ISkillCapableAgent is an optional interface that agents can implement
// to declare support for skill usage. Currently only CustomAgentDef implements this.
// System agents may implement this in the future.
type ISkillCapableAgent interface {
	IAgent
	GetSkillNames() []string
}
