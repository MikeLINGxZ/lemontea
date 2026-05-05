import React, { useEffect, useMemo, useRef, useState } from 'react';
import { App, Spin } from 'antd';
import { Service } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service';
import { getPluginViewDocument } from '@/services/pluginService';
import type { PluginSidePanelContext, PluginSidePanelPayload } from '@/components/chat/plugin_side_panel/utils';

type HostContext = {
  pluginId: string;
  viewId: string;
  location: 'settings_window' | 'chat_side_panel';
  payload?: PluginSidePanelPayload;
  sidePanelContext?: PluginSidePanelContext;
};

type PluginViewRequest =
  | {
      method: 'get_context';
    }
  | {
      method: 'call_tool';
      params?: {
        kind?: string;
        toolId?: string;
        args?: Record<string, any>;
      };
    }
  | {
      method: 'get_settings';
    }
  | {
      method: 'save_settings';
      params?: {
        config?: Record<string, any>;
      };
    }
  | {
      method: 'test_connection';
      params?: {
        protocol?: string;
        config?: Record<string, any>;
      };
    }
  | {
      method: 'update_view';
      params?: {
        payload?: PluginSidePanelPayload;
      };
    }
  | {
      method: 'open_view';
      params?: {
        payload?: PluginSidePanelPayload;
      };
    }
  | {
      method: 'compose_message';
      params?: {
        text?: string;
      };
    };

interface PluginViewFrameProps {
  pluginId: string;
  viewId: string;
  location: HostContext['location'];
  payload?: PluginSidePanelPayload;
  sidePanelContext?: PluginSidePanelContext;
  onUpdateView?: (payload: PluginSidePanelPayload) => void;
  onOpenView?: (payload: PluginSidePanelPayload) => void;
  onComposeMessage?: (text: string) => void;
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return fallback;
}

const frameStyle: React.CSSProperties = {
  width: '100%',
  height: '100%',
  border: 'none',
  background: 'transparent',
};

const PluginViewFrame: React.FC<PluginViewFrameProps> = ({
  pluginId,
  viewId,
  location,
  payload,
  sidePanelContext,
  onUpdateView,
  onOpenView,
  onComposeMessage,
}) => {
  const { message } = App.useApp();
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const hostContext = useMemo<HostContext>(
    () => ({
      pluginId,
      viewId,
      location,
      payload,
      sidePanelContext,
    }),
    [location, payload, pluginId, sidePanelContext, viewId]
  );
  const [srcDoc, setSrcDoc] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    setSrcDoc('');

    void getPluginViewDocument(pluginId, viewId)
      .then((nextDoc) => {
        if (cancelled) {
          return;
        }
        setSrcDoc(nextDoc);
      })
      .catch((err) => {
        if (cancelled) {
          return;
        }
        setError(getErrorMessage(err, 'Failed to load plugin view.'));
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [pluginId, viewId]);

  useEffect(() => {
    const postInit = () => {
      const target = iframeRef.current?.contentWindow;
      if (!target) {
        return;
      }
      target.postMessage(
        {
          type: 'lemontea-plugin-view:init',
          payload: hostContext,
        },
        '*'
      );
    };

    const handleMessage = (event: MessageEvent) => {
      if (event.source !== iframeRef.current?.contentWindow) {
        return;
      }
      const data = event.data;
      if (!data || typeof data !== 'object') {
        return;
      }
      if (data.type === 'lemontea-plugin-view:ready') {
        postInit();
        return;
      }
      if (data.type !== 'lemontea-plugin-view:request') {
        return;
      }

      const envelope = (data.payload || {}) as {
        requestId?: string;
        payload?: PluginViewRequest;
      };
      const requestId = String(data.requestId || envelope.requestId || '');
      const request = (envelope.payload || data.payload || {}) as PluginViewRequest;
      const respond = (ok: boolean, result?: unknown, errorText?: string) => {
        iframeRef.current?.contentWindow?.postMessage(
          {
            type: 'lemontea-plugin-view:response',
            requestId,
            ok,
            result,
            error: errorText || '',
          },
          '*'
        );
      };

      const run = async () => {
        switch (request.method) {
          case 'get_context':
            return hostContext;
          case 'call_tool': {
            const kind = String(request.params?.kind || 'use_tool');
            const toolId = String(request.params?.toolId || '');
            if (!toolId) {
              throw new Error('toolId is required');
            }
            const raw = await Service.CallPluginToolDirect(
              pluginId,
              kind,
              toolId,
              JSON.stringify(request.params?.args || {})
            );
            return JSON.parse(raw);
          }
          case 'get_settings':
            return await Service.GetPluginSettings(pluginId);
          case 'save_settings':
            return await Service.SavePluginSettings(
              pluginId,
              request.params?.config || {}
            );
          case 'test_connection':
            return await Service.TestPluginConnection(
              pluginId,
              String(request.params?.protocol || ''),
              request.params?.config || {}
            );
          case 'update_view': {
            const nextPayload = request.params?.payload;
            if (!nextPayload) {
              throw new Error('payload is required');
            }
            onUpdateView?.(nextPayload);
            return { ok: true };
          }
          case 'open_view': {
            const nextPayload = request.params?.payload;
            if (!nextPayload) {
              throw new Error('payload is required');
            }
            onOpenView?.(nextPayload);
            return { ok: true };
          }
          case 'compose_message': {
            const text = String(request.params?.text || '');
            onComposeMessage?.(text);
            return { ok: true };
          }
          default:
            throw new Error(`Unsupported plugin view method: ${String((request as any).method || '')}`);
        }
      };

      void run()
        .then((result) => respond(true, result))
        .catch((err) => {
          const errorText = getErrorMessage(err, 'Plugin view request failed.');
          message.error(errorText);
          respond(false, undefined, errorText);
        });
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, [hostContext, message, onComposeMessage, onOpenView, onUpdateView, pluginId]);

  if (loading) {
    return (
      <div style={{ display: 'grid', placeItems: 'center', height: '100%', minHeight: 160 }}>
        <Spin size="small" />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 16, color: '#ff7875', fontSize: 13 }}>
        {error}
      </div>
    );
  }

  return (
    <iframe
      ref={iframeRef}
      srcDoc={srcDoc}
      title={`${pluginId}:${viewId}`}
      style={frameStyle}
      sandbox="allow-scripts allow-forms allow-same-origin"
    />
  );
};

export default PluginViewFrame;
