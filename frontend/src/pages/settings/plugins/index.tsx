import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Empty, Modal, Progress, Skeleton, Space, Tag, Typography, message } from 'antd';
import { CloudDownloadOutlined, DeleteOutlined, PlusOutlined, ReloadOutlined, SettingOutlined } from '@ant-design/icons';
import { Events } from '@wailsio/runtime';
import { useTranslation } from 'react-i18next';
import {
  addPluginFromFolder,
  deletePlugin,
  downloadPluginRuntime,
  getPluginRuntimeStatus,
  listPlugins,
  openPluginSettingsWindow,
  selectPluginFolder,
  setPluginEnabled,
} from '@/services/pluginService';
import type { PluginSummary, RuntimeStatus } from '@/services/pluginService';
import styles from './index.module.scss';

const { Paragraph, Text, Title } = Typography;

const typeColor: Record<string, string> = {
  general_plugin: 'blue',
  agent_plugin: 'purple',
};

const statusColor: Record<string, string> = {
  enabled: 'green',
  disabled: 'default',
  error: 'red',
};

const PluginSettingsPage: React.FC<{ className?: string }> = ({ className }) => {
  const { t } = useTranslation();
  const [plugins, setPlugins] = useState<PluginSummary[]>([]);
  const [activeID, setActiveID] = useState('');
  const [loading, setLoading] = useState(false);
  const [adding, setAdding] = useState(false);
  const [pendingPluginID, setPendingPluginID] = useState('');
  const [deletingPluginID, setDeletingPluginID] = useState('');
  const [downloadingRuntime, setDownloadingRuntime] = useState(false);
  const [runtimeStatus, setRuntimeStatus] = useState<RuntimeStatus | null>(null);
  const [error, setError] = useState('');

  const active = useMemo(
    () => plugins.find((plugin) => plugin.id === activeID) || plugins[0] || null,
    [plugins, activeID],
  );

  const refresh = async (options?: { silent?: boolean }) => {
    if (!options?.silent) {
      setLoading(true);
    }
    setError('');
    try {
      const runtime = await getPluginRuntimeStatus();
      setRuntimeStatus(runtime);
      if (!runtime.available) {
        setPlugins([]);
        return;
      }
      const result = await listPlugins();
      setPlugins(result);
      if (!activeID && result.length > 0) {
        setActiveID(result[0].id);
      }
    } catch (e) {
      console.error(e);
      setError(t('settings.plugins.loadFailed'));
    } finally {
      if (!options?.silent) {
        setLoading(false);
      }
    }
  };

  const handleDownloadRuntime = async () => {
    setDownloadingRuntime(true);
    setError('');
    setRuntimeStatus((current) => current ? { ...current, downloading: true, progress: 0, phase: 'preparing' } : current);
    const timer = window.setInterval(async () => {
      try {
        const status = await getPluginRuntimeStatus();
        setRuntimeStatus(status);
      } catch (e) {
        console.error(e);
      }
    }, 500);
    try {
      const status = await downloadPluginRuntime();
      if (status) setRuntimeStatus(status);
      if (!status?.available) {
        setError(t('settings.plugins.runtimeDownloadFailed'));
        return;
      }
      message.success(t('settings.plugins.runtimeDownloadSuccess'));
      await refresh();
    } catch (e: any) {
      console.error(e);
      setError(t('settings.plugins.runtimeDownloadFailed'));
    } finally {
      window.clearInterval(timer);
      setDownloadingRuntime(false);
    }
  };

  useEffect(() => {
    void refresh();
  }, []);

  useEffect(() => {
    const cancel = Events.On('settings:plugins:changed', () => void refresh({ silent: true }));
    return () => {
      cancel?.();
      Events.Off('settings:plugins:changed');
    };
  }, []);

  const handleAdd = async () => {
    setAdding(true);
    try {
      const folder = await selectPluginFolder();
      if (!folder) return;
      const created = await addPluginFromFolder(folder);
      await refresh({ silent: true });
      if (created?.id) setActiveID(created.id);
      message.success(t('settings.plugins.addSuccess'));
    } catch (e: any) {
      console.error(e);
      message.error(e?.message || t('settings.plugins.addFailed'));
    } finally {
      setAdding(false);
    }
  };

  const handleToggle = async (plugin: PluginSummary) => {
    const nextEnabled = !plugin.enabled;
    setPendingPluginID(plugin.id);
    setPlugins((current) => current.map((item) => (
      item.id === plugin.id
        ? {
          ...item,
          enabled: nextEnabled,
          status: nextEnabled ? 'enabled' : 'disabled',
        }
        : item
    )));
    try {
      await setPluginEnabled(plugin.id, nextEnabled);
      await refresh({ silent: true });
      message.success(plugin.enabled ? t('settings.plugins.disableSuccess') : t('settings.plugins.enableSuccess'));
    } catch (e: any) {
      console.error(e);
      message.error(e?.message || t('settings.plugins.updateFailed'));
      await refresh({ silent: true });
    } finally {
      setPendingPluginID('');
    }
  };

  const handleDelete = (plugin: PluginSummary) => {
    Modal.confirm({
      title: t('settings.plugins.deleteConfirmTitle'),
      content: t('settings.plugins.deleteConfirmContent', { name: plugin.name }),
      okText: t('settings.plugins.actions.delete'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        setDeletingPluginID(plugin.id);
        try {
          await deletePlugin(plugin.id);
          await refresh({ silent: true });
          message.success(t('settings.plugins.deleteSuccess'));
        } catch (e: any) {
          console.error(e);
          message.error(e?.message || t('settings.plugins.deleteFailed'));
        } finally {
          setDeletingPluginID('');
        }
      },
    });
  };

  const renderList = () => (
    <Card
      className={styles.listCard}
      title={
        <div className={styles.listTitleRow}>
          <span>{t('settings.plugins.listTitle')}</span>
          <Space size={4}>
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              title={t('common.retry')}
              onClick={() => void refresh()}
            />
            <Button
              type="text"
              size="small"
              icon={<PlusOutlined />}
              title={t('settings.plugins.actions.add')}
              loading={adding}
              onClick={() => void handleAdd()}
            />
          </Space>
        </div>
      }
    >
      {loading ? (
        <div className={styles.loading}><Skeleton active paragraph={{ rows: 6 }} /></div>
      ) : plugins.length === 0 ? (
        <div className={styles.emptyState}><Empty description={t('settings.plugins.empty')} /></div>
      ) : (
        <div className={styles.pluginList}>
          {plugins.map(plugin => (
            <button
              key={plugin.id}
              className={`${styles.pluginItem} ${active?.id === plugin.id ? styles.selected : ''}`}
              type="button"
              onClick={() => setActiveID(plugin.id)}
            >
              <div className={styles.pluginItemHeader}>
                <span className={styles.pluginName}>{plugin.name}</span>
                <Tag color={statusColor[plugin.status] || 'default'} bordered={false}>
                  {t(`settings.plugins.status.${plugin.status}`, plugin.status)}
                </Tag>
              </div>
              <div className={styles.pluginMeta}>{plugin.version} · {plugin.id}</div>
            </button>
          ))}
        </div>
      )}
    </Card>
  );

  const renderCapabilitySection = (title: string, items: { id: string; name: string; description: string }[], tag: string) => (
    <div className={styles.section}>
      <Text strong>{title}</Text>
      {items.length === 0 ? (
        <Text className={styles.emptyText}>{t('settings.plugins.noCapabilities')}</Text>
      ) : (
        <div className={styles.capabilityList}>
          {items.map(item => (
            <div key={`${tag}-${item.id}`} className={styles.capabilityRow}>
              <div>
                <div className={styles.capabilityName}>{item.name || item.id}</div>
                {item.description && <div className={styles.capabilityDesc}>{item.description}</div>}
              </div>
              <Tag bordered={false}>{tag}</Tag>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  const renderDetail = () => (
    <Card className={styles.detailCard}>
      {!active ? (
        <div className={styles.emptyState}><Empty description={t('settings.plugins.empty')} /></div>
      ) : (
        <div className={styles.detailContent}>
          <div className={styles.detailHeader}>
            <div>
              <div className={styles.detailTitleRow}>
                <Title level={4} style={{ margin: 0 }}>{active.name}</Title>
                <Tag color={typeColor[active.type] || 'default'} bordered={false}>{active.type}</Tag>
                <Tag color={statusColor[active.status] || 'default'} bordered={false}>
                  {t(`settings.plugins.status.${active.status}`, active.status)}
                </Tag>
              </div>
              <Text className={styles.pluginMeta}>{active.id} · {active.version}</Text>
            </div>
            <div className={styles.actionRow}>
              <div className={`${styles.segmentSwitch} ${pendingPluginID === active.id ? styles.segmentSwitchLoading : ''}`}>
                <button
                  type="button"
                  className={`${styles.segmentOption} ${active.enabled ? styles.segmentOptionActive : ''}`}
                  disabled={active.enabled || pendingPluginID === active.id}
                  onClick={() => void handleToggle(active)}
                >
                  {t('settings.plugins.actions.enable')}
                </button>
                <button
                  type="button"
                  className={`${styles.segmentOption} ${!active.enabled ? styles.segmentOptionActive : ''}`}
                  disabled={!active.enabled || pendingPluginID === active.id}
                  onClick={() => void handleToggle(active)}
                >
                  {t('settings.plugins.actions.disable')}
                </button>
              </div>
              {active.has_settings && (
                <Button
                  size="small"
                  className={`${styles.actionButton} ${styles.settingsIconButton}`}
                  icon={<SettingOutlined />}
                  children={t('settings.plugins.actions.settings')}
                  aria-label={t('settings.plugins.actions.settings')}
                  title={t('settings.plugins.actions.settings')}
                  onClick={() => void openPluginSettingsWindow(active.id)}
                />
              )}
              <Button
                size="small"
                danger
                className={styles.actionButton}
                icon={<DeleteOutlined />}
                loading={deletingPluginID === active.id}
                onClick={() => handleDelete(active)}
              >
                {t('settings.plugins.actions.delete')}
              </Button>
            </div>
          </div>

          {active.last_error && <Alert type="error" showIcon message={active.last_error} />}
          {active.description && <Paragraph className={styles.description}>{active.description}</Paragraph>}

          <div className={styles.section}>
            <Text strong>{t('settings.plugins.permissions')}</Text>
            <Space wrap>
              {(active.permissions || []).length === 0 ? (
                <Text className={styles.emptyText}>{t('settings.plugins.noPermissions')}</Text>
              ) : active.permissions.map(permission => (
                <Tag key={permission}>{permission}</Tag>
              ))}
            </Space>
          </div>

          {renderCapabilitySection(t('settings.plugins.useTools'), active.use_tools || [], 'use_tool')}
          {renderCapabilitySection(t('settings.plugins.viewTools'), active.view_tools || [], 'view_tool')}
          {renderCapabilitySection(t('settings.plugins.agents'), active.agents || [], 'agent')}
          {renderCapabilitySection(t('settings.plugins.views'), active.views || [], 'view')}
        </div>
      )}
    </Card>
  );

  const runtimePhaseText = () => {
    const phase = runtimeStatus?.phase;
    if (phase === 'extracting') return t('settings.plugins.runtimeExtracting');
    if (phase === 'installing') return t('settings.plugins.runtimeInstalling');
    if (phase === 'ready') return t('settings.plugins.runtimeReady');
    if (downloadingRuntime || runtimeStatus?.downloading || phase === 'downloading') {
      return t('settings.plugins.runtimeDownloading');
    }
    return t('settings.plugins.runtimePreparing');
  };

  const renderRuntimeMissing = () => {
    const showProgress = downloadingRuntime || Boolean(runtimeStatus?.downloading);
    const percent = Math.max(0, Math.min(100, runtimeStatus?.progress || 0));
    return (
      <Card className={styles.runtimeCard}>
        <div className={styles.runtimePrompt}>
          <div className={styles.runtimeIcon}><CloudDownloadOutlined /></div>
          <Title level={4}>{t('settings.plugins.runtimeMissingTitle')}</Title>
          <Paragraph className={styles.description}>
            {t('settings.plugins.runtimeMissingDescription')}
          </Paragraph>
          {showProgress && (
            <div className={styles.runtimeProgress}>
              <Progress percent={percent} status="active" />
              <Text type="secondary">{runtimePhaseText()}</Text>
            </div>
          )}
          <Space>
            <Button
              type="primary"
              icon={<CloudDownloadOutlined />}
              loading={downloadingRuntime}
              onClick={() => void handleDownloadRuntime()}
            >
              {downloadingRuntime ? t('settings.plugins.actions.downloadingRuntime') : t('settings.plugins.actions.downloadRuntime')}
            </Button>
            <Button icon={<ReloadOutlined />} disabled={downloadingRuntime} onClick={() => void refresh()}>
              {t('common.retry')}
            </Button>
          </Space>
        </div>
      </Card>
    );
  };

  return (
    <div className={`${styles.pluginSettings} ${className || ''}`}>
      {error && <Alert type="error" showIcon message={error} style={{ marginBottom: 12 }} />}
      {runtimeStatus && !runtimeStatus.available ? renderRuntimeMissing() : (
        <div className={styles.desktopLayout}>
          <div className={styles.listColumn}>{renderList()}</div>
          <div className={styles.detailColumn}>{renderDetail()}</div>
        </div>
      )}
    </div>
  );
};

export default PluginSettingsPage;
