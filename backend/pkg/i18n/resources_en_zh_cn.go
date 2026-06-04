package i18n

var zhCN = map[string]string{
	"app.window.settings_title":                 "设置",
	"select.folder":                             "选择文件夹",
	"app.window.add_provider_title":             "添加提供商",
	"app.window.add_skill_title":                "新建 Skill",
	"app.window.add_memory_title":               "新建记忆",
	"app.window.quick_chat_title":               "快速对话",
	"provider.deepseek.name":                    "深度求索",
	"provider.deepseek.description":             "成立于2023年，专注于研究世界领先的通用人工智能底层模型与技术，挑战人工智能前沿性难题。",
	"provider.aliyun.name":                      "阿里云百炼",
	"provider.aliyun.description":               "一键部署大模型，支持多种模态的大模型调用服务。",
	"provider.openai_compatibility.name":        "OpenAI 兼容",
	"provider.openai_compatibility.description": "兼容 OpenAI 的模型提供商，可与 OpenAI API 无缝集成。",
	"provider.ollama.name":                      "Ollama",
	"provider.ollama.description":               "一个快速、开源的模型服务器。",
	"select.file":                               "选择文件",
	"ierror.unknown_error":                      "未知错误",

	// Agent
	"ierror.agent.send_message":          "发送消息失败",
	"ierror.agent.no_confirm":            "没有活跃的确认请求",
	"ierror.agent.create_session":        "创建会话失败",
	"ierror.agent.list_sessions":         "获取会话列表失败",
	"ierror.agent.list_messages":         "获取消息列表失败",
	"ierror.agent.count_messages":        "统计消息数量失败",
	"ierror.agent.mark_read":             "标记已读失败",
	"ierror.agent.rename":                "重命名会话失败",
	"ierror.agent.delete":                "删除会话失败",
	"ierror.agent.toggle_star":           "切换收藏失败",
	"ierror.agent.too_many_attachments":  "附件数量过多",
	"ierror.agent.attachment_path_empty": "附件路径为空",
	"ierror.agent.attachment_not_found":  "未找到附件",
	"ierror.agent.attachment_too_large":  "附件文件过大",
	"ierror.agent.stream_error":          "大模型响应出错",
	"ierror.notification.create":         "创建通知失败",
	"ierror.notification.list":           "获取通知列表失败",
	"ierror.notification.resolve":        "处理通知失败",
	"ierror.notification.dismiss":        "删除通知失败",
	"ierror.memory.list":                 "获取记忆列表失败",
	"ierror.memory.get":                  "获取记忆失败",
	"ierror.memory.create":               "创建记忆失败",
	"ierror.memory.update":               "更新记忆失败",
	"ierror.memory.forget":               "遗忘记忆失败",
	"ierror.memory.restore":              "恢复记忆失败",
	"ierror.memory.stats":                "获取记忆统计失败",
	"ierror.memory.settings":             "保存记忆设置失败",

	// File
	"ierror.file.select_folder":  "选择文件夹失败",
	"ierror.file.select_file":    "选择文件失败",
	"ierror.file.open":           "打开文件失败",
	"ierror.file.save_temp_file": "保存临时文件失败",

	// Settings
	"ierror.settings.load_config":         "加载配置失败",
	"ierror.settings.save_config":         "保存配置失败",
	"ierror.settings.create_dir":          "创建目录失败",
	"ierror.settings.copy_file":           "复制文件失败",
	"ierror.settings.read_config":         "读取配置文件失败",
	"ierror.settings.parse_config":        "解析配置失败",
	"ierror.settings.write_locator":       "写入数据目录定位符失败",
	"ierror.settings.target_dir_required": "目标数据目录不能为空",

	// Provider
	"ierror.provider.create":           "创建提供商失败",
	"ierror.provider.create_models":    "创建模型失败",
	"ierror.provider.list_providers":   "获取提供商列表失败",
	"ierror.provider.list_models":      "获取模型列表失败",
	"ierror.provider.delete":           "删除提供商失败",
	"ierror.provider.update":           "更新提供商失败",
	"ierror.provider.delete_model":     "删除模型失败",
	"ierror.provider.invalid_model_id": "无效的模型 ID",
	"ierror.provider.set_default":      "设置默认提供商失败",
	"ierror.provider.fetch_model_list": "获取模型列表失败",

	// Onboarding
	"ierror.onboarding.read_init":  "读取初始化状态失败",
	"ierror.onboarding.write_init": "写入初始化状态失败",
	"ierror.onboarding.complete":   "完成初始化失败",

	// Runtime
	"ierror.runtime.unsupported_os":    "不支持的操作系统或架构",
	"ierror.runtime.fetch_sums":        "获取校验文件失败",
	"ierror.runtime.checksum_mismatch": "运行时安装包校验失败",
	"ierror.runtime.download":          "下载运行时失败",
	"ierror.runtime.extract":           "解压运行时失败",
	"ierror.runtime.write_state":       "写入运行时状态失败",
	"ierror.runtime.read_state":        "读取运行时状态失败",

	// CLI
	"ierror.cli.runtime_unavailable":      "插件运行时尚未就绪，请先完成 onboarding",
	"ierror.cli.install_failed":           "CLI 安装失败",
	"ierror.cli.reset_data_failed":        "重置 CLI 数据失败",
	"ierror.cli.manifest_invalid":         "Manifest 无效",
	"ierror.cli.manifest_save_failed":     "保存 manifest 失败",
	"ierror.cli.manifest_generate_failed": "生成 manifest 失败",
	"ierror.cli.tool_not_found":           "未找到 CLI 工具",

	// CLI login
	"cli.login.session_conflict": "该 CLI 已有进行中的登录会话",
	"cli.login.not_found":        "找不到对应的 CLI 登录会话",
	"cli.login.no_command":       "该 CLI 的 manifest 未声明 login_command",
	"cli.login.start_failed":     "启动 CLI 登录失败：{detail}",

	// Terminal
	"ierror.terminal.list":        "获取终端列表失败",
	"ierror.terminal.read_output": "读取终端输出失败",
	"ierror.terminal.write_input": "写入终端输入失败",
	"ierror.terminal.resize":      "调整终端大小失败",

	// Skills
	"ierror.skills.load_failed":     "加载 skill 失败",
	"ierror.skills.not_found":       "找不到 skill",
	"ierror.skills.invalid_name":    "skill 名称不合法",
	"ierror.skills.invalid_content": "skill 内容不合法",
	"ierror.skills.name_taken":      "skill 名称已被占用",
	"ierror.skills.builtin_locked":  "内置 skill 不可修改",
	"ierror.skills.write_failed":    "保存 skill 失败",
	"ierror.skills.delete_failed":   "删除 skill 失败",
}
