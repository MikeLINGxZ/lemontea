const fs = require('fs');
const path = require('path');
const { ImapFlow } = require('imapflow');
const { simpleParser } = require('mailparser');
const nodemailer = require('nodemailer');
const { definePlugin } = require('../dist/sdk/runtime');

let pluginContext = null;
let configStore = null;
let sentMailStore = null;

function ensureDataDir() {
  const dir = pluginContext && pluginContext.dataDir
    ? pluginContext.dataDir
    : path.join(__dirname, '..', 'data');
  fs.mkdirSync(dir, { recursive: true });
}

function defaultConfig() {
  return {
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
}

function mergeConfig(stored) {
  const defaults = defaultConfig();
  return {
    ...defaults,
    ...(stored || {}),
    account: { ...defaults.account, ...((stored && stored.account) || {}) },
    imap: { ...defaults.imap, ...((stored && stored.imap) || {}) },
    smtp: { ...defaults.smtp, ...((stored && stored.smtp) || {}) },
  };
}

function loadStoredConfig() {
  return mergeConfig(configStore ? configStore.read() : defaultConfig());
}

function persistConfig(config) {
  ensureDataDir();
  if (configStore) {
    configStore.write(config);
  }
}

async function loadConfig() {
  const config = mergeConfig(loadStoredConfig());
  config.imap.passwordSet = await credentialExists(config, 'imap');
  config.smtp.passwordSet = await credentialExists(config, 'smtp');
  return config;
}

function parseArgs(raw) {
  if (!raw || typeof raw !== 'object') return {};
  return raw;
}

function stringValue(value) {
  return String(value || '').trim();
}

function numberValue(value, fallback) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function toArray(value) {
  if (!value) return [];
  if (Array.isArray(value)) {
    return value.map(item => stringValue(item)).filter(Boolean);
  }
  return String(value)
    .split(',')
    .map(item => item.trim())
    .filter(Boolean);
}

function validateEmailAddress(value, fieldName) {
  const email = stringValue(value);
  if (!email) {
    throw new Error(`${fieldName} is required`);
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    throw new Error(`${fieldName} must be a valid email address`);
  }
  return email;
}

function validateEmailList(value, fieldName) {
  const values = toArray(value);
  values.forEach((item, index) => validateEmailAddress(item, `${fieldName}[${index}]`));
  return values;
}

function htmlToText(html) {
  return String(html || '')
    .replace(/<style[\s\S]*?<\/style>/gi, ' ')
    .replace(/<script[\s\S]*?<\/script>/gi, ' ')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function encodeCursor(value) {
  return Buffer.from(JSON.stringify(value), 'utf8').toString('base64');
}

function decodeCursor(value) {
  if (!value) return null;
  try {
    return JSON.parse(Buffer.from(String(value), 'base64').toString('utf8'));
  } catch {
    return null;
  }
}

function normalizeConfig(config) {
  const merged = mergeConfig(config);
  return {
    ...merged,
    account: {
      email: stringValue(merged.account.email),
      displayName: stringValue(merged.account.displayName),
      replyTo: stringValue(merged.account.replyTo),
      preset: stringValue(merged.account.preset) || 'custom',
    },
    imap: {
      host: stringValue(merged.imap.host),
      port: numberValue(merged.imap.port, 993),
      security: ['tls', 'starttls', 'none'].includes(merged.imap.security) ? merged.imap.security : 'tls',
      username: stringValue(merged.imap.username),
      passwordSet: Boolean(merged.imap.passwordSet),
      lastTestedAt: stringValue(merged.imap.lastTestedAt),
      lastTestStatus: stringValue(merged.imap.lastTestStatus),
    },
    smtp: {
      host: stringValue(merged.smtp.host),
      port: numberValue(merged.smtp.port, 465),
      security: ['tls', 'starttls', 'none'].includes(merged.smtp.security) ? merged.smtp.security : 'tls',
      username: stringValue(merged.smtp.username),
      passwordSet: Boolean(merged.smtp.passwordSet),
      lastTestedAt: stringValue(merged.smtp.lastTestedAt),
      lastTestStatus: stringValue(merged.smtp.lastTestStatus),
    },
  };
}

function credentialScope(config, protocol) {
  const account = stringValue(config.account?.email) || stringValue(config[protocol]?.username) || 'default';
  return `mail:${account}:${protocol}`;
}

async function callHost(method, params) {
  if (!pluginContext) {
    throw new Error('Plugin runtime is not initialized.');
  }
  return pluginContext.host.call(method, params);
}

async function getCredential(config, protocol) {
  const response = await callHost('get_credential', {
    scope: credentialScope(config, protocol),
    key: 'password',
  });
  if (!response || !response.set) {
    return '';
  }
  return stringValue(response.value);
}

async function credentialExists(config, protocol) {
  const response = await callHost('get_credential', {
    scope: credentialScope(config, protocol),
    key: 'password',
  });
  return Boolean(response && response.set);
}

async function setCredential(config, protocol, value) {
  await callHost('set_credential', {
    scope: credentialScope(config, protocol),
    key: 'password',
    value,
  });
}

async function deleteCredential(config, protocol) {
  await callHost('delete_credential', {
    scope: credentialScope(config, protocol),
    key: 'password',
  });
}

async function applyCredentialChanges(config, protocol, incoming, storedConfig) {
  const normalizedProtocol = normalizeProtocolConfig(incoming);
  const action = stringValue(normalizedProtocol.passwordAction) || 'keep';
  const nextConfig = { ...config };

  if (action === 'set') {
    const password = stringValue(normalizedProtocol.password);
    if (!password) {
      throw new Error(`${protocol}.password is required`);
    }
    await setCredential(nextConfig, protocol, password);
    nextConfig[protocol].passwordSet = true;
    return nextConfig;
  }

  if (action === 'delete') {
    await deleteCredential(nextConfig, protocol);
    nextConfig[protocol].passwordSet = false;
    return nextConfig;
  }

  const hasStoredPassword = await credentialExists(storedConfig || nextConfig, protocol);
  nextConfig[protocol].passwordSet = hasStoredPassword;
  return nextConfig;
}

function normalizeProtocolConfig(input) {
  if (!input || typeof input !== 'object') {
    return {};
  }
  return input;
}

function ensureMailAddressConfig(config, protocol) {
  validateEmailAddress(config.account.email, 'account.email');
  if (config.account.replyTo) {
    validateEmailAddress(config.account.replyTo, 'account.replyTo');
  }
  if (!config[protocol].host) throw new Error(`${protocol}.host is required`);
  if (!config[protocol].username) throw new Error(`${protocol}.username is required`);
}

async function resolveProtocolConfig(inputConfig, protocol, options = {}) {
  const config = normalizeConfig(inputConfig);
  ensureMailAddressConfig(config, protocol);
  const password =
    stringValue(options.passwordOverride) ||
    await getCredential(config, protocol);
  if (!password) {
    throw new Error(`${protocol}.password is required`);
  }
  return {
    ...config,
    [protocol]: {
      ...config[protocol],
      password,
      passwordSet: true,
    },
  };
}

function imapClientOptions(config) {
  const security = config.imap.security;
  return {
    host: config.imap.host,
    port: config.imap.port,
    secure: security === 'tls',
    auth: {
      user: config.imap.username,
      pass: config.imap.password,
    },
    logger: false,
    tls: {
      rejectUnauthorized: false,
    },
    doSTARTTLS: security === 'starttls',
    disableAutoEnable: security === 'none',
  };
}

function smtpTransportOptions(config) {
  const security = config.smtp.security;
  return {
    host: config.smtp.host,
    port: config.smtp.port,
    secure: security === 'tls',
    requireTLS: security === 'starttls',
    auth: {
      user: config.smtp.username,
      pass: config.smtp.password,
    },
    tls: {
      rejectUnauthorized: false,
    },
  };
}

function formatAddress(input) {
  if (!input) return '';
  if (Array.isArray(input)) {
    return input.map(formatAddress).filter(Boolean).join(', ');
  }
  if (typeof input === 'object') {
    const name = stringValue(input.name);
    const address = stringValue(input.address);
    if (name && address) return `${name} <${address}>`;
    return address || name;
  }
  return String(input);
}

function toAddressArray(input) {
  if (!input) return [];
  if (Array.isArray(input)) {
    return input.flatMap(item => toAddressArray(item));
  }
  if (typeof input === 'object') {
    const address = stringValue(input.address);
    if (!address) return [];
    return [{ name: stringValue(input.name), address }];
  }
  return validateEmailList(input, 'address').map(address => ({ address }));
}

function normalizeAttachment(attachment) {
  return {
    filename: stringValue(attachment.filename),
    contentType: stringValue(attachment.contentType),
    size: Number(attachment.size || 0),
    contentDisposition: stringValue(attachment.contentDisposition),
    cid: stringValue(attachment.cid),
  };
}

function normalizeMessage(record, options = {}) {
  const plainText = stringValue(record.text);
  const htmlText = stringValue(record.html);
  const body = options.includeBody ? (plainText || htmlToText(htmlText) || '') : '';
  const snippet = plainText || htmlToText(htmlText) || '';
  return {
    id: `imap_${record.mailbox}_${record.uid}`,
    uid: record.uid,
    from: formatAddress(record.from),
    to: formatAddress(record.to),
    cc: formatAddress(record.cc),
    subject: stringValue(record.subject) || '(No subject)',
    snippet: snippet.slice(0, 280),
    receivedAt: record.receivedAt,
    unread: !record.seen,
    mailbox: record.mailbox,
    hasAttachments: Array.isArray(record.attachments) && record.attachments.length > 0,
    body,
    html: options.includeHtml ? htmlText : '',
    attachments: Array.isArray(record.attachments) ? record.attachments.map(normalizeAttachment) : [],
    headers: record.headers || {},
  };
}

function matchesDateRange(value, since, before) {
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) return false;
  if (since && timestamp < new Date(since).getTime()) return false;
  if (before && timestamp > new Date(before).getTime()) return false;
  return true;
}

function matchesQuery(message, filters) {
  const textTargets = [
    message.from,
    message.to,
    message.cc,
    message.subject,
    message.body,
    message.snippet,
  ].map(item => String(item || '').toLowerCase());

  if (filters.query && !textTargets.join('\n').includes(filters.query)) {
    return false;
  }
  if (filters.from && !String(message.from || '').toLowerCase().includes(filters.from)) {
    return false;
  }
  if (filters.to) {
    const target = `${message.to || ''}\n${message.cc || ''}`.toLowerCase();
    if (!target.includes(filters.to)) {
      return false;
    }
  }
  if (filters.subject && !String(message.subject || '').toLowerCase().includes(filters.subject)) {
    return false;
  }
  if ((filters.since || filters.before) && !matchesDateRange(message.receivedAt, filters.since, filters.before)) {
    return false;
  }
  return true;
}

function friendlyErrorMessage(error) {
  const message = String(error && error.message ? error.message : error || 'Unknown email error');
  const lower = message.toLowerCase();
  if (lower.includes('auth') || lower.includes('invalid login') || lower.includes('authentication')) {
    return 'Authentication failed. Please check the username and password.';
  }
  if (lower.includes('timed out') || lower.includes('timeout')) {
    return 'Connection timed out. Please verify the server address, port, and network access.';
  }
  if (lower.includes('certificate') || lower.includes('tls') || lower.includes('ssl')) {
    return 'TLS or SSL negotiation failed. Please verify the selected security mode and server certificates.';
  }
  if (lower.includes('mailbox') || lower.includes('no such')) {
    return 'The mailbox could not be opened. Please confirm the folder name exists on the server.';
  }
  return message;
}

async function saveConfig(rawConfig) {
  const storedConfig = await loadConfig();
  let next = normalizeConfig({
    ...storedConfig,
    ...(rawConfig || {}),
    account: {
      ...storedConfig.account,
      ...((rawConfig || {}).account || {}),
    },
    imap: {
      ...storedConfig.imap,
      ...((rawConfig || {}).imap || {}),
    },
    smtp: {
      ...storedConfig.smtp,
      ...((rawConfig || {}).smtp || {}),
    },
    updatedAt: new Date().toISOString(),
  });

  validateEmailAddress(next.account.email, 'account.email');
  next = await applyCredentialChanges(next, 'imap', (rawConfig || {}).imap, storedConfig);
  next = await applyCredentialChanges(next, 'smtp', (rawConfig || {}).smtp, storedConfig);
  next.updatedAt = new Date().toISOString();

  const persisted = normalizeConfig(next);
  persistConfig(persisted);
  return await loadConfig();
}

function loadSentMail() {
  return Array.isArray(sentMailStore && sentMailStore.read()) ? sentMailStore.read() : [];
}

function appendSentMailLog(entry) {
  sentMailStore.update((history) => {
    const nextHistory = Array.isArray(history) ? history.slice() : [];
    nextHistory.unshift(entry);
    return nextHistory.slice(0, 200);
  });
}

async function updateTestStatus(protocol, status) {
  const current = normalizeConfig(loadStoredConfig());
  current[protocol].lastTestedAt = new Date().toISOString();
  current[protocol].lastTestStatus = status;
  current[protocol].passwordSet = await credentialExists(current, protocol);
  persistConfig(current);
}

async function testConnection(protocol, rawConfig) {
  const merged = normalizeConfig({
    ...(await loadConfig()),
    ...(rawConfig || {}),
    account: {
      ...((await loadConfig()).account),
      ...(((rawConfig || {}).account) || {}),
    },
    [protocol]: {
      ...((await loadConfig())[protocol]),
      ...(((rawConfig || {})[protocol]) || {}),
    },
  });
  const rawProtocolConfig = ((rawConfig || {})[protocol]) || {};
  const passwordOverride =
    stringValue(rawProtocolConfig.passwordAction) === 'set'
      ? stringValue(rawProtocolConfig.password)
      : '';

  try {
    if (protocol === 'imap') {
      const config = await resolveProtocolConfig(merged, 'imap', { passwordOverride });
      const client = new ImapFlow(imapClientOptions(config));
      await client.connect();
      await client.mailboxOpen('INBOX').catch(() => {});
      await client.logout().catch(() => {});
    } else {
      const config = await resolveProtocolConfig(merged, 'smtp', { passwordOverride });
      const transporter = nodemailer.createTransport(smtpTransportOptions(config));
      await transporter.verify();
    }
    await updateTestStatus(protocol, 'success');
    return {
      ok: true,
      protocol,
      testedAt: new Date().toISOString(),
      message: `${protocol.toUpperCase()} connection succeeded.`,
    };
  } catch (error) {
    await updateTestStatus(protocol, 'error');
    return {
      ok: false,
      protocol,
      testedAt: new Date().toISOString(),
      message: friendlyErrorMessage(error),
    };
  }
}

async function listFolders() {
  const config = await resolveProtocolConfig(await loadConfig(), 'imap');
  const client = new ImapFlow(imapClientOptions(config));
  try {
    await client.connect();
    const folders = await client.list();
    const items = [];
    for (const folder of folders) {
      let unseen = 0;
      try {
        const status = await client.status(folder.path, { unseen: true, messages: true });
        unseen = Number(status.unseen || 0);
      } catch {
        unseen = 0;
      }
      items.push({
        path: folder.path,
        name: folder.name || folder.path,
        specialUse: folder.specialUse || '',
        unseen,
      });
    }
    return {
      account: config.account.email,
      folders: items,
      count: items.length,
    };
  } finally {
    await client.logout().catch(() => {});
  }
}

async function getMail(rawArgs) {
  const args = parseArgs(rawArgs);
  const mailbox = stringValue(args.mailbox) || 'INBOX';
  const uid = numberValue(args.uid, 0);
  const markAsRead = args.markAsRead !== false;
  if (!uid) {
    throw new Error('uid is required');
  }
  const config = await resolveProtocolConfig(await loadConfig(), 'imap');
  const client = new ImapFlow(imapClientOptions(config));
  try {
    await client.connect();
    const lock = await client.getMailboxLock(mailbox);
    try {
      await client.mailboxOpen(mailbox);
      const matches = await client.search({ uid: String(uid) }, { uid: true }).catch(() => []);
      if (!Array.isArray(matches) || !matches.includes(uid)) {
        throw new Error(`Message ${uid} is no longer available in ${mailbox}. It may have been moved, deleted, or the mailbox has changed.`);
      }
      const item = await client.fetchOne(String(uid), {
        uid: true,
        envelope: true,
        flags: true,
        internalDate: true,
        source: true,
      }, { uid: true });
      if (!item) {
        throw new Error(`Message ${uid} is no longer available in ${mailbox}. It may have been moved, deleted, or the mailbox has changed.`);
      }
      let seen = item.flags?.has('\\Seen') || false;
      if (markAsRead && !seen) {
        await client.messageFlagsAdd(String(uid), ['\\Seen'], { uid: true, silent: true });
        seen = true;
      }
      const parsed = await simpleParser(item.source);
      const message = normalizeMessage({
        uid: item.uid,
        from: parsed.from?.value || item.envelope?.from || [],
        to: parsed.to?.value || item.envelope?.to || [],
        cc: parsed.cc?.value || item.envelope?.cc || [],
        subject: parsed.subject || item.envelope?.subject || '',
        text: parsed.text || '',
        html: parsed.html || '',
        receivedAt: (item.internalDate || parsed.date || new Date()).toISOString(),
        seen,
        mailbox,
        attachments: parsed.attachments || [],
        headers: {
          messageId: stringValue(parsed.messageId),
          inReplyTo: stringValue(parsed.inReplyTo),
          references: toArray(parsed.references),
        },
      }, {
        includeBody: true,
        includeHtml: true,
      });
      return {
        account: config.account.email,
        mailbox,
        message,
      };
    } finally {
      lock.release();
    }
  } finally {
    await client.logout().catch(() => {});
  }
}

async function searchMail(rawArgs) {
  const args = parseArgs(rawArgs);
  const config = await resolveProtocolConfig(await loadConfig(), 'imap');
  const mailbox = stringValue(args.mailbox || args.folder) || 'INBOX';
  const unreadOnly = args.unreadOnly === true;
  const includeBody = args.includeBody === true;
  const includeHtml = args.includeHtml === true;
  const limit = Math.max(1, Math.min(numberValue(args.limit, 10), 50));
  const cursor = decodeCursor(args.cursor);
  const afterUidExclusive = Number(cursor && cursor.afterUidExclusive ? cursor.afterUidExclusive : 0);
  const filters = {
    query: stringValue(args.query).toLowerCase(),
    from: stringValue(args.from).toLowerCase(),
    to: stringValue(args.to).toLowerCase(),
    subject: stringValue(args.subject).toLowerCase(),
    since: stringValue(args.since),
    before: stringValue(args.before),
  };
  const client = new ImapFlow(imapClientOptions(config));
  const messages = [];
  let candidateUids = [];

  try {
    await client.connect();
    const lock = await client.getMailboxLock(mailbox);
    try {
      const mailboxInfo = await client.mailboxOpen(mailbox);
      const exists = mailboxInfo.exists || 0;
      if (exists === 0) {
        return {
          count: 0,
          account: config.account.email,
          mailbox,
          folder: mailbox,
          nextCursor: '',
          hasMore: false,
          messages: [],
        };
      }

      const allUids = await client.search({}, { uid: true });
      candidateUids = (Array.isArray(allUids) ? allUids : [])
        .filter(item => Number.isFinite(Number(item)))
        .map(item => Number(item))
        .filter(item => (afterUidExclusive > 0 ? item < afterUidExclusive : true))
        .sort((a, b) => b - a);

      const batchSize = Math.max(limit * 3, 30);
      for (let index = 0; index < candidateUids.length && messages.length < limit; index += batchSize) {
        const batchUids = candidateUids.slice(index, index + batchSize);
        if (batchUids.length === 0) {
          break;
        }
        for await (const item of client.fetch(batchUids, {
          uid: true,
          envelope: true,
          flags: true,
          internalDate: true,
          source: true,
        }, { uid: true })) {
          const parsed = await simpleParser(item.source);
          const message = normalizeMessage({
            uid: item.uid,
            from: parsed.from?.value || item.envelope?.from || [],
            to: parsed.to?.value || item.envelope?.to || [],
            cc: parsed.cc?.value || item.envelope?.cc || [],
            subject: parsed.subject || item.envelope?.subject || '',
            text: parsed.text || '',
            html: parsed.html || '',
            receivedAt: (item.internalDate || parsed.date || new Date()).toISOString(),
            seen: item.flags?.has('\\Seen') || false,
            mailbox,
            attachments: parsed.attachments || [],
          }, {
            includeBody,
            includeHtml,
          });
          if ((!unreadOnly || message.unread) && matchesQuery(message, filters)) {
            messages.push(message);
          }
        }
        messages.sort((a, b) => Number(b.uid) - Number(a.uid));
        if (messages.length > limit) {
          messages.length = limit;
        }
      }
    } finally {
      lock.release();
    }
  } finally {
    await client.logout().catch(() => {});
  }

  const selected = messages
    .slice(0, limit)
    .sort((a, b) => Number(b.uid) - Number(a.uid));
  const nextCursor = selected.length > 0
    ? encodeCursor({ afterUidExclusive: selected[selected.length - 1].uid })
    : '';
  const oldestSelectedUid = selected.length > 0 ? Number(selected[selected.length - 1].uid || 0) : 0;
  const hasMore = oldestSelectedUid > 0
    ? candidateUids.some(item => item < oldestSelectedUid)
    : false;

  return {
    count: selected.length,
    account: config.account.email,
    mailbox,
    folder: mailbox,
    nextCursor,
    hasMore,
    messages: selected,
  };
}

async function sendMail(rawArgs) {
  const args = parseArgs(rawArgs);
  const config = await resolveProtocolConfig(await loadConfig(), 'smtp');
  const to = validateEmailList(args.to, 'to');
  const cc = validateEmailList(args.cc, 'cc');
  const bcc = validateEmailList(args.bcc, 'bcc');
  const subject = stringValue(args.subject);
  const bodyText = stringValue(args.bodyText || args.body);
  const bodyHtml = stringValue(args.bodyHtml);
  const replyToMessageId = stringValue(args.replyToMessageId);

  if (!subject) throw new Error('subject is required');
  if (!bodyText && !bodyHtml) throw new Error('bodyText or bodyHtml is required');

  const transporter = nodemailer.createTransport(smtpTransportOptions(config));
  await transporter.verify();

  const from = config.account.displayName
    ? `"${config.account.displayName.replace(/"/g, '\\"')}" <${config.account.email}>`
    : config.account.email;

  const info = await transporter.sendMail({
    from,
    to,
    cc: cc.length > 0 ? cc : undefined,
    bcc: bcc.length > 0 ? bcc : undefined,
    replyTo: config.account.replyTo || undefined,
    subject,
    text: bodyText || undefined,
    html: bodyHtml || undefined,
    headers: replyToMessageId ? { 'In-Reply-To': replyToMessageId, References: replyToMessageId } : undefined,
  });

  const sentRecord = {
    id: info.messageId || `smtp_${Date.now()}`,
    from,
    to,
    cc,
    bcc,
    subject,
    bodyText,
    bodyHtml,
    accepted: info.accepted || [],
    rejected: info.rejected || [],
    response: info.response || '',
    sentAt: new Date().toISOString(),
  };
  appendSentMailLog(sentRecord);

  return {
    ok: true,
    message: {
      id: sentRecord.id,
      from,
      to: to.join(', '),
      cc: cc.join(', '),
      subject,
      snippet: bodyText.slice(0, 280) || htmlToText(bodyHtml).slice(0, 280),
      receivedAt: sentRecord.sentAt,
      unread: false,
      sent: true,
      hasAttachments: false,
    },
    accepted: info.accepted || [],
    rejected: info.rejected || [],
    response: info.response || '',
    summary: `Sent email to ${to.join(', ')} with subject "${subject}".`,
  };
}

async function showMailList(rawArgs) {
  const args = parseArgs(rawArgs);
  const result = args.result && typeof args.result === 'object'
    ? args.result
    : {
        mailbox: stringValue(args.mailbox) || 'INBOX',
        folder: stringValue(args.folder) || stringValue(args.mailbox) || 'INBOX',
        count: Array.isArray(args.messages) ? args.messages.length : 0,
        hasMore: false,
        messages: Array.isArray(args.messages) ? args.messages : [],
      };

  return {
    viewId: 'mail_list',
    region: 'chat_side_panel',
    title: args.title || `Mail · ${result.folder || result.mailbox || 'INBOX'}`,
    data: {
      result,
    },
  };
}

const plugin = definePlugin({
  onInitialize(ctx) {
    pluginContext = ctx;
    ensureDataDir();
    configStore = ctx.storage.jsonStore('email-config.json', defaultConfig);
    sentMailStore = ctx.storage.jsonStore('sent-mail.json', []);
  },
  useTools: {
    search_mail: (args) => searchMail(args),
    send_mail: (args) => sendMail(args),
    list_folders: () => listFolders(),
    get_mail: (args) => getMail(args),
  },
  viewTools: {
    show_mail_list: (args) => showMailList(args),
  },
  settings: {
    get: () => loadConfig(),
    save: (config) => saveConfig(config),
    testConnection: (protocol, config) => testConnection(stringValue(protocol), config),
  },
  hooks: {
    beforeLLMSend: (messages) => messages || [],
    afterLLMSend: async () => ({ ok: true }),
  },
});

plugin.start();

process.on('SIGTERM', () => process.exit(0));
