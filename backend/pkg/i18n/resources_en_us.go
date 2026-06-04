package i18n

var enUS = map[string]string{
	"app.window.settings_title":                 "Settings",
	"select.folder":                             "Select Folder",
	"app.window.add_provider_title":             "Add Provider",
	"app.window.add_skill_title":                "New Skill",
	"app.window.add_memory_title":               "New Memory",
	"app.window.quick_chat_title":               "Quick Chat",
	"provider.deepseek.name":                    "DeepSeek",
	"provider.deepseek.description":             "Founded in 2023, focused on world-class general AI foundation models and frontier research.",
	"provider.aliyun.name":                      "Alibaba Cloud Bailian",
	"provider.aliyun.description":               "Deploy large models with one click and access multimodal model APIs.",
	"provider.openai_compatibility.name":        "OpenAI Compatibility",
	"provider.openai_compatibility.description": "OpenAI-compatible model provider for seamless integration with OpenAI APIs.",
	"provider.ollama.name":                      "Ollama",
	"provider.ollama.description":               "A fast, open-source model server.",
	"select.file":                               "Select File",
	"ierror.unknown_error":                      "Unknown error occurred",

	// Agent
	"ierror.agent.send_message":          "Failed to send message",
	"ierror.agent.no_confirm":            "No active confirmation for session",
	"ierror.agent.create_session":        "Failed to create session",
	"ierror.agent.list_sessions":         "Failed to list sessions",
	"ierror.agent.list_messages":         "Failed to list messages",
	"ierror.agent.count_messages":        "Failed to count messages",
	"ierror.agent.mark_read":             "Failed to mark session as read",
	"ierror.agent.rename":                "Failed to rename session",
	"ierror.agent.delete":                "Failed to delete session",
	"ierror.agent.toggle_star":           "Failed to toggle star",
	"ierror.agent.too_many_attachments":  "Too many attachments",
	"ierror.agent.attachment_path_empty": "Attachment path is empty",
	"ierror.agent.attachment_not_found":  "Attachment not found",
	"ierror.agent.attachment_too_large":  "Attachment file too large",
	"ierror.agent.stream_error":          "Model response error",
	"ierror.notification.create":         "Failed to create notification",
	"ierror.notification.list":           "Failed to list notifications",
	"ierror.notification.resolve":        "Failed to resolve notification",
	"ierror.notification.dismiss":        "Failed to dismiss notification",
	"ierror.memory.list":                 "Failed to list memories",
	"ierror.memory.get":                  "Failed to get memory",
	"ierror.memory.create":               "Failed to create memory",
	"ierror.memory.update":               "Failed to update memory",
	"ierror.memory.forget":               "Failed to forget memory",
	"ierror.memory.restore":              "Failed to restore memory",
	"ierror.memory.stats":                "Failed to load memory stats",
	"ierror.memory.settings":             "Failed to save memory settings",

	// File
	"ierror.file.select_folder":  "Failed to select folder",
	"ierror.file.select_file":    "Failed to select file",
	"ierror.file.open":           "Failed to open file",
	"ierror.file.save_temp_file": "Failed to save temporary file",

	// Settings
	"ierror.settings.load_config":         "Failed to load configuration",
	"ierror.settings.save_config":         "Failed to save configuration",
	"ierror.settings.create_dir":          "Failed to create directory",
	"ierror.settings.copy_file":           "Failed to copy file",
	"ierror.settings.read_config":         "Failed to read configuration file",
	"ierror.settings.parse_config":        "Failed to parse configuration",
	"ierror.settings.write_locator":       "Failed to write data directory locator",
	"ierror.settings.target_dir_required": "Target data directory is required",

	// Provider
	"ierror.provider.create":           "Failed to create provider",
	"ierror.provider.create_models":    "Failed to create models",
	"ierror.provider.list_providers":   "Failed to list providers",
	"ierror.provider.list_models":      "Failed to list models",
	"ierror.provider.delete":           "Failed to delete provider",
	"ierror.provider.update":           "Failed to update provider",
	"ierror.provider.delete_model":     "Failed to delete model",
	"ierror.provider.invalid_model_id": "Invalid model ID",
	"ierror.provider.set_default":      "Failed to set default provider",
	"ierror.provider.fetch_model_list": "Failed to fetch model list",

	// Onboarding
	"ierror.onboarding.read_init":  "Failed to read initialization state",
	"ierror.onboarding.write_init": "Failed to write initialization state",
	"ierror.onboarding.complete":   "Failed to complete initialization",

	// Runtime
	"ierror.runtime.unsupported_os":    "Unsupported operating system or architecture",
	"ierror.runtime.fetch_sums":        "Failed to fetch checksums",
	"ierror.runtime.checksum_mismatch": "Runtime archive checksum mismatch",
	"ierror.runtime.download":          "Failed to download runtime",
	"ierror.runtime.extract":           "Failed to extract runtime",
	"ierror.runtime.write_state":       "Failed to write runtime state",
	"ierror.runtime.read_state":        "Failed to read runtime state",

	// CLI
	"ierror.cli.runtime_unavailable":      "Plugin runtime not ready; finish onboarding first",
	"ierror.cli.install_failed":           "CLI install failed",
	"ierror.cli.reset_data_failed":        "Reset CLI data failed",
	"ierror.cli.manifest_invalid":         "Manifest is invalid",
	"ierror.cli.manifest_save_failed":     "Save manifest failed",
	"ierror.cli.manifest_generate_failed": "Generate manifest failed",
	"ierror.cli.tool_not_found":           "CLI tool not found",

	// CLI login
	"cli.login.session_conflict": "A login session is already running for this CLI.",
	"cli.login.not_found":        "No active login session for this CLI.",
	"cli.login.no_command":       "This CLI's manifest has no login_command.",
	"cli.login.start_failed":     "Failed to start CLI login: {detail}",

	// Terminal
	"ierror.terminal.list":        "Failed to list terminals",
	"ierror.terminal.read_output": "Failed to read terminal output",
	"ierror.terminal.write_input": "Failed to write terminal input",
	"ierror.terminal.resize":      "Failed to resize terminal",

	// Skills
	"ierror.skills.load_failed":     "Failed to load skill",
	"ierror.skills.not_found":       "Skill not found",
	"ierror.skills.invalid_name":    "Invalid skill name",
	"ierror.skills.invalid_content": "Invalid skill content",
	"ierror.skills.name_taken":      "Skill name already in use",
	"ierror.skills.builtin_locked":  "Built-in skill is read-only",
	"ierror.skills.write_failed":    "Failed to save skill",
	"ierror.skills.delete_failed":   "Failed to delete skill",
}
