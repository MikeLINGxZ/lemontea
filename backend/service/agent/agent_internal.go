package agent

import (
	"encoding/json"
	"strings"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/agent/agent_dto"
)

// 时间格式化
const timeFormat = time.RFC3339

// 把会话记录转换为前端响应条目，接收会话，返回会话条目
func toSessionItem(s data_models.Session) agent_dto.SessionItem {
	tags := sessionTags(s) // 标签列表
	return agent_dto.SessionItem{
		ID:      s.ID,
		Title:   s.Title,
		Kind:    s.Kind,
		Tags:    tags,
		Starred: s.Starred,
		Status:  s.Status,
		Created: s.CreatedAt.Format(timeFormat),
		Updated: s.UpdatedAt.Format(timeFormat),
	}
}

// 处理会话标签列表，接收会话，返回文本列表
func sessionTags(s data_models.Session) []string {
	tags := parseSessionTags(s.Tags) // 标签列表
	if len(tags) == 0 && s.Kind == "task" {
		return []string{"task"}
	}
	return tags
}

// 规范化会话标签列表，接收标签列表，返回文本列表
func normalizeSessionTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags)) // seen 对应的数据
	out := make([]string, 0, len(tags))          // 本次处理的输出
	// 遍历标签列表，逐项取得当前标签
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if /* 是否成功取得对应值 */ _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

// 从存储文本解析会话标签列表，接收尚未解析的原始数据，返回文本列表
func parseSessionTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// 标签列表
	var tags []string
	err := json.Unmarshal([]byte(raw), &tags) // 解析序列化内容时的错误
	if err != nil {
		return nil
	}
	return normalizeSessionTags(tags)
}

// 将会话标签列表序列化为存储文本，接收标签列表，返回文本
func marshalSessionTags(tags []string) string {
	tags = normalizeSessionTags(tags)
	if len(tags) == 0 {
		return ""
	}
	content, _ := json.Marshal(tags) // 内容
	return string(content)
}

// 检查标签列表是否包含指定标签，接收标签列表、目标，返回是否满足条件
func tagsContain(tags []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	// 遍历标签列表，逐项取得当前标签
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

// 把持久化消息转换为前端响应条目，接收消息，返回消息条目
func toMessageItem(m data_models.Message) agent_dto.MessageItem {
	return agent_dto.MessageItem{
		ID:          m.ID,
		SessionID:   m.SessionID,
		ParentID:    m.ParentID,
		Role:        m.Role,
		ContentType: m.ContentType,
		Content:     m.Content,
		ModelName:   m.ModelName,
		AgentName:   m.AgentName,
		TokensIn:    m.TokensIn,
		TokensOut:   m.TokensOut,
		Extra:       m.Extra,
		Attachments: m.Attachments,
		Created:     m.CreatedAt.Format(timeFormat),
	}
}
