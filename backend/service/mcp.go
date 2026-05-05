package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	mcpcomponent "github.com/cloudwego/eino-ext/components/tool/mcp"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	mcpproto "github.com/mark3labs/mcp-go/mcp"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	llmtools "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/tools"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/utils/ierror"
)

const (
	toolSourceBuiltin   = "builtin"
	toolSourceMCPCustom = "mcp_custom"
	toolSourcePlugin    = "plugin"
)

var invalidToolNameChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

type mcpConfigFile struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type mcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type toolMeta struct {
	ID          string
	Name        string
	Description string
}

func (s *Service) SelectMCPFolder() (string, error) {
	path, err := s.app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(i18n.TCurrent("app.dialog.select_mcp_folder", nil)).
		PromptForSingleSelection()
	if err != nil {
		return "", ierror.NewError(err)
	}
	return path, nil
}

func (s *Service) AddMCPToolFromFolder(path string) (*view_models.Tool, error) {
	ctx := context.Background()

	resolvedPath, configPath, serverName, serverConfig, err := s.parseMCPFolder(path)
	if err != nil {
		return nil, ierror.NewError(err)
	}

	toolID := buildMCPToolID(resolvedPath)
	server, err := s.storage.GetCustomMCPServerBySourcePath(ctx, resolvedPath)
	if err != nil {
		return nil, ierror.NewError(err)
	}
	if server == nil {
		server, err = s.storage.GetCustomMCPServerBySourcePathUnscoped(ctx, resolvedPath)
		if err != nil {
			return nil, ierror.NewError(err)
		}
		if server == nil {
			server = &data_models.CustomMCPServer{}
		} else {
			server.DeletedAt = data_models.OrmModel{}.DeletedAt
		}
	}

	server.Name = serverName
	server.SourcePath = resolvedPath
	server.ConfigPath = configPath
	server.ToolID = toolID
	server.Description = i18n.TCurrent("mcp.service.prefix", map[string]string{"name": serverName})
	server.Enabled = true

	if err := s.storage.SaveCustomMCPServer(ctx, *server); err != nil {
		return nil, ierror.NewError(err)
	}

	serverSnapshot := *server
	go s.refreshMCPServerDescription(serverSnapshot, serverConfig)

	tool := s.customServerToViewTool(*server)
	return &tool, nil
}

func (s *Service) UpdateMCPToolEnabled(toolID string, enabled bool) error {
	ctx := context.Background()

	server, err := s.storage.GetCustomMCPServerByToolID(ctx, toolID)
	if err != nil {
		return ierror.NewError(err)
	}
	if server == nil {
		return ierror.NewError(fmt.Errorf("mcp tool %s not found", toolID))
	}

	server.Enabled = enabled
	if err := s.storage.SaveCustomMCPServer(ctx, *server); err != nil {
		return ierror.NewError(err)
	}
	return nil
}

func (s *Service) DeleteMCPTool(toolID string) error {
	if err := s.storage.DeleteCustomMCPServerByToolID(context.Background(), toolID); err != nil {
		return ierror.NewError(err)
	}
	return nil
}

func (s *Service) syncCustomMCPTools(ctx context.Context) error {
	servers, err := s.storage.ListCustomMCPServers(ctx)
	if err != nil {
		return err
	}

	for _, server := range servers {
		expectedToolID := buildMCPToolID(server.SourcePath)
		changed := false
		if server.ToolID != expectedToolID {
			server.ToolID = expectedToolID
			changed = true
		}

		if shouldRefreshMCPDescription(server.Description) {
			if _, meta, closeFn, err := s.loadMCPServerTools(ctx, server, nil); err == nil {
				server.Description = summarizeMCPServiceDescription(server.Name, meta)
				changed = true
				if closeFn != nil {
					closeFn()
				}
			}
		}

		if changed {
			if err := s.storage.SaveCustomMCPServer(ctx, server); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) refreshMCPServerDescription(server data_models.CustomMCPServer, configOverride *mcpServerConfig) {
	refreshCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, meta, closeFn, err := s.loadMCPServerTools(refreshCtx, server, configOverride)
	if closeFn != nil {
		defer closeFn()
	}
	if err != nil {
		return
	}

	nextDescription := summarizeMCPServiceDescription(server.Name, meta)
	if strings.TrimSpace(nextDescription) == "" || nextDescription == server.Description {
		return
	}

	server.Description = nextDescription
	_ = s.storage.SaveCustomMCPServer(context.Background(), server)
}

func (s *Service) customServerToViewTool(server data_models.CustomMCPServer) view_models.Tool {
	description := strings.TrimSpace(server.Description)
	if description == "" {
		description = i18n.TCurrent("mcp.service.prefix", map[string]string{"name": server.Name})
	}
	return view_models.Tool{
		Id:          server.ToolID,
		Name:        server.Name,
		Description: description,
		SourceType:  toolSourceMCPCustom,
		Enabled:     server.Enabled,
		IsDeletable: true,
	}
}

func (s *Service) resolveSelectedTools(ctx context.Context, toolIDs []string) ([]einotool.BaseTool, map[string]toolMeta, func(), error) {
	var result []einotool.BaseTool
	metaMap := make(map[string]toolMeta)
	var cleanupFns []func()

	cleanup := func() {
		for _, fn := range cleanupFns {
			fn()
		}
	}

	for _, toolID := range toolIDs {
		if strings.HasPrefix(toolID, "plugin:") && !strings.Contains(strings.TrimPrefix(toolID, "plugin:"), ":") {
			if s.plugins == nil {
				continue
			}
			pluginID := strings.TrimPrefix(toolID, "plugin:")
			summary, ok := s.plugins.Get(pluginID)
			if !ok || !summary.Enabled {
				continue
			}
			caps := pluginsSummaryToCapabilities(*summary)
			for _, pluginTool := range llmtools.NewPluginTools(pluginID, summary.Name, caps, s.plugins) {
				result = append(result, pluginTool.Tool())
				metaMap[pluginTool.Id()] = toolMeta{
					ID:          pluginTool.Id(),
					Name:        pluginTool.Name(),
					Description: pluginTool.Description(),
				}
			}
			continue
		}

		if builtinTool, ok := llmtools.ToolRouter.GetToolByID(toolID); ok {
			result = append(result, builtinTool.Tool())
			metaMap[toolID] = toolMeta{
				ID:          builtinTool.Id(),
				Name:        builtinTool.Name(),
				Description: builtinTool.Description(),
			}
			continue
		}

		server, err := s.storage.GetCustomMCPServerByToolID(ctx, toolID)
		if err != nil {
			cleanup()
			return nil, nil, nil, err
		}
		if server == nil {
			cleanup()
			return nil, nil, nil, fmt.Errorf("tool %s not found", toolID)
		}
		if !server.Enabled {
			continue
		}

		_, _, _, serverConfig, err := s.parseMCPFolder(server.SourcePath)
		if err != nil {
			cleanup()
			return nil, nil, nil, err
		}

		tools, serverMeta, closeFn, err := s.loadMCPServerTools(ctx, *server, serverConfig)
		if err != nil {
			cleanup()
			return nil, nil, nil, err
		}
		if closeFn != nil {
			cleanupFns = append(cleanupFns, closeFn)
		}
		result = append(result, tools...)
		for k, v := range serverMeta {
			metaMap[k] = v
		}
	}

	return result, metaMap, cleanup, nil
}

func pluginsSummaryToCapabilities(summary plugins.Summary) plugins.Capabilities {
	return plugins.Capabilities{
		UseTools:  summary.UseTools,
		ViewTools: summary.ViewTools,
		Agents:    summary.Agents,
		Views:     summary.Views,
		Hooks:     summary.Hooks,
	}
}

func (s *Service) loadMCPServerTools(ctx context.Context, server data_models.CustomMCPServer, configOverride *mcpServerConfig) ([]einotool.BaseTool, map[string]toolMeta, func(), error) {
	config := configOverride
	if config == nil {
		_, _, _, parsedConfig, err := s.parseMCPFolder(server.SourcePath)
		if err != nil {
			return nil, nil, nil, err
		}
		config = parsedConfig
	}

	mcpClient, err := client.NewStdioMCPClient(config.Command, buildMCPEnv(config.Env), config.Args...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("start mcp server failed: %w", err)
	}

	closeFn := func() {
		_ = mcpClient.Close()
	}

	initReq := mcpproto.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcpproto.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcpproto.Implementation{
		Name:    "lemon-tea-desktop",
		Version: "1.0.0",
	}
	if _, err := mcpClient.Initialize(ctx, initReq); err != nil {
		closeFn()
		return nil, nil, nil, fmt.Errorf("initialize mcp server failed: %w", err)
	}

	baseTools, err := mcpcomponent.GetTools(ctx, &mcpcomponent.Config{
		Cli: mcpClient,
	})
	if err != nil {
		closeFn()
		return nil, nil, nil, fmt.Errorf("load mcp tools failed: %w", err)
	}
	if len(baseTools) == 0 {
		closeFn()
		return nil, nil, nil, fmt.Errorf("mcp server did not expose any tools")
	}

	result := make([]einotool.BaseTool, 0, len(baseTools))
	metaMap := make(map[string]toolMeta, len(baseTools))
	usedAliases := make(map[string]int)
	for idx, baseTool := range baseTools {
		info, err := baseTool.Info(ctx)
		if err != nil {
			closeFn()
			return nil, nil, nil, fmt.Errorf("read mcp tool info failed: %w", err)
		}

		alias := buildMCPRemoteToolID(server.ToolID, info.Name, idx, usedAliases)
		displayName := fmt.Sprintf("%s / %s", server.Name, info.Name)
		description := info.Desc
		if description == "" {
			description = fmt.Sprintf("来自 MCP 服务 %s 的工具 %s", server.Name, info.Name)
		}

		aliasedTool := &llmtools.AliasedTool{
			AliasID:     alias,
			DisplayName: displayName,
			Desc:        description,
			Base:        baseTool,
		}

		result = append(result, aliasedTool)
		metaMap[alias] = toolMeta{
			ID:          alias,
			Name:        displayName,
			Description: description,
		}
	}

	return result, metaMap, closeFn, nil
}

func (s *Service) parseMCPFolder(path string) (string, string, string, *mcpServerConfig, error) {
	if strings.TrimSpace(path) == "" {
		return "", "", "", nil, fmt.Errorf("mcp folder path is empty")
	}

	resolvedPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", "", nil, err
	}

	stat, err := os.Stat(resolvedPath)
	if err != nil {
		return "", "", "", nil, err
	}
	if !stat.IsDir() {
		return "", "", "", nil, fmt.Errorf("selected path is not a directory")
	}

	configPath := filepath.Join(resolvedPath, "mcp.json")
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return "", "", "", nil, fmt.Errorf("mcp.json not found in selected directory")
		}
		return "", "", "", nil, err
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", "", nil, err
	}

	var cfg mcpConfigFile
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", "", "", nil, fmt.Errorf("invalid mcp.json: %w", err)
	}

	if len(cfg.MCPServers) == 0 {
		return "", "", "", nil, fmt.Errorf("mcp.json does not contain any mcpServers")
	}
	if len(cfg.MCPServers) > 1 {
		return "", "", "", nil, fmt.Errorf("mcp.json contains multiple mcpServers; only one service per folder is supported")
	}

	for name, serverConfig := range cfg.MCPServers {
		if strings.TrimSpace(serverConfig.Command) == "" {
			return "", "", "", nil, fmt.Errorf("mcp server command is empty")
		}
		serverName := strings.TrimSpace(name)
		if serverName == "" {
			serverName = filepath.Base(resolvedPath)
		}
		return resolvedPath, configPath, serverName, &serverConfig, nil
	}

	return "", "", "", nil, fmt.Errorf("mcp.json parsing failed")
}

func buildMCPToolID(sourcePath string) string {
	sum := sha1.Sum([]byte(sourcePath))
	return "mcp_" + hex.EncodeToString(sum[:8])
}

func buildMCPRemoteToolID(serverToolID, remoteName string, index int, used map[string]int) string {
	sanitized := strings.Trim(invalidToolNameChars.ReplaceAllString(remoteName, "_"), "_")
	if sanitized == "" {
		sanitized = fmt.Sprintf("tool_%d", index+1)
	}
	base := fmt.Sprintf("%s_%s", serverToolID, sanitized)
	count := used[base]
	used[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s_%d", base, count+1)
}

func buildMCPEnv(extra map[string]string) []string {
	merged := make(map[string]string)
	for _, item := range os.Environ() {
		key, value, found := strings.Cut(item, "=")
		if !found {
			continue
		}
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	result := make([]string, 0, len(merged))
	for key, value := range merged {
		result = append(result, key+"="+value)
	}
	return result
}

func summarizeMCPServiceDescription(serverName string, meta map[string]toolMeta) string {
	if len(meta) == 0 {
		return i18n.TCurrent("mcp.service.prefix", map[string]string{"name": serverName})
	}

	parts := make([]string, 0, len(meta))
	keys := make([]string, 0, len(meta))
	for key := range meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		item := meta[key]
		desc := strings.TrimSpace(item.Description)
		name := strings.TrimSpace(item.Name)
		if desc == "" {
			if name != "" {
				parts = append(parts, name)
			}
			continue
		}
		if name != "" && !strings.Contains(desc, name) {
			parts = append(parts, fmt.Sprintf("%s: %s", name, desc))
		} else {
			parts = append(parts, desc)
		}
	}

	if len(parts) == 0 {
		return i18n.TCurrent("mcp.service.prefix", map[string]string{"name": serverName})
	}

	if len(parts) == 1 {
		return parts[0]
	}

	if len(parts) > 3 {
		if i18n.CurrentLocale() == i18n.LocaleEnUS {
			return i18n.Sprintf(i18n.CurrentLocale(), "mcp.service.summary.more_tools", strings.Join(parts[:3], "; "), len(parts))
		}
		return i18n.Sprintf(i18n.CurrentLocale(), "mcp.service.summary.more_tools", strings.Join(parts[:3], "；"), len(parts))
	}

	if i18n.CurrentLocale() == i18n.LocaleEnUS {
		return strings.Join(parts, "; ")
	}
	return strings.Join(parts, "；")
}

func shouldRefreshMCPDescription(description string) bool {
	trimmed := strings.TrimSpace(description)
	return trimmed == "" ||
		trimmed == i18n.TCurrent("mcp.service.default_description", nil) ||
		strings.HasPrefix(trimmed, i18n.TCurrent("mcp.service.prefix", map[string]string{"name": ""}))
}
