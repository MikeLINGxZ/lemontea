package onboarding

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/window_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/window_options"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/onboarding/onboarding_dto"
	providerSvc "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto"
)

// Onboarding 管理应用首次启动时的初始化流程
type Onboarding struct {
	// Wails 应用实例
	wailsApp *application.App
	// 提供商服务
	providerSvc *providerSvc.Provider
}

// NewOnboarding 使用提供商服务创建首次启动引导服务
// 接收提供商，返回首次引导
func NewOnboarding(p *providerSvc.Provider) *Onboarding {
	return &Onboarding{providerSvc: p}
}

// IsInitialized 返回应用是否已完成首次启动引导
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Onboarding) IsInitialized(ctx context.Context, input onboarding_dto.IsInitializedInput) (*onboarding_dto.IsInitializedOutput, error) {
	initialized, err := isInitialized() // 初始化完成状态和读取错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrOnboardingReadInit, err)
	}
	return &onboarding_dto.IsInitializedOutput{Initialized: initialized}, nil
}

// SaveProviderAndMarkInitialized 创建首个提供商并写入 init.json 标记文件
// 仅在创建成功后写入标记，创建失败时返回错误
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Onboarding) SaveProviderAndMarkInitialized(ctx context.Context, input onboarding_dto.SaveProviderAndMarkInitializedInput) (*onboarding_dto.SaveProviderAndMarkInitializedOutput, error) {
	// 创建模型提供商时的错误
	_, createErr := s.providerSvc.CreateProvider(ctx, provider_dto.CreateProviderInput{
		ProviderName: input.ProviderName,
		ProviderType: input.ProviderType,
		BaseUrl:      input.BaseUrl,
		ApiKey:       input.ApiKey,
		Enable:       input.Enable,
		DefaultModel: input.DefaultModel,
		Models:       input.Models,
	})
	if createErr != nil {
		return nil, createErr
	}

	markInitializedErr := markInitialized() // 写入首次启动完成标记时的错误
	if markInitializedErr != nil {
		return nil, ierror.Error(ierror.ErrOnboardingWriteInit, markInitializedErr)
	}

	return &onboarding_dto.SaveProviderAndMarkInitializedOutput{}, nil
}

// ExitApp 退出 Wails 应用并终止应用进程
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Onboarding) ExitApp(ctx context.Context, input onboarding_dto.ExitAppInput) (*onboarding_dto.ExitAppOutput, error) {
	if s.wailsApp != nil {
		s.wailsApp.Quit()
	}
	return &onboarding_dto.ExitAppOutput{}, nil
}

// EnterHome 打开主窗口并关闭引导窗口
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Onboarding) EnterHome(ctx context.Context, input onboarding_dto.EnterHomeInput) (*onboarding_dto.EnterHomeOutput, error) {
	if s.wailsApp == nil {
		return &onboarding_dto.EnterHomeOutput{}, nil
	}

	homeWindow, ok := s.wailsApp.Window.GetByName(window_id.Home) // 主界面窗口、是否成功取得对应值
	if !ok {
		homeWindow = s.wailsApp.Window.NewWithOptions(window_options.DefaultHome())
	}
	homeWindow.Show()
	homeWindow.Focus()

	if /* 引导窗口及其是否存在 */ onboardWin, ok := s.wailsApp.Window.GetByName(window_id.Onboarding); ok {
		onboardWin.Close()
	}

	return &onboarding_dto.EnterHomeOutput{}, nil
}
