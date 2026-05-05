package service

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/wrapper_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/agents"
	llmtools "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/tools"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/logger"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/skills"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/tasker"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils/event"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils/ierror"
)

// ChatList 聊天列表
func (s *Service) ChatList(ctx context.Context, offset, limit int, keyword *string, isCollection bool) (*view_models.ChatList, error) {
	chats, total, err := s.storage.GetChats(ctx, offset, limit, keyword, isCollection)
	if err != nil {
		return nil, ierror.NewError(err)
	}

	return &view_models.ChatList{
		Lists: chats,
		Total: total,
	}, nil
}

// ChatInfo 对话信息
func (s *Service) ChatInfo(ctx context.Context, chatUuid string) (*view_models.Chat, error) {
	chat, err := s.storage.GetChat(ctx, chatUuid)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	return chat, nil
}

// ChatMessages 聊天消息
func (s *Service) ChatMessages(ctx context.Context, chatUuid string, offset, limit int) (*view_models.MessageList, error) {
	dataMessages, total, err := s.storage.GetMessage(ctx, chatUuid, offset, limit)
	if err != nil {
		return nil, ierror.NewError(err)
	}

	var messages []view_models.Message
	for _, item := range dataMessages {
		messages = append(messages, item)
	}

	return &view_models.MessageList{
		Messages: messages,
		Total:    total,
	}, nil
}

func preserveWorkflowPreface(message *data_models.Message) {
	if message == nil {
		return
	}
	if message.AssistantMessageExtra == nil {
		message.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	if message.AssistantMessageExtra.PrefaceContent == "" && strings.TrimSpace(message.Content) != "" {
		message.AssistantMessageExtra.PrefaceContent = message.Content
	}
	if message.AssistantMessageExtra.PrefaceReasoningContent == "" && strings.TrimSpace(message.ReasoningContent) != "" {
		message.AssistantMessageExtra.PrefaceReasoningContent = message.ReasoningContent
	}
}

func resetDirectAssistantState(message *data_models.Message) {
	if message == nil {
		return
	}
	if message.AssistantMessageExtra == nil {
		message.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}
	message.Content = ""
	message.ReasoningContent = ""
	message.AssistantMessageExtra.RouteType = ""
	message.AssistantMessageExtra.CurrentStage = ""
	message.AssistantMessageExtra.CurrentAgent = ""
	message.AssistantMessageExtra.PendingApprovals = nil
	message.AssistantMessageExtra.ExecutionTrace = data_models.ExecutionTrace{Steps: []data_models.TraceStep{}}
	message.AssistantMessageExtra.FinishError = ""
}

func buildApprovalDecisionToolResult(toolName string, decision data_models.ToolApprovalDecision, comment string) string {
	switch decision {
	case data_models.ToolApprovalDecisionAllow:
		return i18n.Sprintf(i18n.CurrentLocale(), "chat.approval.allow_result", toolName)
	case data_models.ToolApprovalDecisionCustom:
		if strings.TrimSpace(comment) == "" {
			return i18n.Sprintf(i18n.CurrentLocale(), "chat.approval.custom_result_without_comment", toolName)
		}
		return i18n.Sprintf(i18n.CurrentLocale(), "chat.approval.custom_result_with_comment", toolName, comment)
	default:
		return i18n.Sprintf(i18n.CurrentLocale(), "chat.approval.reject_result", toolName)
	}
}

// Completions 聊天完成接口，负责参数校验、资源初始化和启动异步任务。
// 实际的执行逻辑委托给 completionRunner（见 chat_completion_runner.go）。
func (s *Service) Completions(ctx context.Context, inputMessage view_models.Message) (*view_models.Completions, error) {
	// 参数校验
	if inputMessage.UserMessageExtra == nil {
		return nil, ierror.New(ierror.ErrCodeCompletionsParams)
	}

	// 生成各类标识符
	selectModelId := inputMessage.UserMessageExtra.ModelId
	selectModelName := inputMessage.UserMessageExtra.ModelName
	userMessageUuid := uuid.New().String()
	assistantMessageUuid := uuid.New().String()
	taskUuid := uuid.New().String()
	eventKey := event.GenEventsKey(event.EventTypeTask, taskUuid)
	chatUuid := inputMessage.ChatUuid
	isNewChat := inputMessage.ChatUuid == ""
	if isNewChat {
		chatUuid = uuid.New().String()
	}
	inputMessage.MessageUuid = userMessageUuid
	inputMessage.ChatUuid = chatUuid

	// 获取模型信息
	providerModel, err := s.storage.GetProviderModel(context.Background(), selectModelId, selectModelName)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	if providerModel == nil {
		return nil, ierror.New(ierror.ErrCodeModelNotFound)
	}

	// 获取工作流工具集
	agentTools, toolMetaByID, cleanupTools, err := s.resolveSelectedTools(ctx, inputMessage.UserMessageExtra.Tools)
	if err != nil {
		return nil, ierror.NewError(err)
	}

	// 预加载自定义 agent 定义和工具（sub-agent 实例在 toolMiddleware 构建后创建）
	allCustomAgents := agents.AgentsByType(agents.AgentTypeCustom)
	type customAgentPrep struct {
		def         *agents.CustomAgentDef
		tools       []tool.BaseTool
		instruction string
	}
	var customAgentPreps []customAgentPrep
	var customToolCleanups []func()
	for _, agentDef := range allCustomAgents {
		customDef, ok := agentDef.(*agents.CustomAgentDef)
		if !ok {
			continue
		}

		// 解析该 agent 的工具
		customTools, _, customCleanup, toolErr := s.resolveSelectedTools(ctx, customDef.ToolIDs)
		if toolErr != nil {
			logger.Warm("resolve custom agent tools failed:", toolErr)
			continue
		}
		if customCleanup != nil {
			customToolCleanups = append(customToolCleanups, customCleanup)
		}

		// 如果 agent 有 skill，添加 load_skill 工具
		if len(customDef.SkillIDs) > 0 {
			if loadSkillTool, ok := llmtools.ToolRouter.GetToolByID("load_skill"); ok {
				customTools = append(customTools, loadSkillTool.Tool())
			}
		}

		// 解析 skill 摘要并注入 prompt（渐进注入）
		instruction := customDef.PromptText
		if len(customDef.SkillIDs) > 0 {
			skillSummary := skills.ResolveSkillSummaries(customDef.SkillIDs)
			if skillSummary != "" {
				instruction = instruction + "\n\n" + skillSummary
			}
		}

		// 如果 agent 拥有需要用户确认的工具，追加系统提示：不要自行询问确认，直接调用工具
		for _, toolID := range customDef.ToolIDs {
			if registeredTool, ok := llmtools.ToolRouter.GetToolByID(toolID); ok && registeredTool.RequireConfirmation() {
				instruction = instruction + "\n\n" + i18n.TCurrent("agent.system.tool_approval_hint", nil)
				break
			}
		}
		instruction = instruction + "\n\n" + llmtools.ShellRuntimeInstruction()

		customAgentPreps = append(customAgentPreps, customAgentPrep{
			def:         customDef,
			tools:       customTools,
			instruction: instruction,
		})
	}

	// 合并自定义 agent 工具清理函数到主清理逻辑
	if len(customToolCleanups) > 0 {
		originalCleanup := cleanupTools
		cleanupTools = func() {
			if originalCleanup != nil {
				originalCleanup()
			}
			for _, fn := range customToolCleanups {
				fn()
			}
		}
	}

	// 如果聊天的uuid为空，则新建一个聊天
	if isNewChat {
		title := inputMessage.Content
		err = s.storage.CreateChat(context.Background(), chatUuid, title)
		if err != nil {
			return nil, ierror.NewError(err)
		}
	}

	// 查找历史消息
	historyMessageData, _, err := s.storage.GetMessage(ctx, chatUuid, 0, 10)
	if err != nil {
		return nil, ierror.NewError(err)
	}

	// 创建用户消息
	_, err = s.storage.CreateMessage(ctx, chatUuid, inputMessage)
	if err != nil {
		return nil, ierror.NewError(err)
	}

	// 合并用户当前消息和历史消息，转换为 schema 格式
	historyMessageData = append(historyMessageData, inputMessage)
	var schemaMessages []schema.Message
	for _, item := range historyMessageData {
		schemaMessage, err := item.ToSchemaMessage()
		if err != nil {
			continue
		}
		schemaMessages = append(schemaMessages, *schemaMessage)
	}
	if s.plugins != nil {
		hookPayload := plugins.BeforeLLMSendPayload{
			ChatUUID: chatUuid,
			Messages: schemaMessagesToHookMessages(schemaMessages),
		}
		hookPayload = s.plugins.RunBeforeLLMSend(ctx, hookPayload)
		if len(hookPayload.Messages) > 0 {
			schemaMessages = hookMessagesToSchemaMessages(hookPayload.Messages)
		}
	}

	// 创建助手消息
	assistantMessage := data_models.Message{
		OrmModel:    data_models.OrmModel{},
		ChatUuid:    chatUuid,
		MessageUuid: assistantMessageUuid,
		Role:        schema.Assistant,
		AssistantMessageExtra: &data_models.AssistantMessageExtra{
			ToolUses:       []data_models.ToolUse{},
			ExecutionTrace: data_models.ExecutionTrace{Steps: []data_models.TraceStep{}},
			RouteType:      "",
			CurrentStage:   "chat.stage.pending",
		},
	}
	assistantMessageId, err := s.storage.CreateMessage(ctx, chatUuid, assistantMessage)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	assistantMessage.ID = assistantMessageId

	// 创建任务记录
	task := data_models.Task{
		TaskUuid:             taskUuid,
		ChatUuid:             chatUuid,
		AssistantMessageUuid: assistantMessageUuid,
		Status:               data_models.TaskStatusPending,
		EventKey:             eventKey,
	}
	if err = s.storage.CreateTask(ctx, task); err != nil {
		return nil, ierror.NewError(err)
	}

	// 构建自定义 agent ID 映射（用于在工具中间件中识别子 agent 调用）
	customAgentIDs := make(map[string]string, len(customAgentPreps))
	for _, prep := range customAgentPreps {
		customAgentIDs[prep.def.ID_] = prep.def.DisplayName
	}

	// 创建运行器，封装所有运行时状态
	localizedPrompts := s.promptSetWithCoreMemory(ctx, s.localizedPromptSet())
	runner := newCompletionRunner(
		s, localizedPrompts, inputMessage, providerModel,
		agentTools, toolMetaByID, customAgentIDs, cleanupTools, schemaMessages, isNewChat,
		chatUuid, assistantMessageUuid, taskUuid, eventKey,
		assistantMessage, task,
	)

	// 构建工具中间件（需要访问 runner 状态）
	toolMiddleware := runner.buildToolMiddleware()

	// 创建自定义 sub-agent（需要 toolMiddleware 以支持工具审批）
	var subAgents []adk.Agent
	if len(customAgentPreps) > 0 {
		subChatModel, chatModelErr := llm_provider.NewToolCallingChatModel(ctx, *providerModel)
		if chatModelErr != nil {
			logger.Warm("create chat model for sub-agents failed:", chatModelErr)
		} else {
			for _, prep := range customAgentPreps {
				subAgent, subErr := agents.NewRoleAgent(ctx, subChatModel, prep.def.ID_, prep.def.DisplayName+": "+prep.def.Description, prep.instruction, prep.tools, toolMiddleware)
				if subErr != nil {
					logger.Warm("create custom sub-agent failed:", subErr)
					continue
				}
				subAgents = append(subAgents, subAgent)
			}
		}
	}

	// 构建主 Agent 的 skill 摘要（渐进注入）
	mainSkillSummary := skills.ResolveAllSkillSummaries()

	directTools := append([]tool.BaseTool{newWorkflowHandoffTool(runner.setWorkflowHandoff)}, agentTools...)
	if prefs, prefErr := s.loadAppPreferences(ctx); prefErr == nil && prefs.MemorySystemEnabled {
		directTools = append(directTools, s.buildMemoryRuntimeTools()...)
	}

	// 始终添加 load_skill 工具以支持渐进式技能注入
	if loadSkillTool, ok := llmtools.ToolRouter.GetToolByID("load_skill"); ok {
		directTools = append(directTools, loadSkillTool.Tool())
	}

	provider, err := llm_provider.NewLlmProvider(ctx, *providerModel, subAgents, directTools, toolMiddleware, localizedPrompts, mainSkillSummary)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	runner.provider = provider

	// 启动异步任务
	tasker.Manager.StartTask(tasker.Runtime{
		TaskUUID:             taskUuid,
		ChatUUID:             chatUuid,
		AssistantMessageUUID: assistantMessageUuid,
		EventKey:             eventKey,
	}, runner.run)

	return &view_models.Completions{
		ChatUuid:    chatUuid,
		TaskUuid:    taskUuid,
		MessageUuid: assistantMessageUuid,
		EventKey:    eventKey,
	}, nil
}

func (s *Service) StopCompletions(messageKey string) error {
	tasker.Manager.StopByEventKey(messageKey)
	return nil
}

func (s *Service) StopTask(taskUuid string) error {
	tasker.Manager.StopTask(taskUuid)
	return nil
}

func (s *Service) GetTask(ctx context.Context, taskUuid string) (*view_models.Task, error) {
	task, err := s.storage.GetTask(ctx, taskUuid)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	if task == nil {
		return nil, nil
	}
	viewTask := view_models.Task(*task)
	return &viewTask, nil
}

func (s *Service) GetChatActiveTask(ctx context.Context, chatUuid string) (*view_models.Task, error) {
	for {
		task, err := s.storage.GetChatActiveTask(ctx, chatUuid)
		if err != nil {
			return nil, ierror.NewError(err)
		}
		if task == nil {
			return nil, nil
		}
		task, err = s.repairStaleActiveTask(ctx, task)
		if err != nil {
			return nil, ierror.NewError(err)
		}
		if task == nil {
			continue
		}
		viewTask := view_models.Task(*task)
		return &viewTask, nil
	}
}

func (s *Service) GetRunningTasks(ctx context.Context) (*view_models.TaskList, error) {
	tasks, err := s.storage.GetRunningTasks(ctx)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	viewTasks := make([]view_models.Task, 0, len(tasks))
	for _, task := range tasks {
		liveTask, err := s.repairStaleActiveTask(ctx, &task)
		if err != nil {
			return nil, ierror.NewError(err)
		}
		if liveTask == nil {
			continue
		}
		viewTasks = append(viewTasks, view_models.Task(*liveTask))
	}
	return &view_models.TaskList{Tasks: viewTasks}, nil
}

func (s *Service) emitTaskEvent(task data_models.Task, assistantMessage data_models.Message, traceDelta []data_models.TraceStep) {
	s.app.Event.Emit(task.EventKey, view_models.TaskStreamEvent{
		TaskUuid:         task.TaskUuid,
		ChatUuid:         task.ChatUuid,
		EventKey:         task.EventKey,
		Status:           task.Status,
		FinishReason:     task.FinishReason,
		FinishError:      task.FinishError,
		ExecutionTrace:   assistantMessage.AssistantMessageExtra.ExecutionTrace,
		TraceDelta:       traceDelta,
		CurrentStage:     assistantMessage.AssistantMessageExtra.CurrentStage,
		CurrentAgent:     assistantMessage.AssistantMessageExtra.CurrentAgent,
		RetryCount:       assistantMessage.AssistantMessageExtra.RetryCount,
		AssistantMessage: assistantMessage,
	})
}

// DeleteChat 删除聊天
func (s *Service) DeleteChat(chatUuid string) error {
	err := s.storage.DeleteChat(context.Background(), chatUuid)
	if err != nil {
		return ierror.NewError(err)
	}
	return nil
}

// RenameChat 重命名聊天
func (s *Service) RenameChat(chatUuid, title string) error {
	err := s.storage.RenameChat(context.Background(), chatUuid, title)
	if err != nil {
		return ierror.NewError(err)
	}
	payload := struct {
		ChatUuid string `json:"chat_uuid"`
		Title    string `json:"title"`
	}{
		ChatUuid: chatUuid,
		Title:    title,
	}
	s.app.Event.Emit(event.GenEventsKey(event.EventTypeChatTitle, chatUuid), payload)
	s.app.Event.Emit(event.GenEventsKey(event.EventTypeChatTitle, "all"), payload)
	return nil
}

// CollectionChat 收藏/取消收藏对话
func (s *Service) CollectionChat(chatUuid string, isCollection bool) error {
	err := s.storage.CollectionChat(context.Background(), chatUuid, isCollection)
	if err != nil {
		return ierror.NewError(err)
	}

	return nil
}

// GenChatTitle 创建聊天标题
func (s *Service) GenChatTitle(ctx context.Context, chatUuid string, modelId uint, modelName string, update bool) (string, error) {

	// 获取模型信息
	providerModel, err := s.storage.GetProviderModel(context.Background(), modelId, modelName)
	if err != nil {
		return "", ierror.NewError(err)
	}
	if providerModel == nil {
		return "", ierror.New(ierror.ErrCodeModelNotFound)
	}

	title, err := s.genChatTitle(ctx, chatUuid, *providerModel, update)
	if err != nil {
		return "", ierror.NewError(err)
	}

	return title, nil
}

func (s *Service) genChatTitle(ctx context.Context, chatUuid string, providerModel wrapper_models.ProviderModel, update bool) (string, error) {
	// 新建供应商
	provider, err := llm_provider.NewLlmProvider(ctx, providerModel, []adk.Agent{}, []tool.BaseTool{}, compose.ToolMiddleware{}, s.localizedPromptSet(), "")
	if err != nil {
		return "", err
	}

	historyMessages, _, err := s.storage.GetMessage(context.Background(), chatUuid, 0, 2)
	if err != nil {
		return "", err
	}
	var messages []schema.Message
	for _, item := range historyMessages {
		schemaMessage, err := item.ToSchemaMessage()
		if err != nil {
			return "", err
		}
		messages = append(messages, *schemaMessage)
	}

	title, err := provider.GenChatTitle(ctx, messages)
	if err != nil {
		return "", err
	}

	if update {
		err := s.storage.RenameChat(context.Background(), chatUuid, title)
		if err != nil {
			return "", err
		}
		payload := struct {
			ChatUuid string `json:"chat_uuid"`
			Title    string `json:"title"`
		}{
			ChatUuid: chatUuid,
			Title:    title,
		}
		s.app.Event.Emit(event.GenEventsKey(event.EventTypeChatTitle, chatUuid), payload)
		s.app.Event.Emit(event.GenEventsKey(event.EventTypeChatTitle, "all"), payload)
	}

	return title, nil
}
