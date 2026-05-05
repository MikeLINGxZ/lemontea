import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import { useSearchParams } from 'react-router-dom';
import { Service } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service';
import styles from './formWindow.module.scss';

const { Paragraph, Title, Text } = Typography;

type ProtocolConfig = {
  host?: string;
  port?: number;
  security?: string;
  username?: string;
  passwordSet?: boolean;
  lastTestedAt?: string;
  lastTestStatus?: string;
};

type EmailPluginConfig = {
  account?: {
    email?: string;
    displayName?: string;
    replyTo?: string;
    preset?: string;
  };
  imap?: ProtocolConfig;
  smtp?: ProtocolConfig;
  updatedAt?: string;
};

type CredentialAction = 'keep' | 'set' | 'delete';
type MailProtocol = 'imap' | 'smtp';

const { Option } = Select;

const providerPresets: Record<string, { imap: Partial<ProtocolConfig>; smtp: Partial<ProtocolConfig> }> = {
  gmail: {
    imap: { host: 'imap.gmail.com', port: 993, security: 'tls' },
    smtp: { host: 'smtp.gmail.com', port: 465, security: 'tls' },
  },
  outlook: {
    imap: { host: 'outlook.office365.com', port: 993, security: 'tls' },
    smtp: { host: 'smtp.office365.com', port: 587, security: 'starttls' },
  },
  qq: {
    imap: { host: 'imap.qq.com', port: 993, security: 'tls' },
    smtp: { host: 'smtp.qq.com', port: 465, security: 'tls' },
  },
  '163': {
    imap: { host: 'imap.163.com', port: 993, security: 'tls' },
    smtp: { host: 'smtp.163.com', port: 465, security: 'tls' },
  },
  custom: {
    imap: { host: '', port: 993, security: 'tls' },
    smtp: { host: '', port: 465, security: 'tls' },
  },
};

const defaultConfig: Required<EmailPluginConfig> = {
  account: {
    email: '',
    displayName: '',
    replyTo: '',
    preset: 'custom',
  },
  imap: {
    host: '',
    port: 993,
    security: 'tls',
    username: '',
    passwordSet: false,
    lastTestedAt: '',
    lastTestStatus: '',
  },
  smtp: {
    host: '',
    port: 465,
    security: 'tls',
    username: '',
    passwordSet: false,
    lastTestedAt: '',
    lastTestStatus: '',
  },
  updatedAt: '',
};

const mergeConfig = (input?: EmailPluginConfig): Required<EmailPluginConfig> => ({
  account: { ...defaultConfig.account, ...(input?.account || {}) },
  imap: { ...defaultConfig.imap, ...(input?.imap || {}) },
  smtp: { ...defaultConfig.smtp, ...(input?.smtp || {}) },
  updatedAt: input?.updatedAt || '',
});

function formatTestStatus(status: string): { color: string; text: string } | null {
  if (!status) return null;
  if (status === 'success') return { color: 'success', text: 'Last test succeeded' };
  if (status === 'error') return { color: 'error', text: 'Last test failed' };
  return { color: 'default', text: status };
}

const EmailPluginSettingsPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const pluginID = searchParams.get('id') || 'com.lemontea.examples.email';
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testingProtocol, setTestingProtocol] = useState<MailProtocol | ''>('');
  const [error, setError] = useState('');
  const [configSnapshot, setConfigSnapshot] = useState<Required<EmailPluginConfig>>(defaultConfig);
  const [imapPasswordAction, setImapPasswordAction] = useState<CredentialAction>('keep');
  const [smtpPasswordAction, setSmtpPasswordAction] = useState<CredentialAction>('keep');

  const title = useMemo(() => 'Email Plugin', []);

  const applyFormValues = (config: Required<EmailPluginConfig>) => {
    form.setFieldsValue({
      providerPreset: config.account.preset,
      accountEmail: config.account.email,
      accountDisplayName: config.account.displayName,
      accountReplyTo: config.account.replyTo,
      imapHost: config.imap.host,
      imapPort: config.imap.port,
      imapSecurity: config.imap.security,
      imapUsername: config.imap.username,
      imapPassword: '',
      smtpHost: config.smtp.host,
      smtpPort: config.smtp.port,
      smtpSecurity: config.smtp.security,
      smtpUsername: config.smtp.username,
      smtpPassword: '',
    });
  };

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const config = mergeConfig(await Service.GetPluginSettings(pluginID) as EmailPluginConfig);
      setConfigSnapshot(config);
      setImapPasswordAction('keep');
      setSmtpPasswordAction('keep');
      applyFormValues(config);
    } catch (e: any) {
      console.error(e);
      setError(e?.message || 'Failed to load plugin settings');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [pluginID]);

  const applyPreset = (preset: string) => {
    const nextPreset = providerPresets[preset] || providerPresets.custom;
    form.setFieldsValue({
      providerPreset: preset,
      imapHost: nextPreset.imap.host,
      imapPort: nextPreset.imap.port,
      imapSecurity: nextPreset.imap.security,
      smtpHost: nextPreset.smtp.host,
      smtpPort: nextPreset.smtp.port,
      smtpSecurity: nextPreset.smtp.security,
    });
  };

  const buildPayload = async (): Promise<EmailPluginConfig> => {
    const values = await form.validateFields();
    return {
      account: {
        email: values.accountEmail,
        displayName: values.accountDisplayName,
        replyTo: values.accountReplyTo,
        preset: values.providerPreset,
      },
      imap: {
        host: values.imapHost,
        port: values.imapPort,
        security: values.imapSecurity,
        username: values.imapUsername,
        password: values.imapPassword || '',
        passwordAction: values.imapPassword ? 'set' : imapPasswordAction,
      } as Record<string, unknown>,
      smtp: {
        host: values.smtpHost,
        port: values.smtpPort,
        security: values.smtpSecurity,
        username: values.smtpUsername,
        password: values.smtpPassword || '',
        passwordAction: values.smtpPassword ? 'set' : smtpPasswordAction,
      } as Record<string, unknown>,
    };
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      const payload = await buildPayload();
      const saved = mergeConfig(await Service.SavePluginSettings(pluginID, payload) as EmailPluginConfig);
      setConfigSnapshot(saved);
      setImapPasswordAction('keep');
      setSmtpPasswordAction('keep');
      applyFormValues(saved);
      message.success('Settings saved');
    } catch (e: any) {
      console.error(e);
      setError(e?.message || 'Failed to save plugin settings');
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async (protocol: MailProtocol, saveFirst = false) => {
    setTestingProtocol(protocol);
    setError('');
    try {
      const payload = await buildPayload();
      if (saveFirst) {
        const saved = mergeConfig(await Service.SavePluginSettings(pluginID, payload) as EmailPluginConfig);
        setConfigSnapshot(saved);
        setImapPasswordAction('keep');
        setSmtpPasswordAction('keep');
        applyFormValues(saved);
      }
      const result = await Service.TestPluginConnection(pluginID, protocol, payload);
      if (result?.ok) {
        message.success(result?.message || `${protocol.toUpperCase()} connection succeeded`);
      } else {
        message.error(result?.message || `${protocol.toUpperCase()} connection failed`);
      }
      await load();
    } catch (e: any) {
      console.error(e);
      setError(e?.message || `Failed to test ${protocol.toUpperCase()} connection`);
    } finally {
      setTestingProtocol('');
    }
  };

  const handleSaveAndTest = async () => {
    setSaving(true);
    setError('');
    try {
      const payload = await buildPayload();
      const saved = mergeConfig(await Service.SavePluginSettings(pluginID, payload) as EmailPluginConfig);
      setConfigSnapshot(saved);
      setImapPasswordAction('keep');
      setSmtpPasswordAction('keep');
      applyFormValues(saved);
      const imapResult = await Service.TestPluginConnection(pluginID, 'imap', payload);
      const smtpResult = await Service.TestPluginConnection(pluginID, 'smtp', payload);
      if (imapResult?.ok && smtpResult?.ok) {
        message.success('Settings saved and both connections succeeded');
      } else {
        message.warning([
          imapResult?.message || 'IMAP test failed',
          smtpResult?.message || 'SMTP test failed',
        ].join(' | '));
      }
      await load();
    } catch (e: any) {
      console.error(e);
      setError(e?.message || 'Failed to save and test settings');
    } finally {
      setSaving(false);
    }
  };

  const renderProtocolStatus = (protocol: MailProtocol) => {
    const current = configSnapshot[protocol];
    const status = formatTestStatus(current.lastTestStatus || '');
    return (
      <Space size={8} wrap>
        {current.passwordSet ? <Tag color="green">Credential saved</Tag> : <Tag>Credential missing</Tag>}
        {status ? <Tag color={status.color}>{status.text}</Tag> : null}
        {current.lastTestedAt ? (
          <Text type="secondary">Tested at {new Date(current.lastTestedAt).toLocaleString()}</Text>
        ) : null}
      </Space>
    );
  };

  return (
    <div className={styles.formWindow}>
      <Card bordered={false}>
        <Space direction="vertical" size={20} style={{ width: '100%' }}>
          <div>
            <Title level={3} style={{ marginBottom: 8 }}>{title}</Title>
            <Paragraph type="secondary" style={{ marginBottom: 0 }}>
              Configure mailbox identity, choose a provider preset, save host-managed credentials, and test IMAP or SMTP connectivity before using the plugin.
            </Paragraph>
          </div>

          {error && <Alert type="error" showIcon message={error} />}

          {loading ? (
            <div style={{ minHeight: 240, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <Spin />
            </div>
          ) : (
            <Form form={form} layout="vertical" autoComplete="off">
              <Title level={5}>Account</Title>
              <Form.Item label="Provider preset" name="providerPreset">
                <Select onChange={applyPreset}>
                  <Option value="gmail">Gmail</Option>
                  <Option value="outlook">Outlook / Office 365</Option>
                  <Option value="qq">QQ Mail</Option>
                  <Option value="163">163 Mail</Option>
                  <Option value="custom">Custom</Option>
                </Select>
              </Form.Item>
              <Form.Item label="Email address" name="accountEmail" rules={[{ required: true, type: 'email' }]}>
                <Input />
              </Form.Item>
              <Form.Item label="Display name" name="accountDisplayName">
                <Input />
              </Form.Item>
              <Form.Item label="Reply-to address" name="accountReplyTo" rules={[{ type: 'email' }]}>
                <Input />
              </Form.Item>

              <Title level={5}>Incoming mail server (IMAP)</Title>
              <div style={{ marginBottom: 12 }}>{renderProtocolStatus('imap')}</div>
              <Form.Item label="Host" name="imapHost" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item label="Port" name="imapPort" rules={[{ required: true }]}>
                <InputNumber min={1} max={65535} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item label="Security" name="imapSecurity" rules={[{ required: true }]}>
                <Select options={[
                  { label: 'TLS', value: 'tls' },
                  { label: 'STARTTLS', value: 'starttls' },
                  { label: 'None', value: 'none' },
                ]} />
              </Form.Item>
              <Form.Item label="Username" name="imapUsername" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item label="Password" name="imapPassword">
                <Input.Password
                  placeholder={imapPasswordAction === 'delete' ? 'Password will be deleted on save' : 'Leave blank to keep the current credential'}
                  onChange={(event) => {
                    setImapPasswordAction(event.target.value ? 'set' : 'keep');
                  }}
                />
              </Form.Item>
              <Space style={{ marginBottom: 24 }} wrap>
                <Button onClick={() => setImapPasswordAction('delete')}>Delete saved IMAP password</Button>
                <Button loading={testingProtocol === 'imap'} onClick={() => void handleTest('imap')}>Test IMAP connection</Button>
              </Space>

              <Title level={5}>Outgoing mail server (SMTP)</Title>
              <div style={{ marginBottom: 12 }}>{renderProtocolStatus('smtp')}</div>
              <Form.Item label="Host" name="smtpHost" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item label="Port" name="smtpPort" rules={[{ required: true }]}>
                <InputNumber min={1} max={65535} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item label="Security" name="smtpSecurity" rules={[{ required: true }]}>
                <Select options={[
                  { label: 'TLS', value: 'tls' },
                  { label: 'STARTTLS', value: 'starttls' },
                  { label: 'None', value: 'none' },
                ]} />
              </Form.Item>
              <Form.Item label="Username" name="smtpUsername" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item label="Password" name="smtpPassword">
                <Input.Password
                  placeholder={smtpPasswordAction === 'delete' ? 'Password will be deleted on save' : 'Leave blank to keep the current credential'}
                  onChange={(event) => {
                    setSmtpPasswordAction(event.target.value ? 'set' : 'keep');
                  }}
                />
              </Form.Item>
              <Space style={{ marginBottom: 24 }} wrap>
                <Button onClick={() => setSmtpPasswordAction('delete')}>Delete saved SMTP password</Button>
                <Button loading={testingProtocol === 'smtp'} onClick={() => void handleTest('smtp')}>Test SMTP connection</Button>
              </Space>

              <Space wrap>
                <Button type="primary" loading={saving} onClick={() => void handleSave()}>
                  Save
                </Button>
                <Button loading={saving} onClick={() => void handleSaveAndTest()}>
                  Save and test
                </Button>
                <Button loading={testingProtocol === 'imap'} onClick={() => void handleTest('imap', true)}>
                  Save, then test IMAP
                </Button>
                <Button loading={testingProtocol === 'smtp'} onClick={() => void handleTest('smtp', true)}>
                  Save, then test SMTP
                </Button>
              </Space>
            </Form>
          )}
        </Space>
      </Card>
    </div>
  );
};

export default EmailPluginSettingsPage;
