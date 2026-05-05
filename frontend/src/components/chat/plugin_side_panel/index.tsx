import React from 'react';
import { CloseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import PluginViewFrame from '@/components/plugin/PluginViewFrame';
import styles from './index.module.scss';
import type { PluginSidePanelContext, PluginSidePanelPayload } from './utils';

export interface PluginSidePanelTab {
  callId: string;
  pluginName: string;
  title: string;
  viewId: string;
}

interface PluginSidePanelProps {
  pluginId: string;
  tabs: PluginSidePanelTab[];
  activeCallId: string;
  context: PluginSidePanelContext;
  payload: PluginSidePanelPayload;
  width?: number;
  status?: 'loading' | 'ready' | 'empty' | 'error' | 'stale';
  errorMessage?: string;
  onSelectTab?: (callId: string) => void;
  onCloseTab?: (callId: string) => void;
  onUpdateView?: (payload: PluginSidePanelPayload) => void;
  onOpenView?: (payload: PluginSidePanelPayload) => void;
  onComposeMessage?: (text: string) => void;
}

const GenericView: React.FC<{ payload: PluginSidePanelPayload }> = ({ payload }) => (
  <pre className={styles.genericResult}>{JSON.stringify(payload.data ?? payload, null, 2)}</pre>
);

const PluginSidePanel: React.FC<PluginSidePanelProps> = ({
  pluginId,
  tabs,
  activeCallId,
  context,
  payload,
  width,
  status = 'ready',
  errorMessage = '',
  onSelectTab,
  onCloseTab,
  onUpdateView,
  onOpenView,
  onComposeMessage,
}) => {
  const { t } = useTranslation();

  return (
    <aside className={styles.sidePanel} style={width ? { width: `${width}px` } : undefined}>
      <div className={styles.tabBar}>
        <div className={styles.tabScroller}>
          {tabs.map((tab) => {
            const isActive = tab.callId === activeCallId;
            return (
              <button
                key={tab.callId}
                type="button"
                className={`${styles.tabButton} ${isActive ? styles.tabButtonActive : ''}`}
                onClick={() => onSelectTab?.(tab.callId)}
                onMouseDown={(event) => {
                  if (event.button === 1) {
                    event.preventDefault();
                    onCloseTab?.(tab.callId);
                  }
                }}
              >
                <span className={styles.tabTitle}>{tab.title}</span>
                <span className={styles.tabPluginName}>{tab.pluginName}</span>
                <span
                  className={styles.tabClose}
                  role="button"
                  tabIndex={0}
                  onClick={(event) => {
                    event.stopPropagation();
                    onCloseTab?.(tab.callId);
                  }}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      event.stopPropagation();
                      onCloseTab?.(tab.callId);
                    }
                  }}
                >
                  <CloseOutlined />
                </span>
              </button>
            );
          })}
        </div>
      </div>
      {status === 'error' && errorMessage ? (
        <div className={styles.panelStatusError}>{errorMessage}</div>
      ) : null}
      {status === 'stale' ? (
        <div className={styles.panelStatusStale}>{t('chat.pluginPanel.stale')}</div>
      ) : null}
      <div className={styles.panelBody}>
        {status === 'loading' ? (
          <div className={styles.loadingState}>{t('chat.pluginPanel.loading')}</div>
        ) : status === 'empty' ? (
          <div className={styles.emptyState}>{t('chat.pluginPanel.empty')}</div>
        ) : pluginId ? (
          <PluginViewFrame
            key={context.callId}
            pluginId={pluginId}
            viewId={payload.viewId}
            location="chat_side_panel"
            payload={payload}
            sidePanelContext={context}
            onUpdateView={onUpdateView}
            onOpenView={onOpenView}
            onComposeMessage={onComposeMessage}
          />
        ) : (
          <GenericView payload={payload} />
        )}
      </div>
    </aside>
  );
};

export default PluginSidePanel;
