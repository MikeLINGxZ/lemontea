package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	rdebug "runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/wrapper_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/tools"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/logger"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/prompts"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/tool_approval"
)

// =============================================================================
// completionRunner —— 封装一次 Completions 调用的全部运行时状态与方法
//
// 原 Completions 函数包含 30+ 个嵌套闭包，它们共享 assistantMessage、task 等
// 可变状态，可读性差。此结构体将所有闭包转换为命名方法，共享变量变为字段，
// 使代码结构清晰、职责分明。
// =============================================================================

// completionRunner 封装一次 Completions 调用的全部运行时状态。
type completionRunner struct {
	// ---- 外部依赖（创建后只读） ----
	svc              *Service                      // 服务实例，用于访问 storage 和事件系统
	provider         *llm_provider.Provider        // LLM 供应商，执行 AI 推理
	localizedPrompts prompts.PromptSet             // 本地化的系统提示词集合
	inputMessage     view_models.Message           // 用户输入消息
	providerModel    *wrapper_models.ProviderModel // 选中的模型配置
	agentTools       []tool.BaseTool               // 用户选择的工具集
	toolMetaByID     map[string]toolMeta           // 工具元数据索引（toolID → meta）
	customAgentIDs   map[string]string             // 自定义 agent ID → 显示名（用于识别子 agent 工具调用）
	cleanupTools     func()                        // 工具资源清理回调
	schemaMessages   []schema.Message              // 转换后的历史消息（含当前用户消息）
	isNewChat        bool                          // 是否为新建对话

	// ---- 标识符（不可变） ----
	chatUuid             string
	assistantMessageUuid string
	taskUuid             string
	eventKey             string

	// ---- 受 mu 保护的可变状态 ----
	// 所有以 Locked 结尾的方法都假设调用者已持有 mu 锁
	mu                            sync.Mutex
	assistantMessage              data_models.Message     // 当前助手消息（持续更新）
	task                          data_models.Task        // 当前任务状态
	pendingTraceDelta             []data_models.TraceStep // 待发送的追踪步骤增量
	lastSnapshotPersistAt         time.Time               // 上次持久化时间（用于节流）
	lastSnapshotPersistContentLen int                     // 上次持久化时的内容长度（用于节流）

	// ---- 工作流切换状态（受 handoffMu 保护） ----
	handoffMu               sync.Mutex
	workflowHandoffDecision *workflowHandoff

	// ---- 运行时原子状态（仅在 run 方法体内使用） ----
	userStopped          atomic.Bool // 用户是否主动停止
	terminalEventEmitted atomic.Bool // 终结事件是否已发射（保证仅执行一次）

	// ---- 优化组件 ----
	fsm            *TaskFSM
	persistWriter  *PersistWriter
	eventBus       *AgentEventBus
	ctxManager     *ContextManager
	workflowConfig WorkflowConfig
}

type WorkflowConfig struct {
	MaxRetries             int
	AllowPartialSuccess    bool
	MaxConsecutiveFailures int
}

func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		MaxRetries:             2,
		AllowPartialSuccess:    true,
		MaxConsecutiveFailures: 2,
	}
}

// newCompletionRunner 创建一个新的 completionRunner 实例。
func newCompletionRunner(
	svc *Service,
	localizedPrompts prompts.PromptSet,
	inputMessage view_models.Message,
	providerModel *wrapper_models.ProviderModel,
	agentTools []tool.BaseTool,
	toolMetaByID map[string]toolMeta,
	customAgentIDs map[string]string,
	cleanupTools func(),
	schemaMessages []schema.Message,
	isNewChat bool,
	chatUuid, assistantMessageUuid, taskUuid, eventKey string,
	assistantMessage data_models.Message,
	task data_models.Task,
) *completionRunner {
	r := &completionRunner{
		svc:                  svc,
		localizedPrompts:     localizedPrompts,
		inputMessage:         inputMessage,
		providerModel:        providerModel,
		agentTools:           agentTools,
		toolMetaByID:         toolMetaByID,
		customAgentIDs:       customAgentIDs,
		cleanupTools:         cleanupTools,
		schemaMessages:       schemaMessages,
		isNewChat:            isNewChat,
		chatUuid:             chatUuid,
		assistantMessageUuid: assistantMessageUuid,
		taskUuid:             taskUuid,
		eventKey:             eventKey,
		assistantMessage:     assistantMessage,
		task:                 task,
		fsm:                  NewTaskFSM(StateIdle),
		eventBus:             NewAgentEventBus(),
		ctxManager:           NewContextManager(DefaultContextConfig()),
		workflowConfig:       DefaultWorkflowConfig(),
	}
	r.persistWriter = NewPersistWriter(svc, r)
	r.fsm.SetOnTransition(func(from, to TaskState, meta *transitionMetadata) {
		r.eventBus.Publish(AgentEvent{
			Type: EventTaskStateChanged,
			Metadata: map[string]interface{}{
				"from":   string(from),
				"to":     string(to),
				"reason": meta.Reason,
				"error":  meta.Error,
			},
		})
	})
	return r
}

// =============================================================================
// 深拷贝 —— 线程安全的消息快照
// =============================================================================

// cloneAssistantMessageLocked 深拷贝 assistantMessage，用于在持锁期间创建
// 安全的快照传递给事件系统，避免数据竞争。
func (r *completionRunner) cloneAssistantMessageLocked() data_models.Message {
	src := r.assistantMessage
	clone := src
	if src.UserMessageExtra != nil {
		userExtra := *src.UserMessageExtra
		if len(src.UserMessageExtra.Files) > 0 {
			userExtra.Files = append([]data_models.File(nil), src.UserMessageExtra.Files...)
		}
		if len(src.UserMessageExtra.Tools) > 0 {
			userExtra.Tools = append([]string(nil), src.UserMessageExtra.Tools...)
		}
		if len(src.UserMessageExtra.Agents) > 0 {
			userExtra.Agents = append([]string(nil), src.UserMessageExtra.Agents...)
		}
		clone.UserMessageExtra = &userExtra
	}
	if src.AssistantMessageExtra != nil {
		assistantExtra := *src.AssistantMessageExtra
		if len(src.AssistantMessageExtra.ToolUses) > 0 {
			assistantExtra.ToolUses = append([]data_models.ToolUse(nil), src.AssistantMessageExtra.ToolUses...)
		}
		if len(src.AssistantMessageExtra.PendingApprovals) > 0 {
			assistantExtra.PendingApprovals = append([]data_models.ToolApprovalSummary(nil), src.AssistantMessageExtra.PendingApprovals...)
		}
		if len(src.AssistantMessageExtra.SubAgentTasks) > 0 {
			clonedTasks := make([]data_models.SubAgentTask, len(src.AssistantMessageExtra.SubAgentTasks))
			for i, t := range src.AssistantMessageExtra.SubAgentTasks {
				clonedTasks[i] = t
				if len(t.ToolCalls) > 0 {
					clonedTasks[i].ToolCalls = append([]data_models.ToolUse(nil), t.ToolCalls...)
				}
			}
			assistantExtra.SubAgentTasks = clonedTasks
		}
		if len(src.AssistantMessageExtra.ExecutionTrace.Steps) > 0 {
			assistantExtra.ExecutionTrace.Steps = make([]data_models.TraceStep, 0, len(src.AssistantMessageExtra.ExecutionTrace.Steps))
			for _, step := range src.AssistantMessageExtra.ExecutionTrace.Steps {
				clonedStep := step
				if len(step.Metadata) > 0 {
					clonedStep.Metadata = make(map[string]interface{}, len(step.Metadata))
					for key, value := range step.Metadata {
						clonedStep.Metadata[key] = value
					}
				}
				if len(step.DetailBlocks) > 0 {
					clonedStep.DetailBlocks = append([]data_models.TraceDetailBlock(nil), step.DetailBlocks...)
				}
				assistantExtra.ExecutionTrace.Steps = append(assistantExtra.ExecutionTrace.Steps, clonedStep)
			}
		}
		clone.AssistantMessageExtra = &assistantExtra
	}
	return clone
}

// =============================================================================
// 持久化与事件发射 —— 消息快照的存储和推送
// =============================================================================

// emitSnapshotLocked 将当前助手消息快照和追踪增量推送给前端订阅者。
func (r *completionRunner) emitSnapshotLocked() {
	traceDelta := append([]data_models.TraceStep(nil), r.pendingTraceDelta...)
	r.pendingTraceDelta = nil
	r.svc.emitTaskEvent(r.task, r.cloneAssistantMessageLocked(), traceDelta)
}

// persistSnapshotLocked 将助手消息持久化到数据库，并发射快照事件。
func (r *completionRunner) persistSnapshotLocked(updateTask bool) error {
	if err := r.svc.storage.SaveOrUpdateMessage(context.Background(), r.assistantMessage); err != nil {
		return err
	}
	if updateTask {
		if err := r.svc.storage.SaveTask(context.Background(), r.task); err != nil {
			return err
		}
	}
	r.lastSnapshotPersistAt = time.Now()
	r.lastSnapshotPersistContentLen = len([]rune(r.assistantMessage.Content))
	r.emitSnapshotLocked()
	return nil
}

// persistSnapshotThrottledLocked 节流版持久化：控制写入频率，避免在流式输出时
// 频繁写数据库。策略：最小间隔 350ms 或内容增量 >= 48 字符时才实际持久化，
// 但每次都会发射快照事件以保证前端实时性。
func (r *completionRunner) persistSnapshotThrottledLocked(updateTask bool) error {
	const minPersistInterval = 350 * time.Millisecond
	const minPersistContentDelta = 48

	currentContentLen := len([]rune(r.assistantMessage.Content))
	shouldPersist := r.lastSnapshotPersistAt.IsZero() ||
		time.Since(r.lastSnapshotPersistAt) >= minPersistInterval ||
		currentContentLen-r.lastSnapshotPersistContentLen >= minPersistContentDelta
	if shouldPersist {
		return r.persistSnapshotLocked(updateTask)
	}
	r.emitSnapshotLocked()
	return nil
}

// =============================================================================
// 索引查找 —— 在切片中按 ID 查找元素位置
// =============================================================================

// findToolUseIndexLocked 根据 callID 查找工具调用记录的索引，未找到返回 -1。
func (r *completionRunner) findToolUseIndexLocked(callID string) int {
	if r.assistantMessage.AssistantMessageExtra == nil {
		return -1
	}
	for idx := range r.assistantMessage.AssistantMessageExtra.ToolUses {
		if r.assistantMessage.AssistantMessageExtra.ToolUses[idx].CallID == callID {
			return idx
		}
	}
	return -1
}

// upsertSubAgentTaskLocked 创建或更新子 agent 任务记录。
func (r *completionRunner) upsertSubAgentTaskLocked(callID, agentID, agentName string, status data_models.ToolUseStatus, input, output string, startedAt, finishedAt *time.Time) {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	tasks := r.assistantMessage.AssistantMessageExtra.SubAgentTasks
	idx := -1
	for i := range tasks {
		if tasks[i].TaskID == callID {
			idx = i
			break
		}
	}
	if idx == -1 {
		// 新建
		task := data_models.SubAgentTask{
			TaskID:       callID,
			AgentID:      agentID,
			AgentName:    agentName,
			Status:       status,
			Input:        input,
			CreatorAgent: "MainChatAgent",
			StartedAt:    startedAt,
		}
		r.assistantMessage.AssistantMessageExtra.SubAgentTasks = append(r.assistantMessage.AssistantMessageExtra.SubAgentTasks, task)
	} else {
		// 更新
		t := &r.assistantMessage.AssistantMessageExtra.SubAgentTasks[idx]
		t.Status = status
		if output != "" {
			t.Output = output
		}
		if finishedAt != nil {
			t.FinishedAt = finishedAt
			if t.StartedAt != nil {
				t.ElapsedMs = finishedAt.Sub(*t.StartedAt).Milliseconds()
			}
		}
	}
}

// findTraceStepIndexLocked 根据 stepID 查找追踪步骤的索引，未找到返回 -1。
func (r *completionRunner) findTraceStepIndexLocked(stepID string) int {
	if r.assistantMessage.AssistantMessageExtra == nil {
		return -1
	}
	for idx := range r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps {
		if r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[idx].StepID == stepID {
			return idx
		}
	}
	return -1
}

// findPendingApprovalIndexLocked 根据 approvalID 查找待审批记录的索引，未找到返回 -1。
func (r *completionRunner) findPendingApprovalIndexLocked(approvalID string) int {
	if r.assistantMessage.AssistantMessageExtra == nil {
		return -1
	}
	for idx := range r.assistantMessage.AssistantMessageExtra.PendingApprovals {
		if r.assistantMessage.AssistantMessageExtra.PendingApprovals[idx].ApprovalID == approvalID {
			return idx
		}
	}
	return -1
}

// =============================================================================
// 追踪与审批管理 —— 执行追踪步骤和工具审批的增删改
// =============================================================================

// upsertPendingApprovalLocked 新增或更新待审批记录。
func (r *completionRunner) upsertPendingApprovalLocked(summary data_models.ToolApprovalSummary) {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	idx := r.findPendingApprovalIndexLocked(summary.ApprovalID)
	if idx == -1 {
		r.assistantMessage.AssistantMessageExtra.PendingApprovals = append(r.assistantMessage.AssistantMessageExtra.PendingApprovals, summary)
		return
	}
	r.assistantMessage.AssistantMessageExtra.PendingApprovals[idx] = summary
}

// removePendingApprovalLocked 移除指定的待审批记录。
func (r *completionRunner) removePendingApprovalLocked(approvalID string) {
	if r.assistantMessage.AssistantMessageExtra == nil {
		return
	}
	idx := r.findPendingApprovalIndexLocked(approvalID)
	if idx == -1 {
		return
	}
	r.assistantMessage.AssistantMessageExtra.PendingApprovals = append(
		r.assistantMessage.AssistantMessageExtra.PendingApprovals[:idx],
		r.assistantMessage.AssistantMessageExtra.PendingApprovals[idx+1:]...,
	)
}

// appendTraceStepLocked 追加一个新的追踪步骤，并立即持久化。
func (r *completionRunner) appendTraceStepLocked(step data_models.TraceStep) error {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	if step.StartedAt == nil {
		now := time.Now()
		step.StartedAt = &now
	}
	r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps = append(r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps, step)
	r.pendingTraceDelta = append(r.pendingTraceDelta, step)
	r.task.LastOutputAt = step.StartedAt
	return r.persistSnapshotLocked(true)
}

// updateTraceStepLocked 更新已有的追踪步骤（按 StepID 匹配），若不存在则追加。
func (r *completionRunner) updateTraceStepLocked(step data_models.TraceStep) error {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	idx := r.findTraceStepIndexLocked(step.StepID)
	if idx == -1 {
		return r.appendTraceStepLocked(step)
	}
	r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[idx] = step
	r.pendingTraceDelta = append(r.pendingTraceDelta, step)
	now := time.Now()
	r.task.LastOutputAt = &now
	return r.persistSnapshotLocked(true)
}

// startTraceStepLocked 创建并追加一个「运行中」状态的追踪步骤。
func (r *completionRunner) startTraceStepLocked(stepID, parentStepID string, stepType data_models.TraceStepType, title, summary, inputPreview, stage, agentName string, detailBlocks []data_models.TraceDetailBlock, metadata map[string]interface{}) error {
	now := time.Now()
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	r.assistantMessage.AssistantMessageExtra.CurrentStage = stage
	r.assistantMessage.AssistantMessageExtra.CurrentAgent = agentName
	return r.appendTraceStepLocked(data_models.TraceStep{
		StepID:       stepID,
		ParentStepID: parentStepID,
		Type:         stepType,
		Title:        title,
		Summary:      summary,
		InputPreview: inputPreview,
		Status:       data_models.TraceStepStatusRunning,
		AgentName:    agentName,
		StartedAt:    &now,
		DetailBlocks: detailBlocks,
		Metadata:     metadata,
	})
}

// finishTraceStepLocked 标记追踪步骤为已完成，记录耗时和结果。
func (r *completionRunner) finishTraceStepLocked(stepID, summary, outputPreview, stage, agentName string, status data_models.TraceStepStatus, detailBlocks []data_models.TraceDetailBlock, metadata map[string]interface{}) error {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	idx := r.findTraceStepIndexLocked(stepID)
	if idx == -1 {
		return nil
	}
	step := r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[idx]
	now := time.Now()
	if step.StartedAt == nil {
		step.StartedAt = &now
	}
	step.Status = status
	step.Summary = summary
	step.OutputPreview = outputPreview
	step.FinishedAt = &now
	step.ElapsedMs = now.Sub(*step.StartedAt).Milliseconds()
	if detailBlocks != nil {
		step.DetailBlocks = detailBlocks
	}
	if metadata != nil {
		step.Metadata = metadata
	}
	r.assistantMessage.AssistantMessageExtra.CurrentStage = stage
	r.assistantMessage.AssistantMessageExtra.CurrentAgent = agentName
	return r.updateTraceStepLocked(step)
}

// updateCurrentStageLocked 更新当前执行阶段和代理名称。
func (r *completionRunner) updateCurrentStageLocked(stage, agentName string) {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	r.assistantMessage.AssistantMessageExtra.CurrentStage = stage
	r.assistantMessage.AssistantMessageExtra.CurrentAgent = agentName
}

// =============================================================================
// 工具调用生命周期 —— 记录工具调用的开始和结束
// =============================================================================

// startToolUseLocked 记录工具调用开始，创建 ToolUse 记录和对应的追踪步骤。
func (r *completionRunner) startToolUseLocked(toolCtx context.Context, callID, toolName, toolArgs string) error {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	now := time.Now()
	contentPos := len([]rune(strings.TrimRight(r.assistantMessage.Content, "\n")))

	// 查找工具元信息：优先从用户自定义工具中查找，其次从全局注册表查找
	toolID := toolName
	displayName := toolName
	description := ""
	if meta, ok := r.toolMetaByID[toolName]; ok {
		toolID = meta.ID
		displayName = meta.Name
		description = meta.Description
	} else if registeredTool, ok := tools.ToolRouter.GetToolByID(toolName); ok {
		toolID = registeredTool.Id()
		displayName = registeredTool.Name()
		description = registeredTool.Description()
	}

	idx := r.findToolUseIndexLocked(callID)
	if idx == -1 {
		// 新增工具调用记录
		r.assistantMessage.AssistantMessageExtra.ToolUses = append(r.assistantMessage.AssistantMessageExtra.ToolUses, data_models.ToolUse{
			Index:           len(r.assistantMessage.AssistantMessageExtra.ToolUses) + 1,
			CallID:          callID,
			ContentPos:      contentPos,
			ToolID:          toolID,
			ToolName:        displayName,
			ToolDescription: description,
			Status:          data_models.ToolUseStatusRunning,
			StartedAt:       &now,
		})
	} else {
		// 更新已有的工具调用记录（例如从 pending 恢复为 running）
		toolUse := &r.assistantMessage.AssistantMessageExtra.ToolUses[idx]
		toolUse.ToolID = toolID
		toolUse.ToolName = displayName
		toolUse.ToolDescription = description
		toolUse.Status = data_models.ToolUseStatusRunning
		if toolUse.Index == 0 {
			toolUse.Index = idx + 1
		}
		if toolUse.StartedAt == nil {
			toolUse.StartedAt = &now
		}
		if toolUse.ContentPos == 0 {
			toolUse.ContentPos = contentPos
		}
		toolUse.FinishedAt = nil
	}

	r.task.LastOutputAt = &now
	parentStepID, _ := toolCtx.Value(traceParentStepIDContextKey).(string)
	agentName, _ := toolCtx.Value(traceAgentNameContextKey).(string)
	r.assistantMessage.AssistantMessageExtra.CurrentStage = "chat.stage.running_tasks"
	r.assistantMessage.AssistantMessageExtra.CurrentAgent = agentName

	// 如果是子 agent 调用，同步创建 SubAgentTask 记录
	if displayNameFromAgent, isSubAgent := r.customAgentIDs[toolName]; isSubAgent {
		r.upsertSubAgentTaskLocked(callID, toolName, displayNameFromAgent, data_models.ToolUseStatusRunning, toolArgs, "", &now, nil)
	}

	return r.appendTraceStepLocked(data_models.TraceStep{
		StepID:       callID,
		ParentStepID: parentStepID,
		Type:         data_models.TraceStepTypeToolCall,
		Title:        i18n.TCurrent("chat.trace.tool_call_title", map[string]string{"name": displayName}),
		Summary:      description,
		Status:       data_models.TraceStepStatusRunning,
		AgentName:    agentName,
		ToolName:     displayName,
		StartedAt:    &now,
		InputPreview: compactText(toolArgs, 180),
		DetailBlocks: []data_models.TraceDetailBlock{
			{
				Kind:    "tool_args",
				Title:   i18n.TCurrent("chat.trace.tool_parameters", nil),
				Content: toolArgs,
				Format:  data_models.TraceDetailFormatJSON,
			},
		},
		Metadata: func() map[string]interface{} {
			m := map[string]interface{}{
				"tool_id": toolID,
			}
			if _, isSubAgent := r.customAgentIDs[toolName]; isSubAgent {
				m["is_sub_agent"] = true
			}
			return m
		}(),
	})
}

// finishToolUseWithStatusLocked 以指定状态完成工具调用，记录结果和耗时。
func (r *completionRunner) finishToolUseWithStatusLocked(toolCtx context.Context, callID, toolName, toolResult string, toolStatus data_models.ToolUseStatus, traceStatus data_models.TraceStepStatus, runErr error) error {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	now := time.Now()

	idx := r.findToolUseIndexLocked(callID)
	if idx == -1 {
		// 找不到记录时补建一条（防御性处理）
		r.assistantMessage.AssistantMessageExtra.ToolUses = append(r.assistantMessage.AssistantMessageExtra.ToolUses, data_models.ToolUse{
			Index:      len(r.assistantMessage.AssistantMessageExtra.ToolUses) + 1,
			CallID:     callID,
			ContentPos: len([]rune(strings.TrimRight(r.assistantMessage.Content, "\n"))),
		})
		idx = len(r.assistantMessage.AssistantMessageExtra.ToolUses) - 1
	}

	toolUse := &r.assistantMessage.AssistantMessageExtra.ToolUses[idx]
	if toolUse.Index == 0 {
		toolUse.Index = idx + 1
	}
	toolUse.CallID = callID

	// 更新工具元信息
	if toolName != "" {
		if meta, ok := r.toolMetaByID[toolName]; ok {
			toolUse.ToolID = meta.ID
			toolUse.ToolName = meta.Name
			toolUse.ToolDescription = meta.Description
		} else if registeredTool, ok := tools.ToolRouter.GetToolByID(toolName); ok {
			toolUse.ToolID = registeredTool.Id()
			toolUse.ToolName = registeredTool.Name()
			toolUse.ToolDescription = registeredTool.Description()
		} else {
			if toolUse.ToolID == "" {
				toolUse.ToolID = toolName
			}
			if toolUse.ToolName == "" {
				toolUse.ToolName = toolName
			}
		}
	}

	if toolUse.StartedAt == nil {
		toolUse.StartedAt = &now
	}
	toolUse.FinishedAt = &now
	toolUse.ToolResult = toolResult
	toolUse.ElapsedMs = now.Sub(*toolUse.StartedAt).Milliseconds()
	toolUse.Status = toolStatus
	if runErr != nil && toolUse.ToolResult == "" {
		toolUse.ToolResult = runErr.Error()
	}
	r.task.LastOutputAt = &now

	// 如果是子 agent 调用，更新 SubAgentTask 记录
	if _, isSubAgent := r.customAgentIDs[toolName]; isSubAgent {
		r.upsertSubAgentTaskLocked(callID, toolName, "", toolStatus, "", toolResult, nil, &now)
	}

	// 更新对应的追踪步骤
	traceIdx := r.findTraceStepIndexLocked(callID)
	if traceIdx != -1 {
		traceStep := r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[traceIdx]
		if traceStep.StartedAt == nil {
			traceStep.StartedAt = &now
		}
		traceStep.Status = traceStatus
		traceStep.OutputPreview = compactText(toolUse.ToolResult, 240)
		traceStep.Summary = toolUse.ToolDescription
		traceStep.ToolName = toolUse.ToolName
		traceStep.AgentName, _ = toolCtx.Value(traceAgentNameContextKey).(string)
		traceStep.FinishedAt = &now
		traceStep.ElapsedMs = now.Sub(*traceStep.StartedAt).Milliseconds()
		traceStep.DetailBlocks = append(traceStep.DetailBlocks[:0:0], traceStep.DetailBlocks...)
		traceStep.DetailBlocks = append(traceStep.DetailBlocks, data_models.TraceDetailBlock{
			Kind:    "tool_result",
			Title:   i18n.TCurrent("chat.trace.tool_result", nil),
			Content: toolUse.ToolResult,
			Format:  data_models.TraceDetailFormatText,
		})
		if runErr != nil {
			traceStep.DetailBlocks = append(traceStep.DetailBlocks, data_models.TraceDetailBlock{
				Kind:    "tool_result",
				Title:   i18n.TCurrent("chat.trace.error_info", nil),
				Content: runErr.Error(),
				Format:  data_models.TraceDetailFormatText,
			})
		}
		return r.updateTraceStepLocked(traceStep)
	}
	return r.persistSnapshotLocked(true)
}

// finishToolUseLocked 根据错误自动判断状态来完成工具调用。
func (r *completionRunner) finishToolUseLocked(toolCtx context.Context, callID, toolName, toolResult string, runErr error) error {
	toolStatus := data_models.ToolUseStatusDone
	traceStatus := data_models.TraceStepStatusDone
	if runErr != nil {
		toolStatus = data_models.ToolUseStatusError
		traceStatus = data_models.TraceStepStatusError
	}
	return r.finishToolUseWithStatusLocked(toolCtx, callID, toolName, toolResult, toolStatus, traceStatus, runErr)
}

// =============================================================================
// 工具审批 —— 人工审批流程的状态管理
// =============================================================================

// setToolApprovalPendingLocked 将工具调用标记为「等待审批」状态，
// 暂停任务执行直到用户做出审批决定。
func (r *completionRunner) setToolApprovalPendingLocked(toolCtx context.Context, callID string, approval data_models.ToolApproval) error {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}

	summary := approval.Summary()
	r.upsertPendingApprovalLocked(summary)
	r.task.Status = data_models.TaskStatusWaitingApproval
	r.assistantMessage.AssistantMessageExtra.CurrentStage = "chat.stage.awaiting_approval"
	r.assistantMessage.AssistantMessageExtra.CurrentAgent, _ = toolCtx.Value(traceAgentNameContextKey).(string)

	if idx := r.findToolUseIndexLocked(callID); idx != -1 {
		r.assistantMessage.AssistantMessageExtra.ToolUses[idx].Status = data_models.ToolUseStatusAwaitingApproval
		r.assistantMessage.AssistantMessageExtra.ToolUses[idx].ToolResult = approval.Message
	}

	if traceIdx := r.findTraceStepIndexLocked(callID); traceIdx != -1 {
		traceStep := r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[traceIdx]
		traceStep.Status = data_models.TraceStepStatusAwaitingApproval
		traceStep.DetailBlocks = append(traceStep.DetailBlocks[:0:0], traceStep.DetailBlocks...)
		traceStep.DetailBlocks = append(traceStep.DetailBlocks, data_models.TraceDetailBlock{
			Kind:    "approval_request",
			Title:   i18n.TCurrent("chat.trace.confirm_request", nil),
			Content: approval.Message,
			Format:  data_models.TraceDetailFormatMarkdown,
		})
		if traceStep.Metadata == nil {
			traceStep.Metadata = map[string]interface{}{}
		}
		traceStep.Metadata["approval_id"] = approval.ApprovalID
		traceStep.Metadata["approval_status"] = approval.Status
		traceStep.Metadata["approval_decision_required"] = true
		traceStep.Metadata["approval_title"] = approval.Title
		traceStep.Metadata["approval_message"] = approval.Message
		traceStep.Metadata["approval_scope"] = approval.Scope
		r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[traceIdx] = traceStep
		r.pendingTraceDelta = append(r.pendingTraceDelta, traceStep)
	}
	return r.persistSnapshotLocked(true)
}

// resumeApprovedToolLocked 用户批准后恢复工具执行，清除审批状态。
func (r *completionRunner) resumeApprovedToolLocked(toolCtx context.Context, callID string, approval data_models.ToolApproval) error {
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}

	r.removePendingApprovalLocked(approval.ApprovalID)
	r.task.Status = data_models.TaskStatusRunning
	r.assistantMessage.AssistantMessageExtra.CurrentStage = "chat.stage.running_tasks"
	r.assistantMessage.AssistantMessageExtra.CurrentAgent, _ = toolCtx.Value(traceAgentNameContextKey).(string)

	if idx := r.findToolUseIndexLocked(callID); idx != -1 {
		toolUse := &r.assistantMessage.AssistantMessageExtra.ToolUses[idx]
		toolUse.Status = data_models.ToolUseStatusRunning
		toolUse.ToolResult = ""
		toolUse.FinishedAt = nil
	}
	if traceIdx := r.findTraceStepIndexLocked(callID); traceIdx != -1 {
		traceStep := r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[traceIdx]
		traceStep.Status = data_models.TraceStepStatusRunning
		traceStep.FinishedAt = nil
		traceStep.OutputPreview = ""
		if traceStep.Metadata == nil {
			traceStep.Metadata = map[string]interface{}{}
		}
		traceStep.Metadata["approval_id"] = approval.ApprovalID
		traceStep.Metadata["approval_status"] = approval.Status
		traceStep.Metadata["approval_decision_required"] = false
		traceStep.Metadata["approval_decision"] = approval.Decision
		traceStep.DetailBlocks = append(traceStep.DetailBlocks[:0:0], traceStep.DetailBlocks...)
		traceStep.DetailBlocks = append(traceStep.DetailBlocks, data_models.TraceDetailBlock{
			Kind:    "approval_response",
			Title:   i18n.TCurrent("chat.trace.confirm_result", nil),
			Content: i18n.TCurrent("chat.trace.confirm_allowed", nil),
			Format:  data_models.TraceDetailFormatText,
		})
		// 折叠确认请求详情块
		for i := range traceStep.DetailBlocks {
			if traceStep.DetailBlocks[i].Kind == "approval_request" {
				traceStep.DetailBlocks[i].Collapsed = true
			}
		}
		r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[traceIdx] = traceStep
		r.pendingTraceDelta = append(r.pendingTraceDelta, traceStep)
	}
	return r.persistSnapshotLocked(true)
}

// collapseApprovalDetailBlocksLocked 折叠确认请求详情块。
func (r *completionRunner) collapseApprovalDetailBlocksLocked(callID string) {
	traceIdx := r.findTraceStepIndexLocked(callID)
	if traceIdx == -1 {
		return
	}
	step := &r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[traceIdx]
	for i := range step.DetailBlocks {
		if step.DetailBlocks[i].Kind == "approval_request" {
			step.DetailBlocks[i].Collapsed = true
		}
	}
}

// finalizeRunningToolUsesLocked 将所有仍在运行/等待的工具调用和追踪步骤
// 标记为错误状态（用于任务终结时的清理）。
func (r *completionRunner) finalizeRunningToolUsesLocked() {
	if r.assistantMessage.AssistantMessageExtra == nil {
		return
	}
	now := time.Now()
	for idx := range r.assistantMessage.AssistantMessageExtra.ToolUses {
		toolUse := &r.assistantMessage.AssistantMessageExtra.ToolUses[idx]
		if toolUse.Status != data_models.ToolUseStatusRunning &&
			toolUse.Status != data_models.ToolUseStatusPending &&
			toolUse.Status != data_models.ToolUseStatusAwaitingApproval {
			continue
		}
		if toolUse.StartedAt == nil {
			toolUse.StartedAt = &now
		}
		toolUse.FinishedAt = &now
		toolUse.ElapsedMs = now.Sub(*toolUse.StartedAt).Milliseconds()
		toolUse.Status = data_models.ToolUseStatusError
	}
	for idx := range r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps {
		step := &r.assistantMessage.AssistantMessageExtra.ExecutionTrace.Steps[idx]
		if step.Status != data_models.TraceStepStatusRunning &&
			step.Status != data_models.TraceStepStatusPending &&
			step.Status != data_models.TraceStepStatusAwaitingApproval {
			continue
		}
		if step.StartedAt == nil {
			step.StartedAt = &now
		}
		step.FinishedAt = &now
		step.ElapsedMs = now.Sub(*step.StartedAt).Milliseconds()
		step.Status = data_models.TraceStepStatusError
	}
	r.assistantMessage.AssistantMessageExtra.PendingApprovals = nil
}

// =============================================================================
// 工作流切换 —— 直答模式到工作流模式的路由决策
// =============================================================================

// setWorkflowHandoff 记录工作流切换决策（线程安全）。
func (r *completionRunner) setWorkflowHandoff(handoff workflowHandoff) {
	r.handoffMu.Lock()
	defer r.handoffMu.Unlock()
	cloned := handoff
	r.workflowHandoffDecision = &cloned
}

// getWorkflowHandoff 获取工作流切换决策（线程安全），未触发时返回 nil。
func (r *completionRunner) getWorkflowHandoff() *workflowHandoff {
	r.handoffMu.Lock()
	defer r.handoffMu.Unlock()
	if r.workflowHandoffDecision == nil {
		return nil
	}
	cloned := *r.workflowHandoffDecision
	return &cloned
}

// =============================================================================
// 工具中间件 —— 拦截工具调用，处理追踪记录和审批流程
// =============================================================================

// buildToolMiddleware 构建工具调用中间件，拦截每次工具调用以实现：
// 1. 记录工具调用开始/结束到追踪系统
// 2. 对需要审批的工具发起审批流程，阻塞等待用户决定
// 3. 处理审批结果（允许/拒绝/自定义回复）
func (r *completionRunner) buildToolMiddleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(toolCtx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				// 工具调用出错时的统一处理：记录失败并返回错误
				finishWithMiddlewareError := func(err error) (*compose.ToolOutput, error) {
					if input.Name == workflowHandoffToolName {
						return nil, err
					}
					r.mu.Lock()
					finishErr := r.finishToolUseLocked(toolCtx, input.CallID, input.Name, "", err)
					r.mu.Unlock()
					if finishErr != nil {
						return nil, finishErr
					}
					return nil, err
				}

				// 非工作流切换工具：记录调用开始 + 检查是否需要审批
				if input.Name != workflowHandoffToolName {
					r.mu.Lock()
					err := r.startToolUseLocked(toolCtx, input.CallID, input.Name, input.Arguments)
					r.mu.Unlock()
					if err != nil {
						return nil, err
					}

					// 检查工具是否需要用户确认
					if registeredTool, ok := tools.ToolRouter.GetToolByID(input.Name); ok {
						if registeredTool.RequireConfirmation() {
							approvalTool, ok := registeredTool.(tool_approval.ApprovalAwareTool)
							if !ok {
								return finishWithMiddlewareError(fmt.Errorf("tool %s requires confirmation but does not implement BuildApprovalPrompt", input.Name))
							}
							prompt, err := approvalTool.BuildApprovalPrompt(toolCtx, input.Arguments)
							if err != nil {
								return finishWithMiddlewareError(err)
							}

							// 创建审批请求
							requestedAt := time.Now()
							approval := data_models.ToolApproval{
								ApprovalID:           uuid.NewString(),
								TaskUuid:             r.task.TaskUuid,
								ChatUuid:             r.task.ChatUuid,
								AssistantMessageUuid: r.task.AssistantMessageUuid,
								ToolCallID:           input.CallID,
								ToolID:               registeredTool.Id(),
								ToolName:             registeredTool.Name(),
								Status:               data_models.ToolApprovalStatusPending,
								Title:                prompt.Title,
								Message:              prompt.Message,
								Scope:                prompt.Scope,
								ArgumentsJSON:        input.Arguments,
								RequestedAt:          &requestedAt,
							}

							// 注册审批并持久化
							if err := tool_approval.Manager.Register(approval.ApprovalID); err != nil {
								return finishWithMiddlewareError(err)
							}
							if err := r.svc.storage.CreateToolApproval(context.Background(), approval); err != nil {
								tool_approval.Manager.Cancel(approval.ApprovalID)
								return finishWithMiddlewareError(err)
							}

							// 更新状态为等待审批
							r.mu.Lock()
							err = r.setToolApprovalPendingLocked(toolCtx, input.CallID, approval)
							r.mu.Unlock()
							if err != nil {
								tool_approval.Manager.Cancel(approval.ApprovalID)
								return finishWithMiddlewareError(err)
							}

							// 阻塞等待用户审批决定
							waitResult, err := tool_approval.Manager.Wait(toolCtx, approval.ApprovalID)
							if err != nil {
								// 审批超时或取消
								expiredAt := time.Now()
								approval.Status = data_models.ToolApprovalStatusExpired
								approval.Decision = data_models.ToolApprovalDecisionReject
								approval.ResponseComment = err.Error()
								approval.RespondedAt = &expiredAt
								_ = r.svc.storage.SaveToolApproval(context.Background(), approval)
								return finishWithMiddlewareError(err)
							}

							// 处理审批结果
							approval.Status = data_models.ToolApprovalStatusResolved
							approval.Decision = waitResult.Decision
							approval.ResponseComment = waitResult.Comment
							approval.RespondedAt = &waitResult.RespondedAt

							switch waitResult.Decision {
							case data_models.ToolApprovalDecisionAllow:
								// 用户批准：恢复执行
								r.mu.Lock()
								err = r.resumeApprovedToolLocked(toolCtx, input.CallID, approval)
								r.mu.Unlock()
								if err != nil {
									return finishWithMiddlewareError(err)
								}
							case data_models.ToolApprovalDecisionReject, data_models.ToolApprovalDecisionCustom:
								// 用户拒绝或自定义回复：终止工具执行，返回拒绝结果给 LLM
								result := buildApprovalDecisionToolResult(approval.ToolName, waitResult.Decision, waitResult.Comment)
								r.mu.Lock()
								r.removePendingApprovalLocked(approval.ApprovalID)
								r.collapseApprovalDetailBlocksLocked(input.CallID)
								r.task.Status = data_models.TaskStatusRunning
								agentName, _ := toolCtx.Value(traceAgentNameContextKey).(string)
								r.updateCurrentStageLocked("chat.stage.running_tasks", agentName)
								err = r.finishToolUseWithStatusLocked(
									toolCtx,
									input.CallID,
									input.Name,
									result,
									data_models.ToolUseStatusRejected,
									data_models.TraceStepStatusRejected,
									nil,
								)
								r.mu.Unlock()
								if err != nil {
									return nil, err
								}
								return &compose.ToolOutput{Result: result}, nil
							default:
								return finishWithMiddlewareError(fmt.Errorf("unknown approval decision: %s", waitResult.Decision))
							}
						}
					}
				}

				// 如果是子 agent 调用，注入父级上下文，使其内部工具调用能正确嵌套
				execCtx := toolCtx
				if displayName, isSubAgent := r.customAgentIDs[input.Name]; isSubAgent {
					execCtx = context.WithValue(execCtx, traceParentStepIDContextKey, input.CallID)
					execCtx = context.WithValue(execCtx, traceAgentNameContextKey, displayName)
				}

				// 执行工具调用
				output, runErr := next(execCtx, input)
				result := ""
				if output != nil {
					result = output.Result
				}

				// 记录工具调用结束
				if input.Name != workflowHandoffToolName {
					r.mu.Lock()
					err := r.finishToolUseLocked(toolCtx, input.CallID, input.Name, result, runErr)
					r.mu.Unlock()
					if err != nil {
						return nil, err
					}
				}
				if runErr != nil {
					return nil, runErr
				}
				return output, nil
			}
		},
	}
}

func (r *completionRunner) createToolApproval(ctx context.Context, callID, toolName, toolArgs string) (*ApprovalHandle, error) {
	registeredTool, ok := tools.ToolRouter.GetToolByID(toolName)
	if !ok {
		return nil, fmt.Errorf("tool %s not found in registry", toolName)
	}
	approvalTool, ok := registeredTool.(tool_approval.ApprovalAwareTool)
	if !ok {
		return nil, fmt.Errorf("tool %s does not implement BuildApprovalPrompt", toolName)
	}

	prompt, err := approvalTool.BuildApprovalPrompt(ctx, toolArgs)
	if err != nil {
		return nil, err
	}

	requestedAt := time.Now()
	approval := data_models.ToolApproval{
		ApprovalID:           uuid.NewString(),
		TaskUuid:             r.task.TaskUuid,
		ChatUuid:             r.task.ChatUuid,
		AssistantMessageUuid: r.task.AssistantMessageUuid,
		ToolCallID:           callID,
		ToolID:               registeredTool.Id(),
		ToolName:             registeredTool.Name(),
		Status:               data_models.ToolApprovalStatusPending,
		Title:                prompt.Title,
		Message:              prompt.Message,
		Scope:                prompt.Scope,
		ArgumentsJSON:        toolArgs,
		RequestedAt:          &requestedAt,
	}

	if err := tool_approval.Manager.Register(approval.ApprovalID); err != nil {
		return nil, err
	}
	if err := r.svc.storage.CreateToolApproval(context.Background(), approval); err != nil {
		tool_approval.Manager.Cancel(approval.ApprovalID)
		return nil, err
	}

	r.mu.Lock()
	err = r.setToolApprovalPendingLocked(ctx, callID, approval)
	r.mu.Unlock()
	if err != nil {
		tool_approval.Manager.Cancel(approval.ApprovalID)
		return nil, err
	}

	return &ApprovalHandle{
		ApprovalID: approval.ApprovalID,
		ToolCallID: callID,
		ToolName:   toolName,
	}, nil
}

func (r *completionRunner) handleApprovalDecision(ctx context.Context, callID, toolName string, waitResult tool_approval.WaitResult) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	approval := data_models.ToolApproval{
		ApprovalID:      waitResult.ApprovalID,
		Status:          data_models.ToolApprovalStatusResolved,
		Decision:        waitResult.Decision,
		ResponseComment: waitResult.Comment,
	}
	respondedAt := waitResult.RespondedAt
	approval.RespondedAt = &respondedAt

	switch waitResult.Decision {
	case data_models.ToolApprovalDecisionAllow:
		err := r.resumeApprovedToolLocked(ctx, callID, approval)
		if err != nil {
			return "", err
		}
		return "", nil

	case data_models.ToolApprovalDecisionReject, data_models.ToolApprovalDecisionCustom:
		result := buildApprovalDecisionToolResult(toolName, waitResult.Decision, waitResult.Comment)
		r.removePendingApprovalLocked(approval.ApprovalID)
		r.collapseApprovalDetailBlocksLocked(callID)
		r.task.Status = data_models.TaskStatusRunning
		agentName, _ := ctx.Value(traceAgentNameContextKey).(string)
		r.updateCurrentStageLocked("chat.stage.running_tasks", agentName)
		err := r.finishToolUseWithStatusLocked(
			ctx,
			callID,
			toolName,
			result,
			data_models.ToolUseStatusRejected,
			data_models.TraceStepStatusRejected,
			nil,
		)
		if err != nil {
			return "", err
		}
		return result, nil

	default:
		return "", fmt.Errorf("unknown approval decision: %s", waitResult.Decision)
	}
}

// =============================================================================
// 任务执行 —— 完整的任务生命周期管理
// =============================================================================

// finalizeTaskTerminal 执行任务终结逻辑，保证仅执行一次（通过 terminalEventEmitted 原子标志）。
// 负责：关闭所有运行中的工具调用、设置最终状态、持久化、发射终结事件、
// 以及为新对话自动生成标题。
func (r *completionRunner) finalizeTaskTerminal(finishReason, finishError string) {
	if !r.terminalEventEmitted.CompareAndSwap(false, true) {
		return
	}

	var finalTaskSnapshot data_models.Task
	var finalAssistantSnapshot data_models.Message
	var persistErr error
	var shouldGenTitle bool

	r.mu.Lock()
	if r.assistantMessage.AssistantMessageExtra == nil {
		r.assistantMessage.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	r.finalizeRunningToolUsesLocked()
	r.assistantMessage.AssistantMessageExtra.CurrentStage = "chat.stage.finished"
	r.assistantMessage.AssistantMessageExtra.FinishReason = finishReason
	r.assistantMessage.AssistantMessageExtra.FinishError = finishError

	finishedAt := time.Now()
	r.task.FinishReason = finishReason
	r.task.FinishError = finishError
	r.task.FinishedAt = &finishedAt
	switch finishReason {
	case "done":
		r.task.Status = data_models.TaskStatusCompleted
	case "user stop":
		r.task.Status = data_models.TaskStatusStopped
	default:
		r.task.Status = data_models.TaskStatusFailed
	}

	finalTaskSnapshot = r.task
	finalAssistantSnapshot = r.cloneAssistantMessageLocked()
	if err := r.svc.storage.SaveOrUpdateMessage(context.Background(), r.assistantMessage); err != nil {
		persistErr = err
	} else if err := r.svc.storage.SaveTask(context.Background(), r.task); err != nil {
		persistErr = err
	}
	shouldGenTitle = r.isNewChat && finishError == ""
	r.mu.Unlock()

	if persistErr != nil {
		logger.Error("finalizeTaskTerminal persist error: ", persistErr)
	}
	r.svc.emitTaskEvent(finalTaskSnapshot, finalAssistantSnapshot, nil)
	if r.svc.plugins != nil {
		go r.svc.plugins.RunAfterLLMSend(context.Background(), plugins.AfterLLMSendPayload{
			ChatUUID:      r.chatUuid,
			MessageUUID:   r.assistantMessageUuid,
			FinishReason:  finishReason,
			FinishError:   finishError,
			AssistantText: finalAssistantSnapshot.Content,
		})
	}

	_ = r.fsm.TransitionToTerminal(StateCompleted, finishReason, finishError)
	if r.eventBus != nil {
		r.eventBus.Publish(AgentEvent{
			Type:    EventTerminal,
			Payload: map[string]string{"finish_reason": finishReason, "finish_error": finishError},
		})
	}

	// 新对话且无错误时，异步生成对话标题
	if shouldGenTitle {
		go func() {
			_, titleErr := r.svc.genChatTitle(context.Background(), r.chatUuid, *r.providerModel, true)
			if titleErr != nil {
				logger.Error("gen chat title error", titleErr)
			}
		}()
	}

}

// failWithError 处理执行错误：区分用户主动取消和真正的错误。
func (r *completionRunner) failWithError(err error, runCtx context.Context) {
	if r.userStopped.Load() || errors.Is(runCtx.Err(), context.Canceled) {
		r.finalizeTaskTerminal("user stop", "")
		return
	}
	if err != nil {
		logger.Error("failWithError: ", err)
		r.finalizeTaskTerminal("error", err.Error())
	}
}

// appendContentLocked 追加流式输出的内容和推理内容（节流持久化）。
func (r *completionRunner) appendContentLocked(content, reasoning string) error {
	r.assistantMessage.Content += content
	r.assistantMessage.ReasoningContent += reasoning
	lastOutputAt := time.Now()
	r.task.LastOutputAt = &lastOutputAt
	return r.persistSnapshotThrottledLocked(true)
}

func shouldAppendStreamChunk(content, reasoning string) bool {
	return content != "" || reasoning != ""
}

// resetStateLocked 重置助手消息状态（用于从直答模式切换到工作流模式前）。
func (r *completionRunner) resetStateLocked() {
	resetDirectAssistantState(&r.assistantMessage)
}

// runSinglePassEntry 执行直答模式：向 LLM 发送消息并流式接收响应。
// 在接收过程中持续检测工作流切换信号，一旦检测到则提前退出。
func (r *completionRunner) runSinglePassEntry(runCtx context.Context) error {
	entryCtx, entryCancel := context.WithCancel(runCtx)
	defer entryCancel()
	entryCtx = context.WithValue(entryCtx, traceAgentNameContextKey, "MainAgent")

	entryMessages := append([]schema.Message{{
		Role:    schema.System,
		Content: r.localizedPrompts.EntrySystem,
	}}, r.schemaMessages...)

	iter, err := r.provider.AgentCompletions(entryCtx, entryMessages)
	if err != nil {
		logger.Error("get agent completions error", err)
		return err
	}

	for {
		// 检查是否已触发工作流切换
		if r.getWorkflowHandoff() != nil {
			entryCancel()
			break
		}

		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mo := event.Output.MessageOutput

		// 工具调用消息：检查工作流切换后跳过
		if mo.Role == schema.Tool {
			if r.getWorkflowHandoff() != nil {
				entryCancel()
				break
			}
			continue
		}
		if mo.Role != schema.Assistant {
			continue
		}

		// 处理单个消息块（流式或非流式）
		handleChunk := func(msg *schema.Message) error {
			if msg == nil {
				return nil
			}
			if !shouldAppendStreamChunk(msg.Content, msg.ReasoningContent) {
				return nil
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			return r.appendContentLocked(msg.Content, msg.ReasoningContent)
		}

		// 流式响应：逐块接收
		if mo.IsStreaming && mo.MessageStream != nil {
			for {
				msg, streamErr := mo.MessageStream.Recv()
				if streamErr == io.EOF {
					break
				}
				if streamErr != nil {
					return streamErr
				}
				if err := handleChunk(msg); err != nil {
					return err
				}
				if r.getWorkflowHandoff() != nil {
					entryCancel()
					break
				}
			}
			mo.MessageStream.Close()
			continue
		}

		// 非流式响应
		if err := handleChunk(mo.Message); err != nil {
			return err
		}
	}
	return nil
}

// executeWorkflow 执行工作流模式，包含完整的多阶段流程：
// 1. 路由分类（记录切换决策）
// 2. 规划（生成任务执行计划）
// 3. 批量执行（按依赖关系分批并行执行子任务）
// 4. 综合（将所有子任务输出整合为最终答案）
// 5. 审核（检查答案是否满足目标）
// 6. 重试（审核不通过时重新执行受影响的任务）
// 7. 终结（输出最终答案）
func (r *completionRunner) executeWorkflow(runCtx context.Context, handoff *workflowHandoff) error {
	userRequest := r.inputMessage.Content
	hasAttachedFiles := r.inputMessage.UserMessageExtra != nil && len(r.inputMessage.UserMessageExtra.Files) > 0
	originalUserMessage := findLatestUserContextMessage(r.schemaMessages)

	// 根据是否有附件选择不同的提示文本
	planningSummary := "PlannerAgent 正在生成执行计划"
	planningInputLabel := "任务请求"
	synthesisSummary := "SynthesizerAgent 正在整合所有子任务产出"
	reviewSummaryText := "ReviewerAgent 正在检查答案是否满足目标"
	if hasAttachedFiles {
		planningSummary = "PlannerAgent 正在基于原始多模态输入生成执行计划"
		planningInputLabel = "任务请求（沿用原始多模态输入）"
		synthesisSummary = "SynthesizerAgent 正在结合原始多模态输入整合结果"
		reviewSummaryText = "ReviewerAgent 正在结合原始多模态输入审核答案"
	}

	// ---- 阶段 1：记录路由分类 ----
	if handoff != nil {
		r.mu.Lock()
		preserveWorkflowPreface(&r.assistantMessage)
		r.resetStateLocked()
		err := r.persistSnapshotLocked(false)
		if err == nil {
			r.assistantMessage.AssistantMessageExtra.RouteType = data_models.RouteTypeWorkflow
			summary := strings.TrimSpace(handoff.Summary)
			if summary == "" {
				summary = handoff.Reason
			}
			err = r.startTraceStepLocked("workflow_handoff", "", data_models.TraceStepTypeClassify, i18n.TCurrent("chat.trace.workflow_handoff", nil), summary, userRequest, "chat.stage.classify", "MainRouterAgent", []data_models.TraceDetailBlock{
				{Kind: "input", Title: i18n.TCurrent("chat.trace.user_input", nil), Content: userRequest, Format: data_models.TraceDetailFormatText},
				{Kind: "review", Title: i18n.TCurrent("chat.trace.handoff_result", nil), Content: fmt.Sprintf("{\"reason\":%q,\"summary\":%q,\"rule_name\":%q}", handoff.Reason, handoff.Summary, handoff.RuleName), Format: data_models.TraceDetailFormatJSON},
			}, map[string]interface{}{
				"route_source": handoff.Source,
				"rule_name":    handoff.RuleName,
			})
		}
		if err == nil {
			err = r.finishTraceStepLocked("workflow_handoff", handoff.Reason, handoff.Summary, "chat.stage.classify", "MainRouterAgent", data_models.TraceStepStatusDone, []data_models.TraceDetailBlock{
				{Kind: "input", Title: i18n.TCurrent("chat.trace.user_input", nil), Content: userRequest, Format: data_models.TraceDetailFormatText},
				{Kind: "review", Title: i18n.TCurrent("chat.trace.handoff_result", nil), Content: fmt.Sprintf("{\"reason\":%q,\"summary\":%q,\"rule_name\":%q}", handoff.Reason, handoff.Summary, handoff.RuleName), Format: data_models.TraceDetailFormatJSON},
			}, map[string]interface{}{
				"route_source": handoff.Source,
				"rule_name":    handoff.RuleName,
			})
		}
		r.mu.Unlock()
		if err != nil {
			return err
		}
	}

	// ---- 阶段 2：生成执行计划 ----
	r.mu.Lock()
	if err := r.startTraceStepLocked("plan", "", data_models.TraceStepTypePlan, i18n.TCurrent("chat.trace.plan_title", nil), planningSummary, userRequest, "chat.stage.plan", "PlannerAgent", []data_models.TraceDetailBlock{
		{Kind: "input", Title: planningInputLabel, Content: userRequest, Format: data_models.TraceDetailFormatText},
	}, nil); err != nil {
		r.mu.Unlock()
		return err
	}
	r.mu.Unlock()

	plan, err := generateWorkflowPlan(runCtx, r.provider, userRequest, r.schemaMessages, r.agentTools)
	if err != nil {
		return err
	}

	taskTitles := make([]string, 0, len(plan.Tasks))
	for _, item := range plan.Tasks {
		taskTitles = append(taskTitles, item.Title)
	}

	r.mu.Lock()
	err = r.finishTraceStepLocked("plan", i18n.Sprintf(i18n.CurrentLocale(), "chat.trace.plan_done", len(plan.Tasks)), strings.Join(taskTitles, " | "), "chat.stage.plan", "PlannerAgent", data_models.TraceStepStatusDone, []data_models.TraceDetailBlock{
		{Kind: "plan", Title: i18n.TCurrent("chat.trace.plan_complete", nil), Content: formatWorkflowPlanForTrace(plan), Format: data_models.TraceDetailFormatMarkdown},
	}, map[string]interface{}{
		"goal":                plan.Goal,
		"completion_criteria": plan.CompletionCriteria,
	})
	r.mu.Unlock()
	if err != nil {
		return err
	}

	// ---- 阶段 3：批量执行子任务 ----
	results := map[string]workflowTaskResult{}
	if err := r.executeBatches(runCtx, plan, results, nil, "", originalUserMessage); err != nil {
		return err
	}

	// ---- 阶段 4-6：综合、审核、重试循环 ----
	maxRetries := r.workflowConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}
	reviewFeedback := ""
	draft := ""
	review := reviewDecision{}
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 综合阶段
		r.mu.Lock()
		err := r.startTraceStepLocked(fmt.Sprintf("synthesize_%d", attempt), "", data_models.TraceStepTypeSynthesize, i18n.TCurrent("chat.trace.synthesize_title", nil), synthesisSummary, userRequest, "chat.stage.synthesize", "SynthesizerAgent", []data_models.TraceDetailBlock{
			{Kind: "input", Title: planningInputLabel, Content: userRequest, Format: data_models.TraceDetailFormatText},
		}, nil)
		r.mu.Unlock()
		if err != nil {
			return err
		}

		draft, err = synthesizeWorkflowAnswer(runCtx, r.provider, userRequest, originalUserMessage, plan, results, reviewFeedback)
		if err != nil {
			return err
		}

		r.mu.Lock()
		err = r.finishTraceStepLocked(fmt.Sprintf("synthesize_%d", attempt), i18n.TCurrent("chat.trace.candidate_generated", nil), compactText(draft, 240), "chat.stage.synthesize", "SynthesizerAgent", data_models.TraceStepStatusDone, []data_models.TraceDetailBlock{
			{Kind: "output", Title: i18n.TCurrent("chat.trace.candidate_answer", nil), Content: draft, Format: data_models.TraceDetailFormatMarkdown},
		}, nil)
		r.mu.Unlock()
		if err != nil {
			return err
		}

		// 审核阶段
		r.mu.Lock()
		err = r.startTraceStepLocked(fmt.Sprintf("review_%d", attempt), "", data_models.TraceStepTypeReview, i18n.TCurrent("chat.trace.review_title", nil), reviewSummaryText, draft, "chat.stage.review", "ReviewerAgent", []data_models.TraceDetailBlock{
			{Kind: "output", Title: "待审核答案", Content: draft, Format: data_models.TraceDetailFormatMarkdown},
		}, nil)
		r.mu.Unlock()
		if err != nil {
			return err
		}

		review, err = reviewWorkflowAnswer(runCtx, r.provider, userRequest, originalUserMessage, plan, results, draft)
		if err != nil {
			return err
		}

		reviewSummary := "审核通过"
		reviewStatus := data_models.TraceStepStatusDone
		if !review.Approved {
			reviewSummary = strings.Join(review.Issues, "；")
			reviewStatus = data_models.TraceStepStatusError
		}

		r.mu.Lock()
		err = r.finishTraceStepLocked(fmt.Sprintf("review_%d", attempt), reviewSummary, compactText(review.RetryInstructions, 240), "chat.stage.review", "ReviewerAgent", reviewStatus, []data_models.TraceDetailBlock{
			{Kind: "review", Title: i18n.TCurrent("chat.trace.review_result", nil), Content: formatReviewDecisionForTrace(review), Format: data_models.TraceDetailFormatJSON},
		}, map[string]interface{}{
			"approved":           review.Approved,
			"affected_task_ids":  review.AffectedTaskIDs,
			"retry_instructions": review.RetryInstructions,
		})
		r.mu.Unlock()
		if err != nil {
			return err
		}

		if review.Approved {
			break
		}
		if attempt == maxRetries-1 {
			break
		}

		// 重试阶段：重新执行受影响的子任务
		retryStepID := fmt.Sprintf("retry_%d", attempt+1)
		r.mu.Lock()
		r.assistantMessage.AssistantMessageExtra.RetryCount = attempt + 1
		err = r.startTraceStepLocked(retryStepID, "", data_models.TraceStepTypeRetry, i18n.Sprintf(i18n.CurrentLocale(), "chat.trace.retry_title", attempt+1), strings.Join(review.Issues, "；"), review.RetryInstructions, "chat.stage.retry", "MainRouterAgent", []data_models.TraceDetailBlock{
			{Kind: "retry", Title: i18n.TCurrent("chat.trace.retry_reason", nil), Content: strings.Join(review.Issues, "\n"), Format: data_models.TraceDetailFormatMarkdown},
			{Kind: "retry", Title: i18n.TCurrent("chat.trace.retry_instruction", nil), Content: review.RetryInstructions, Format: data_models.TraceDetailFormatText},
		}, map[string]interface{}{
			"retry_instructions": review.RetryInstructions,
		})
		r.mu.Unlock()
		if err != nil {
			return err
		}

		filterIDs := map[string]struct{}{}
		if len(review.AffectedTaskIDs) > 0 {
			for _, taskID := range review.AffectedTaskIDs {
				filterIDs[taskID] = struct{}{}
			}
		}
		if len(filterIDs) == 0 {
			filterIDs = nil
		}
		if err := r.executeBatches(runCtx, plan, results, filterIDs, review.RetryInstructions, originalUserMessage); err != nil {
			return err
		}
		reviewFeedback = strings.Join(review.Issues, "；") + "\n" + review.RetryInstructions

		r.mu.Lock()
		err = r.finishTraceStepLocked(retryStepID, i18n.TCurrent("chat.trace.retry_done", nil), review.RetryInstructions, "chat.stage.retry", "MainRouterAgent", data_models.TraceStepStatusDone, []data_models.TraceDetailBlock{
			{Kind: "retry", Title: i18n.TCurrent("chat.trace.retry_result", nil), Content: formatRetrySummaryForTrace(review), Format: data_models.TraceDetailFormatMarkdown},
		}, map[string]interface{}{
			"affected_task_ids": review.AffectedTaskIDs,
		})
		r.mu.Unlock()
		if err != nil {
			return err
		}
	}

	// ---- 阶段 7：终结，输出最终答案 ----
	finalSummary := "答案已通过审核"
	if !review.Approved {
		finalSummary = "达到最大重试次数，输出当前最佳答案"
		draft = strings.TrimSpace(draft + "\n\n注意：系统已进行一次自动修正，但仍建议你根据上面的内容做最终确认。")
	}

	r.mu.Lock()
	err = r.startTraceStepLocked("finalize_workflow", "", data_models.TraceStepTypeFinalize, i18n.TCurrent("chat.trace.finalize_title", nil), finalSummary, draft, "chat.stage.finished", "MainRouterAgent", []data_models.TraceDetailBlock{
		{Kind: "output", Title: i18n.TCurrent("chat.trace.final_draft", nil), Content: draft, Format: data_models.TraceDetailFormatMarkdown},
	}, nil)
	if err == nil {
		r.assistantMessage.Content = draft
		err = r.persistSnapshotLocked(true)
	}
	if err == nil {
		err = r.finishTraceStepLocked("finalize_workflow", finalSummary, compactText(draft, 240), "chat.stage.finished", "MainRouterAgent", data_models.TraceStepStatusDone, []data_models.TraceDetailBlock{
			{Kind: "output", Title: i18n.TCurrent("chat.trace.final_answer", nil), Content: draft, Format: data_models.TraceDetailFormatMarkdown},
		}, map[string]interface{}{
			"approved": review.Approved,
		})
	}
	r.mu.Unlock()
	return err
}

// executeBatches 按依赖关系将任务分批，每批内并行执行。
// filterIDs 不为 nil 时仅执行指定的任务（用于重试场景）。
func (r *completionRunner) executeBatches(runCtx context.Context, plan workflowPlan, results map[string]workflowTaskResult, filterIDs map[string]struct{}, retryInstructions string, originalUserMessage *schema.Message) error {
	batches := batchTasksByDependencies(plan.Tasks, filterIDs)
	toolMiddleware := r.buildToolMiddleware()

	r.mu.Lock()
	retryCount := r.assistantMessage.AssistantMessageExtra.RetryCount
	r.mu.Unlock()

	for batchIndex, batch := range batches {
		// 快照当前已有的结果，供本批任务引用
		priorResults := make(map[string]workflowTaskResult, len(results))
		for key, value := range results {
			priorResults[key] = value
		}

		type taskRun struct {
			task   workflowPlanTask
			result workflowTaskResult
			err    error
		}
		resultCh := make(chan taskRun, len(batch))
		var wg sync.WaitGroup

		for _, baseTask := range batch {
			taskForRun := baseTask
			if strings.TrimSpace(retryInstructions) != "" {
				taskForRun.Description += "\n补充修正要求：" + retryInstructions
			}
			wg.Add(1)
			go func(taskItem workflowPlanTask, batchNo int) {
				defer wg.Done()

				// 记录任务分发追踪步骤
				dispatchStepID := fmt.Sprintf("dispatch_%s_%d", taskItem.ID, retryCount)
				r.mu.Lock()
				dispatchErr := r.startTraceStepLocked(dispatchStepID, "", data_models.TraceStepTypeDispatch, i18n.TCurrent("chat.trace.dispatch_task", nil), fmt.Sprintf("第 %d 批：%s", batchNo+1, taskItem.Title), taskItem.Description, "chat.stage.running_tasks", "MainRouterAgent", []data_models.TraceDetailBlock{
					{Kind: "plan", Title: i18n.TCurrent("chat.trace.dispatch_content", nil), Content: formatDispatchedTaskForTrace(taskItem, batchNo+1), Format: data_models.TraceDetailFormatMarkdown},
				}, map[string]interface{}{
					"task_id": taskItem.ID,
				})
				if dispatchErr == nil {
					dispatchErr = r.finishTraceStepLocked(dispatchStepID, i18n.TCurrent("chat.trace.dispatch_done", nil), taskItem.Description, "chat.stage.running_tasks", "MainRouterAgent", data_models.TraceStepStatusDone, []data_models.TraceDetailBlock{
						{Kind: "plan", Title: i18n.TCurrent("chat.trace.dispatch_content", nil), Content: formatDispatchedTaskForTrace(taskItem, batchNo+1), Format: data_models.TraceDetailFormatMarkdown},
					}, map[string]interface{}{
						"task_id": taskItem.ID,
					})
				}
				r.mu.Unlock()
				if dispatchErr != nil {
					resultCh <- taskRun{task: taskItem, err: dispatchErr}
					return
				}

				// 记录代理执行追踪步骤
				agentStepID := fmt.Sprintf("agent_%s_retry_%d", taskItem.ID, retryCount)
				r.mu.Lock()
				agentErr := r.startTraceStepLocked(agentStepID, "", data_models.TraceStepTypeAgentRun, taskItem.Title, taskItem.Description, buildWorkerPrompt(plan, taskItem, priorResults), "chat.stage.running_tasks", taskItem.SuggestedAgent, buildAgentTraceDetails(plan, taskItem, priorResults, retryInstructions), map[string]interface{}{
					"task_id":         taskItem.ID,
					"expected_output": taskItem.ExpectedOutput,
				})
				r.mu.Unlock()
				if agentErr != nil {
					resultCh <- taskRun{task: taskItem, err: agentErr}
					return
				}

				// 执行子任务
				execResult, execErr := executePlanTask(runCtx, r.provider, taskItem, plan, priorResults, originalUserMessage, r.agentTools, toolMiddleware, agentStepID)

				// 记录执行结果
				r.mu.Lock()
				finishStatus := data_models.TraceStepStatusDone
				summary := i18n.TCurrent("chat.trace.agent_finished", nil)
				outputPreview := compactText(execResult.Output, 240)
				metadata := map[string]interface{}{
					"task_id":    taskItem.ID,
					"used_tools": execResult.UsedTools,
				}
				if execErr != nil {
					finishStatus = data_models.TraceStepStatusError
					summary = execErr.Error()
				}
				agentErr = r.finishTraceStepLocked(agentStepID, summary, outputPreview, "chat.stage.running_tasks", taskItem.SuggestedAgent, finishStatus, buildAgentResultTraceDetails(plan, taskItem, priorResults, retryInstructions, execResult, execErr), metadata)
				r.mu.Unlock()
				if agentErr != nil && execErr == nil {
					execErr = agentErr
				}
				resultCh <- taskRun{task: taskItem, result: execResult, err: execErr}
			}(taskForRun, batchIndex)
		}

		wg.Wait()
		close(resultCh)
		var multiErr error
		var consecutiveFailures int
		for item := range resultCh {
			if item.err != nil {
				consecutiveFailures++
				if !r.workflowConfig.AllowPartialSuccess {
					return item.err
				}
				if r.workflowConfig.MaxConsecutiveFailures > 0 && consecutiveFailures > r.workflowConfig.MaxConsecutiveFailures {
					return fmt.Errorf("too many consecutive failures in batch: %w", item.err)
				}
				results[item.task.ID] = workflowTaskResult{
					TaskID: item.task.ID,
					Title:  item.task.Title + " (执行失败)",
					Output: fmt.Sprintf("任务执行失败: %s", item.err.Error()),
				}
				multiErr = errors.Join(multiErr, item.err)
				consecutiveFailures = 0
				continue
			}
			results[item.task.ID] = item.result
		}
		if multiErr != nil && !r.workflowConfig.AllowPartialSuccess {
			return multiErr
		}
	}
	return nil
}

// run 是传递给 tasker.Manager.StartTask 的回调函数，
// 包含完整的任务执行生命周期：初始化 → 直答/工作流执行 → 终结。
func (r *completionRunner) run(userStop <-chan struct{}) {
	now := time.Now()

	// 使用 ContextManager 创建带超时的任务上下文
	if r.ctxManager == nil {
		r.ctxManager = NewContextManager(DefaultContextConfig())
	}
	runCtx, cancel := r.ctxManager.NewTaskContext(context.Background())
	defer cancel()

	// 启动独立 I/O 写入协程（可选，当前默认不启用）
	// r.persistWriter.Start(runCtx)
	// defer r.persistWriter.Stop()

	// 监听用户主动停止信号
	go func() {
		<-userStop
		r.userStopped.Store(true)
		r.fsm.TransitionToTerminal(StateStopped, "user stop", "")
		cancel()
	}()

	// FSM: idle → preparing
	_ = r.fsm.Transition(StatePreparing, "task start", "")

	// 标记任务为运行中
	r.mu.Lock()
	r.task.Status = data_models.TaskStatusRunning
	r.task.StartedAt = &now
	r.updateCurrentStageLocked("chat.stage.preparing", "")
	saveErr := r.svc.storage.SaveTask(context.Background(), r.task)
	r.mu.Unlock()
	if saveErr != nil {
		logger.Error("save task running status error", saveErr)
	}

	// FSM: preparing → running
	_ = r.fsm.Transition(StateRunning, "task running", "")

	// defer 兜底：确保任务一定会发射终结事件（即使发生 panic 恢复后）
	defer func() {
		if rec := recover(); rec != nil {
			stack := rdebug.Stack()
			logger.Error("completionRunner panic recovered:", rec, "\nstack:", string(stack))
			r.fsm.TransitionToTerminal(StateFailed, "panic", fmt.Sprintf("%v", rec))
		}
		if r.cleanupTools != nil {
			r.cleanupTools()
		}
		if r.fsm.IsTerminal() && r.terminalEventEmitted.Load() {
			return
		}

		finishReason := "error"
		finishError := i18n.TCurrent("errors.internal", nil)
		r.mu.Lock()
		if r.assistantMessage.AssistantMessageExtra != nil {
			if r.assistantMessage.AssistantMessageExtra.FinishReason != "" {
				finishReason = r.assistantMessage.AssistantMessageExtra.FinishReason
			}
			if r.assistantMessage.AssistantMessageExtra.FinishError != "" {
				finishError = r.assistantMessage.AssistantMessageExtra.FinishError
			} else if finishReason == "done" || finishReason == "user stop" {
				finishError = ""
			}
		}
		r.mu.Unlock()
		r.finalizeTaskTerminal(finishReason, finishError)
	}()

	// ---- 主执行流程 ----
	// 检查是否强制进入工作流模式（例如用户选择了 agent）
	if guard := shouldForceWorkflow(r.inputMessage); guard.Force {
		_ = r.fsm.Transition(StateWorkflowPlanning, "guard rule forced workflow", "")
		if err := r.executeWorkflow(runCtx, &workflowHandoff{
			Reason:   guard.Reason,
			Summary:  guard.Reason,
			Source:   routeSourceGuardRule,
			RuleName: guard.RuleName,
		}); err != nil {
			r.fsm.TransitionToTerminal(StateFailed, "workflow error", err.Error())
			r.failWithError(err, runCtx)
			return
		}
	} else {
		// 默认路径：先尝试直答，LLM 可能在直答过程中触发工作流切换
		r.mu.Lock()
		r.resetStateLocked()
		saveErr := r.persistSnapshotLocked(true)
		r.mu.Unlock()
		if saveErr != nil {
			r.fsm.TransitionToTerminal(StateFailed, "persist error", saveErr.Error())
			r.failWithError(saveErr, runCtx)
			return
		}

		_ = r.fsm.Transition(StateStreaming, "entry direct answer", "")
		if err := r.runSinglePassEntry(runCtx); err != nil {
			r.failWithError(err, runCtx)
			return
		}

		// 检查直答过程中是否触发了工作流切换
		handoff := r.getWorkflowHandoff()
		if handoff != nil {
			_ = r.fsm.Transition(StateWorkflowPlanning, "workflow handoff", "")
			if err := r.executeWorkflow(runCtx, handoff); err != nil {
				r.fsm.TransitionToTerminal(StateFailed, "workflow error", err.Error())
				r.failWithError(err, runCtx)
				return
			}
		} else {
			// 直答完成：清理并持久化最终内容
			r.mu.Lock()
			r.assistantMessage.Content = strings.TrimSpace(r.assistantMessage.Content)
			if r.assistantMessage.Content == "" {
				r.assistantMessage.Content = "抱歉，我暂时没有生成内容，请重试。"
			}
			saveErr = r.persistSnapshotLocked(true)
			r.mu.Unlock()
			if saveErr != nil {
				r.fsm.TransitionToTerminal(StateFailed, "final persist error", saveErr.Error())
				r.failWithError(saveErr, runCtx)
				return
			}
		}
	}

	// ---- 记忆系统：异步策展（工作流路径不触发） ----
	if r.getWorkflowHandoff() == nil {
		msgsCopy := make([]schema.Message, len(r.schemaMessages))
		copy(msgsCopy, r.schemaMessages)
		providerModel := r.providerModel

		go r.safeMemoryOp("encode", func() {
			r.svc.encodeMemoriesAsync(providerModel, msgsCopy)
		})
	}

	_ = r.fsm.TransitionToTerminal(StateCompleted, "done", "")
	r.finalizeTaskTerminal("done", "")
}
