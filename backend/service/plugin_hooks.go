package service

import (
	"github.com/cloudwego/eino/schema"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
)

func schemaMessagesToHookMessages(messages []schema.Message) []plugins.HookMessage {
	res := make([]plugins.HookMessage, 0, len(messages))
	for _, msg := range messages {
		res = append(res, plugins.HookMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}
	return res
}

func hookMessagesToSchemaMessages(messages []plugins.HookMessage) []schema.Message {
	res := make([]schema.Message, 0, len(messages))
	for _, msg := range messages {
		res = append(res, schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		})
	}
	return res
}
