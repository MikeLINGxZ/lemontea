package ierror

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"

// 定义错误代码的类型
type errorCode string

// 取得用于展示的应用错误消息，返回文本
func (e errorCode) Msg() string {
	return i18n.TCurrent(string(e), nil)
}

const (
	// 智能体错误码
	ErrAgentSendMessage errorCode = "ierror.agent.send_message"
	// 智能体No确认对应的错误
	ErrAgentNoConfirm errorCode = "ierror.agent.no_confirm"
	// 智能体创建会话对应的错误
	ErrAgentCreateSession errorCode = "ierror.agent.create_session"
	// 智能体列表会话列表对应的错误
	ErrAgentListSessions errorCode = "ierror.agent.list_sessions"
	// 智能体列表消息列表对应的错误
	ErrAgentListMessages errorCode = "ierror.agent.list_messages"
	// 智能体数量消息列表对应的错误
	ErrAgentCountMessages errorCode = "ierror.agent.count_messages"
	// 智能体标记已读对应的错误
	ErrAgentMarkRead errorCode = "ierror.agent.mark_read"
	// 智能体重命名对应的错误
	ErrAgentRename errorCode = "ierror.agent.rename"
	// 智能体删除对应的错误
	ErrAgentDelete errorCode = "ierror.agent.delete"
	// 智能体切换收藏对应的错误
	ErrAgentToggleStar errorCode = "ierror.agent.toggle_star"
	// 智能体过多Many附件列表对应的错误
	ErrAgentTooManyAttachments errorCode = "ierror.agent.too_many_attachments"
	// 智能体附件路径对应的错误
	ErrAgentAttachmentPath errorCode = "ierror.agent.attachment_path_empty"
	// 智能体附件Not是否找到对应的错误
	ErrAgentAttachmentNotFound errorCode = "ierror.agent.attachment_not_found"
	// 智能体附件尺寸对应的错误
	ErrAgentAttachmentSize errorCode = "ierror.agent.attachment_too_large"
	// 智能体流式输出错误对应的错误
	ErrAgentStreamError errorCode = "ierror.agent.stream_error"
	// 通知创建对应的错误
	ErrNotificationCreate errorCode = "ierror.notification.create"
	// 通知列表对应的错误
	ErrNotificationList errorCode = "ierror.notification.list"
	// 通知解析对应的错误
	ErrNotificationResolve errorCode = "ierror.notification.resolve"
	// 通知关闭对应的错误
	ErrNotificationDismiss errorCode = "ierror.notification.dismiss"
	// 记忆列表对应的错误
	ErrMemoryList errorCode = "ierror.memory.list"
	// 记忆获取对应的错误
	ErrMemoryGet errorCode = "ierror.memory.get"
	// 记忆创建对应的错误
	ErrMemoryCreate errorCode = "ierror.memory.create"
	// 记忆更新对应的错误
	ErrMemoryUpdate errorCode = "ierror.memory.update"
	// 记忆遗忘对应的错误
	ErrMemoryForget errorCode = "ierror.memory.forget"
	// 记忆恢复对应的错误
	ErrMemoryRestore errorCode = "ierror.memory.restore"
	// 记忆统计对应的错误
	ErrMemoryStats errorCode = "ierror.memory.stats"
	// 记忆设置对应的错误
	ErrMemorySettings errorCode = "ierror.memory.settings"

	// 文件错误码
	ErrFileSelectFolder errorCode = "ierror.file.select_folder"
	// 文件选择文件对应的错误
	ErrFileSelectFile errorCode = "ierror.file.select_file"
	// 文件展开对应的错误
	ErrFileOpen errorCode = "ierror.file.open"
	// 文件保存临时文件对应的错误
	ErrFileSaveTempFile errorCode = "ierror.file.save_temp_file"

	// 设置错误码
	ErrSettingsLoadConfig errorCode = "ierror.settings.load_config"
	// 设置保存配置对应的错误
	ErrSettingsSaveConfig errorCode = "ierror.settings.save_config"
	// 设置创建目录对应的错误
	ErrSettingsCreateDir errorCode = "ierror.settings.create_dir"
	// 设置复制文件对应的错误
	ErrSettingsCopyFile errorCode = "ierror.settings.copy_file"
	// 设置已读配置对应的错误
	ErrSettingsReadConfig errorCode = "ierror.settings.read_config"
	// 设置解析配置对应的错误
	ErrSettingsParseConfig errorCode = "ierror.settings.parse_config"
	// 设置写入定位器对应的错误
	ErrSettingsWriteLocator errorCode = "ierror.settings.write_locator"
	// 设置目标目录对应的错误
	ErrSettingsTargetDir errorCode = "ierror.settings.target_dir_required"

	// 供应商错误码
	ErrProviderCreate errorCode = "ierror.provider.create"
	// 提供商创建模型列表对应的错误
	ErrProviderCreateModels errorCode = "ierror.provider.create_models"
	// 提供商列表提供商列表对应的错误
	ErrProviderListProviders errorCode = "ierror.provider.list_providers"
	// 提供商列表模型列表对应的错误
	ErrProviderListModels errorCode = "ierror.provider.list_models"
	// 提供商删除对应的错误
	ErrProviderDelete errorCode = "ierror.provider.delete"
	// 提供商更新对应的错误
	ErrProviderUpdate errorCode = "ierror.provider.update"
	// 提供商删除模型对应的错误
	ErrProviderDeleteModel errorCode = "ierror.provider.delete_model"
	// 提供商无效模型对应的错误
	ErrProviderInvalidModel errorCode = "ierror.provider.invalid_model_id"
	// 提供商集合默认对应的错误
	ErrProviderSetDefault errorCode = "ierror.provider.set_default"
	// 提供商拉取模型列表对应的错误
	ErrProviderFetchModels errorCode = "ierror.provider.fetch_model_list"

	// 首次启动引导错误码
	ErrOnboardingReadInit errorCode = "ierror.onboarding.read_init"
	// 首次引导写入初始化对应的错误
	ErrOnboardingWriteInit errorCode = "ierror.onboarding.write_init"
	// 首次引导完成对应的错误
	ErrOnboardingComplete errorCode = "ierror.onboarding.complete"

	// 运行时错误码
	ErrRuntimeUnsupportedOS errorCode = "ierror.runtime.unsupported_os"
	// 运行环境拉取校验和列表对应的错误
	ErrRuntimeFetchSums errorCode = "ierror.runtime.fetch_sums"
	// 运行环境校验和不匹配对应的错误
	ErrRuntimeChecksumMismatch errorCode = "ierror.runtime.checksum_mismatch"
	// 运行环境下载对应的错误
	ErrRuntimeDownload errorCode = "ierror.runtime.download"
	// 运行环境提取对应的错误
	ErrRuntimeExtract errorCode = "ierror.runtime.extract"
	// 运行环境写入状态对应的错误
	ErrRuntimeWriteState errorCode = "ierror.runtime.write_state"
	// 运行环境已读状态对应的错误
	ErrRuntimeReadState errorCode = "ierror.runtime.read_state"

	// CLI 插件错误码
	ErrCliRuntimeUnavailable errorCode = "ierror.cli.runtime_unavailable"
	// CLI安装失败对应的错误
	ErrCliInstallFailed errorCode = "ierror.cli.install_failed"
	// CLI重置数据失败对应的错误
	ErrCliResetDataFailed errorCode = "ierror.cli.reset_data_failed"
	// CLI清单无效对应的错误
	ErrCliManifestInvalid errorCode = "ierror.cli.manifest_invalid"
	// CLI清单保存失败对应的错误
	ErrCliManifestSaveFailed errorCode = "ierror.cli.manifest_save_failed"
	// CLI清单生成失败对应的错误
	ErrCliManifestGenerateFailed errorCode = "ierror.cli.manifest_generate_failed"
	// CLI工具Not是否找到对应的错误
	ErrCliToolNotFound errorCode = "ierror.cli.tool_not_found"

	// CLI 登录错误码
	ErrCliLoginSessionConflict errorCode = "cli.login.session_conflict"
	// CLI登录Not是否找到对应的错误
	ErrCliLoginNotFound errorCode = "cli.login.not_found"
	// CLI登录No命令对应的错误
	ErrCliLoginNoCommand errorCode = "cli.login.no_command"
	// CLI登录开始失败对应的错误
	ErrCliLoginStartFailed errorCode = "cli.login.start_failed"

	// 终端错误码
	ErrTerminalList errorCode = "ierror.terminal.list"
	// 终端已读输出对应的错误
	ErrTerminalReadOutput errorCode = "ierror.terminal.read_output"
	// 终端写入输入对应的错误
	ErrTerminalWriteInput errorCode = "ierror.terminal.write_input"
	// 终端调整尺寸对应的错误
	ErrTerminalResize errorCode = "ierror.terminal.resize"

	// 技能错误码
	ErrSkillsLoadFailed errorCode = "ierror.skills.load_failed"
	// 技能列表Not是否找到对应的错误
	ErrSkillsNotFound errorCode = "ierror.skills.not_found"
	// 技能列表无效名称对应的错误
	ErrSkillsInvalidName errorCode = "ierror.skills.invalid_name"
	// 技能列表无效内容对应的错误
	ErrSkillsInvalidContent errorCode = "ierror.skills.invalid_content"
	// 技能列表名称已占用对应的错误
	ErrSkillsNameTaken errorCode = "ierror.skills.name_taken"
	// 技能列表内置已加锁对应的错误
	ErrSkillsBuiltinLocked errorCode = "ierror.skills.builtin_locked"
	// 技能列表写入失败对应的错误
	ErrSkillsWriteFailed errorCode = "ierror.skills.write_failed"
	// 技能列表删除失败对应的错误
	ErrSkillsDeleteFailed errorCode = "ierror.skills.delete_failed"
)
