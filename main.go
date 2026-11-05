package main

import (
	"context"
	"embed"
	_ "embed"
	"log"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/global_hotkey"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/window_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/migration"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/skills"
	pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/window_options"
	agentSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/agent"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/config"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file"
	memorySvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory"
	notificationSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/notification"
	onboardingSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/onboarding"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/onboarding/onboarding_dto"
	pluginSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/process"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider"
	runtimeSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/runtime"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/settings"
	skillsSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills"
	terminalSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/terminal"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window/window_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// 嵌入 Go 可执行文件的前端构建产物
//
//go:embed all:frontend/dist
var assets embed.FS

// 注册前端可订阅的自定义时间事件
func init() {
	application.RegisterEvent[string]("time")
}

// 初始化桌面应用并阻塞在 Wails 事件循环中
// 先打开存储、迁移旧版扩展目录，再按依赖顺序构造服务
// 将服务、快捷键和窗口注册到应用后启动事件循环
func main() {
	istorage, err := storage.NewStorage() // 存储接口、构建存储时的错误
	if err != nil {
		log.Fatal(err)
	}

	dataDir, derr := dir.GetDataDir() // 应用数据目录和读取错误
	if derr == nil {
		// 迁移旧版扩展目录并收集可继续启动的警告
		result, merr := migration.MigrateExtensionDirs(dataDir)
		if merr != nil {
			log.Printf("extension dir migration failed: %v", merr)
		} else if len(result.Warnings) > 0 {
			// 逐条记录迁移警告
			for _, w := range result.Warnings {
				log.Printf("extension dir migration warning: %s", w)
			}
		}
	} else {
		log.Printf("skip extension dir migration: cannot resolve data dir: %v", derr)
	}

	providerService := provider.NewProvider(istorage)                 // 提供商服务
	onboardingService := onboardingSvc.NewOnboarding(providerService) // 首次引导服务

	// 使用技能根目录创建技能管理器
	skillsManager := skills.NewManager(dir.SkillsRoot(dataDir))
	refreshErr := skillsManager.Refresh(nil) // 刷新技能列表时的错误
	if refreshErr != nil {
		// 技能是可选功能，初始化失败时记录警告并继续启动
		log.Printf("warning: failed to refresh skills: %v", refreshErr)
	}

	// 先创建插件服务，再创建 Agent 服务，以便接入插件的 CLI 桥接能力
	pluginService := pluginSvc.NewPlugin()
	memoryService := memorySvc.NewMemory(istorage)                         // 记忆服务
	notificationService := notificationSvc.NewNotification(istorage)       // 通知服务
	terminalManager := pkgterminal.NewManager(istorage, nil)               // 终端管理器
	terminalService := terminalSvc.NewTerminalWithManager(terminalManager) // 终端服务
	windowService := &window.Window{}                                      // 窗口服务
	// 快捷聊天快捷键、加载快捷聊天快捷键对应启动时的错误
	quickChatShortcut, shortcutErr := settings.LoadQuickChatShortcutForStartup()
	if shortcutErr != nil {
		log.Printf("load quick chat shortcut failed, using default: %v", shortcutErr)
		quickChatShortcut = global_hotkey.DefaultQuickChatShortcut()
	}
	// 快捷聊天服务
	quickChatService := global_hotkey.NewQuickChatServiceWithShortcut( /* 接到全局快捷键事件后切换快捷聊天窗口 */ func() {
		log.Printf("quick chat global hotkey pressed")
		// 切换快捷聊天时的错误
		_, err := windowService.ToggleQuickChat(context.Background(), window_dto.ToggleQuickChatInput{})
		if err != nil {
			log.Printf("quick chat toggle failed: %v", err)
		}
	}, quickChatShortcut)
	// 设置服务
	settingsService := settings.NewSettings(settings.Dependencies{QuickChatShortcuts: quickChatService})
	// 智能体服务
	agentService := agentSvc.NewAgent(istorage, agentSvc.Dependencies{
		SkillProvider:        skillsManager,
		AttentionRequester:   notificationService,
		SkillCreator:         skillsManager,
		CliInstaller:         pluginService,
		CliInstallProgress:   pluginService,
		CliManifestGenerator: pluginService,
		CliCommandRunner:     pluginService,
		TerminalRunner:       terminalManager,
	})

	// Wails 应用实例
	app := application.New(application.Options{
		Name:        window_id.Home,
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(settingsService),
			application.NewService(agentService),
			application.NewService(providerService),
			application.NewService(onboardingService),
			application.NewService(quickChatService),
			application.NewService(runtimeSvc.NewRuntime()),
			application.NewService(pluginService),
			application.NewService(terminalService),
			application.NewService(&process.Process{}),
			application.NewService(windowService),
			application.NewService(&file.File{}),
			application.NewService(&config.Config{}),
			application.NewService(skillsSvc.NewSkills(skillsManager)),
			application.NewService(memoryService),
			application.NewService(notificationService),
		},
		KeyBindings: map[string]func(window application.Window){
			quickChatShortcut: /* 执行当前步骤的回调，接收窗口，无返回值 */ func(window application.Window) {
				log.Printf("quick chat app-level shortcut pressed")
				// 切换快捷聊天时的错误
				_, err := windowService.ToggleQuickChat(context.Background(), window_dto.ToggleQuickChatInput{})
				if err != nil {
					log.Printf("quick chat toggle failed: %v", err)
				}
			},
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 初始化状态
	initState, err := onboardingService.IsInitialized(context.Background(), onboarding_dto.IsInitializedInput{})
	if err != nil {
		log.Fatal(err)
	}

	if initState != nil && initState.Initialized {
		win := app.Window.NewWithOptions(window_options.DefaultHome()) // 待操作的窗口
		win.OnWindowEvent(events.Common.WindowFilesDropped /* 处理窗口事件时调用的回调，接收事件，无返回值 */, func(event *application.WindowEvent) {
			application.Get().Event.Emit("files-dropped", map[string]any{
				"files": event.Context().DroppedFiles(),
			})
		})
	} else {
		app.Window.NewWithOptions(window_options.DefaultOnboarding())
	}
	quickChatOptions := window_options.DefaultQuickChat() // 快捷聊天选项列表
	quickChatOptions.Title = i18n.TCurrent("app.window.quick_chat_title", nil)
	app.Window.NewWithOptions(quickChatOptions)

	runErr := app.Run() // 运行 Wails 应用时的错误
	if runErr != nil {
		log.Fatal(runErr)
	}
}
