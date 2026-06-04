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

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[string]("time")
}

func main() {
	istorage, err := storage.NewStorage()
	if err != nil {
		log.Fatal(err)
	}

	dataDir, derr := dir.GetDataDir()
	if derr == nil {
		if result, merr := migration.MigrateExtensionDirs(dataDir); merr != nil {
			log.Printf("extension dir migration failed: %v", merr)
		} else if len(result.Warnings) > 0 {
			for _, w := range result.Warnings {
				log.Printf("extension dir migration warning: %s", w)
			}
		}
	} else {
		log.Printf("skip extension dir migration: cannot resolve data dir: %v", derr)
	}

	providerService := provider.NewProvider(istorage)
	onboardingService := onboardingSvc.NewOnboarding(providerService)

	// Create skills manager (needs the skills root directory).
	skillsManager := skills.NewManager(dir.SkillsRoot(dataDir))
	if err := skillsManager.Refresh(nil); err != nil {
		// Log warning but don't fail startup — skills are optional.
		log.Printf("warning: failed to refresh skills: %v", err)
	}

	// Create the plugin and agent services — plugin must come first so its CLI bridge can be wired in.
	pluginService := pluginSvc.NewPlugin()
	memoryService := memorySvc.NewMemory(istorage)
	notificationService := notificationSvc.NewNotification(istorage)
	terminalManager := pkgterminal.NewManager(istorage, nil)
	terminalService := terminalSvc.NewTerminalWithManager(terminalManager)
	windowService := &window.Window{}
	quickChatShortcut, shortcutErr := settings.LoadQuickChatShortcutForStartup()
	if shortcutErr != nil {
		log.Printf("load quick chat shortcut failed, using default: %v", shortcutErr)
		quickChatShortcut = global_hotkey.DefaultQuickChatShortcut()
	}
	quickChatService := global_hotkey.NewQuickChatServiceWithShortcut(func() {
		log.Printf("quick chat global hotkey pressed")
		if _, err := windowService.ToggleQuickChat(context.Background(), window_dto.ToggleQuickChatInput{}); err != nil {
			log.Printf("quick chat toggle failed: %v", err)
		}
	}, quickChatShortcut)
	settingsService := settings.NewSettings(settings.Dependencies{QuickChatShortcuts: quickChatService})
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
			quickChatShortcut: func(window application.Window) {
				log.Printf("quick chat app-level shortcut pressed")
				if _, err := windowService.ToggleQuickChat(context.Background(), window_dto.ToggleQuickChatInput{}); err != nil {
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

	initState, err := onboardingService.IsInitialized(context.Background(), onboarding_dto.IsInitializedInput{})
	if err != nil {
		log.Fatal(err)
	}

	if initState != nil && initState.Initialized {
		win := app.Window.NewWithOptions(window_options.DefaultHome())
		win.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
			application.Get().Event.Emit("files-dropped", map[string]any{
				"files": event.Context().DroppedFiles(),
			})
		})
	} else {
		app.Window.NewWithOptions(window_options.DefaultOnboarding())
	}
	quickChatOptions := window_options.DefaultQuickChat()
	quickChatOptions.Title = i18n.TCurrent("app.window.quick_chat_title", nil)
	app.Window.NewWithOptions(quickChatOptions)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
