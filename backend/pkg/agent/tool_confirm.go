package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/agent/tools"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/event_id"
	"trpc.group/trpc-go/trpc-agent-go/model"
	toolpkg "trpc.group/trpc-go/trpc-agent-go/tool"
)

// 创建一轮会话的工具调用前后回调，返回回调集合
// 调用前读取工具元数据并执行确认策略，需要确认时通知前端并等待用户选择
// 拒绝和附带意见分别转换为工具结果，调用后按确认信息补充模型可见的说明
func (m *Manager) newToolCallbacks() *toolpkg.Callbacks {
	return toolpkg.NewCallbacks().
		RegisterBeforeTool( /* 构建回调列表时调用的回调，接收本次调用的上下文、参数，返回Before工具结果及可能发生的错误 */ func(
			ctx context.Context,
			args *toolpkg.BeforeToolArgs,
		) (*toolpkg.BeforeToolResult, error) {
			meta, ok := m.lookupToolMeta(args.ToolName) // 元信息、是否成功取得对应值
			if !ok {
				return nil, nil
			}

			runInfo, ok := getRunContext(ctx) // 运行信息
			if !ok {
				return nil, nil
			}

			purpose := "" // 用途
			if meta.FormatPurpose != nil && len(args.Arguments) > 0 {
				purpose = meta.FormatPurpose(json.RawMessage(args.Arguments))
			}

			argsText := "{}" // 参数文本
			if len(args.Arguments) > 0 {
				argsText = string(args.Arguments)
			}

			// 决策、依据工具配置和运行策略决定是否需要用户确认时的错误
			decision, err := m.decideToolConfirmation(ctx, ToolConfirmDecisionInput{
				ToolName:     args.ToolName,
				Arguments:    argsText,
				Purpose:      purpose,
				Meta:         meta,
				ModelName:    runInfo.ModelName,
				BaseURL:      runInfo.BaseURL,
				APIKey:       runInfo.APIKey,
				ProviderType: runInfo.ProviderType,
				SessionID:    runInfo.SessionID,
			})
			if err != nil {
				decision = ToolConfirmDecision{RequireConfirm: true, Reason: "fallback after policy error", Source: "fallback"}
			}
			if decision.Deny {
				return &toolpkg.BeforeToolResult{
					CustomResult: map[string]any{
						"error":  buildDeniedToolMessage(decision.Reason),
						"reason": decision.Reason,
					},
				}, nil
			}
			shouldConfirm := decision.RequireConfirm // 是否需确认
			if !shouldConfirm {
				return nil, nil
			}

			m.emitToolConfirmRequest(runInfo.SessionID, args.ToolCallID, args.ToolName, argsText, purpose)
			m.updateSessionStatus(runInfo.SessionID, "waiting-unread")

			resp, err := m.Streams().WaitForConfirm(ctx, runInfo.SessionID) // 远程服务响应
			if err != nil {
				m.updateSessionStatus(runInfo.SessionID, "idle")
				return nil, err
			}

			action := normalizeConfirmAction(resp)                       // 操作
			comment := strings.TrimSpace(resp.Message)                   // 备注
			confirmContent := formatConfirmContent(action, resp.Message) // 确认内容

			_, _ = m.Storage().CreateMessage(data_models.Message{
				SessionID:   runInfo.SessionID,
				Role:        "user",
				ContentType: "confirm_response",
				Content:     confirmContent,
				Extra:       marshalConfirmMetadata(action, comment),
			})
			m.updateSessionStatus(runInfo.SessionID, "loading")

			if action == "comment" {
				return &toolpkg.BeforeToolResult{
					Context: withToolConfirmation(ctx, toolConfirmation{
						Approved: false,
						Message:  comment,
						Action:   action,
					}),
					CustomResult: map[string]any{
						"status":  "user_comment",
						"comment": comment,
					},
				}, nil
			}

			if action == "reject" {
				return &toolpkg.BeforeToolResult{
					Context: withToolConfirmation(ctx, toolConfirmation{
						Approved: false,
						Message:  comment,
						Action:   action,
					}),
					CustomResult: map[string]any{
						"error":   buildRejectedToolMessage(resp.Message),
						"comment": comment,
					},
				}, nil
			}

			return &toolpkg.BeforeToolResult{
				Context: withToolConfirmation(ctx, toolConfirmation{
					Approved: true,
					Message:  comment,
					Action:   action,
				}),
			}, nil
		}).
		RegisterToolResultMessages( /* 构建回调列表时调用的回调，接收本次调用的上下文、输入，返回待检查的动态数据及可能发生的错误 */ func(
			ctx context.Context,
			in *toolpkg.ToolResultMessagesInput,
		) (any, error) {
			confirmation, ok := getToolConfirmation(ctx) // 确认、是否成功取得对应值
			if !ok {
				return nil, nil
			}

			defaultMsg, ok := in.DefaultToolMessage.(model.Message) // 默认消息
			if !ok {
				return nil, nil
			}

			note := formatToolConfirmationNote(confirmation) // 说明
			if note == "" {
				return nil, nil
			}

			if confirmation.Action == "comment" && strings.TrimSpace(confirmation.Message) != "" {
				defaultMsg.Content = note
				return defaultMsg, nil
			}

			if strings.TrimSpace(defaultMsg.Content) == "" {
				defaultMsg.Content = note
			} else {
				defaultMsg.Content = fmt.Sprintf("%s\n\n%s", note, defaultMsg.Content)
			}

			return defaultMsg, nil
		})
}

// 依据工具配置和运行策略决定是否需要用户确认，接收调用上下文和请求参数，返回工具确认决策及可能发生的错误
func (m *Manager) decideToolConfirmation(ctx context.Context, input ToolConfirmDecisionInput) (ToolConfirmDecision, error) {
	m.mu.RLock()
	decider := m.toolConfirmDecider // 决策器
	m.mu.RUnlock()
	if decider == nil {
		return ToolConfirmDecision{RequireConfirm: true, Reason: "missing decider", Source: "fallback"}, nil
	}
	return decider(ctx, input)
}

// 按工具名称查找可用于确认提示的元数据，接收工具名称，返回工具元信息、是否满足条件
func (m *Manager) lookupToolMeta(toolName string) (tools.ToolMeta, bool) {
	if /* 元信息、是否成功取得对应值 */ meta, ok := m.ToolRegistry().Get(toolName); ok {
		return meta, true
	}
	config, err := loadAgentConfig() // 配置、加载智能体配置时的错误
	if err != nil {
		return tools.ToolMeta{}, false
	}
	// 遍历扩展列表，逐项取得当前条目
	for _, item := range config.Extensions {
		// 遍历工具列表，逐项取得当前
		for _, current := range item.Tools {
			if current.ToolID == toolName {
				return tools.ToolMeta{
					Name:            current.ToolID,
					Description:     current.Description,
					Category:        tools.CategoryUser,
					RequiresConfirm: current.RequiresConfirm,
					FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
						return fmt.Sprintf("Run %s with arguments %s", current.Name, string(args))
					},
				}, true
			}
		}
	}
	return tools.ToolMeta{}, false
}

// 向前端发布待确认的工具调用，接收会话编号、请求编号、工具名称、参数、用途，无返回值
func (m *Manager) emitToolConfirmRequest(sessionID uint, requestID, toolName, args, purpose string) {
	app := m.App() // Wails 应用实例
	if app == nil {
		return
	}

	app.Event.Emit(event_id.AgentStreamConfirmRequest, map[string]any{
		"sessionId": sessionID,
		"requestId": requestID,
		"toolName":  toolName,
		"args":      args,
		"purpose":   purpose,
	})
}

// 更新会话状态，接收会话编号、状态，无返回值
func (m *Manager) updateSessionStatus(sessionID uint, status string) {
	_ = m.Storage().UpdateSessionStatus(sessionID, status)

	app := m.App() // Wails 应用实例
	if app == nil {
		return
	}

	app.Event.Emit(event_id.AgentSessionStatus, map[string]any{
		"sessionId": sessionID,
		"status":    status,
	})
}

// 构造用户拒绝执行时反馈给模型的工具消息，接收消息，返回文本
func buildRejectedToolMessage(message string) string {
	if message == "" {
		return "tool execution rejected by user"
	}
	return fmt.Sprintf("tool execution rejected by user: %s", message)
}

// 构造策略禁止执行时反馈给模型的工具消息，接收原因，返回文本
func buildDeniedToolMessage(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "tool execution denied by policy"
	}
	return fmt.Sprintf("tool execution denied by policy: %s", reason)
}

// 将前端确认操作归一化为后端支持的动作，接收远程服务响应，返回文本
func normalizeConfirmAction(resp ConfirmResponse) string {
	action := strings.TrimSpace(resp.Action) // 操作
	switch action {
	case "approve", "reject", "comment":
		return action
	}
	if resp.Approved {
		return "approve"
	}
	return "reject"
}

// 格式化确认内容，接收操作、消息，返回文本
func formatConfirmContent(action, message string) string {
	message = strings.TrimSpace(message)
	status := action // 状态
	switch action {
	case "approve":
		status = "approved"
	case "reject":
		status = "rejected"
	case "comment":
		status = "commented"
	}
	if message == "" {
		return status
	}
	return fmt.Sprintf("%s\ncomment: %s", status, message)
}

// 格式化工具确认说明，接收确认，返回文本
func formatToolConfirmationNote(confirmation toolConfirmation) string {
	message := strings.TrimSpace(confirmation.Message) // 消息

	if confirmation.Action == "comment" && message != "" {
		return fmt.Sprintf(
			"Tool execution paused. The user submitted this comment: %s. Do not execute the previous tool call unchanged. Revise the tool call to incorporate the comment, then continue.",
			message,
		)
	}

	status := "Tool confirmation: approved." // 状态
	if !confirmation.Approved {
		status = "Tool confirmation: rejected."
	}
	if message == "" {
		return status
	}
	return fmt.Sprintf("%s User comment: %s", status, message)
}

// 将工具确认动作与说明序列化为存储元数据，接收操作、备注，返回文本
func marshalConfirmMetadata(action, comment string) string {
	// 载荷、将数据序列化时的错误
	payload, err := json.Marshal(map[string]string{
		"action":  action,
		"comment": comment,
	})
	if err != nil {
		return ""
	}
	return string(payload)
}
