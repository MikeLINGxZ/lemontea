import React, {useEffect, useLayoutEffect, useMemo, useRef, useState} from "react";
import { createPortal } from "react-dom";
import { EyeOutlined } from "@ant-design/icons";
import { useTranslation } from 'react-i18next';
import styles from "./index.module.scss";
import ReasoningContent from "@/components/chat/reasoning_message";
import ExecutionTracePanel from "@/components/chat/execution_trace";
import {Service} from "@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service";
import type {Message, Tool as ViewTool} from "@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models";
import {ToolUseStatus, type ToolUse} from "@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models/models";
import {RoleType} from "@bindings/github.com/cloudwego/eino/schema/models";
import MarkdownRenderer from "@/components/markdown_renderer";
import { isDirectMode, buildInterleavedSegments } from "./interleave_utils";
import InterleavedContent from "./interleaved_content";
import { extractMessagePluginSidePanelContexts } from "@/components/chat/plugin_side_panel/utils";

interface ChatMessageProps {
    message: Message
    isLoading?: boolean
    onApprovalDecision?: (approvalId: string, decision: 'allow' | 'reject') => void
    onSendApprovalComment?: (approvalId: string, comment: string) => Promise<void> | void
    onReopenPluginView?: (callId: string) => void
    openPluginViewCallIds?: string[]
    activePluginViewCallId?: string
}

let cachedToolsPromise: Promise<ViewTool[]> | null = null;

function loadAvailableTools(): Promise<ViewTool[]> {
    if (!cachedToolsPromise) {
        cachedToolsPromise = Service.GetTools()
            .then((tools) => tools ?? [])
            .catch(() => []);
    }
    return cachedToolsPromise;
}

function resolveToolMeta(
    toolUse: ToolUse,
    toolDefinitions: Map<string, ViewTool>,
    t: (key: string, options?: Record<string, unknown>) => string,
): { id: string; name: string; description: string } {
    let matchedTool: ViewTool | undefined;

    if (toolUse.tool_id) {
        matchedTool = toolDefinitions.get(toolUse.tool_id);
    }
    if (!matchedTool && toolUse.tool_name) {
        matchedTool = [...toolDefinitions.values()].find((tool) =>
            tool.id === toolUse.tool_name || tool.name === toolUse.tool_name
        );
    }

    return {
        id: matchedTool?.id || toolUse.tool_id || toolUse.tool_name || 'unknown',
        name: matchedTool?.name || toolUse.tool_name || t('chat.message.unnamedTool'),
        description: matchedTool?.description || toolUse.tool_description || t('chat.message.noDescription'),
    };
}

function getToolUseDisplayIndex(toolUse: ToolUse, fallbackIndex: number): number {
    return toolUse.index > 0 ? toolUse.index : fallbackIndex + 1;
}

function parseTime(value: unknown): number | null {
    if (!value) {
        return null;
    }
    const date = new Date(value as string);
    const timestamp = date.getTime();
    return Number.isNaN(timestamp) ? null : timestamp;
}

function isToolUseRunning(toolUse: ToolUse): boolean {
    return toolUse.status === ToolUseStatus.ToolUseStatusRunning;
}

function getToolUseElapsedMs(toolUse: ToolUse, nowMs: number): number {
    const startedAtMs = parseTime(toolUse.started_at);
    const finishedAtMs = parseTime(toolUse.finished_at);
    const baseElapsedMs = toolUse.elapsed_ms ?? 0;

    if (startedAtMs !== null && finishedAtMs !== null) {
        return Math.max(baseElapsedMs, finishedAtMs - startedAtMs, 0);
    }
    if (isToolUseRunning(toolUse) && startedAtMs !== null) {
        return Math.max(baseElapsedMs, nowMs - startedAtMs, 0);
    }
    return Math.max(baseElapsedMs, 0);
}

function formatDuration(elapsedMs: number): string {
    const seconds = Math.max(0, Math.floor(elapsedMs / 1000));
    if (seconds < 60) {
        return `${seconds}s`;
    }
    const minutes = Math.floor(seconds / 60);
    const remainSeconds = seconds % 60;
    return `${minutes}m ${remainSeconds}s`;
}

function getStatusLabel(toolUse: ToolUse, t: (key: string) => string): string {
    if (toolUse.status === ToolUseStatus.ToolUseStatusDone) {
        return t('chat.message.done');
    }
    if (toolUse.status === ToolUseStatus.ToolUseStatusAwaitingApproval) {
        return t('chat.message.awaitingApproval');
    }
    if (toolUse.status === ToolUseStatus.ToolUseStatusRejected) {
        return t('chat.message.rejected');
    }
    if (toolUse.status === ToolUseStatus.ToolUseStatusError) {
        return t('chat.message.failed');
    }
    if (toolUse.status === ToolUseStatus.ToolUseStatusPending) {
        return t('chat.message.pending');
    }
    return t('chat.message.running');
}

function buildToolMetaTooltip(
    toolUse: ToolUse,
    fallbackIndex: number,
    toolDefinitions: Map<string, ViewTool>,
    t: (key: string, options?: Record<string, unknown>) => string,
): string {
    const displayIndex = getToolUseDisplayIndex(toolUse, fallbackIndex);
    const toolMeta = resolveToolMeta(toolUse, toolDefinitions, t);
    const lines = [
        t('chat.message.toolNumber', { index: displayIndex }),
        t('chat.message.id', { value: toolMeta.id }),
        t('chat.message.name', { value: toolMeta.name }),
        t('chat.message.description', { value: toolMeta.description }),
    ];

    return lines.join("\n");
}

function buildContentWithToolMarkers(content: string, toolUses: ToolUse[]): string {
    if (!content || toolUses.length === 0) {
        return content;
    }

    const runes = Array.from(content);
    const markersByPos = new Map<number, string[]>();

    toolUses.forEach((toolUse, idx) => {
        const displayIndex = getToolUseDisplayIndex(toolUse, idx);
        const rawPos = typeof toolUse.content_pos === "number" ? toolUse.content_pos : runes.length;
        const pos = Math.max(0, Math.min(rawPos, runes.length));
        const currentMarkers = markersByPos.get(pos) ?? [];
        currentMarkers.push(`[${displayIndex}]`);
        markersByPos.set(pos, currentMarkers);
    });

    const chunks: string[] = [];
    for (let i = 0; i <= runes.length; i++) {
        const markers = markersByPos.get(i);
        if (markers?.length) {
            chunks.push(markers.join(""));
        }
        if (i < runes.length) {
            chunks.push(runes[i]);
        }
    }

    return chunks.join("");
}

function buildFriendlyFinishError(rawError: string): string {
    const trimmed = rawError.trim();
    if (!trimmed) {
        return "";
    }

    const hasMeaningfulText = (value: string): boolean => /[\p{L}\p{N}\u4e00-\u9fff]/u.test(value);
    const normalizeCandidate = (value: string): string => value.replace(/\s+/g, " ").trim();
    const finalizeCandidate = (value: string): string => {
        const normalized = normalizeCandidate(value)
            .replace(/^[\s"'`[{(,:;]+/, "")
            .replace(/[\s"'`}\]),:;]+$/, "")
            .trim();

        return hasMeaningfulText(normalized) ? normalized : "";
    };

    const timeoutMatch = trimmed.match(/Error:\s*([^"}\]]+)/i);
    if (timeoutMatch?.[1]) {
        return finalizeCandidate(timeoutMatch[1]);
    }

    const mcpMatch = trimmed.match(/mcp server return error:\s*(.+)$/i);
    if (mcpMatch?.[1]) {
        const sanitized = mcpMatch[1]
            .replace(/^\{?["']?content["']?:/i, "")
            .replace(/["'{}\[\]]/g, " ")
            .replace(/\s+/g, " ")
            .trim();

        const extractedError = sanitized.match(/Error:\s*(.+)$/i);
        if (extractedError?.[1]) {
            return finalizeCandidate(extractedError[1]);
        }

        return finalizeCandidate(sanitized);
    }

    return finalizeCandidate(trimmed.replace(/\{.*\}/g, " "));
}

function getVisibleReasoningContent(reasoningContent: string, prefaceReasoningContent: string): string {
    if (!reasoningContent) {
        return "";
    }
    if (prefaceReasoningContent && reasoningContent === prefaceReasoningContent) {
        return "";
    }
    return reasoningContent;
}

const TOOLTIP_OFFSET = 8;
const TOOLTIP_VIEWPORT_GAP = 12;

const HoverTooltip: React.FC<{
    content: string;
    children: React.ReactNode;
    wrapperClassName?: string;
    tooltipClassName?: string;
}> = ({ content, children, wrapperClassName, tooltipClassName }) => {
    const triggerRef = useRef<HTMLSpanElement | null>(null);
    const tooltipRef = useRef<HTMLSpanElement | null>(null);
    const [visible, setVisible] = useState(false);
    const [tooltipStyle, setTooltipStyle] = useState<React.CSSProperties>({});

    useLayoutEffect(() => {
        if (!visible || !triggerRef.current || !tooltipRef.current) {
            return;
        }

        const updatePosition = () => {
            const rect = triggerRef.current?.getBoundingClientRect();
            const tooltipRect = tooltipRef.current?.getBoundingClientRect();
            if (!rect || !tooltipRect) {
                return;
            }

            const centerX = rect.left + rect.width / 2;
            const minLeft = TOOLTIP_VIEWPORT_GAP;
            const maxLeft = Math.max(
                minLeft,
                window.innerWidth - tooltipRect.width - TOOLTIP_VIEWPORT_GAP
            );
            const left = Math.min(
                maxLeft,
                Math.max(minLeft, centerX - tooltipRect.width / 2)
            );

            const preferredTop = rect.top - tooltipRect.height - TOOLTIP_OFFSET;
            const hasEnoughSpaceAbove = preferredTop >= TOOLTIP_VIEWPORT_GAP;
            const top = hasEnoughSpaceAbove
                ? preferredTop
                : Math.min(
                    window.innerHeight - tooltipRect.height - TOOLTIP_VIEWPORT_GAP,
                    rect.bottom + TOOLTIP_OFFSET
                );

            setTooltipStyle({
                position: "fixed",
                left,
                top,
            });
        };

        updatePosition();
        window.addEventListener("scroll", updatePosition, true);
        window.addEventListener("resize", updatePosition);

        return () => {
            window.removeEventListener("scroll", updatePosition, true);
            window.removeEventListener("resize", updatePosition);
        };
    }, [visible]);

    return (
        <>
            <span
                ref={triggerRef}
                className={wrapperClassName}
                onMouseEnter={() => setVisible(true)}
                onMouseLeave={() => setVisible(false)}
            >
                {children}
            </span>
            {visible && typeof document !== "undefined" && createPortal(
                <span
                    ref={tooltipRef}
                    className={`${styles.inlineToolTooltip} ${styles.inlineToolTooltipPortal} ${tooltipClassName ?? ""}`}
                    style={tooltipStyle}
                    role="tooltip"
                >
                    {content}
                </span>,
                document.body
            )}
        </>
    );
};

const InlineToolMarker: React.FC<{
    label: string;
    tooltip: string;
}> = ({ label, tooltip }) => {
    return (
        <HoverTooltip
            content={tooltip}
            wrapperClassName={styles.inlineToolMarkerWrap}
        >
            <sup className={styles.inlineToolMarker}>
                {label}
            </sup>
        </HoverTooltip>
    );
};

function renderTextWithToolMarkers(
    value: string,
    toolUsesByIndex: Map<number, { toolUse: ToolUse; fallbackIndex: number }>,
    toolDefinitions: Map<string, ViewTool>,
    t: (key: string, options?: Record<string, unknown>) => string,
): React.ReactNode[] {
    const parts = value.split(/(\[\d+\])/g);

    return parts.filter(Boolean).map((part, idx) => {
        const match = /^\[(\d+)\]$/.exec(part);
        if (!match) {
            return <React.Fragment key={`text-${idx}`}>{part}</React.Fragment>;
        }

        const displayIndex = Number(match[1]);
        const toolUseInfo = toolUsesByIndex.get(displayIndex);
        if (!toolUseInfo) {
            return <React.Fragment key={`text-${idx}`}>{part}</React.Fragment>;
        }

        return (
            <InlineToolMarker
                key={`marker-${displayIndex}-${idx}`}
                label={part}
                tooltip={buildToolMetaTooltip(toolUseInfo.toolUse, toolUseInfo.fallbackIndex, toolDefinitions, t)}
            />
        );
    });
}

function withInlineToolMarkers(
    children: React.ReactNode,
    toolUsesByIndex: Map<number, { toolUse: ToolUse; fallbackIndex: number }>,
    toolDefinitions: Map<string, ViewTool>,
    t: (key: string, options?: Record<string, unknown>) => string,
): React.ReactNode {
    return React.Children.map(children, (child) => {
        if (typeof child === 'string') {
            return renderTextWithToolMarkers(child, toolUsesByIndex, toolDefinitions, t);
        }
        if (React.isValidElement<{ children?: React.ReactNode }>(child) && child.props.children) {
            return React.cloneElement(child, {
                ...child.props,
                children: withInlineToolMarkers(child.props.children, toolUsesByIndex, toolDefinitions, t),
            });
        }
        return child;
    });
}

const ToolUseItem: React.FC<{ toolUse: ToolUse; fallbackIndex: number; nowMs: number; toolDefinitions: Map<string, ViewTool> }> = ({ toolUse, fallbackIndex, nowMs, toolDefinitions }) => {
    const { t } = useTranslation();
    const [expanded, setExpanded] = useState(false);
    const result = toolUse.tool_result?.trim() || '';
    const isLong = result.length > 120;
    const displayResult = isLong && !expanded ? result.slice(0, 120) + '…' : result;
    const elapsedLabel = formatDuration(getToolUseElapsedMs(toolUse, nowMs));
    const displayIndex = getToolUseDisplayIndex(toolUse, fallbackIndex);
    const toolMeta = resolveToolMeta(toolUse, toolDefinitions, t);
    const statusLabel = getStatusLabel(toolUse, t);
    const statusClassName = isToolUseRunning(toolUse)
        ? styles.toolUseStatusRunning
        : toolUse.status === ToolUseStatus.ToolUseStatusError || toolUse.status === ToolUseStatus.ToolUseStatusRejected
            ? styles.toolUseStatusError
            : styles.toolUseStatusDone;
    const tooltip = buildToolMetaTooltip(toolUse, fallbackIndex, toolDefinitions, t);

    return (
        <div className={styles.toolUseItem}>
            <div
                className={`${styles.toolUseHeader} ${isLong ? styles.toolUseHeaderClickable : ''}`}
                onClick={() => isLong && setExpanded(!expanded)}
                role={isLong ? 'button' : undefined}
            >
                <div className={styles.toolUseMain}>
                    <span className={styles.toolUseBadge}>#{displayIndex}</span>
                    <span className={styles.toolUseName}>{toolMeta.name}</span>
                </div>
                <div className={styles.toolUseMeta}>
                    <span className={`${styles.toolUseStatus} ${statusClassName}`}>
                        {elapsedLabel} · {statusLabel}
                    </span>
                    {isLong && (
                        <span className={styles.toolUseToggle}>
                            {expanded ? t('chat.message.collapse') : t('chat.message.expand')}
                        </span>
                    )}
                </div>
            </div>
            {result && (
                <pre className={styles.toolUseResult}>{displayResult}</pre>
            )}
            <div className={styles.toolUseTooltip} role="tooltip">
                {tooltip}
            </div>
        </div>
    );
};

const ToolUsesSection: React.FC<{ toolUses: ToolUse[]; toolDefinitions: Map<string, ViewTool> }> = ({ toolUses, toolDefinitions }) => {
    const { t } = useTranslation();
    const [nowMs, setNowMs] = useState(() => Date.now());
    const hasRunningTool = toolUses.some(isToolUseRunning);

    useEffect(() => {
        if (!hasRunningTool) {
            return;
        }
        setNowMs(Date.now());
        const timer = window.setInterval(() => {
            setNowMs(Date.now());
        }, 1000);
        return () => window.clearInterval(timer);
    }, [hasRunningTool]);

    return (
        <div className={styles.toolUsesSection}>
            <div className={styles.toolUsesHeader}>
                <svg className={styles.toolUsesIcon} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>
                </svg>
                <span>{t('chat.message.toolCalls')}</span>
                <span className={styles.toolUsesCount}>({toolUses.length})</span>
            </div>
            <div className={styles.toolUsesList}>
                {toolUses.map((toolUse, idx) => (
                    <ToolUseItem
                        key={toolUse.call_id || `${toolUse.tool_name}-${idx}`}
                        toolUse={toolUse}
                        fallbackIndex={idx}
                        nowMs={nowMs}
                        toolDefinitions={toolDefinitions}
                    />
                ))}
            </div>
        </div>
    );
};

const ChatMessage: React.FC<ChatMessageProps> = ({
    message,
    isLoading = false,
    onApprovalDecision,
    onSendApprovalComment,
    onReopenPluginView,
    openPluginViewCallIds = [],
    activePluginViewCallId = '',
}: ChatMessageProps) => {
    const { t } = useTranslation();
    const [toolDefinitions, setToolDefinitions] = useState<Map<string, ViewTool>>(new Map());
    const isUser = message.role === RoleType.User;
    const wrapperClass = isUser ? styles.userMessageWrapper : styles.assistantMessageWrapper;
    const toolUses = useMemo(() => {
        const currentToolUses = message.assistant_message_extra?.tool_uses ?? [];
        return [...currentToolUses].sort((a, b) => {
            const aIndex = getToolUseDisplayIndex(a, 0);
            const bIndex = getToolUseDisplayIndex(b, 0);
            return aIndex - bIndex;
        });
    }, [message.assistant_message_extra?.tool_uses]);
    const traceSteps = message.assistant_message_extra?.execution_trace?.steps ?? [];
    const toolUsesByIndex = useMemo(() => {
        const map = new Map<number, { toolUse: ToolUse; fallbackIndex: number }>();
        toolUses.forEach((toolUse, idx) => {
            map.set(getToolUseDisplayIndex(toolUse, idx), { toolUse, fallbackIndex: idx });
        });
        return map;
    }, [toolUses]);
    const pluginViewContexts = useMemo(
        () => extractMessagePluginSidePanelContexts(message),
        [message]
    );

    useEffect(() => {
        let active = true;
        loadAvailableTools().then((tools) => {
            if (!active) {
                return;
            }
            setToolDefinitions(new Map(tools.map((tool) => [tool.id, tool])));
        });
        return () => {
            active = false;
        };
    }, []);

    const messageContent = message.content?.trim() ?? "";
    const reasoningContent = message.reasoning_content?.trim() ?? "";
    const prefaceContent = message.assistant_message_extra?.preface_content?.trim() ?? "";
    const prefaceReasoningContent = message.assistant_message_extra?.preface_reasoning_content?.trim() ?? "";
    const visibleReasoningContent = getVisibleReasoningContent(reasoningContent, prefaceReasoningContent);
    const finishReason = message.assistant_message_extra?.finish_reason?.trim() ?? "";
    const finishError = message.assistant_message_extra?.finish_error?.trim() ?? "";
    const currentStage = message.assistant_message_extra?.current_stage?.trim() ?? "";
    const friendlyFinishError = useMemo(() => buildFriendlyFinishError(finishError), [finishError]);
    const isReasoningStreaming = !isUser && isLoading && !finishReason && messageContent.length === 0;
    const isEmptyAssistant = !isUser &&
        !messageContent &&
        !visibleReasoningContent &&
        !prefaceContent &&
        !prefaceReasoningContent &&
        traceSteps.length === 0 &&
        toolUses.length === 0 &&
        (message.assistant_message_extra?.finish_error == "");
    const hasTrace = traceSteps.length > 0;
    const hasVisibleProgress = hasTrace || currentStage.length > 0 || visibleReasoningContent.length > 0 || prefaceContent.length > 0 || prefaceReasoningContent.length > 0;
    const shouldShowHeadLoading = isLoading && isEmptyAssistant && !hasVisibleProgress;
    const shouldShowTailLoading = !isUser && isLoading && !finishReason && hasVisibleProgress;
    const useInterleaved = useMemo(() => {
        return !isUser && isDirectMode(message.assistant_message_extra) && toolUses.length > 0;
    }, [isUser, message.assistant_message_extra, toolUses.length]);
    const interleavedSegments = useMemo(() => {
        if (!useInterleaved) return [];
        return buildInterleavedSegments(message.content ?? '', toolUses, traceSteps);
    }, [useInterleaved, message.content, toolUses, traceSteps]);

    const getDisplayContent = () => {
        if (messageContent) {
            return message.content;
        }
        return '';
    };

    if (isEmptyAssistant && !isLoading) {
        return null;
    }

    const handleFileClick = (filePath: string) => {
        if (filePath) {
            Service.OpenFile(filePath).catch((err) => {
                console.error('打开文件失败:', err);
            });
        }
    };

    return (
        <div className={styles.ChatMessage}>
            <div className={`${styles.message} ${wrapperClass}`}>
                <div className={styles.messageContainer}>
                    {isUser ? (
                        <>
                            <div className={`${styles.messageContent} ${styles.markdownContent}`}>
                                <MarkdownRenderer
                                    content={getDisplayContent()}
                                    variant="user"
                                />
                            </div>
                            {(message.user_message_extra?.files?.length ?? 0) > 0 && (
                                <div className={styles.fileList}>
                                    {message.user_message_extra!.files!.map((file, index) => (
                                        <div
                                            key={index}
                                            className={styles.fileItem}
                                            onClick={() => handleFileClick(file.path)}
                                            title={t('chat.message.openFile', { name: file.name })}
                                        >
                                            <span className={styles.fileType}>{file.mine_type}</span>
                                            <span className={styles.fileName}>{file.name}</span>
                                            {file.mine_type && (
                                                <span className={styles.fileMimeType}>{file.mine_type}</span>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </>
                    ) : (
                        <div>
                            {shouldShowHeadLoading && (
                                <div className={styles.loadingIndicator}>
                                    <span className={styles.loadingDot} />
                                    <span className={styles.loadingDot} />
                                    <span className={styles.loadingDot} />
                                </div>
                            )}

                            {prefaceReasoningContent && (
                                <ReasoningContent
                                    content={prefaceReasoningContent}
                                />
                            )}

                            {prefaceContent && (
                                <div className={`${styles.messageContent} ${styles.markdownContent}`}>
                                    <MarkdownRenderer
                                        content={prefaceContent}
                                        variant="assistant"
                                        decorateText={useInterleaved ? undefined : (children) => withInlineToolMarkers(children, toolUsesByIndex, toolDefinitions, t)}
                                    />
                                </div>
                            )}

                            {useInterleaved ? (
                                <>
                                    {visibleReasoningContent && (
                                        <ReasoningContent
                                            content={visibleReasoningContent}
                                            isStreaming={isReasoningStreaming}
                                        />
                                    )}
                                    <div className={`${styles.messageContent} ${styles.markdownContent}`}>
                                        <InterleavedContent
                                            segments={interleavedSegments}
                                            isStreaming={isLoading}
                                            onApprovalDecision={onApprovalDecision}
                                            onSendApprovalComment={onSendApprovalComment}
                                        />
                                    </div>
                                </>
                            ) : (
                                <>
                                    <ExecutionTracePanel
                                        trace={message.assistant_message_extra?.execution_trace}
                                        currentStage={message.assistant_message_extra?.current_stage}
                                        retryCount={message.assistant_message_extra?.retry_count}
                                        isStreaming={isLoading}
                                        onApprovalDecision={onApprovalDecision}
                                        onSendApprovalComment={onSendApprovalComment}
                                    />

                                    {visibleReasoningContent && (
                                        <ReasoningContent
                                            content={visibleReasoningContent}
                                            isStreaming={isReasoningStreaming}
                                        />
                                    )}

                                    <div className={`${styles.messageContent} ${styles.markdownContent}`}>
                                        <MarkdownRenderer
                                            content={getDisplayContent()}
                                            variant="assistant"
                                            transformContent={(content) => buildContentWithToolMarkers(content, toolUses)}
                                            decorateText={(children) => withInlineToolMarkers(children, toolUsesByIndex, toolDefinitions, t)}
                                        />
                                    </div>

                                    {toolUses.length > 0 && traceSteps.length === 0 && (
                                        <ToolUsesSection toolUses={toolUses} toolDefinitions={toolDefinitions} />
                                    )}

                                </>
                            )}

                            {pluginViewContexts.length > 0 && (
                                <div className={styles.pluginViewActions}>
                                    {pluginViewContexts.map((context) => {
                                        const isOpen = openPluginViewCallIds.includes(context.callId);
                                        const isActive = activePluginViewCallId === context.callId;

                                        return (
                                            <button
                                                key={context.callId}
                                                type="button"
                                                className={`${styles.pluginViewActionButton} ${isOpen ? styles.pluginViewActionButtonOpen : ''} ${isActive ? styles.pluginViewActionButtonActive : ''}`}
                                                onClick={() => onReopenPluginView?.(context.callId)}
                                                title={t('chat.message.reopenPluginView', {
                                                    title: context.payload.title || context.payload.viewId,
                                                }) + `\n${context.pluginName} · 插件`}
                                            >
                                                <span className={`${styles.pluginViewActionIcon} ${isOpen ? styles.pluginViewActionIconOpen : ''}`}>
                                                    <EyeOutlined />
                                                </span>
                                                <span className={styles.pluginViewActionContent}>
                                                    <span className={styles.pluginViewActionTitle}>
                                                        {context.payload.title || context.payload.viewId}
                                                    </span>
                                                </span>
                                            </button>
                                        );
                                    })}
                                </div>
                            )}

                            {!messageContent && finishReason === 'error' && friendlyFinishError && (
                                <div
                                    className={styles.errorSummary}
                                    title={finishError}
                                >
                                    {friendlyFinishError}
                                </div>
                            )}

                            {shouldShowTailLoading && (
                                <div className={`${styles.loadingIndicator} ${styles.tailLoadingIndicator}`}>
                                    <span className={styles.loadingDot} />
                                    <span className={styles.loadingDot} />
                                    <span className={styles.loadingDot} />
                                </div>
                            )}
                        </div>
                    )}
                    {message.assistant_message_extra?.finish_reason === 'error' && (
                        finishError ? (
                            <HoverTooltip
                                content={finishError}
                                wrapperClassName={styles.finishReasonErrorWrap}
                                tooltipClassName={styles.finishReasonTooltip}
                            >
                                <div className={styles.finishReasonError}>{t('chat.message.stoppedByError')}</div>
                            </HoverTooltip>
                        ) : (
                            <div className={styles.finishReasonError}>{t('chat.message.stoppedByError')}</div>
                        )
                    )}
                    {message.assistant_message_extra?.finish_reason === 'user stop' && (
                        <div className={styles.finishReasonUserStop}>{t('chat.message.stoppedByUser')}</div>
                    )}
                </div>
            </div>
        </div>
    );
};

ChatMessage.displayName = 'ChatMessage';
export default ChatMessage;
