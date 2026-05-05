import type { Message } from "@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models";
import type { ToolUse, TraceStep } from "@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models/models";

export interface PluginSidePanelPayload {
  viewId: string;
  region: string;
  title?: string;
  data?: Record<string, any>;
}

export interface PluginSidePanelContext {
  callId: string;
  toolAliasId: string;
  toolName: string;
  pluginName: string;
  payload: PluginSidePanelPayload;
  args: string;
  messageUuid: string;
  traceStep?: TraceStep;
  toolUse: ToolUse;
  sourceKind: "view_tool" | "use_tool";
}

interface PluginToolLike {
  tool_id?: string;
  tool_name?: string;
}

export function parsePluginSidePanelPayload(raw: string): PluginSidePanelPayload | null {
  const text = raw.trim();
  if (!text) {
    return null;
  }
  try {
    const parsed = JSON.parse(text);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    if (parsed.region !== "chat_side_panel" || typeof parsed.viewId !== "string" || !parsed.viewId.trim()) {
      return null;
    }
    return parsed as PluginSidePanelPayload;
  } catch {
    return null;
  }
}

function buildFallbackMailListPayload(raw: string, toolUse: PluginToolLike): PluginSidePanelPayload | null {
  const text = raw.trim();
  if (!text) {
    return null;
  }
  try {
    const parsed = JSON.parse(text);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    const toolID = String(toolUse.tool_id || "");
    const toolName = String(toolUse.tool_name || "");
    const looksLikeEmailSearch = (
      toolID.includes("com_lemontea_examples_email") ||
      toolName.toLowerCase().includes("search mail") ||
      toolName.toLowerCase().includes("email / search mail")
    );
    if (!looksLikeEmailSearch) {
      return null;
    }
    const result = typeof (parsed as any).messages !== "undefined"
      ? parsed
      : (parsed as any).result;
    if (!result || !Array.isArray((result as any).messages)) {
      return null;
    }
    return {
      viewId: "mail_list",
      region: "chat_side_panel",
      title: "Mail",
      data: {
        result,
      },
    };
  } catch {
    return null;
  }
}

function buildFallbackMailDetailPayload(raw: string, toolUse: PluginToolLike): PluginSidePanelPayload | null {
  const text = raw.trim();
  if (!text) {
    return null;
  }
  try {
    const parsed = JSON.parse(text);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    const toolID = String(toolUse.tool_id || "");
    const toolName = String(toolUse.tool_name || "").toLowerCase();
    const looksLikeEmailDetail = (
      toolID.includes("com_lemontea_examples_email") ||
      toolName.includes("get mail") ||
      toolName.includes("email / get mail")
    );
    if (!looksLikeEmailDetail) {
      return null;
    }
    const result = typeof (parsed as any).message !== "undefined"
      ? parsed
      : (parsed as any).result;
    const message = (result as any)?.message;
    if (!message || typeof message !== "object") {
      return null;
    }
    return {
      viewId: "mail_detail",
      region: "chat_side_panel",
      title: String(message.subject || "Mail Detail"),
      data: {
        result,
      },
    };
  } catch {
    return null;
  }
}

function getTraceStepForToolUse(message: Message, toolUse: ToolUse): TraceStep | undefined {
  const steps = message.assistant_message_extra?.execution_trace?.steps || [];
  return steps.find((step) => step.step_id === toolUse.call_id);
}

function getToolArgs(step?: TraceStep): string {
  if (!step?.detail_blocks?.length) {
    return "{}";
  }
  const block = step.detail_blocks.find((item) => item.kind === "tool_args" && (item.content?.trim().length || 0) > 0);
  return block?.content?.trim() || "{}";
}

function inferPluginName(toolName: string): string {
  const parts = toolName.split(" / ");
  return parts[0]?.trim() || toolName || "Plugin";
}

export function extractMessagePluginSidePanelContexts(message: Message): PluginSidePanelContext[] {
  const toolUses = message.assistant_message_extra?.tool_uses || [];
  const contexts: PluginSidePanelContext[] = [];

  toolUses.forEach((toolUse) => {
    const rawResult = toolUse.tool_result?.trim() || "";
    if (!rawResult) {
      return;
    }
    const payload = (
      parsePluginSidePanelPayload(rawResult) ||
      buildFallbackMailListPayload(rawResult, toolUse) ||
      buildFallbackMailDetailPayload(rawResult, toolUse)
    );
    if (!payload) {
      return;
    }
    const traceStep = getTraceStepForToolUse(message, toolUse);
    const toolName = String(toolUse.tool_name || "");
    contexts.push({
      callId: String(toolUse.call_id || `${message.message_uuid}-${toolUse.tool_id}`),
      toolAliasId: String(toolUse.tool_id || ""),
      toolName,
      pluginName: inferPluginName(toolName),
      payload,
      args: getToolArgs(traceStep),
      messageUuid: String(message.message_uuid || ""),
      traceStep,
      toolUse,
      sourceKind: payload.viewId === "mail_list" && !toolName.toLowerCase().includes("show mail list") ? "use_tool" : "view_tool",
    });
  });

  return contexts;
}

export function extractAllPluginSidePanelContexts(messages: Message[]): PluginSidePanelContext[] {
  return messages.flatMap((message) => extractMessagePluginSidePanelContexts(message));
}

export function buildPluginSidePanelPayloadFromResult(raw: string, toolUse: PluginToolLike): PluginSidePanelPayload | null {
  return (
    parsePluginSidePanelPayload(raw) ||
    buildFallbackMailListPayload(raw, toolUse) ||
    buildFallbackMailDetailPayload(raw, toolUse)
  );
}
