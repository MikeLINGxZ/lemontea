package service

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/agents/memory/lifecycle"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/agents/memory/search"
	memory_storage "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/agents/memory/storage"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/llm_provider/agents"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/logger"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/plugins"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/prompts"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

const (
	WindowNameHome         = "window_home"
	WindowNameOnboarding   = "window_onboarding"
	WindowNameSettings     = "window_settings"
	WindowNameFormProvider = "window_form_provider"
	WindowNameFormAgent    = "window_form_agent"
	WindowNameFormSkill    = "window_form_skill"
	WindowNameFormMemory   = "window_form_memory"
)

const (
	EventSettingsProvidersChanged = "settings:providers:changed"
	EventSettingsAgentsChanged    = "settings:agents:changed"
	EventSettingsSkillsChanged    = "settings:skills:changed"
	EventSettingsMemoriesChanged  = "settings:memories:changed"
)

type Service struct {
	storage         *storage.Storage
	app             *application.App
	prompts         prompts.PromptSet
	memoryStorage   *memory_storage.Storage
	memoryCache     *memoryPrefetchCache
	memoryLifecycle *lifecycle.Manager
	memorySearcher  *search.HybridSearcher
	plugins         *plugins.Manager
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {

	istorage, err := storage.NewStorage()
	if err != nil {
		return err
	}

	s.storage = istorage
	s.app = application.Get()
	if pluginManager, pluginErr := plugins.NewManager(); pluginErr != nil {
		logger.Warm("plugin manager init failed:", pluginErr)
	} else {
		s.plugins = pluginManager
	}

	// 初始化记忆系统存储（复用主数据库连接）
	memStorage, memErr := memory_storage.NewStorage(istorage.DB())
	if memErr != nil {
		logger.Warm("memory storage init failed:", memErr)
	} else {
		s.memoryStorage = memStorage
		// 初始化嵌入表
		if embErr := memStorage.AutoMigrateEmbeddings(); embErr != nil {
			logger.Warm("memory embeddings migration failed:", embErr)
		}
		// 一次性迁移历史记忆：把旧结构化字段拼入 content，并映射旧类型
		if migErr := memStorage.MigrateLegacyFieldsToContent(ctx); migErr != nil {
			logger.Warm("memory legacy-fields migration failed:", migErr)
		}
		if migErr := memStorage.MigrateCoreMemoryMetadata(ctx); migErr != nil {
			logger.Warm("memory core metadata migration failed:", migErr)
		}
		// 创建混合检索引擎（默认无 embedder）
		s.memorySearcher = search.NewHybridSearcher(memStorage, nil, "")
		// 启动记忆生命周期管理（巩固/遗忘/矛盾检测）
		s.memoryLifecycle = lifecycle.NewManager(memStorage)
		s.memoryLifecycle.Start()
	}
	s.memoryCache = newMemoryPrefetchCache()
	if prefs, prefsErr := s.loadAppPreferences(ctx); prefsErr == nil {
		i18n.SetCurrentLocale(string(prefs.Language))
		if s.memoryStorage != nil && prefs.MemorySystemEnabled && prefs.VectorSearchEnabled {
			if err := s.configureEmbeddingInternal(ctx, search.EmbeddingConfig{
				Provider: search.EmbeddingProvider(prefs.EmbeddingProvider),
				BaseURL:  prefs.EmbeddingBaseURL,
				APIKey:   prefs.EmbeddingAPIKey,
				Model:    prefs.EmbeddingModel,
			}, true); err != nil {
				logger.Warm("restore embedding config failed, falling back to FTS5/LIKE:", err)
				s.DisableEmbedding()
			}
		}
	}
	if err := s.reloadPromptSet(); err != nil {
		logger.Warm("load prompt set fallback:", err)
	}

	agents.SyncCustomAgentsToRegistry()

	if err := s.syncCustomMCPTools(ctx); err != nil {
		return err
	}
	if s.plugins != nil {
		s.plugins.StartEnabled(ctx)
	}

	if err := s.recoverStaleRunningTasks(ctx); err != nil {
		return err
	}

	return nil
}
