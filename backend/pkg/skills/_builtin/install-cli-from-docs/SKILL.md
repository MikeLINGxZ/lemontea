---
name: install-cli-from-docs
description: Install a CLI plugin from official documentation. Use when the user provides a documentation URL, pasted documentation text, or a natural-language description of a CLI tool they want to integrate. Handles package discovery, npm install, manifest generation, and post-install initialization orchestration.
---

# Install CLI from Official Documentation

You are installing a CLI plugin based on user-provided documentation or description.

Your job is not finished when the package is installed. You should continue until the CLI is initialized and meaningfully usable, unless you are blocked on user action or missing information.

## Input

The user's message contains one or more of:
- A documentation URL (e.g. `https://github.com/foo/bar-cli#readme`)
- Pasted documentation text (Markdown or plain text)
- A natural-language description of the CLI tool

## Steps

1. **Identify the npm package name.**
   - If the user provided a URL, fetch it (via WebSearch or WebFetch) and extract the npm package name from the documentation.
   - If the user pasted documentation, look for `npm install <package>` commands, `npx <package>` references, or a "Package" / "Installation" section.
   - If the user gave a natural-language description, search for the most popular npm package matching that description.
   - Common patterns: `@anthropic-ai/claude-code`, `@modelcontextprotocol/server-filesystem`, `lark-cli`, etc.

2. **If you cannot determine the package name with confidence, call `RequestUserAttention`.**
   - `title`: "需要确认 npm 包名"
   - `message`: Explain what you found and ask the user to confirm or provide the exact package name.
   - Wait for the user's reply before proceeding.

3. **Call `InstallCli` with the package name.**
   ```json
   {"npm_package": "<determined-package-name>", "name": "<short-cli-name>"}
   ```
   - `name` should be a short kebab-case identifier derived from the package name (e.g. `@scope/foo-cli` → `foo-cli`).
   - `InstallCli` installs the package, creates usable tools, and reports those stages automatically.
   - Keep the returned extension ID for setup and verification.

4. **Decide whether post-install initialization is needed.**
   - Read the generated manifest and the documentation you already have.
   - Decide whether the CLI still needs setup steps such as auth, config initialization, API key entry, profile creation, or a verification command.
   - Prefer dynamic judgment from the docs + CLI output. Do not assume all CLIs follow the same fixed flow.

5. **If initialization is needed, continue the workflow using `RunCliCommand`.**
   - Use `RunCliCommand` to run setup/auth commands against the installed CLI with Lemontea's bundled runtime and isolated env.
   - Use `ReportCliInstallProgress` with the returned extension ID to report `initializing`, `waiting_auth`, `verifying`, or `failed`. The installation tool already reports download and tool preparation.
   - Prefer structured outputs when possible:
     - use `output_mode: "json"` when the CLI offers JSON
     - otherwise use `text` or `lines`
   - Re-check CLI output after each step and decide the next step dynamically.
   - When the flow may need to resume in a later turn, persist state with `SaveTaskState` and recover it with `LoadTaskState`.

6. **For device-code or browser-based login flows:**
   - Prefer a non-blocking structured command first when the CLI supports it.
   - For `lark-cli`, prefer:
     - `["auth","login","--no-wait","--json"]`
   - Extract structured fields such as:
     - `verification_url`
     - `device_code`
     - expiry / ttl if available
   - Show the original `verification_url` to the user unchanged.
   - Also send `ReportCliInstallProgress` with phase `waiting_auth` and put the unchanged URL into `action_url`.
   - If the product can generate a QR code image from the URL, use that capability and place the QR code under the URL.
   - Save any temporary state needed to resume later; do not request a fresh device code unless the original one expired.
   - Prefer storing temporary state keys such as `device_code`, `verification_url`, and `auth_expires_at` with `SaveTaskState`.
   - Do not immediately start blocking polling in the same turn after showing the URL if the user still needs to authorize in the browser.

7. **After the user completes external authorization, resume with the saved state.**
   - Reuse the original device code or saved context.
   - Load saved values with `LoadTaskState` instead of assuming they are still present in model context.
   - Report progress with phase `verifying` before resuming the polling or verification command.
   - For `lark-cli`, continue with:
     - `["auth","login","--device-code","<saved_device_code>"]`
   - Then run a lightweight verification step if the CLI provides one.

8. **Report success only when the CLI is usable.**
   - Send a final `ReportCliInstallProgress` update with phase `done` only after successful verification.
   - Summarize:
     - what was installed
     - whether initialization/auth completed
     - any remaining manual action, if there is one

## Error Handling

- If `InstallCli` fails (network error, package not found), report the error clearly and suggest alternatives.
- If initialization is partially complete but blocked on user action, present the next required user action clearly and preserve resume state.
- If a device code or login session expires, explain that it expired and restart the authorization flow rather than hanging indefinitely.
- If the user's input is completely empty or nonsensical, call `RequestUserAttention` asking for valid documentation.

## Constraints

- Do NOT install packages that are not published on npm.
- Do NOT modify the CLI's source code.
- Do NOT hard-code a permanent product-side special case for one CLI; rely on docs, manifest, and runtime output to decide the flow.
- Prefer `RunCliCommand` over generic shell execution for installed CLI setup steps, so the bundled runtime and isolated env are preserved.
- Treat `requires_confirm`-style manifest hints as guidance, not the only source of truth for risk.
