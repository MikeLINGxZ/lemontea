import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Events } from '@wailsio/runtime';
import styles from '@/pages/home/chat/index.module.scss';
import MessageList, {
  type MessageListRef,
} from '@/components/chat/message_list';
import ChatTitle from '@/components/chat/title';
import ChatInput from '@/components/chat/input';
import WelcomeEmpty from '@/components/chat/welcome';
import PluginSidePanel from '@/components/chat/plugin_side_panel';
import {
  buildPluginSidePanelPayloadFromResult,
  extractAllPluginSidePanelContexts,
  type PluginSidePanelContext,
} from '@/components/chat/plugin_side_panel/utils';
import {
  type Chat as ChatType,
  FileInfo,
  type Message,
  Model,
  Task,
  Tool,
} from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models';
import { Service } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service';
import {
  RouteType,
  TaskStatus,
  ToolApprovalDecision,
  ToolApprovalResponse,
} from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models/models';
import { RoleType } from '@bindings/github.com/cloudwego/eino/schema';
import {
  BuildTaskFromCompletions,
  CompletionsUtils,
  SubscribeTaskStream,
  type TaskStreamEvent,
} from '@/utils/completions.ts';
import { useNavigate } from 'react-router-dom';
import { notify } from '@/utils/notification.ts';
import { useTranslation } from 'react-i18next';
import { getDefaultModelConfig } from '@/utils/defaultModel';

function readStoredSelectedToolIds(): {
  ids: string[];
  hasStoredValue: boolean;
} {
  try {
    const raw = localStorage.getItem('chat_selected_tools');
    if (!raw) {
      return { ids: [], hasStoredValue: false };
    }
    const parsed = JSON.parse(raw) as string[];
    return {
      ids: Array.isArray(parsed) ? parsed : [],
      hasStoredValue: true,
    };
  } catch {
    return { ids: [], hasStoredValue: false };
  }
}

interface ChatProps {
  // 对话uuid
  chatUuid?: string;
  // 是否折叠菜单栏
  isSidebarCollapsed: boolean;
  // 点击菜单栏事件
  onToggleSidebar: () => void;
  //
  onChatChange: (chatUuid: string) => void;
  // 刷新聊天列表
  refreshChatList: (() => void) | null;
  /** 同步「正在生成」的会话 uuid 列表给侧边栏等 */
  onGeneratingUuidsChange?: (uuids: string[]) => void;
  /** 注册按 chatUuid 停止生成（与输入框停止一致） */
  onRegisterStopGenerationForChat?: (fn: (chatUuid: string) => void) => void;
}

function assistantHasSubstantiveOutput(message: Message): boolean {
  if (message.role !== RoleType.Assistant) return false;
  const content = message.content?.trim() ?? '';
  const reasoning = message.reasoning_content?.trim() ?? '';
  const prefaceContent =
    message.assistant_message_extra?.preface_content?.trim() ?? '';
  const prefaceReasoning =
    message.assistant_message_extra?.preface_reasoning_content?.trim() ?? '';
  const toolUses = message.assistant_message_extra?.tool_uses?.length ?? 0;
  const traceSteps =
    message.assistant_message_extra?.execution_trace?.steps?.length ?? 0;
  return (
    content.length > 0 ||
    reasoning.length > 0 ||
    prefaceContent.length > 0 ||
    prefaceReasoning.length > 0 ||
    toolUses > 0 ||
    traceSteps > 0
  );
}

function isAssistantPlaceholderMessage(message: Message): boolean {
  if (message.role !== RoleType.Assistant) return false;
  if (assistantHasSubstantiveOutput(message)) return false;
  const finishReason =
    message.assistant_message_extra?.finish_reason?.trim() ?? '';
  const finishError =
    message.assistant_message_extra?.finish_error?.trim() ?? '';
  return finishReason === '' && finishError === '';
}

function toTaskStatus(status: string): TaskStatus {
  switch (status) {
    case TaskStatus.TaskStatusPending:
      return TaskStatus.TaskStatusPending;
    case TaskStatus.TaskStatusRunning:
      return TaskStatus.TaskStatusRunning;
    case TaskStatus.TaskStatusWaitingApproval:
      return TaskStatus.TaskStatusWaitingApproval;
    case TaskStatus.TaskStatusCompleted:
      return TaskStatus.TaskStatusCompleted;
    case TaskStatus.TaskStatusFailed:
      return TaskStatus.TaskStatusFailed;
    case TaskStatus.TaskStatusStopped:
      return TaskStatus.TaskStatusStopped;
    default:
      return TaskStatus.$zero;
  }
}

function isTerminalTaskStatus(
  status: string | TaskStatus | undefined | null
): boolean {
  return (
    status === TaskStatus.TaskStatusCompleted ||
    status === TaskStatus.TaskStatusFailed ||
    status === TaskStatus.TaskStatusStopped
  );
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return fallback;
}

function mergeAssistantMessage(current: Message, incoming: Message): Message {
  return {
    ...current,
    ...incoming,
    assistant_message_extra: incoming.assistant_message_extra
      ? {
          ...(current.assistant_message_extra || {}),
          ...incoming.assistant_message_extra,
          execution_trace:
            incoming.assistant_message_extra.execution_trace ||
            current.assistant_message_extra?.execution_trace,
        }
      : current.assistant_message_extra,
  };
}

function findMessageIndexForUpsert(list: Message[], incoming: Message): number {
  if (incoming.message_uuid) {
    const exactIndex = list.findIndex(
      item => item.message_uuid === incoming.message_uuid
    );
    if (exactIndex !== -1) {
      return exactIndex;
    }
  }

  if (incoming.role !== RoleType.Assistant) {
    return -1;
  }

  for (let index = list.length - 1; index >= 0; index -= 1) {
    const message = list[index];
    if (!isAssistantPlaceholderMessage(message)) {
      continue;
    }
    const incomingChatUuid = incoming.chat_uuid ?? '';
    const currentChatUuid = message.chat_uuid ?? '';
    if (
      !incomingChatUuid ||
      !currentChatUuid ||
      incomingChatUuid === currentChatUuid
    ) {
      return index;
    }
  }

  return -1;
}

function upsertMessage(list: Message[], incoming: Message): Message[] {
  const index = findMessageIndexForUpsert(list, incoming);
  if (index === -1) {
    return [...list, incoming];
  }
  const next = [...list];
  next[index] = mergeAssistantMessage(next[index], incoming);
  return next;
}

function mergeStreamingAssistant(
  propUuid: string,
  list: Message[],
  cache: Record<string, Message>
): Message[] {
  const cached = cache[propUuid];
  if (!cached) return list;
  return upsertMessage(list, cached);
}

const HIDDEN_BUILTIN_TOOL_IDS = new Set(['file_tool']);
const SIDE_PANEL_WIDTH_KEY = 'chat_plugin_side_panel_width';
const SIDE_PANEL_CLOSED_VIEW_KEY = 'chat_plugin_side_panel_closed_views';
const SIDE_PANEL_MIN_WIDTH = 280;
const CHAT_MAIN_MIN_WIDTH = 580;
const SIDE_PANEL_RESIZE_HANDLE_WIDTH = 6;
type SidePanelStatus = 'loading' | 'ready' | 'empty' | 'error' | 'stale';

function getPluginSidePanelMessageCount(context: PluginSidePanelContext | null): number {
  if (!context) {
    return 0;
  }
  const result = context.payload.data?.result;
  if (Array.isArray(result?.messages)) {
    return result.messages.length;
  }
  if (Array.isArray(context.payload.data?.messages)) {
    return context.payload.data.messages.length;
  }
  return 0;
}

function getPluginSidePanelAutoStatus(
  context: PluginSidePanelContext,
  currentStatus?: SidePanelStatus
): SidePanelStatus {
  if (
    currentStatus === 'loading' ||
    currentStatus === 'error' ||
    currentStatus === 'stale'
  ) {
    return currentStatus;
  }
  if (context.payload.viewId === 'mail_list') {
    return getPluginSidePanelMessageCount(context) === 0 ? 'empty' : 'ready';
  }
  if (context.payload.viewId === 'mail_detail') {
    return context.payload.data?.result?.message ? 'ready' : 'empty';
  }
  return 'ready';
}

function sanitizePluginToolNamePart(value: string): string {
  const sanitized = value.replace(/[^a-zA-Z0-9_-]+/g, '_').replace(/^_+|_+$/g, '');
  return sanitized || 'item';
}

function resolvePluginIDForContext(context: PluginSidePanelContext, tools: Tool[]): string {
  const aliasId = String(context.toolAliasId || '');
  const pluginCandidates = tools.filter(tool => tool.source_type === 'plugin');

  const aliasMatch = pluginCandidates.find(tool => {
    const pluginID = String(tool.id || '').replace(/^plugin:/, '');
    return aliasId.startsWith(`plugin_${sanitizePluginToolNamePart(pluginID)}_`);
  });
  if (aliasMatch?.id) {
    return String(aliasMatch.id).replace(/^plugin:/, '');
  }

  const nameMatch = pluginCandidates.find(tool => {
    if (tool.name !== context.pluginName) {
      return false;
    }
    return Array.isArray(tool.use_tools) && tool.use_tools.some(item => item.id === 'get_mail');
  });
  if (nameMatch?.id) {
    return String(nameMatch.id).replace(/^plugin:/, '');
  }

  return '';
}

function getPluginToolResultError(raw: string): string {
  try {
    const parsed = JSON.parse(raw) as { ok?: boolean; error?: unknown };
    if (parsed && parsed.ok === false && parsed.error) {
      return String(parsed.error);
    }
  } catch {
    // The normal plugin result is not guaranteed to be an object with ok/error.
  }
  return '';
}

function readStoredClosedSidePanelViews(): Record<string, string[]> {
  try {
    const raw = localStorage.getItem(SIDE_PANEL_CLOSED_VIEW_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {};
    }
    return Object.entries(parsed).reduce<Record<string, string[]>>(
      (result, [chatUuid, callIds]) => {
        if (typeof chatUuid === 'string') {
          if (Array.isArray(callIds)) {
            result[chatUuid] = callIds.filter(
              (callId): callId is string => typeof callId === 'string'
            );
          } else if (typeof callIds === 'string') {
            result[chatUuid] = [callIds];
          }
        }
        return result;
      },
      {}
    );
  } catch {
    return {};
  }
}

function writeStoredClosedSidePanelViews(views: Record<string, string[]>): void {
  try {
    const entries = Object.entries(views).reduce<Record<string, string[]>>(
      (result, [chatUuid, callIds]) => {
        const nextCallIds = callIds.filter(callId => callId.trim() !== '');
        if (chatUuid.trim() !== '' && nextCallIds.length > 0) {
          result[chatUuid] = nextCallIds;
        }
        return result;
      },
      {}
    );
    if (Object.keys(entries).length === 0) {
      localStorage.removeItem(SIDE_PANEL_CLOSED_VIEW_KEY);
      return;
    }
    localStorage.setItem(SIDE_PANEL_CLOSED_VIEW_KEY, JSON.stringify(entries));
  } catch {
    // ignore storage errors
  }
}

const Chat: React.FC<ChatProps> = ({
  chatUuid,
  isSidebarCollapsed,
  onToggleSidebar,
  onChatChange,
  refreshChatList,
  onGeneratingUuidsChange,
  onRegisterStopGenerationForChat,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const propUuid = chatUuid ?? '';
  const [loading, setLoading] = useState<boolean>(!!chatUuid);
  const [chatInfo, setChatInfo] = useState<ChatType | null>(null);
  // 当前对话uuid
  const [currentChatUuid, setCurrentChatUuid] = useState<string>('');
  const currentChatUuidRef = useRef<string>('');
  // 所选择的模型
  const [selectModel, setSelectModelId] = useState<number>(-1);
  const [selectModelName, setSelectModelName] = useState<string>('');
  // 可用模型
  const [availableModels, setAvailableModels] = useState<Model[]>([]);
  const availableModelsRef = useRef<Model[]>([]);
  // 可用工具
  const [availableTools, setAvailableTools] = useState<Tool[]>([]);
  // 用户选中的自定义 MCP 工具 id 列表（持久化到 localStorage）
  const initialStoredToolSelection = useMemo(
    () => readStoredSelectedToolIds(),
    []
  );
  const [selectedToolIds, setSelectedToolIds] = useState<string[]>(
    initialStoredToolSelection.ids
  );
  const [sidePanelWidth, setSidePanelWidth] = useState<number>(() => {
    try {
      const raw = localStorage.getItem(SIDE_PANEL_WIDTH_KEY);
      const width = Number(raw);
      if (Number.isFinite(width)) {
        return Math.max(SIDE_PANEL_MIN_WIDTH, width);
      }
    } catch {
      // ignore storage errors
    }
    return 360;
  });
  const chatWorkspaceRef = useRef<HTMLDivElement | null>(null);
  const [activeSidePanelCallId, setActiveSidePanelCallId] = useState('');
  const [sidePanelStatusByCallId, setSidePanelStatusByCallId] = useState<
    Record<string, SidePanelStatus>
  >({});
  const [sidePanelErrorByCallId, setSidePanelErrorByCallId] = useState<
    Record<string, string>
  >({});
  const [sidePanelOverridesByCallId, setSidePanelOverridesByCallId] = useState<
    Record<string, PluginSidePanelContext>
  >({});
  const [sidePanelExtraContextsByCallId, setSidePanelExtraContextsByCallId] =
    useState<Record<string, PluginSidePanelContext>>({});
  const [closedSidePanelByChat, setClosedSidePanelByChat] = useState<
    Record<string, string[]>
  >(() => readStoredClosedSidePanelViews());
  // 后端已开始流式推送的会话 uuid（支持多会话后台生成）
  const [generatingUuids, setGeneratingUuids] = useState<string[]>([]);
  const [activeTasksByChat, setActiveTasksByChat] = useState<
    Record<string, Task>
  >({});
  const activeTasksByChatRef = useRef<Record<string, Task>>({});
  const [pendingExistingChatUuids, setPendingExistingChatUuids] = useState<
    string[]
  >([]);
  const [pendingNewChatCount, setPendingNewChatCount] = useState(0);
  // 输入框文字内容
  const [inputMessage, setInputMessage] = useState<string>('');
  // 输入框选择文件
  const [inputFiles, setInputFiles] = useState<FileInfo[]>([]);
  const [displayTitle, setDisplayTitle] = useState<string>('');
  // 当前聊天消息
  const [messages, setMessages] = useState<Message[]>([]);
  // 消息列表引用
  const messageListRef = useRef<MessageListRef>(null);
  // 标记：当新对话首次获得 UUID 时，跳过因 prop 变化触发的数据重新加载
  const skipNextFetchRef = useRef<string | null>(null);
  // 用于重置 ChatInput 内部状态（切换对话/新建对话时递增）
  const [inputResetKey, setInputResetKey] = useState<number>(0);
  // 欢迎页建议预填到输入框的草稿文本
  const [pendingInitialInput, setPendingInitialInput] = useState<string>('');
  // 子Agent面板是否打开

  // 当前路由可见会话，用于将流式事件路由到正确会话
  const visibleChatUuidRef = useRef<string>('');
  // 非当前可见会话的进行中 assistant 快照（切回时与 DB 合并）
  const streamingAssistantByChatRef = useRef<Record<string, Message>>({});
  // 活跃任务订阅
  const taskSubscriptionsRef = useRef<Map<string, () => void>>(new Map());
  // 切换会话时的拉取序号，避免快速切换或 Strict Mode 下二次 setState 造成「闪两次 loading」
  const chatFetchSeqRef = useRef(0);

  const refreshAvailableTools = useCallback(async () => {
    const tools = await Service.GetTools();
    setAvailableTools(tools);
    return tools;
  }, []);

  const defaultBuiltinToolIds = useMemo(
    () =>
      availableTools
        .filter(
          tool =>
            tool.source_type === 'builtin' &&
            !HIDDEN_BUILTIN_TOOL_IDS.has(tool.id)
        )
        .map(tool => tool.id),
    [availableTools]
  );

  const customTools = useMemo(
    () =>
      availableTools.filter(
        tool =>
          tool.source_type === 'mcp_custom' || tool.source_type === 'plugin'
      ),
    [availableTools]
  );

  const effectiveSelectedToolIds = useMemo(
    () => [...new Set([...defaultBuiltinToolIds, ...selectedToolIds])],
    [defaultBuiltinToolIds, selectedToolIds]
  );

  const sidePanelContexts = useMemo(
    () => extractAllPluginSidePanelContexts(messages),
    [messages]
  );

  const closedSidePanelCallIds = useMemo(
    () => (propUuid ? closedSidePanelByChat[propUuid] || [] : []),
    [closedSidePanelByChat, propUuid]
  );

  const sidePanelContextsWithOverrides = useMemo(
    () =>
      sidePanelContexts.map(
        context => sidePanelOverridesByCallId[context.callId] || context
      ),
    [sidePanelContexts, sidePanelOverridesByCallId]
  );

  const mergedSidePanelContexts = useMemo(
    () => [
      ...sidePanelContextsWithOverrides,
      ...Object.values(sidePanelExtraContextsByCallId),
    ],
    [sidePanelContextsWithOverrides, sidePanelExtraContextsByCallId]
  );

  const latestSidePanelContext = useMemo(
    () => mergedSidePanelContexts[mergedSidePanelContexts.length - 1] || null,
    [mergedSidePanelContexts]
  );

  const openSidePanelContexts = useMemo(
    () =>
      mergedSidePanelContexts.filter(
        context => !closedSidePanelCallIds.includes(context.callId)
      ),
    [closedSidePanelCallIds, mergedSidePanelContexts]
  );

  const activeSidePanelContext = useMemo(() => {
    if (activeSidePanelCallId) {
      const active = mergedSidePanelContexts.find(
        item =>
          item.callId === activeSidePanelCallId &&
          !closedSidePanelCallIds.includes(item.callId)
      );
      if (active) {
        return active;
      }
    }
    for (let index = mergedSidePanelContexts.length - 1; index >= 0; index -= 1) {
      const item = mergedSidePanelContexts[index];
      if (!closedSidePanelCallIds.includes(item.callId)) {
        return item;
      }
    }
    return null;
  }, [activeSidePanelCallId, closedSidePanelCallIds, mergedSidePanelContexts]);

  const getMaxAllowedSidePanelWidth = useCallback(() => {
    const workspaceWidth = chatWorkspaceRef.current?.clientWidth || 0;
    if (workspaceWidth <= 0) {
      return sidePanelWidth;
    }
    return Math.max(
      SIDE_PANEL_MIN_WIDTH,
      workspaceWidth - CHAT_MAIN_MIN_WIDTH - SIDE_PANEL_RESIZE_HANDLE_WIDTH
    );
  }, [sidePanelWidth]);

  const effectiveSidePanelWidth = useMemo(
    () => Math.min(sidePanelWidth, getMaxAllowedSidePanelWidth()),
    [getMaxAllowedSidePanelWidth, sidePanelWidth]
  );

  const handleSidePanelResizeStart = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      if (window.innerWidth <= 768) return;
      event.preventDefault();
      const startX = event.clientX;
      const startWidth = effectiveSidePanelWidth;
      let latestWidth = startWidth;
      let rafId = 0;

      const onMove = (e: MouseEvent) => {
        const maxAllowedWidth = getMaxAllowedSidePanelWidth();
        latestWidth = Math.min(
          maxAllowedWidth,
          Math.max(SIDE_PANEL_MIN_WIDTH, startWidth - (e.clientX - startX))
        );
        if (rafId) return;
        rafId = requestAnimationFrame(() => {
          rafId = 0;
          setSidePanelWidth(latestWidth);
        });
      };

      const onUp = () => {
        if (rafId) cancelAnimationFrame(rafId);
        setSidePanelWidth(latestWidth);
        try {
          localStorage.setItem(SIDE_PANEL_WIDTH_KEY, String(latestWidth));
        } catch {
          // ignore storage errors
        }
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup', onUp);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      };

      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup', onUp);
      document.body.style.cursor = 'ew-resize';
      document.body.style.userSelect = 'none';
    },
    [effectiveSidePanelWidth, getMaxAllowedSidePanelWidth]
  );

  const handleReopenPluginView = useCallback((callId: string) => {
    setActiveSidePanelCallId(callId);
    if (!propUuid) {
      return;
    }
    setClosedSidePanelByChat(current => {
      const currentClosedCallIds = current[propUuid] || [];
      if (!currentClosedCallIds.includes(callId)) {
        return current;
      }
      const next = { ...current };
      const remainingClosedCallIds = currentClosedCallIds.filter(
        item => item !== callId
      );
      if (remainingClosedCallIds.length === 0) {
        delete next[propUuid];
      } else {
        next[propUuid] = remainingClosedCallIds;
      }
      writeStoredClosedSidePanelViews(next);
      return next;
    });
  }, [propUuid]);

  const handleSelectSidePanelTab = useCallback((callId: string) => {
    setActiveSidePanelCallId(callId);
  }, []);

  const handleOpenMailDetail = useCallback(
    async (
      sourceContext: PluginSidePanelContext,
      message: Record<string, any>
    ) => {
      const uid = Number(message?.uid || 0);
      const mailbox = String(
        message?.mailbox ||
          sourceContext.payload.data?.result?.mailbox ||
          sourceContext.payload.data?.result?.folder ||
          'INBOX'
      );

      if (!uid) {
        notify.error('Unable to open email', 'The selected email is missing a valid uid.');
        return;
      }

      const detailCallId = `${sourceContext.callId}::mail_detail::${mailbox}::${uid}`;
      const pluginID = resolvePluginIDForContext(sourceContext, availableTools);
      const detailTitle = String(message?.subject || `Mail ${uid}`);

      setActiveSidePanelCallId(detailCallId);
      if (propUuid) {
        setClosedSidePanelByChat(current => {
          const currentClosedCallIds = current[propUuid] || [];
          if (!currentClosedCallIds.includes(detailCallId)) {
            return current;
          }
          const next = { ...current };
          const remainingClosedCallIds = currentClosedCallIds.filter(
            item => item !== detailCallId
          );
          if (remainingClosedCallIds.length === 0) {
            delete next[propUuid];
          } else {
            next[propUuid] = remainingClosedCallIds;
          }
          writeStoredClosedSidePanelViews(next);
          return next;
        });
      }

      setSidePanelExtraContextsByCallId(current => ({
        ...current,
        [detailCallId]: current[detailCallId] || {
          ...sourceContext,
          callId: detailCallId,
          toolAliasId: sourceContext.toolAliasId,
          toolName: `${sourceContext.pluginName} / Get Mail`,
          payload: {
            viewId: 'mail_detail',
            region: 'chat_side_panel',
            title: detailTitle,
            data: {
              result: {
                mailbox,
                message,
              },
            },
          },
          sourceKind: 'use_tool',
        },
      }));
      setSidePanelErrorByCallId(current => ({ ...current, [detailCallId]: '' }));
      setSidePanelStatusByCallId(current => ({ ...current, [detailCallId]: 'loading' }));

      if (!pluginID) {
        setSidePanelStatusByCallId(current => ({ ...current, [detailCallId]: 'error' }));
        setSidePanelErrorByCallId(current => ({
          ...current,
          [detailCallId]: 'Could not resolve the email plugin runtime.',
        }));
        return;
      }

      try {
        const raw = await Service.CallPluginToolDirect(
          pluginID,
          'use_tool',
          'get_mail',
          JSON.stringify({ mailbox, uid })
        );
        const pluginError = getPluginToolResultError(raw);
        if (pluginError) {
          throw new Error(pluginError);
        }
        const payload = buildPluginSidePanelPayloadFromResult(raw, {
          tool_id: `${pluginID}:get_mail`,
          tool_name: `${sourceContext.pluginName} / Get Mail`,
        });
        if (!payload) {
          throw new Error('The email detail response could not be rendered.');
        }
        setSidePanelExtraContextsByCallId(current => {
          const existing = current[detailCallId];
          if (!existing) {
            return current;
          }
          return {
            ...current,
            [detailCallId]: {
              ...existing,
              payload: {
                ...payload,
                title: payload.title || detailTitle,
              },
            },
          };
        });
        setSidePanelStatusByCallId(current => ({
          ...current,
          [detailCallId]: 'ready',
        }));
      } catch (error) {
        setSidePanelStatusByCallId(current => ({
          ...current,
          [detailCallId]: 'error',
        }));
        setSidePanelErrorByCallId(current => ({
          ...current,
          [detailCallId]: getErrorMessage(
            error,
            'Unable to load the selected email.'
          ),
        }));
      }
    },
    [availableTools, propUuid]
  );

  const handleCloseSidePanelTab = useCallback(
    (callId: string) => {
      if (!propUuid) {
        return;
      }
      if (callId === activeSidePanelContext?.callId) {
        const remainingContexts = openSidePanelContexts.filter(
          context => context.callId !== callId
        );
        setActiveSidePanelCallId(
          remainingContexts[remainingContexts.length - 1]?.callId || ''
        );
      }
      setClosedSidePanelByChat(current => {
        const currentClosedCallIds = current[propUuid] || [];
        if (currentClosedCallIds.includes(callId)) {
          return current;
        }
        const next = {
          ...current,
          [propUuid]: [...currentClosedCallIds, callId],
        };
        writeStoredClosedSidePanelViews(next);
        return next;
      });
    },
    [activeSidePanelContext, openSidePanelContexts, propUuid]
  );

  useEffect(() => {
    if (!activeSidePanelContext || window.innerWidth <= 768) {
      return;
    }
    const clampWidth = () => {
      const nextWidth = Math.min(sidePanelWidth, getMaxAllowedSidePanelWidth());
      if (nextWidth !== sidePanelWidth) {
        setSidePanelWidth(nextWidth);
        try {
          localStorage.setItem(SIDE_PANEL_WIDTH_KEY, String(nextWidth));
        } catch {
          // ignore storage errors
        }
      }
    };
    clampWidth();
    window.addEventListener('resize', clampWidth);
    return () => window.removeEventListener('resize', clampWidth);
  }, [activeSidePanelContext, getMaxAllowedSidePanelWidth, sidePanelWidth]);

  useEffect(() => {
    if (!activeSidePanelContext) {
      return;
    }
    const callId = activeSidePanelContext.callId;
    setSidePanelStatusByCallId(current => ({
      ...current,
      [callId]: getPluginSidePanelAutoStatus(
        activeSidePanelContext,
        current[callId]
      ),
    }));
  }, [activeSidePanelContext]);

  useEffect(() => {
    setActiveSidePanelCallId('');
    setSidePanelExtraContextsByCallId({});
    setSidePanelOverridesByCallId({});
    setSidePanelErrorByCallId({});
    setSidePanelStatusByCallId({});
  }, [propUuid]);

  useEffect(() => {
    const availableContexts = mergedSidePanelContexts.filter(
      item => !closedSidePanelCallIds.includes(item.callId)
    );
    if (availableContexts.length === 0) {
      if (activeSidePanelCallId) {
        setActiveSidePanelCallId('');
      }
      return;
    }
    if (
      !activeSidePanelCallId ||
      !availableContexts.some(item => item.callId === activeSidePanelCallId)
    ) {
      setActiveSidePanelCallId(
        availableContexts[availableContexts.length - 1]?.callId || ''
      );
    }
  }, [activeSidePanelCallId, closedSidePanelCallIds, mergedSidePanelContexts]);

  useEffect(() => {
    if (!latestSidePanelContext) {
      return;
    }
    if (closedSidePanelCallIds.includes(latestSidePanelContext.callId)) {
      return;
    }
    setActiveSidePanelCallId(latestSidePanelContext.callId);
  }, [closedSidePanelCallIds, latestSidePanelContext]);

  const applyDefaultModel = useCallback((models: Model[]) => {
    const config = getDefaultModelConfig();
    if (!config) {
      return false;
    }
    const defaultModel = models.find(model => model.id === config.modelId);
    if (!defaultModel) {
      return false;
    }
    setSelectModelId(defaultModel.id);
    setSelectModelName(defaultModel.model);
    return true;
  }, []);

  const activeTitleChatUuid = currentChatUuid || propUuid;

  const isGenerating = useMemo(() => {
    if (propUuid !== '') {
      if (generatingUuids.includes(propUuid)) return true;
      if (pendingExistingChatUuids.includes(propUuid)) return true;
      return false;
    }
    return pendingNewChatCount > 0;
  }, [
    propUuid,
    generatingUuids,
    pendingExistingChatUuids,
    pendingNewChatCount,
  ]);

  // 侧边栏：已开始流式 + 已发请求但尚未 onStreamStarted 的已有会话
  const sidebarGeneratingUuids = useMemo(() => {
    const set = new Set(generatingUuids);
    pendingExistingChatUuids.forEach(u => {
      if (u) set.add(u);
    });
    return [...set];
  }, [generatingUuids, pendingExistingChatUuids]);

  useEffect(() => {
    onGeneratingUuidsChange?.(sidebarGeneratingUuids);
  }, [sidebarGeneratingUuids, onGeneratingUuidsChange]);

  useEffect(() => {
    if (!onRegisterStopGenerationForChat) return;
    onRegisterStopGenerationForChat((targetUuid: string) => {
      const task = activeTasksByChatRef.current[targetUuid];
      if (!task?.task_uuid) {
        return;
      }
      Service.StopTask(task.task_uuid);
    });
  }, [onRegisterStopGenerationForChat]);

  useEffect(() => {
    activeTasksByChatRef.current = activeTasksByChat;
  }, [activeTasksByChat]);

  useEffect(() => {
    currentChatUuidRef.current = currentChatUuid;
  }, [currentChatUuid]);

  useEffect(() => {
    availableModelsRef.current = availableModels;
  }, [availableModels]);

  useEffect(() => {
    Service.GetModels(true, true).then((models: Model[]) => {
      setAvailableModels(models);
      if (!currentChatUuidRef.current) {
        applyDefaultModel(models);
      }
    });
    refreshAvailableTools().catch(() => {});
  }, [applyDefaultModel, refreshAvailableTools]);

  useEffect(() => {
    const handleDefaultModelChanged = () => {
      Service.GetModels(true, true)
        .then((models: Model[]) => {
          setAvailableModels(models);
          if (!currentChatUuidRef.current) {
            applyDefaultModel(models);
          }
        })
        .catch(err => {
          console.error(
            'Failed to refresh models after default model changed:',
            err
          );
        });
    };
    const cancelDefaultModelEvent = Events.On(
      'chat-default-model-changed',
      handleDefaultModelChanged
    );
    window.addEventListener(
      'chat-default-model-changed',
      handleDefaultModelChanged
    );
    return () => {
      cancelDefaultModelEvent?.();
      window.removeEventListener(
        'chat-default-model-changed',
        handleDefaultModelChanged
      );
    };
  }, [applyDefaultModel]);

  useEffect(() => {
    const cancelPluginChanged = Events.On('settings:plugins:changed', () => {
      refreshAvailableTools().catch(err => {
        console.error('Failed to refresh tools after plugin changed:', err);
      });
    });
    return () => {
      cancelPluginChanged?.();
      Events.Off('settings:plugins:changed');
    };
  }, [refreshAvailableTools]);

  // 当可用工具加载后，仅保留有效的自定义工具，并自动纳入已启用项
  useEffect(() => {
    if (availableTools.length === 0) return;
    const validCustomIds = new Set(
      availableTools
        .filter(
          tool =>
            tool.source_type === 'mcp_custom' || tool.source_type === 'plugin'
        )
        .map(tool => tool.id)
    );
    const enabledCustomIds = availableTools
      .filter(
        tool =>
          (tool.source_type === 'mcp_custom' ||
            tool.source_type === 'plugin') &&
          tool.enabled
      )
      .map(tool => tool.id);
    setSelectedToolIds(prev => {
      const next = [
        ...new Set([
          ...prev.filter(id => validCustomIds.has(id)),
          ...enabledCustomIds,
        ]),
      ];
      return next.length === prev.length &&
        next.every((id, index) => id === prev[index])
        ? prev
        : next;
    });
  }, [availableTools]);

  // 持久化用户选择的 tools
  useEffect(() => {
    if (selectedToolIds.length === 0) {
      localStorage.removeItem('chat_selected_tools');
    } else {
      localStorage.setItem(
        'chat_selected_tools',
        JSON.stringify(selectedToolIds)
      );
    }
  }, [selectedToolIds]);

  const shouldApplyEventToMessages = useCallback(
    (list: Message[], event: TaskStreamEvent): boolean => {
      if (visibleChatUuidRef.current === event.chat_uuid) {
        return true;
      }
      if (currentChatUuidRef.current === event.chat_uuid) {
        return true;
      }
      if (list.some(message => (message.chat_uuid ?? '') === event.chat_uuid)) {
        return true;
      }
      if (
        visibleChatUuidRef.current !== '' ||
        currentChatUuidRef.current !== ''
      ) {
        return false;
      }
      return list.some(
        message =>
          isAssistantPlaceholderMessage(message) &&
          (message.chat_uuid ?? '') === ''
      );
    },
    []
  );

  const syncStreamingAssistantToMessages = useCallback(
    (chatUuidToSync: string) => {
      const cached = streamingAssistantByChatRef.current[chatUuidToSync];
      if (!cached) {
        return;
      }
      setMessages(prev => upsertMessage(prev, cached));
    },
    []
  );

  const finishTaskTracking = (event: TaskStreamEvent) => {
    const cancel = taskSubscriptionsRef.current.get(event.task_uuid);
    cancel?.();
    taskSubscriptionsRef.current.delete(event.task_uuid);
    setGeneratingUuids(prev => prev.filter(x => x !== event.chat_uuid));
    setActiveTasksByChat(prev => {
      const next = { ...prev };
      delete next[event.chat_uuid];
      return next;
    });
    if (
      visibleChatUuidRef.current === event.chat_uuid ||
      currentChatUuidRef.current === event.chat_uuid
    ) {
      delete streamingAssistantByChatRef.current[event.chat_uuid];
    }
    refreshChatList?.();
  };

  const finishTaskTrackingByTask = useCallback(
    (task: Task) => {
      const cancel = taskSubscriptionsRef.current.get(task.task_uuid);
      cancel?.();
      taskSubscriptionsRef.current.delete(task.task_uuid);
      setGeneratingUuids(prev => prev.filter(x => x !== task.chat_uuid));
      setPendingExistingChatUuids(prev =>
        prev.filter(uuid => uuid !== task.chat_uuid)
      );
      setActiveTasksByChat(prev => {
        const next = { ...prev };
        delete next[task.chat_uuid];
        return next;
      });
      if (
        visibleChatUuidRef.current === task.chat_uuid ||
        currentChatUuidRef.current === task.chat_uuid
      ) {
        delete streamingAssistantByChatRef.current[task.chat_uuid];
      }
      refreshChatList?.();
    },
    [refreshChatList]
  );

  const handleTaskEvent = (event: TaskStreamEvent) => {
    const assistantMessage = event.assistant_message;
    if (assistantMessage?.message_uuid) {
      streamingAssistantByChatRef.current[event.chat_uuid] = assistantMessage;
    }

    setActiveTasksByChat(prev => ({
      ...prev,
      [event.chat_uuid]: new Task({
        ...(prev[event.chat_uuid] || {}),
        task_uuid: event.task_uuid,
        chat_uuid: event.chat_uuid,
        assistant_message_uuid:
          assistantMessage?.message_uuid ??
          prev[event.chat_uuid]?.assistant_message_uuid ??
          '',
        event_key: event.event_key,
        status: toTaskStatus(event.status),
        finish_reason: event.finish_reason,
        finish_error: event.finish_error,
      }),
    }));

    if (event.status === 'pending' || event.status === 'running') {
      setGeneratingUuids(prev =>
        prev.includes(event.chat_uuid) ? prev : [...prev, event.chat_uuid]
      );
    } else {
      setMessages(prev => {
        if (!shouldApplyEventToMessages(prev, event)) {
          return prev;
        }
        return upsertMessage(prev, assistantMessage);
      });
      return;
    }

    if (visibleChatUuidRef.current !== event.chat_uuid) {
      return;
    }
    setMessages(prev => upsertMessage(prev, assistantMessage));
  };

  const ensureTaskSubscription = (task: Task | null | undefined) => {
    if (!task?.task_uuid || !task?.event_key) {
      return;
    }
    setActiveTasksByChat(prev => ({ ...prev, [task.chat_uuid]: task }));
    if (task.status === 'pending' || task.status === 'running') {
      setGeneratingUuids(prev =>
        prev.includes(task.chat_uuid) ? prev : [...prev, task.chat_uuid]
      );
    }
    if (taskSubscriptionsRef.current.has(task.task_uuid)) {
      return;
    }
    const cancel = SubscribeTaskStream(
      task,
      handleTaskEvent,
      error => {
        console.error(error);
      },
      event => {
        finishTaskTracking(event);
      }
    );
    if (cancel) {
      taskSubscriptionsRef.current.set(task.task_uuid, cancel);
    }
  };

  const reconcileTaskAfterSubscription = useCallback(
    async (task: Task | null | undefined) => {
      if (!task?.task_uuid) {
        return;
      }
      try {
        const latestTask = await Service.GetTask(task.task_uuid);
        if (!latestTask || !isTerminalTaskStatus(latestTask.status)) {
          return;
        }

        console.warn('missed terminal event, reconciled from GetTask', {
          task_uuid: latestTask.task_uuid,
          chat_uuid: latestTask.chat_uuid,
          event_key: latestTask.event_key,
          status: latestTask.status,
          finish_reason: latestTask.finish_reason,
          finish_error: latestTask.finish_error,
        });

        finishTaskTrackingByTask(latestTask);

        if (
          visibleChatUuidRef.current !== latestTask.chat_uuid &&
          currentChatUuidRef.current !== latestTask.chat_uuid
        ) {
          return;
        }

        const messageList = await Service.ChatMessages(
          latestTask.chat_uuid,
          0,
          200
        );
        const raw = messageList?.messages ?? [];
        const merged = mergeStreamingAssistant(
          latestTask.chat_uuid,
          raw,
          streamingAssistantByChatRef.current
        );
        setMessages(merged);
        delete streamingAssistantByChatRef.current[latestTask.chat_uuid];
      } catch (error) {
        console.error('Failed to reconcile task after subscription:', error);
      }
    },
    [finishTaskTrackingByTask]
  );

  useEffect(() => {
    Service.GetRunningTasks()
      .then(taskList => {
        taskList?.tasks?.forEach(task => ensureTaskSubscription(task));
      })
      .catch(err => {
        console.error('Failed to restore running tasks:', err);
      });

    return () => {
      taskSubscriptionsRef.current.forEach(cancel => cancel());
      taskSubscriptionsRef.current.clear();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const v = chatUuid ?? '';
    visibleChatUuidRef.current = v;

    // 如果这次 chatUuid 变化是由内部 navigate 引起的（新对话获得 UUID），只同步 currentChatUuid，不重新拉取，避免打断流式
    if (skipNextFetchRef.current && skipNextFetchRef.current === v) {
      skipNextFetchRef.current = null;
      setCurrentChatUuid(v);
      syncStreamingAssistantToMessages(v);
      return;
    }

    setCurrentChatUuid(v);

    if (!v) {
      setChatInfo(null);
      setDisplayTitle('');
      setLoading(false);
      setMessages([]);
      setInputMessage('');
      setInputFiles([]);
      setPendingInitialInput('');
      setInputResetKey(prev => prev + 1);
      applyDefaultModel(availableModelsRef.current);
      return;
    }
    const fetchSeq = ++chatFetchSeqRef.current;
    setLoading(true);
    setPendingInitialInput('');
    setInputResetKey(prev => prev + 1);
    Promise.all([
      Service.ChatInfo(v)
        .then((info: ChatType | null) => {
          if (fetchSeq !== chatFetchSeqRef.current) return;
          setChatInfo(info);
          setDisplayTitle(info?.title?.trim() || '');
        })
        .catch(err => {
          console.error('Failed to fetch chat info:', err);
          if (fetchSeq !== chatFetchSeqRef.current) return;
          setChatInfo(null);
          setDisplayTitle('');
        }),
      Service.ChatMessages(v, 0, 200)
        .then(messageList => {
          if (fetchSeq !== chatFetchSeqRef.current) return;
          const raw = messageList!.messages;
          const merged = mergeStreamingAssistant(
            v,
            raw,
            streamingAssistantByChatRef.current
          );
          setMessages(merged);
          delete streamingAssistantByChatRef.current[v];
        })
        .catch(err => {
          console.error('Failed to fetch chat messages info:', err);
          if (fetchSeq !== chatFetchSeqRef.current) return;
          setMessages([]);
        }),
      Service.GetChatActiveTask(v)
        .then(task => {
          if (fetchSeq !== chatFetchSeqRef.current || !task) return;
          ensureTaskSubscription(task);
        })
        .catch(err => {
          console.error('Failed to fetch active task:', err);
        }),
    ]).finally(() => {
      setTimeout(() => {
        if (fetchSeq !== chatFetchSeqRef.current) return;
        setLoading(false);
      }, 300);
    });
  }, [applyDefaultModel, chatUuid]);

  // onModelSelectorClick 模型选择框点击事件
  const onModelSelectorClick = () => {
    Service.GetModels(true, true).then((models: Model[]) => {
      setAvailableModels(models);
    });
  };

  // onSelectModelChange 所选模型变更
  const onSelectModelChange = (modelId: number, modelName: string) => {
    setSelectModelId(modelId);
    setSelectModelName(modelName);
  };

  // onMessageChange 输入消息变更
  const onMessageChange = (message: string) => {
    setInputMessage(message);
  };

  // onSelectFileChange 输入文件变更
  const onSelectFileChange = (paths: FileInfo[]) => {
    setInputFiles(paths);
  };

  // onSendButtonClick 发送按钮点击
  const onSendButtonClick = async () => {
    if (selectModel <= 0 || !selectModelName.trim()) {
      return;
    }
    let fromEmptyChat = false;
    const prevMessages = messages;
    const pendingTitle = inputMessage.trim();
    try {
      fromEmptyChat = currentChatUuid === '';
      if (fromEmptyChat) {
        setPendingNewChatCount(n => n + 1);
      } else if (currentChatUuid) {
        setPendingExistingChatUuids(prev => [...prev, currentChatUuid]);
      }
      const userMessage: Message = {
        id: 0,
        created_at: null,
        updated_at: null,
        deleted_at: null,
        role: RoleType.User,
        chat_uuid: currentChatUuid,
        message_uuid: '',
        content: inputMessage,
        reasoning_content: '',
        user_message_extra: {
          model_id: selectModel,
          model_name: selectModelName,
          files: inputFiles,
          tools: effectiveSelectedToolIds,
          agents: [],
        },
        user_message_extra_content: '',
        assistant_message_extra: null,
        assistant_message_extra_content: '',
      };
      const assistantMessage: Message = {
        id: 0,
        created_at: null,
        updated_at: null,
        deleted_at: null,
        role: RoleType.Assistant,
        chat_uuid: currentChatUuid,
        content: '',
        reasoning_content: '',
        message_uuid: '',
        user_message_extra: null,
        user_message_extra_content: '',
        assistant_message_extra: {
          execution_trace: {
            steps: [],
          },
          route_type: RouteType.$zero,
          retry_count: 0,
          current_stage: '',
          current_agent: '',
          preface_content: '',
          preface_reasoning_content: '',
          finish_reason: '',
          finish_error: '',
          tool_uses: [],
          pending_approvals: [],
        },
        assistant_message_extra_content: '',
      };

      setMessages(prev => [...prev, userMessage, assistantMessage]);

      const resp = await CompletionsUtils(userMessage);
      if (!resp?.chat_uuid || !resp?.task_uuid) {
        setMessages(prevMessages);
        if (fromEmptyChat) {
          setPendingNewChatCount(n => Math.max(0, n - 1));
        } else if (currentChatUuid) {
          setPendingExistingChatUuids(prev =>
            prev.filter(uuid => uuid !== currentChatUuid)
          );
        }
        return;
      }

      const streamChatUuid = resp.chat_uuid;
      assistantMessage.chat_uuid = streamChatUuid;
      assistantMessage.message_uuid = resp.message_uuid;

      if (fromEmptyChat) {
        setPendingNewChatCount(n => Math.max(0, n - 1));
        skipNextFetchRef.current = streamChatUuid;
        setCurrentChatUuid(streamChatUuid);
        if (pendingTitle) {
          setDisplayTitle(pendingTitle);
        }
        setMessages(prev =>
          prev.map(m => ({ ...m, chat_uuid: streamChatUuid }))
        );
        navigate(`/home/${streamChatUuid}`, { replace: true });
      } else {
        setPendingExistingChatUuids(prev =>
          prev.filter(uuid => uuid !== streamChatUuid)
        );
      }

      setGeneratingUuids(prev =>
        prev.includes(streamChatUuid) ? prev : [...prev, streamChatUuid]
      );
      const task = BuildTaskFromCompletions(resp, assistantMessage);
      ensureTaskSubscription(task);
      console.log('chat task created:', {
        task_uuid: task.task_uuid,
        event_key: task.event_key,
        chat_uuid: task.chat_uuid,
      });
      void reconcileTaskAfterSubscription(task);
      refreshChatList?.();
    } catch (e) {
      setMessages(prevMessages);
      if (fromEmptyChat) {
        setPendingNewChatCount(n => Math.max(0, n - 1));
      } else if (currentChatUuid) {
        setPendingExistingChatUuids(prev =>
          prev.filter(uuid => uuid !== currentChatUuid)
        );
      }
    }
  };

  // onStopGeneration 停止生成点击
  const onStopGeneration = () => {
    const v = chatUuid ?? '';
    if (!v) {
      return;
    }
    const task = activeTasksByChatRef.current[v];
    if (!task?.task_uuid) {
      return;
    }
    Service.StopTask(task.task_uuid);
  };

  const handleTitleChange = useCallback(
    async (newTitle: string) => {
      const activeChatUuid = currentChatUuid || propUuid;
      if (!activeChatUuid) {
        return;
      }
      await Service.RenameChat(activeChatUuid, newTitle);
      setChatInfo(prev => (prev ? { ...prev, title: newTitle } : prev));
      setDisplayTitle(newTitle);
    },
    [currentChatUuid, propUuid]
  );

  const handleApprovalDecision = useCallback(
    async (approvalId: string, decision: 'allow' | 'reject') => {
      try {
        await Service.RespondToolApproval(
          new ToolApprovalResponse({
            approval_id: approvalId,
            decision:
              decision === 'allow'
                ? ToolApprovalDecision.ToolApprovalDecisionAllow
                : ToolApprovalDecision.ToolApprovalDecisionReject,
            comment: '',
          })
        );
      } catch (error: any) {
        notify.error(
          t('home.chat.approvalFailed'),
          getErrorMessage(error, t('home.chat.approvalFailedDesc'))
        );
      }
    },
    [t]
  );

  const handleSendApprovalComment = useCallback(
    async (approvalId: string, comment: string) => {
      try {
        await Service.RespondToolApproval(
          new ToolApprovalResponse({
            approval_id: approvalId,
            decision: ToolApprovalDecision.ToolApprovalDecisionCustom,
            comment,
          })
        );
      } catch (error: any) {
        notify.error(
          t('home.chat.approvalCommentFailed'),
          getErrorMessage(error, t('home.chat.approvalCommentFailedDesc'))
        );
      }
    },
    [t]
  );

  useEffect(() => {
    if (!chatInfo?.title?.trim()) {
      return;
    }
    setDisplayTitle(chatInfo.title);
  }, [chatInfo?.title]);

  useEffect(() => {
    if (!activeTitleChatUuid) {
      return;
    }
    const eventKey = `event:chat_title:${activeTitleChatUuid}`;
    const cancel = Events.On(eventKey, event => {
      const payload = event.data as { chat_uuid?: string; title?: string };
      const nextTitle = payload?.title;
      if (payload?.chat_uuid !== activeTitleChatUuid || !nextTitle) {
        return;
      }
      setDisplayTitle(nextTitle);
      setChatInfo(prev => (prev ? { ...prev, title: nextTitle } : prev));
    });

    return () => {
      cancel?.();
      Events.Off(eventKey);
    };
  }, [activeTitleChatUuid]);

  return (
    <div className={`${styles.chatPage}`}>
      {/* 主内容始终渲染 */}
      <div className={styles.chatWorkspace} ref={chatWorkspaceRef}>
        <div className={styles.chatMain}>
          <ChatTitle
            title={displayTitle}
            uuid={propUuid}
            onTitleChange={handleTitleChange}
            isSidebarCollapsed={isSidebarCollapsed}
            onToggleSidebar={onToggleSidebar}
          />
          <div className={`${styles.chatMessagesContent}`}>
            {!currentChatUuid && messages.length === 0 && !isGenerating ? (
              <WelcomeEmpty
                onSuggestionClick={prompt => {
                  setPendingInitialInput(prompt);
                  setInputMessage(prompt);
                  setInputResetKey(prev => prev + 1);
                }}
              />
            ) : (
                            <MessageList
                                key={currentChatUuid || 'new'}
                                ref={messageListRef}
                                messages={messages}
                                isGenerating={isGenerating}
                                useInstantScrollOnFirstLoad
                                onApprovalDecision={handleApprovalDecision}
                                onSendApprovalComment={handleSendApprovalComment}
                                onReopenPluginView={handleReopenPluginView}
                                openPluginViewCallIds={openSidePanelContexts.map(context => context.callId)}
                                activePluginViewCallId={activeSidePanelContext?.callId || ''}
                            />
            )}
          </div>
          <div className={`${styles.chatInput}`}>
            <ChatInput
              key={inputResetKey}
              selectedModelId={selectModel}
              hasSelectedModel={selectModel > 0 && !!selectModelName.trim()}
              availableModels={availableModels}
              availableTools={customTools}
              selectedToolIds={selectedToolIds}
              onSelectedToolsChange={setSelectedToolIds}
              onRefreshTools={refreshAvailableTools}
              isGenerating={isGenerating}
              initialValue={pendingInitialInput}
              onMessageChange={onMessageChange}
              onSendButtonClick={onSendButtonClick}
              onSelectModelChange={onSelectModelChange}
              onSelectFileChange={onSelectFileChange}
              onStopGeneration={onStopGeneration}
              onModelSelectorClick={onModelSelectorClick}
              onMessageListScrollToBottom={() => {
                messageListRef.current?.scrollToBottom();
              }}
            />
          </div>
        </div>
                {activeSidePanelContext ? (
                    <>
                        <div
                            className={styles.sidePanelResizeHandle}
                            onMouseDown={handleSidePanelResizeStart}
                            title="Drag to resize"
              role="separator"
              aria-orientation="vertical"
            />
                        <PluginSidePanel
                            tabs={openSidePanelContexts.map(context => ({
                              callId: context.callId,
                              pluginName: context.pluginName,
                              title: context.payload.title || context.payload.viewId,
                              viewId: context.payload.viewId,
                            }))}
                            activeCallId={activeSidePanelContext.callId}
                            context={activeSidePanelContext}
                            payload={activeSidePanelContext.payload}
                            width={effectiveSidePanelWidth}
                            status={
                              sidePanelStatusByCallId[
                                activeSidePanelContext.callId
                              ] || 'ready'
                            }
                            errorMessage={
                              sidePanelErrorByCallId[
                                activeSidePanelContext.callId
                              ] || ''
                            }
                            onSelectTab={handleSelectSidePanelTab}
                            onCloseTab={handleCloseSidePanelTab}
                            onOpenMailDetail={handleOpenMailDetail}
                        />
                    </>
                ) : null}
      </div>
      {/* loading 蒙层覆盖在主内容之上 */}
      {loading && (
        <div className={styles.chatLoadingContainer}>
          <div className={styles.loadingSpinner} />
        </div>
      )}
    </div>
  );
};

export default Chat;
export type { ChatProps };
