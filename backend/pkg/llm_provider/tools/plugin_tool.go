package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
)

type PluginInvoker interface {
	CallTool(ctx context.Context, pluginID, kind, toolID, args string) (string, error)
}

type PluginTool struct {
	AliasID     string
	DisplayName string
	Desc        string
	PluginID    string
	ToolID      string
	Kind        string
	InputSchema map[string]interface{}
	Confirm     bool
	Invoker     PluginInvoker
}

var invalidPluginToolNameChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (p *PluginTool) Id() string { return p.AliasID }

func (p *PluginTool) Name() string { return p.DisplayName }

func (p *PluginTool) Description() string { return p.Desc }

func (p *PluginTool) RequireConfirmation() bool { return p.Confirm }

func (p *PluginTool) Tool() einotool.BaseTool { return p }

func (p *PluginTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        p.AliasID,
		Desc:        p.Desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(pluginSchemaToParams(p.InputSchema)),
	}, nil
}

func (p *PluginTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einotool.Option) (string, error) {
	if p.Invoker == nil {
		return "", fmt.Errorf("plugin invoker is nil")
	}
	return p.Invoker.CallTool(ctx, p.PluginID, p.Kind, p.ToolID, argumentsInJSON)
}

func PluginAggregateID(pluginID string) string {
	return "plugin:" + pluginID
}

func PluginUseToolID(pluginID, toolID string) string {
	return pluginToolAliasID(pluginID, "tool", toolID)
}

func PluginViewToolID(pluginID, toolID string) string {
	return pluginToolAliasID(pluginID, "view_tool", toolID)
}

func pluginLegacyViewToolID(pluginID, toolID string) string {
	return pluginToolAliasID(pluginID, "tool", "view_tool_"+toolID)
}

func NewPluginTools(pluginID, pluginName string, caps plugins.Capabilities, invoker PluginInvoker) []ITool {
	var result []ITool
	for _, t := range caps.UseTools {
		result = append(result, &PluginTool{
			AliasID:     PluginUseToolID(pluginID, t.ID),
			DisplayName: pluginName + " / " + t.Name,
			Desc:        t.Description,
			PluginID:    pluginID,
			ToolID:      t.ID,
			Kind:        "use_tool",
			InputSchema: t.InputSchema,
			Confirm:     t.RequireConfirmation,
			Invoker:     invoker,
		})
	}
	for _, t := range caps.ViewTools {
		result = append(result, &PluginTool{
			AliasID:     PluginViewToolID(pluginID, t.ID),
			DisplayName: pluginName + " / " + t.Name,
			Desc:        t.Description,
			PluginID:    pluginID,
			ToolID:      t.ID,
			Kind:        "view_tool",
			InputSchema: t.InputSchema,
			Confirm:     t.RequireConfirmation,
			Invoker:     invoker,
		})
		// Some models infer a legacy alias shape from use_tool names and call
		// `plugin_<id>_tool_view_tool_<toolID>`. Register a compatibility alias
		// so the ToolNode can still resolve the requested plugin view tool.
		result = append(result, &PluginTool{
			AliasID:     pluginLegacyViewToolID(pluginID, t.ID),
			DisplayName: pluginName + " / " + t.Name,
			Desc:        t.Description,
			PluginID:    pluginID,
			ToolID:      t.ID,
			Kind:        "view_tool",
			InputSchema: t.InputSchema,
			Confirm:     t.RequireConfirmation,
			Invoker:     invoker,
		})
	}
	return result
}

func pluginToolAliasID(pluginID, kind, toolID string) string {
	parts := []string{
		"plugin",
		sanitizePluginToolNamePart(pluginID),
		sanitizePluginToolNamePart(kind),
		sanitizePluginToolNamePart(toolID),
	}
	return strings.Join(parts, "_")
}

func sanitizePluginToolNamePart(value string) string {
	sanitized := strings.Trim(invalidPluginToolNameChars.ReplaceAllString(value, "_"), "_")
	if sanitized == "" {
		return "item"
	}
	return sanitized
}

func pluginSchemaToParams(input map[string]interface{}) map[string]*schema.ParameterInfo {
	params := map[string]*schema.ParameterInfo{}
	props, _ := input["properties"].(map[string]interface{})
	required := map[string]bool{}
	if req, ok := input["required"].([]interface{}); ok {
		for _, item := range req {
			if s, ok := item.(string); ok {
				required[s] = true
			}
		}
	}
	for name, raw := range props {
		prop, _ := raw.(map[string]interface{})
		params[name] = pluginParamInfo(prop, required[name])
	}
	return params
}

func pluginParamInfo(prop map[string]interface{}, required bool) *schema.ParameterInfo {
	typ, _ := prop["type"].(string)
	desc, _ := prop["description"].(string)
	info := &schema.ParameterInfo{Type: schema.DataType(typ), Desc: desc, Required: required}
	if info.Type == "" {
		info.Type = schema.String
	}
	if enumRaw, ok := prop["enum"].([]interface{}); ok {
		for _, item := range enumRaw {
			if s, ok := item.(string); ok {
				info.Enum = append(info.Enum, s)
			}
		}
	}
	if info.Type == schema.Array {
		if items, ok := prop["items"].(map[string]interface{}); ok {
			info.ElemInfo = pluginParamInfo(items, false)
		} else {
			info.ElemInfo = &schema.ParameterInfo{Type: schema.String}
		}
	}
	if info.Type == schema.Object {
		info.SubParams = pluginSchemaToParams(prop)
	}
	return info
}
