package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	pkgAgent "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent/tools"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/event_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/prompt_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/prompt"
	pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/agent/agent_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
)

// Agent 向前端提供智能体操作
type Agent struct {
	// 管理器
	manager *pkgAgent.Manager
}

// Dependencies 集中声明应用启动时可选的智能体集成
type Dependencies struct {
	// 技能提供商
	SkillProvider tools.SkillProvider
	// 注意提醒请求器
	AttentionRequester tools.AttentionRequester
	// 技能创建器
	SkillCreator tools.SkillCreator
	// CLI安装器
	CliInstaller tools.CliInstaller
	// CLI安装进度
	CliInstallProgress tools.CliInstallProgressReporter
	// CLI清单生成器
	CliManifestGenerator tools.CliManifestGenerator
	// CLI命令Runner
	CliCommandRunner tools.CliCommandRunner
	// 终端Runner
	TerminalRunner *pkgterminal.Manager
}

// NewAgent 使用应用存储创建智能体服务
// 接收存储接口、deps 对应的数据，返回智能体
func NewAgent(istorage *storage.Storage, deps ...Dependencies) *Agent {
	svc := &Agent{manager: pkgAgent.NewManager(istorage)} // 服务
	if len(deps) > 0 {
		if deps[0].SkillProvider != nil {
			svc.manager.SetSkillProvider(deps[0].SkillProvider)
		}
		if deps[0].AttentionRequester != nil {
			svc.manager.SetAttentionRequester(deps[0].AttentionRequester)
		}
		if deps[0].SkillCreator != nil {
			svc.manager.SetSkillCreator(deps[0].SkillCreator)
		}
		if deps[0].CliInstaller != nil {
			svc.manager.SetCliInstaller(deps[0].CliInstaller)
		}
		if deps[0].CliInstallProgress != nil {
			svc.manager.SetCliInstallProgressReporter(deps[0].CliInstallProgress)
		}
		if deps[0].CliManifestGenerator != nil {
			svc.manager.SetCliManifestGenerator(deps[0].CliManifestGenerator)
		}
		if deps[0].CliCommandRunner != nil {
			svc.manager.SetCliCommandRunner(deps[0].CliCommandRunner)
		}
		if deps[0].TerminalRunner != nil {
			svc.manager.SetTerminalRunner(deps[0].TerminalRunner)
		}
	}
	return svc
}

// 启动指定会话的模型请求并将处理结果返回给前端，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) SendMessage(ctx context.Context, input agent_dto.SendMessageInput) (*agent_dto.SendMessageOutput, error) {
	atts, err := convertAttachments(input.Attachments) // 附件列表、转换附件列表时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentSendMessage, err)
	}

	handler := a.manager.NewChatHandler() // 处理器
	err = handler.SendMessage(ctx, pkgAgent.SendMessageParams{
		SessionID:        input.SessionID,
		Content:          input.Content,
		SystemPrompt:     input.SystemPrompt,
		PrimedSkill:      input.SkillName,
		BaseURL:          input.BaseURL,
		ApiKey:           input.ApiKey,
		ModelName:        input.ModelName,
		ProviderType:     input.ProviderType,
		EnabledUserTools: input.EnabledUserTools,
		Attachments:      atts,
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentSendMessage, err)
	}
	return &agent_dto.SendMessageOutput{}, nil
}

// 取消指定会话仍在运行的模型请求，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) StopGeneration(ctx context.Context, input agent_dto.StopGenerationInput) (*agent_dto.StopGenerationOutput, error) {
	a.manager.Streams().Stop(input.SessionID)
	_ = a.manager.Storage().UpdateSessionStatus(input.SessionID, "idle")
	return &agent_dto.StopGenerationOutput{}, nil
}

// 把用户对工具调用的确认结果交给正在等待的会话，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) RespondToConfirm(ctx context.Context, input agent_dto.RespondToConfirmInput) (*agent_dto.RespondToConfirmOutput, error) {
	// 是否成功取得对应值
	ok := a.manager.Streams().SendConfirmResponse(input.SessionID, pkgAgent.ConfirmResponse{
		Approved: input.Approved,
		Message:  input.Message,
		Action:   input.Action,
	})
	if !ok {
		return nil, ierror.Error(ierror.ErrAgentNoConfirm, fmt.Errorf("no active confirmation for session %d", input.SessionID))
	}
	return &agent_dto.RespondToConfirmOutput{}, nil
}

// 创建并保存一个新会话，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) CreateSession(ctx context.Context, input agent_dto.CreateSessionInput) (*agent_dto.CreateSessionOutput, error) {
	title := strings.TrimSpace(input.Title) // 标题
	if title == "" {
		title = "New Chat"
	}
	tags := normalizeSessionTags(input.Tags) // 标签列表
	kind := "user"                           // 类别标识
	if tagsContain(tags, "task") {
		kind = "task"
	}

	// 会话、处理存储时的错误
	session, err := a.manager.Storage().CreateSession(data_models.Session{
		Title:  title,
		Status: "idle",
		Kind:   kind,
		Tags:   marshalSessionTags(tags),
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentCreateSession, err)
	}

	return &agent_dto.CreateSessionOutput{
		SessionID: session.ID,
		Title:     session.Title,
		Tags:      tags,
	}, nil
}

// 按查询条件和游标返回会话列表，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) ListSessions(ctx context.Context, input agent_dto.ListSessionsInput) (*agent_dto.ListSessionsOutput, error) {
	limit := input.Limit // 上限
	if limit <= 0 {
		limit = 20
	}

	// 会话列表、处理存储时的错误
	sessions, err := a.manager.Storage().ListSessions(input.Cursor, limit+1, input.StarredOnly, input.IncludeHidden)
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentListSessions, err)
	}

	hasMore := len(sessions) > limit // 是否有更多
	if hasMore {
		sessions = sessions[:limit]
	}

	items := make([]agent_dto.SessionItem, 0, len(sessions)) // 条目列表
	// 遍历会话列表，逐项取得会话
	for _, session := range sessions {
		items = append(items, toSessionItem(session))
	}

	// 下一项游标
	var nextCursor uint
	if hasMore && len(sessions) > 0 {
		nextCursor = sessions[len(sessions)-1].ID
	}

	return &agent_dto.ListSessionsOutput{
		Sessions:   items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// SpawnTaskSession 创建任务会话并异步提交首轮请求
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) SpawnTaskSession(ctx context.Context, input agent_dto.SpawnTaskSessionInput) (*agent_dto.SpawnTaskSessionOutput, error) {
	title := strings.TrimSpace(input.Title) // 标题
	if title == "" {
		title = "Task"
	}

	// 会话、处理存储时的错误
	session, err := a.manager.Storage().CreateSession(data_models.Session{
		Title:  title,
		Status: "idle",
		Kind:   "task",
		Tags:   marshalSessionTags([]string{"task"}),
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentCreateSession, err)
	}

	if /* Wails 应用实例 */ app := a.manager.App(); app != nil {
		app.Event.Emit(event_id.AgentSessionSpawned, map[string]any{
			"sessionId":   session.ID,
			"title":       session.Title,
			"kind":        session.Kind,
			"tags":        []string{"task"},
			"userMessage": input.UserMessage,
		})
	}

	go /* 处理技能名称时调用的回调，无返回值 */ func() {
		handler := a.manager.NewChatHandler() // 处理器
		_ = handler.SendMessage(context.Background(), pkgAgent.SendMessageParams{
			SessionID:    session.ID,
			Content:      input.UserMessage,
			SystemPrompt: input.SystemPrompt,
			PrimedSkill:  input.SkillName,
			BaseURL:      input.BaseURL,
			ApiKey:       input.ApiKey,
			ModelName:    input.ModelName,
			ProviderType: input.ProviderType,
		})
	}()

	return &agent_dto.SpawnTaskSessionOutput{
		SessionID: session.ID,
		Title:     session.Title,
	}, nil
}

// 按分页参数加载指定会话的历史消息，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) LoadSessionMessages(ctx context.Context, input agent_dto.LoadSessionMessagesInput) (*agent_dto.LoadSessionMessagesOutput, error) {
	limit := input.Limit // 上限
	if limit <= 0 {
		limit = 50
	}

	// 消息列表、处理存储时的错误
	msgs, err := a.manager.Storage().ListMessagesForSession(input.SessionID, input.Offset, limit)
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentListMessages, err)
	}

	total, err := a.manager.Storage().CountMessagesForSession(input.SessionID) // 总量
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentCountMessages, err)
	}

	items := make([]agent_dto.MessageItem, 0, len(msgs)) // 条目列表
	// 遍历消息列表，逐项取得消息
	for _, msg := range msgs {
		items = append(items, toMessageItem(msg))
	}

	return &agent_dto.LoadSessionMessagesOutput{
		Messages: items,
		Total:    total,
		HasMore:  int64(input.Offset+len(msgs)) < total,
	}, nil
}

// 清除指定会话的未读状态，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) MarkSessionRead(ctx context.Context, input agent_dto.MarkSessionReadInput) (*agent_dto.MarkSessionReadOutput, error) {
	err := a.manager.Storage().UpdateSessionStatus(input.SessionID, "idle") // 处理存储时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentMarkRead, err)
	}
	return &agent_dto.MarkSessionReadOutput{}, nil
}

// 重命名会话，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) RenameSession(ctx context.Context, input agent_dto.RenameSessionInput) (*agent_dto.RenameSessionOutput, error) {
	// 处理存储时的错误
	err := a.manager.Storage().UpdateSessionTitle(input.SessionID, strings.TrimSpace(input.Title))
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentRename, err)
	}
	return &agent_dto.RenameSessionOutput{}, nil
}

// 删除会话，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) DeleteSession(ctx context.Context, input agent_dto.DeleteSessionInput) (*agent_dto.DeleteSessionOutput, error) {
	a.manager.Streams().Stop(input.SessionID)
	err := a.manager.Storage().DeleteSession(input.SessionID) // 处理存储时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentDelete, err)
	}
	return &agent_dto.DeleteSessionOutput{}, nil
}

// 根据请求更新会话的收藏状态，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (a *Agent) ToggleStarSession(ctx context.Context, input agent_dto.ToggleStarSessionInput) (*agent_dto.ToggleStarSessionOutput, error) {
	err := a.manager.Storage().UpdateSessionStarred(input.SessionID, input.Starred) // 处理存储时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrAgentToggleStar, err)
	}
	return &agent_dto.ToggleStarSessionOutput{}, nil
}

// 处理生成标题，接收调用上下文和请求参数，返回操作结果及可能发生的错误
// 取得或创建模型运行器
// 运行智能体
func (a *Agent) GenerateTitle(ctx context.Context, input agent_dto.GenerateTitleInput) (*agent_dto.GenerateTitleOutput, error) {
	fmt.Println("Generating title for session:", input.SessionID)

	msgs, err := a.manager.Storage().ListMessagesForSession(input.SessionID, 0, 4) // 消息列表、处理存储时的错误
	if err != nil || len(msgs) == 0 {
		return &agent_dto.GenerateTitleOutput{Title: "New Chat"}, nil
	}

	// 预览
	var preview strings.Builder
	// 遍历消息列表，逐项取得消息
	for _, msg := range msgs {
		if msg.ContentType != "text" || msg.Content == "" {
			continue
		}
		content := msg.Content // 内容
		if len(content) > 200 {
			content = content[:200]
		}
		preview.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, content))
	}

	if preview.Len() == 0 {
		return &agent_dto.GenerateTitleOutput{Title: "New Chat"}, nil
	}

	// 读取源
	r, err := a.manager.GetOrCreateRunner(input.BaseURL, input.ApiKey, input.ModelName, input.ProviderType, nil, "user", input.SessionID)
	if err != nil {
		return &agent_dto.GenerateTitleOutput{Title: "New Chat"}, nil
	}

	titlePrompt := buildTitlePrompt(preview.String()) // 标题提示词
	// 事件列表
	events, err := r.Run(ctx, "local", fmt.Sprintf("title-%d", input.SessionID), model.NewUserMessage(titlePrompt))
	if err != nil {
		return &agent_dto.GenerateTitleOutput{Title: "New Chat"}, nil
	}

	result := collectTitleFromEvents(events) // 从模型事件中提取生成的会话标题得到的结果
	if result == "" {
		result = "New Chat"
	}

	_ = a.manager.Storage().UpdateSessionTitle(input.SessionID, result)
	return &agent_dto.GenerateTitleOutput{Title: result}, nil
}

// 从模型事件中提取生成的会话标题，接收事件列表，返回文本
func collectTitleFromEvents(events <-chan *event.Event) string {
	// streamed标题
	var streamedTitle strings.Builder
	// 最终标题
	var finalTitle string

	// 遍历事件列表，逐项取得事件
	for evt := range events {
		if evt == nil || evt.Error != nil {
			continue
		}
		// 遍历Choices 对应的数据，逐项取得选择
		for _, choice := range evt.Choices {
			if choice.Delta.Content != "" {
				streamedTitle.WriteString(choice.Delta.Content)
			}
			if finalTitle == "" && choice.Message.Content != "" {
				finalTitle = choice.Message.Content
			}
		}
	}

	result := strings.TrimSpace(streamedTitle.String()) // 去掉首尾空白得到的结果
	if result != "" {
		return result
	}

	return strings.TrimSpace(finalTitle)
}

// 根据历史消息构造会话标题生成提示词，接收预览，返回文本
func buildTitlePrompt(preview string) string {
	titleInstruction, err := prompt.Load(prompt_id.GenChatTitle) // 标题指令、加载当前数据时的错误
	if err != nil || strings.TrimSpace(titleInstruction) == "" {
		titleInstruction = "Generate a concise title under 20 characters for this conversation. Return only the title."
	}

	return fmt.Sprintf("%s\n\n%s", titleInstruction, preview)
}

const (
	// 上限附件数量
	maxAttachmentCount = 10
	// 上限附件字节
	maxAttachmentBytes = 20 * 1024 * 1024
)

// convertAttachments 校验附件数量、大小及文件是否存在
// 返回归一化后的 pkgAgent.Attachment 列表，校验失败时返回错误
// 接收输入，返回附件列表及可能发生的错误
func convertAttachments(in []agent_dto.AttachmentInput) ([]pkgAgent.Attachment, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if len(in) > maxAttachmentCount {
		return nil, ierror.Error(ierror.ErrAgentTooManyAttachments, fmt.Errorf("too many attachments: %d (max %d)", len(in), maxAttachmentCount))
	}
	out := make([]pkgAgent.Attachment, 0, len(in)) // 本次处理的输出
	// 遍历输入，逐项取得当前条目
	for _, item := range in {
		path := strings.TrimSpace(item.Path) // 文件路径
		if path == "" {
			return nil, ierror.Error(ierror.ErrAgentAttachmentPath, fmt.Errorf("attachment path is empty"))
		}
		info, err := os.Stat(path) // 信息、读取文件元数据时的错误
		if err != nil {
			return nil, ierror.Error(ierror.ErrAgentAttachmentNotFound, fmt.Errorf("attachment not found: %s", path))
		}
		if info.Size() > maxAttachmentBytes {
			return nil, ierror.Error(ierror.ErrAgentAttachmentSize, fmt.Errorf("attachment %q exceeds %d bytes", info.Name(), maxAttachmentBytes))
		}
		out = append(out, pkgAgent.NormalizeAttachment(pkgAgent.Attachment{
			Path: path,
			Name: item.Name,
			Mime: item.Mime,
		}))
	}
	return out, nil
}
