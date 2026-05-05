# Email Plugin

Email is a Lemon Tea `general_plugin` for real mailbox workflows over standard `IMAP + SMTP`.

It provides:

- Secure host-managed credentials for IMAP and SMTP passwords
- A host-rendered settings page with provider presets and connection tests
- `list_folders` to inspect available mailboxes
- `search_mail` with folder, sender, recipient, subject, unread, date-range, and cursor-based pagination controls
- `get_mail` for message detail, headers, and attachment metadata
- `send_mail` for text or HTML email delivery with multi-recipient support
- `show_mail_list` for rendering mailbox results in the chat side panel

## Install

1. Make sure Lemon Tea can find a Node.js runtime at:

   ```text
   <GetDataPath>/plugin_runtime/bin/node
   ```

   or one of the fallback runtime paths supported by the host.

2. Open `Settings -> Plugins`.
3. Click `Add`.
4. Select this folder:

   ```text
   examples/email_plugin
   ```

5. Enable the plugin.

## Supported Account Mode

- Username + password IMAP
- Username + password SMTP
- Common provider presets: Gmail, Outlook / Office 365, QQ Mail, 163 Mail
- Custom server configuration

This version does not include OAuth, Gmail API, Exchange Web Services, or Microsoft Graph integration.

## Security Model

- Email passwords are not stored in `email-config.json`
- The plugin requests credentials from the Lemon Tea host through the plugin RPC bridge
- The host persists credentials in its own encrypted credential store
- The config file only keeps non-sensitive server settings and test status metadata

## Configure

Open the plugin detail page and click `Settings`.

The host-rendered settings page supports:

- Account identity fields
- Provider presets
- Separate IMAP and SMTP host, port, security mode, and username
- Save / delete credential actions
- `Test IMAP connection`
- `Test SMTP connection`
- `Save and test`

## Tool Examples

List folders:

```text
List my mail folders and unread counts.
```

Search mailbox results and show them in the side panel:

```text
Search unread email in INBOX about project launch, then show the mail list.
```

Read one message in detail:

```text
Get the full detail for the latest matching email from INBOX.
```

Send a message:

```text
Send an email to alice@example.com and bob@example.com with subject "Launch update" and a short HTML summary.
```

## Development

Source lives in:

```text
src/main.js
```

`dist/main.js` is a small runtime bootstrap that loads the source entrypoint. If you need to refresh the bootstrap file, run:

```text
npm run build
```

## Known Limitations

- No OAuth or provider-specific APIs yet
- No attachment upload or download workflow yet
- Message search still focuses on recent mailbox windows instead of a full indexed search backend
- Mailbox rendering is currently optimized for the Lemon Tea chat side panel rather than a standalone inbox UI
