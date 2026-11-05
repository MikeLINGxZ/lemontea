package window

import (
	"context"
	"fmt"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/event_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/id/window_id"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/window_options"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window/window_dto"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Window 管理应用窗口
type Window struct {
	// Wails 应用实例
	wailsApp *application.App
}

// OpenSettings 打开设置窗口
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) OpenSettings(ctx context.Context, input window_dto.OpenSettingsInput) (*window_dto.OpenSettingsOutput, error) {
	tab := "settings" // 标签页
	if input.Tab != "" {
		tab = input.Tab
	}
	pageUrl := fmt.Sprintf("/?entry=settings&tab=%s", tab) // 页面地址

	settingsWindow, ok := p.wailsApp.Window.GetByName(window_id.Settings) // 设置窗口、是否成功取得对应值
	if ok {
		settingsWindow.SetURL(pageUrl)
	} else {
		settingsWindow = p.wailsApp.Window.NewWithOptions(p.childWindowOptions(application.WebviewWindowOptions{
			Name:   window_id.Settings,
			Title:  i18n.TCurrent("app.window.settings_title", nil),
			Hidden: true,
			Mac: application.MacWindow{
				// 设置窗口的拖动和双击由前端标题栏声明
				Backdrop: application.MacBackdropTranslucent,
				TitleBar: application.MacTitleBarHiddenInsetUnified,
			},
			BackgroundColour: application.NewRGB(27, 38, 54),
			URL:              pageUrl,
			Width:            1200,
			Height:           800,
			MinWidth:         550,
			MinHeight:        550,
		}))
	}

	// 已最大化或全屏时保持系统管理的窗口位置
	if !settingsWindow.IsMaximised() && !settingsWindow.IsFullscreen() {
		home, hasHome := p.wailsApp.Window.GetByName(window_id.Home) // 主窗口及是否存在
		if !hasHome || !home.IsVisible() || home.IsMinimised() {
			p.showCenteredOnHomeScreen(settingsWindow)
			return &window_dto.OpenSettingsOutput{}, nil
		}
		homeBounds := home.Bounds()                      // 主窗口的桌面坐标和尺寸
		width, height := settingsWindow.Size()           // 设置窗口当前的宽高
		x := homeBounds.X + (homeBounds.Width-width)/2   // 相对主窗口居中的横坐标
		y := homeBounds.Y + (homeBounds.Height-height)/2 // 相对主窗口居中的纵坐标

		// 主窗口靠近屏幕边缘时，将设置窗口限制在屏幕工作区内
		screen := p.homeWindowScreen() // 主窗口所在屏幕
		if screen != nil && screen.WorkArea.Width > 0 && screen.WorkArea.Height > 0 {
			area := screen.WorkArea // 扣除系统菜单栏和任务栏后的可用区域
			x = max(area.X, min(x, area.X+area.Width-width))
			y = max(area.Y, min(y, area.Y+area.Height-height))
		}
		settingsWindow.SetPosition(x, y)
	}
	settingsWindow.Show()
	settingsWindow.Focus()
	return &window_dto.OpenSettingsOutput{}, nil
}

// OpenAddProvider 打开添加提供商窗口
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) OpenAddProvider(ctx context.Context, input window_dto.OpenAddProviderInput) (*window_dto.OpenAddProviderOutput, error) {

	addProviderWindow, ok := p.wailsApp.Window.GetByName(window_id.AddProvider) // 增加提供商窗口、是否成功取得对应值
	if ok {
		p.showCenteredOnHomeScreen(addProviderWindow)
		return &window_dto.OpenAddProviderOutput{}, nil
	}
	addProviderWindow = p.wailsApp.Window.NewWithOptions(p.childWindowOptions(application.WebviewWindowOptions{
		Name:  window_id.AddProvider,
		Title: i18n.TCurrent("app.window.add_provider_title", nil),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 48,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/?entry=add_provider",
		Width:            750,
		Height:           800,
		MinWidth:         550,
		MinHeight:        550,
	}))
	p.showCenteredOnHomeScreen(addProviderWindow)
	return &window_dto.OpenAddProviderOutput{}, nil
}

// CloseAddProvider 关闭添加提供商窗口
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) CloseAddProvider(ctx context.Context, input window_dto.CloseAddProviderInput) (*window_dto.CloseAddProviderOutput, error) {
	addProviderWindow, ok := p.wailsApp.Window.GetByName(window_id.AddProvider) // 增加提供商窗口、是否成功取得对应值
	if ok {
		addProviderWindow.Close()
		return &window_dto.CloseAddProviderOutput{}, nil
	}
	return &window_dto.CloseAddProviderOutput{}, nil
}

// OpenAddSkill 打开添加技能窗口，窗口已存在时将其聚焦
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) OpenAddSkill(ctx context.Context, input window_dto.OpenAddSkillInput) (*window_dto.OpenAddSkillOutput, error) {
	addSkillWindow, ok := p.wailsApp.Window.GetByName(window_id.AddSkill) // 增加技能窗口、是否成功取得对应值
	if ok {
		p.showCenteredOnHomeScreen(addSkillWindow)
		return &window_dto.OpenAddSkillOutput{}, nil
	}

	addSkillWindow = p.wailsApp.Window.NewWithOptions(p.childWindowOptions(application.WebviewWindowOptions{
		Name:  window_id.AddSkill,
		Title: i18n.TCurrent("app.window.add_skill_title", nil),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 48,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/?entry=add_skill",
		Width:            900,
		Height:           820,
		MinWidth:         700,
		MinHeight:        620,
	}))
	p.showCenteredOnHomeScreen(addSkillWindow)
	return &window_dto.OpenAddSkillOutput{}, nil
}

// CloseAddSkill 关闭添加技能窗口
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) CloseAddSkill(ctx context.Context, input window_dto.CloseAddSkillInput) (*window_dto.CloseAddSkillOutput, error) {
	addSkillWindow, ok := p.wailsApp.Window.GetByName(window_id.AddSkill) // 增加技能窗口、是否成功取得对应值
	if ok {
		addSkillWindow.Close()
		return &window_dto.CloseAddSkillOutput{}, nil
	}
	return &window_dto.CloseAddSkillOutput{}, nil
}

// OpenAddMemory 打开添加记忆窗口，窗口已存在时将其聚焦
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) OpenAddMemory(ctx context.Context, input window_dto.OpenAddMemoryInput) (*window_dto.OpenAddMemoryOutput, error) {
	addMemoryWindow, ok := p.wailsApp.Window.GetByName(window_id.AddMemory) // 增加记忆窗口、是否成功取得对应值
	if ok {
		p.showCenteredOnHomeScreen(addMemoryWindow)
		return &window_dto.OpenAddMemoryOutput{}, nil
	}

	addMemoryWindow = p.wailsApp.Window.NewWithOptions(p.childWindowOptions(application.WebviewWindowOptions{
		Name:  window_id.AddMemory,
		Title: i18n.TCurrent("app.window.add_memory_title", nil),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 48,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/?entry=add_memory",
		Width:            900,
		Height:           820,
		MinWidth:         550,
		MinHeight:        620,
	}))
	p.showCenteredOnHomeScreen(addMemoryWindow)
	return &window_dto.OpenAddMemoryOutput{}, nil
}

// CloseAddMemory 关闭添加记忆窗口
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) CloseAddMemory(ctx context.Context, input window_dto.CloseAddMemoryInput) (*window_dto.CloseAddMemoryOutput, error) {
	addMemoryWindow, ok := p.wailsApp.Window.GetByName(window_id.AddMemory) // 增加记忆窗口、是否成功取得对应值
	if ok {
		addMemoryWindow.Close()
		return &window_dto.CloseAddMemoryOutput{}, nil
	}
	return &window_dto.CloseAddMemoryOutput{}, nil
}

// OpenOnboarding 打开首次启动引导窗口，窗口已存在时将其聚焦
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) OpenOnboarding(ctx context.Context, input window_dto.OpenOnboardingInput) (*window_dto.OpenOnboardingOutput, error) {
	if /* 首次引导窗口、是否成功取得对应值 */ onboardingWindow, ok := p.wailsApp.Window.GetByName(window_id.Onboarding); ok {
		p.showCenteredOnHomeScreen(onboardingWindow)
		return &window_dto.OpenOnboardingOutput{}, nil
	}

	onboardingWindow := p.wailsApp.Window.NewWithOptions(window_options.DefaultOnboarding()) // 首次引导窗口
	p.showCenteredOnHomeScreen(onboardingWindow)
	return &window_dto.OpenOnboardingOutput{}, nil
}

// OpenQuickChat 打开快捷聊天浮窗，窗口已存在时将其聚焦
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) OpenQuickChat(ctx context.Context, input window_dto.OpenQuickChatInput) (*window_dto.OpenQuickChatOutput, error) {
	if p.wailsApp == nil {
		return &window_dto.OpenQuickChatOutput{}, nil
	}
	if /* 快捷聊天窗口、是否成功取得对应值 */ quickChatWindow, ok := p.wailsApp.Window.GetByName(window_id.QuickChat); ok {
		p.resetAndShowQuickChatWindow(quickChatWindow)
		return &window_dto.OpenQuickChatOutput{}, nil
	}

	options := window_options.DefaultQuickChat() // 选项列表
	options.Title = i18n.TCurrent("app.window.quick_chat_title", nil)
	quickChatWindow := p.wailsApp.Window.NewWithOptions(p.childWindowOptions(options)) // 快捷聊天窗口
	p.showCenteredOnHomeScreen(quickChatWindow)
	return &window_dto.OpenQuickChatOutput{}, nil
}

// CloseQuickChat 隐藏已存在的快捷聊天浮窗
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) CloseQuickChat(ctx context.Context, input window_dto.CloseQuickChatInput) (*window_dto.CloseQuickChatOutput, error) {
	if p.wailsApp == nil {
		return &window_dto.CloseQuickChatOutput{}, nil
	}
	if /* 快捷聊天窗口、是否成功取得对应值 */ quickChatWindow, ok := p.wailsApp.Window.GetByName(window_id.QuickChat); ok {
		quickChatWindow.Hide()
	}
	return &window_dto.CloseQuickChatOutput{}, nil
}

// ToggleQuickChat 切换快捷聊天浮窗的显示状态
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) ToggleQuickChat(ctx context.Context, input window_dto.ToggleQuickChatInput) (*window_dto.ToggleQuickChatOutput, error) {
	if p.wailsApp == nil {
		return &window_dto.ToggleQuickChatOutput{}, nil
	}
	if /* 快捷聊天窗口、是否成功取得对应值 */ quickChatWindow, ok := p.wailsApp.Window.GetByName(window_id.QuickChat); ok {
		if quickChatWindow.IsVisible() {
			quickChatWindow.Hide()
			return &window_dto.ToggleQuickChatOutput{}, nil
		}
		p.resetAndShowQuickChatWindow(quickChatWindow)
		return &window_dto.ToggleQuickChatOutput{}, nil
	}

	_, err := p.OpenQuickChat(ctx, window_dto.OpenQuickChatInput{}) // 打开快捷聊天时的错误
	return &window_dto.ToggleQuickChatOutput{}, err
}

// SelectQuickChat 向主窗口发送选中的会话并关闭快捷聊天浮窗
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Window) SelectQuickChat(ctx context.Context, input window_dto.SelectQuickChatInput) (*window_dto.SelectQuickChatOutput, error) {
	if p.wailsApp == nil {
		return &window_dto.SelectQuickChatOutput{}, nil
	}
	p.wailsApp.Event.Emit(event_id.QuickChatSelected, map[string]uint{
		"sessionId": input.SessionID,
	})
	if /* 主界面窗口、是否成功取得对应值 */ homeWindow, ok := p.wailsApp.Window.GetByName(window_id.Home); ok {
		homeWindow.Show()
		homeWindow.Focus()
	}

	if /* 快捷聊天窗口、是否成功取得对应值 */ quickChatWindow, ok := p.wailsApp.Window.GetByName(window_id.QuickChat); ok {
		quickChatWindow.Hide()
	}
	return &window_dto.SelectQuickChatOutput{}, nil
}

// 重置快捷聊天窗口的界面状态并显示窗口，接收快捷聊天窗口，无返回值
func (p *Window) resetAndShowQuickChatWindow(quickChatWindow application.Window) {
	options := window_options.DefaultQuickChat() // 选项列表
	quickChatWindow.SetSize(options.Width, options.Height)
	quickChatWindow.EmitEvent(event_id.QuickChatReset)
	p.showCenteredOnHomeScreen(quickChatWindow)
}
