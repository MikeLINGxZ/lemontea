package view_models

import (
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
)

type ChatList struct {
	Lists []Chat `json:"lists"`
	Total int    `json:"total"`
}

type Chat = data_models.Chat

type MatchMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Completions struct {
	ChatUuid    string `json:"chat_uuid"`
	TaskUuid    string `json:"task_uuid"`
	MessageUuid string `json:"message_uuid"`
	EventKey    string `json:"event_key"`
}

type Tool struct {
	Id          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	SourceType  string                `json:"source_type"`
	Enabled     bool                  `json:"enabled"`
	IsDeletable bool                  `json:"is_deletable"`
	PluginType  string                `json:"plugin_type,omitempty"`
	UseTools    []plugins.PluginTool  `json:"use_tools,omitempty"`
	ViewTools   []plugins.PluginTool  `json:"view_tools,omitempty"`
	Agents      []plugins.PluginAgent `json:"agents,omitempty"`
}
