package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/event_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
	agentpkg "trpc.group/trpc-go/trpc-agent-go/agent"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
)

// ChatHandler 处理聊天请求，将运行器事件转发给 Wails 和存储层
type ChatHandler struct {
	// 管理器
	manager *Manager
}

// SendMessageParams 包含运行一轮对话所需的输入
type SendMessageParams struct {
	// 会话编号
	SessionID uint
	// 内容
	Content string
	// 系统提示词
	SystemPrompt string
	// 已预热技能
	PrimedSkill string
	// 服务基础地址
	BaseURL string
	// API 密钥
	ApiKey string
	// 模型名称
	ModelName string
	// 提供商类型
	ProviderType pkgProvider.Type
	// 已启用的用户工具列表
	EnabledUserTools []string
	// 附件列表
	Attachments []Attachment
}

// NewChatHandler 为指定的管理器创建聊天处理器
// 接收管理器，返回聊天处理器
func NewChatHandler(m *Manager) *ChatHandler {
	return &ChatHandler{manager: m}
}

// 发布事件，接收当前名称、待处理的数据，无返回值
func (ch *ChatHandler) emitEvent(name string, data any) {
	app := ch.manager.App() // Wails 应用实例
	if app != nil {
		app.Event.Emit(name, data)
	}
}

// 更新会话状态，接收会话编号、状态，无返回值
func (ch *ChatHandler) updateSessionStatus(sessionID uint, status string) {
	_ = ch.manager.Storage().UpdateSessionStatus(sessionID, status)
	ch.emitEvent(event_id.AgentSessionStatus, map[string]any{
		"sessionId": sessionID,
		"status":    status,
	})
}

// 从存储读取会话历史并重建模型请求所需的消息列表，接收会话编号，返回消息列表及可能发生的错误
func (ch *ChatHandler) loadSessionHistory(sessionID uint) ([]model.Message, error) {
	// 已持久化消息列表、处理存储时的错误
	storedMessages, err := ch.manager.Storage().ListMessagesForSession(sessionID, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("list session messages: %w", err)
	}

	history := make([]model.Message, 0, len(storedMessages)) // 历史

	// 遍历已持久化消息列表，逐项取得已持久化
	for _, stored := range storedMessages {
		msg, ok := toModelMessage(stored) // 消息、是否成功取得对应值
		if !ok {
			continue
		}
		history = append(history, msg)
	}

	return history, nil
}

// 将持久化消息转换为模型 SDK 使用的消息，接收已持久化，返回消息、是否满足条件
func toModelMessage(stored data_models.Message) (model.Message, bool) {
	switch stored.ContentType {
	case "text":
		if stored.Role == "user" {
			// 每次重放历史消息时按需从磁盘加载附件
			// 按约定只存储附件路径，不缓存附件内容
			// JSON 格式错误时静默降级为纯文本
			// 与下方 tool_call 分支的降级方式一致
			atts, err := UnmarshalAttachments(stored.Attachments)
			if err == nil && len(atts) > 0 {
				return BuildUserMessage(stored.Content, atts), true
			}
			return model.NewUserMessage(stored.Content), true
		}
		return model.NewAssistantMessage(stored.Content), true
	case "thinking":
		return model.Message{
			Role:             model.RoleAssistant,
			ReasoningContent: stored.Content,
		}, true
	case "tool_result":
		return model.Message{
			Role:     model.RoleTool,
			ToolName: stored.AgentName,
			Content:  stored.Content,
		}, true
	case "confirm_response":
		return model.NewUserMessage(stored.Content), true
	case "tool_call":
		// 载荷
		var payload struct {
			// 唯一标识
			ID string `json:"id"`
			// 名称
			Name string `json:"name"`
			// 参数
			Args json.RawMessage `json:"args"`
		}
		decodeErr := json.Unmarshal([]byte(stored.Content), &payload) // 解析序列化内容时的错误
		if decodeErr != nil {
			return model.NewAssistantMessage(stored.Content), true
		}
		return model.Message{
			Role: model.RoleAssistant,
			ToolCalls: []model.ToolCall{
				{
					Type: "function",
					ID:   payload.ID,
					Function: model.FunctionDefinitionParam{
						Name:      payload.Name,
						Arguments: payload.Args,
					},
				},
			},
		}, true
	default:
		return model.Message{}, false
	}
}

// 规范化工具实参，接收尚未解析的原始数据，返回文本、是否满足条件
func normalizeToolArguments(raw json.RawMessage) (string, bool) {
	trimmed := bytes.TrimSpace(raw) // 已去除空白
	if len(trimmed) == 0 {
		return "{}", true
	}
	if !json.Valid(trimmed) {
		return "", false
	}

	// 紧凑
	var compact bytes.Buffer
	err := json.Compact(&compact, trimmed) // 压缩 JSON 空白时的错误
	if err != nil {
		return "", false
	}

	return compact.String(), true
}

// 按工具名称和规范化参数构造调用指纹，接收工具名称、参数，返回文本
func toolCallFingerprint(toolName, args string) string {
	if toolName == "" || args == "" {
		return ""
	}
	return toolName + "\x00" + args
}

// 启动指定会话的一轮流式模型请求，接收调用上下文和会话运行参数，返回启动阶段的错误
// 先读取历史并持久化用户消息，再用提供商、模型和工具配置获取运行器
// 注册取消回调后启动运行器，在后台消费模型事件
func (ch *ChatHandler) SendMessage(ctx context.Context, params SendMessageParams) error {
	stor := ch.manager.Storage()                            // 存储
	history, err := ch.loadSessionHistory(params.SessionID) // 历史、从存储读取会话历史并重建模型请求所需的消息列表时的错误
	if err != nil {
		return err
	}

	attachmentsJSON, err := MarshalAttachments(params.Attachments) // 附件列表JSON
	if err != nil {
		return fmt.Errorf("marshal attachments: %w", err)
	}

	session, err := stor.GetSession(params.SessionID) // 会话
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	_, err = stor.CreateMessage(data_models.Message{
		SessionID:   params.SessionID,
		Role:        "user",
		ContentType: "text",
		Content:     params.Content,
		Attachments: attachmentsJSON,
	})
	if err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	_ = stor.TouchSession(params.SessionID)
	ch.updateSessionStatus(params.SessionID, "loading")

	// 读取源
	r, err := ch.manager.GetOrCreateRunner(
		params.BaseURL,
		params.ApiKey,
		params.ModelName,
		params.ProviderType,
		params.EnabledUserTools,
		session.Kind,
		params.SessionID,
	)
	if err != nil {
		ch.updateSessionStatus(params.SessionID, "error-unread")
		return fmt.Errorf("get runner: %w", err)
	}

	// 流式输出上下文、用于取消本次操作的回调
	streamCtx, cancel := ch.manager.Streams().Start(context.WithoutCancel(ctx), params.SessionID)
	userID := "local"                                                // 用户编号
	sessionIDStr := strconv.FormatUint(uint64(params.SessionID), 10) // 会话编号文本

	runOptions := make([]agentpkg.RunOption, 0, 1) // 运行选项列表
	if len(history) > 0 {
		runOptions = append(runOptions, agentpkg.WithMessages(history))
	}

	// 运行上下文
	runCtx := withRunContext(streamCtx, runContext{
		SessionID:    params.SessionID,
		ModelName:    params.ModelName,
		BaseURL:      params.BaseURL,
		APIKey:       params.ApiKey,
		ProviderType: params.ProviderType,
	})

	systemPrompt := ch.memorySystemPrompt(params.SystemPrompt, params.Content) // 系统提示词
	// 用户消息
	userMsg := BuildUserMessage(buildRunContent(systemPrompt, params.PrimedSkill, params.Content), params.Attachments)

	// 事件列表
	events, err := r.Run(
		runCtx,
		userID,
		sessionIDStr,
		userMsg,
		runOptions...,
	)
	if err != nil {
		cancel()
		ch.manager.Streams().Remove(params.SessionID)
		ch.updateSessionStatus(params.SessionID, "error-unread")
		return fmt.Errorf("runner.Run: %w", err)
	}

	go ch.consumeEventsWithParams(streamCtx, params, events)
	return nil
}

// buildRunContent 将隐藏会话的初始化指令合入第一轮用户消息
// 保持共享运行器的全局提示词约定
// 接收系统提示词、已预热技能、内容，返回文本
func buildRunContent(systemPrompt, primedSkill, content string) string {
	systemPrompt = strings.TrimSpace(systemPrompt)
	primedSkill = strings.TrimSpace(primedSkill)
	content = strings.TrimSpace(content)

	parts := make([]string, 0, 3) // 组成部分
	if systemPrompt != "" {
		parts = append(parts, "System instructions:\n"+systemPrompt)
	}
	if primedSkill != "" {
		parts = append(parts, "Before doing any other work, call the Skill tool with {\"name\":\""+primedSkill+"\"}.")
	}
	if content != "" {
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n")
}

// 消费运行器事件并将结果同步给存储和前端，接收本次调用的上下文、会话编号、模型名称、事件列表，无返回值
func (ch *ChatHandler) consumeEvents(ctx context.Context, sessionID uint, modelName string, events <-chan *event.Event) {
	ch.consumeEventsWithParams(ctx, SendMessageParams{SessionID: sessionID, ModelName: modelName}, events)
}

// 消费一轮模型事件，接收调用上下文、会话参数及事件通道，无返回值
// 只处理选定的响应候选，累计并发送正文和推理文本的增量
// 按调用编号和参数指纹过滤重复工具事件，将工具调用及结果写入会话
// 结束时保存助手消息及用量、清理流状态，并在合适时安排记忆提取
func (ch *ChatHandler) consumeEventsWithParams(ctx context.Context, params SendMessageParams, events <-chan *event.Event) {
	sessionID := params.SessionID // 会话编号
	modelName := params.ModelName // 模型名称
	defer ch.manager.Streams().Remove(sessionID)

	stor := ch.manager.Storage() // 存储
	// 完整内容
	var fullContent string
	// 完整思考内容
	var fullThinking string
	// Token输入
	var tokensIn int
	// Token输出
	var tokensOut int
	// 流式输出序号
	var streamSeq int64
	// 主操作选择索引
	var primaryChoiceIndex *int
	handledToolCalls := make(map[string]struct{}) // 已处理工具调用列表
	lastToolCallFingerprint := ""                 // 最后一个工具调用指纹

	// 发布分片
	emitChunk := /* 实现发布分片回调，接收增量、内容类型、内容，无返回值 */ func(delta, contentType, content string) {
		streamSeq++
		ch.emitEvent(event_id.AgentStreamChunk, map[string]any{
			"sessionId":   sessionID,
			"seq":         streamSeq,
			"delta":       delta,
			"content":     content,
			"contentType": contentType,
		})
	}

	// 遍历事件列表，逐项取得事件
	for evt := range events {
		if evt == nil {
			continue
		}

		select {
		case <-ctx.Done():
			ch.updateSessionStatus(sessionID, "idle")
			return
		default:
		}

		if evt.Error != nil {
			ch.emitEvent(event_id.AgentStreamError, map[string]any{
				"sessionId": sessionID,
				"error":     evt.Error.Message,
			})
			ch.updateSessionStatus(sessionID, "error-unread")
			// 错误内容
			errContent, _ := json.Marshal(map[string]string{
				"msg":    ierror.ErrAgentStreamError.Msg(),
				"detail": evt.Error.Message,
			})
			_, _ = stor.CreateMessage(data_models.Message{
				SessionID:   sessionID,
				Role:        "assistant",
				ContentType: "error",
				Content:     string(errContent),
				ModelName:   modelName,
			})
			return
		}

		// 遍历Choices 对应的数据，逐项取得选择
		for _, choice := range evt.Choices {
			if primaryChoiceIndex == nil {
				idx := choice.Index // 索引
				primaryChoiceIndex = &idx
			}
			if primaryChoiceIndex != nil && choice.Index != *primaryChoiceIndex {
				continue
			}

			delta := choice.Delta                       // 增量
			message := choice.Message                   // 消息
			isToolResultMessage := message.ToolID != "" // 是否工具结果消息

			if !isToolResultMessage && delta.ReasoningContent != "" {
				lastToolCallFingerprint = ""
				fullThinking += delta.ReasoningContent
				emitChunk(delta.ReasoningContent, "thinking", fullThinking)
			}
			if !isToolResultMessage && delta.ReasoningContent == "" && message.ReasoningContent != "" && fullThinking == "" {
				lastToolCallFingerprint = ""
				fullThinking = message.ReasoningContent
				emitChunk(message.ReasoningContent, "thinking", fullThinking)
			}

			if !isToolResultMessage && delta.Content != "" {
				lastToolCallFingerprint = ""
				fullContent += delta.Content
				emitChunk(delta.Content, "text", fullContent)
			}
			if !isToolResultMessage && delta.Content == "" && message.Content != "" && fullContent == "" {
				lastToolCallFingerprint = ""
				fullContent = message.Content
				emitChunk(message.Content, "text", fullContent)
			}

			toolCalls := delta.ToolCalls // 工具调用列表
			if len(toolCalls) == 0 && len(message.ToolCalls) > 0 {
				toolCalls = message.ToolCalls
			}

			// 遍历工具调用列表，逐项取得tc 对应的数据
			for _, tc := range toolCalls {
				normalizedArgs, ok := normalizeToolArguments(tc.Function.Arguments) // 规范化值参数、是否成功取得对应值
				if !ok {
					continue
				}

				if tc.ID != "" {
					if /* 是否存在 */ _, exists := handledToolCalls[tc.ID]; exists {
						continue
					}
				}

				toolName := tc.Function.Name                                 // 工具名称
				fingerprint := toolCallFingerprint(toolName, normalizedArgs) // 指纹
				if fingerprint != "" && fingerprint == lastToolCallFingerprint {
					if tc.ID != "" {
						handledToolCalls[tc.ID] = struct{}{}
					}
					continue
				}
				lastToolCallFingerprint = fingerprint
				if tc.ID != "" {
					handledToolCalls[tc.ID] = struct{}{}
				}

				args := normalizedArgs // 参数

				meta, hasMeta := ch.manager.ToolRegistry().Get(toolName) // 元信息、是否有元信息
				purpose := ""                                            // 用途
				if hasMeta && meta.FormatPurpose != nil {
					purpose = meta.FormatPurpose(json.RawMessage(args))
				}

				// 载荷
				payload, _ := json.Marshal(map[string]any{
					"id":   tc.ID,
					"name": toolName,
					"args": json.RawMessage(args),
				})

				ch.emitEvent(event_id.AgentStreamToolCall, map[string]any{
					"sessionId": sessionID,
					"toolName":  toolName,
					"args":      args,
					"purpose":   purpose,
				})

				_, _ = stor.CreateMessage(data_models.Message{
					SessionID:   sessionID,
					Role:        "assistant",
					ContentType: "tool_call",
					Content:     string(payload),
					ModelName:   modelName,
					Extra:       purpose,
				})

			}

			if message.ToolID != "" {
				lastToolCallFingerprint = ""
				ch.emitEvent(event_id.AgentStreamToolResult, map[string]any{
					"sessionId": sessionID,
					"toolName":  message.ToolName,
					"result":    message.Content,
				})
				_, _ = stor.CreateMessage(data_models.Message{
					SessionID:   sessionID,
					Role:        "tool",
					ContentType: "tool_result",
					Content:     message.Content,
					ModelName:   modelName,
					AgentName:   message.ToolName,
				})
			}
		}

		if evt.Usage != nil {
			tokensIn = evt.Usage.PromptTokens
			tokensOut = evt.Usage.CompletionTokens
		}

		if evt.Done {
			break
		}
	}

	if fullThinking != "" {
		_, _ = stor.CreateMessage(data_models.Message{
			SessionID:   sessionID,
			Role:        "assistant",
			ContentType: "thinking",
			Content:     fullThinking,
			ModelName:   modelName,
		})
	}

	if fullContent != "" {
		_, _ = stor.CreateMessage(data_models.Message{
			SessionID:   sessionID,
			Role:        "assistant",
			ContentType: "text",
			Content:     fullContent,
			ModelName:   modelName,
			TokensIn:    tokensIn,
			TokensOut:   tokensOut,
		})
	}

	ch.emitEvent(event_id.AgentStreamDone, map[string]any{
		"sessionId": sessionID,
		"usage": map[string]int{
			"input":  tokensIn,
			"output": tokensOut,
		},
	})
	ch.updateSessionStatus(sessionID, "done-unread")
	ch.enqueueMemoryEncoding(params, fullContent)
}
