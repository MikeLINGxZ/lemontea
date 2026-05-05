import React from 'react';
import { CloseOutlined } from '@ant-design/icons';
import DOMPurify from 'dompurify';
import { useTranslation } from 'react-i18next';
import styles from './index.module.scss';
import type { PluginSidePanelContext, PluginSidePanelPayload } from './utils';

export interface PluginSidePanelTab {
  callId: string;
  pluginName: string;
  title: string;
  viewId: string;
}

interface PluginSidePanelProps {
  tabs: PluginSidePanelTab[];
  activeCallId: string;
  context: PluginSidePanelContext;
  payload: PluginSidePanelPayload;
  width?: number;
  status?: 'loading' | 'ready' | 'empty' | 'error' | 'stale';
  errorMessage?: string;
  onSelectTab?: (callId: string) => void;
  onCloseTab?: (callId: string) => void;
  onOpenMailDetail?: (context: PluginSidePanelContext, message: Record<string, any>) => void;
}

function formatDateTime(value: unknown): string {
  const text = String(value || '').trim();
  if (!text) return '';
  const date = new Date(text);
  if (Number.isNaN(date.getTime())) {
    return text;
  }
  return date.toLocaleString();
}

function formatMailPreview(value: unknown): string {
  return String(value || '').replace(/\s+/g, ' ').trim();
}

const MailListView: React.FC<{
  context: PluginSidePanelContext;
  payload: PluginSidePanelPayload;
  onOpenMailDetail?: (context: PluginSidePanelContext, message: Record<string, any>) => void;
}> = ({ context, payload, onOpenMailDetail }) => {
  const { t } = useTranslation();
  const result = payload.data?.result || {};
  const messages = Array.isArray(result?.messages) ? result.messages : [];

  if (messages.length === 0) {
    return <div className={styles.emptyState}>{t('chat.pluginPanel.empty')}</div>;
  }

  return (
    <div className={styles.mailList}>
      <div className={styles.mailListSummary}>
        <span>{result.folder || result.mailbox || 'INBOX'}</span>
        <span>{result.count || messages.length} messages</span>
        {result.hasMore ? <span>More available</span> : null}
      </div>
      {messages.map((message: any, index: number) => (
        <button
          key={message?.id || message?.uid || `${message?.subject || 'mail'}-${index}`}
          type="button"
          className={styles.mailItem}
          onClick={() => onOpenMailDetail?.(context, message)}
        >
          <div className={styles.mailHeader}>
            <div className={styles.mailSubject}>{message?.subject || '(No subject)'}</div>
            <div className={styles.mailTime}>
              {message?.receivedAt ? formatDateTime(message.receivedAt) : ''}
            </div>
          </div>
          <div className={styles.mailPreviewRow}>
            {message?.unread ? <span className={styles.unreadDot} /> : null}
            <div className={styles.mailBody}>
              {formatMailPreview(message?.snippet || message?.body || '') || t('chat.pluginPanel.empty')}
            </div>
          </div>
        </button>
      ))}
    </div>
  );
};

const MailDetailView: React.FC<{ payload: PluginSidePanelPayload }> = ({ payload }) => {
  const result = payload.data?.result || {};
  const message = result?.message || {};
  const attachments = Array.isArray(message?.attachments) ? message.attachments : [];
  const html = String(message?.html || '').trim();
  const sanitizedHtml = html
    ? DOMPurify.sanitize(html, {
        USE_PROFILES: { html: true },
      })
    : '';
  const plainBody = String(message?.body || '').trim();

  return (
    <div className={styles.mailDetail}>
      <section className={styles.mailDetailHero}>
        <div className={styles.mailDetailEyebrow}>
          <span>{result.mailbox || 'INBOX'}</span>
          {message?.receivedAt ? <span>{formatDateTime(message.receivedAt)}</span> : null}
        </div>
        <h2 className={styles.mailDetailSubject}>{message?.subject || '(No subject)'}</h2>
        <div className={styles.mailDetailMetaGrid}>
          {message?.from ? (
            <div className={styles.mailDetailMetaCard}>
              <span className={styles.mailDetailMetaLabel}>From</span>
              <span className={styles.mailDetailMetaValue}>{message.from}</span>
            </div>
          ) : null}
          {message?.to ? (
            <div className={styles.mailDetailMetaCard}>
              <span className={styles.mailDetailMetaLabel}>To</span>
              <span className={styles.mailDetailMetaValue}>{message.to}</span>
            </div>
          ) : null}
          {message?.cc ? (
            <div className={styles.mailDetailMetaCard}>
              <span className={styles.mailDetailMetaLabel}>Cc</span>
              <span className={styles.mailDetailMetaValue}>{message.cc}</span>
            </div>
          ) : null}
          {attachments.length > 0 ? (
            <div className={styles.mailDetailMetaCard}>
              <span className={styles.mailDetailMetaLabel}>Attachments</span>
              <span className={styles.mailDetailMetaValue}>{attachments.length}</span>
            </div>
          ) : null}
        </div>
      </section>

      {attachments.length > 0 ? (
        <section className={styles.mailDetailSection}>
          <div className={styles.mailDetailSectionTitle}>Attachments</div>
          <div className={styles.attachmentList}>
            {attachments.map((attachment: any, index: number) => (
              <div
                key={`${attachment?.filename || 'attachment'}-${index}`}
                className={styles.attachmentItem}
              >
                <div className={styles.attachmentName}>
                  {attachment?.filename || `Attachment ${index + 1}`}
                </div>
                <div className={styles.attachmentMeta}>
                  {attachment?.contentType || 'unknown type'}
                  {attachment?.size ? ` · ${Math.max(1, Math.round(Number(attachment.size) / 1024))} KB` : ''}
                </div>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      <section className={styles.mailDetailBodyCard}>
        {sanitizedHtml ? (
          <div
            className={styles.mailHtmlBody}
            dangerouslySetInnerHTML={{ __html: sanitizedHtml }}
          />
        ) : (
          <div className={styles.mailPlainBody}>{plainBody || message?.snippet || ''}</div>
        )}
      </section>
    </div>
  );
};

const GenericView: React.FC<{ payload: PluginSidePanelPayload }> = ({ payload }) => (
  <pre className={styles.genericResult}>{JSON.stringify(payload.data ?? payload, null, 2)}</pre>
);

const PluginSidePanel: React.FC<PluginSidePanelProps> = ({
  tabs,
  activeCallId,
  context,
  payload,
  width,
  status = 'ready',
  errorMessage = '',
  onSelectTab,
  onCloseTab,
  onOpenMailDetail,
}) => {
  const { t } = useTranslation();
  const result = payload.data?.result || {};
  const messageCount = Array.isArray(result?.messages) ? result.messages.length : 0;
  const showEmptyState = status === 'empty' || (payload.viewId === 'mail_list' && messageCount === 0);

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
        ) : showEmptyState ? (
          <div className={styles.emptyState}>{t('chat.pluginPanel.empty')}</div>
        ) : payload.viewId === 'mail_list' ? (
          <MailListView context={context} payload={payload} onOpenMailDetail={onOpenMailDetail} />
        ) : payload.viewId === 'mail_detail' ? (
          <MailDetailView payload={payload} />
        ) : (
          <GenericView payload={payload} />
        )}
      </div>
    </aside>
  );
};

export default PluginSidePanel;
