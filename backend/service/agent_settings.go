package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/agents"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/logger"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils/ierror"
)

var customAgentIDPattern = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// ListAgents 返回所有已注册 Agent 的摘要信息。
func (s *Service) ListAgents() ([]view_models.AgentSummary, error) {
	allAgents := agents.AllAgents()
	result := make([]view_models.AgentSummary, 0, len(allAgents))
	for _, a := range allAgents {
		promptNames := a.PromptNames()
		if promptNames == nil {
			promptNames = []string{}
		}
		summary := view_models.AgentSummary{
			Name:        a.Name(),
			Description: a.Desc(),
			AgentType:   string(a.Type()),
			AgentRole:   string(a.Role()),
			PromptNames: promptNames,
			IsDeletable: a.Type() == agents.AgentTypeCustom,
		}
		if cad, ok := a.(*agents.CustomAgentDef); ok {
			summary.DisplayName = cad.DisplayName
			summary.Description = cad.Description
			summary.Tools = cad.ToolIDs
			summary.Skills = cad.SkillIDs
		} else {
			summary.DisplayName = a.Desc()
		}
		if summary.Tools == nil {
			summary.Tools = []string{}
		}
		if summary.Skills == nil {
			summary.Skills = []string{}
		}
		result = append(result, summary)
	}
	return result, nil
}

// GetAgent 返回指定 Agent 的详情，包含其关联的提示词内容。
func (s *Service) GetAgent(name string) (*view_models.AgentDetail, error) {
	agentDef, ok := agents.FindAgent(name)
	if !ok {
		return nil, ierror.NewError(fmt.Errorf("agent not found: %s", name))
	}

	promptNames := agentDef.PromptNames()
	if promptNames == nil {
		promptNames = []string{}
	}

	defaults := agentDef.DefaultPrompts()
	metas := agentDef.PromptMetas()

	agentPrompts := make([]view_models.AgentPrompt, 0, len(metas))
	for _, meta := range metas {
		defaultContent := defaults[meta.FileName]
		content, _ := agents.LoadAgentPrompt(agentDef.Name(), meta.FileName, defaultContent)

		agentPrompts = append(agentPrompts, view_models.AgentPrompt{
			Name:        meta.FileName,
			Title:       meta.Title,
			Description: meta.Description,
			Content:     content,
			IsSystem:    meta.IsSystem,
		})
	}

	summary := view_models.AgentSummary{
		Name:        agentDef.Name(),
		Description: agentDef.Desc(),
		AgentType:   string(agentDef.Type()),
		AgentRole:   string(agentDef.Role()),
		PromptNames: promptNames,
		IsDeletable: agentDef.Type() == agents.AgentTypeCustom,
	}
	if cad, ok := agentDef.(*agents.CustomAgentDef); ok {
		summary.DisplayName = cad.DisplayName
		summary.Description = cad.Description
		summary.Tools = cad.ToolIDs
		summary.Skills = cad.SkillIDs
	} else {
		summary.DisplayName = agentDef.Desc()
	}
	if summary.Tools == nil {
		summary.Tools = []string{}
	}
	if summary.Skills == nil {
		summary.Skills = []string{}
	}

	return &view_models.AgentDetail{
		AgentSummary: summary,
		Prompts:      agentPrompts,
	}, nil
}

// UpdateAgentPrompt 更新指定 Agent 的某个提示词内容。
func (s *Service) UpdateAgentPrompt(agentName, promptName, content string) (*view_models.AgentDetail, error) {
	agentDef, ok := agents.FindAgent(agentName)
	if !ok {
		return nil, ierror.NewError(fmt.Errorf("agent not found: %s", agentName))
	}

	// 校验 promptName 属于该 Agent
	owned := false
	for _, pn := range agentDef.PromptNames() {
		if pn == promptName {
			owned = true
			break
		}
	}
	if !owned {
		return nil, ierror.NewError(fmt.Errorf("prompt %s does not belong to agent %s", promptName, agentName))
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ierror.NewError(fmt.Errorf("prompt content cannot be empty"))
	}

	if err := agents.SaveAgentPrompt(agentName, promptName, content); err != nil {
		return nil, ierror.NewError(err)
	}
	if err := s.reloadPromptSet(); err != nil {
		logger.Warm("reload prompt set fallback:", err)
	}

	return s.GetAgent(agentName)
}

// ResetAgentPrompt 将指定 Agent 的某个提示词恢复为默认内容。
func (s *Service) ResetAgentPrompt(agentName, promptName string) (*view_models.AgentDetail, error) {
	defaultContent, ok := agents.DefaultAgentPromptContent(agentName, promptName)
	if !ok {
		return nil, ierror.NewError(fmt.Errorf("default prompt not found: %s/%s", agentName, promptName))
	}
	return s.UpdateAgentPrompt(agentName, promptName, defaultContent)
}

// CreateCustomAgent creates a new custom agent.
func (s *Service) CreateCustomAgent(input view_models.CustomAgentInput) (*view_models.AgentDetail, error) {
	if !customAgentIDPattern.MatchString(input.ID) {
		return nil, ierror.NewError(fmt.Errorf("invalid agent ID: must match [a-zA-Z0-9_-]+"))
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, ierror.NewError(fmt.Errorf("agent name cannot be empty"))
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return nil, ierror.NewError(fmt.Errorf("agent prompt cannot be empty"))
	}
	if _, exists := agents.FindAgent(input.ID); exists {
		return nil, ierror.NewError(fmt.Errorf("agent with ID %q already exists", input.ID))
	}

	now := time.Now().Format(time.RFC3339)
	def := agents.CustomAgentDef{
		ID_:         input.ID,
		DisplayName: input.Name,
		Description: input.Description,
		PromptText:  input.Prompt,
		ToolIDs:     input.Tools,
		SkillIDs:    input.Skills,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if def.ToolIDs == nil {
		def.ToolIDs = []string{}
	}
	if def.SkillIDs == nil {
		def.SkillIDs = []string{}
	}

	if err := agents.SaveCustomAgent(def); err != nil {
		return nil, ierror.NewError(err)
	}
	if err := agents.SaveAgentPrompt(def.Name(), def.PromptNames()[0], def.PromptText); err != nil {
		return nil, ierror.NewError(err)
	}
	agents.SyncCustomAgentsToRegistry()

	return s.GetAgent(input.ID)
}

// UpdateCustomAgent updates an existing custom agent.
func (s *Service) UpdateCustomAgent(input view_models.CustomAgentInput) (*view_models.AgentDetail, error) {
	existing, ok := agents.FindAgent(input.ID)
	if !ok {
		return nil, ierror.NewError(fmt.Errorf("agent not found: %s", input.ID))
	}
	if existing.Type() != agents.AgentTypeCustom {
		return nil, ierror.NewError(fmt.Errorf("agent %q is not a custom agent", input.ID))
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, ierror.NewError(fmt.Errorf("agent name cannot be empty"))
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return nil, ierror.NewError(fmt.Errorf("agent prompt cannot be empty"))
	}

	// Preserve the original creation timestamp.
	oldDef, err := agents.LoadCustomAgent(input.ID)
	if err != nil {
		return nil, ierror.NewError(err)
	}

	def := agents.CustomAgentDef{
		ID_:         input.ID,
		DisplayName: input.Name,
		Description: input.Description,
		PromptText:  input.Prompt,
		ToolIDs:     input.Tools,
		SkillIDs:    input.Skills,
		CreatedAt:   oldDef.CreatedAt,
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}
	if def.ToolIDs == nil {
		def.ToolIDs = []string{}
	}
	if def.SkillIDs == nil {
		def.SkillIDs = []string{}
	}

	if err := agents.SaveCustomAgent(def); err != nil {
		return nil, ierror.NewError(err)
	}
	if err := agents.SaveAgentPrompt(def.Name(), def.PromptNames()[0], def.PromptText); err != nil {
		return nil, ierror.NewError(err)
	}
	agents.SyncCustomAgentsToRegistry()

	return s.GetAgent(input.ID)
}

// DeleteCustomAgent deletes a custom agent.
func (s *Service) DeleteCustomAgent(id string) error {
	existing, ok := agents.FindAgent(id)
	if !ok {
		return ierror.NewError(fmt.Errorf("agent not found: %s", id))
	}
	if existing.Type() != agents.AgentTypeCustom {
		return ierror.NewError(fmt.Errorf("agent %q is not a custom agent", id))
	}

	if err := agents.DeleteCustomAgent(id); err != nil {
		return ierror.NewError(err)
	}
	agents.SyncCustomAgentsToRegistry()
	return nil
}
