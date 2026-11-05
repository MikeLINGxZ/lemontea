//go:build windows

package plugin

import (
	"context"
	"errors"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto"
)

// CLI登录不受支持在Windows对应的错误
var errCliLoginUnsupportedOnWindows = errors.New("cli login is not supported on Windows")

// LoginCli 在 Windows 上返回平台不支持交互式 CLI 登录的错误
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) LoginCli(ctx context.Context, input plugin_dto.LoginCliInput) (*plugin_dto.LoginCliOutput, error) {
	_ = ctx
	_ = input
	return nil, ierror.Error(ierror.ErrRuntimeUnsupportedOS, errCliLoginUnsupportedOnWindows)
}

// SendLoginStdin 在 Windows 上返回平台不支持写入登录会话的错误
// 接收调用上下文和请求参数，返回操作错误
func (p *Plugin) SendLoginStdin(ctx context.Context, input plugin_dto.SendLoginStdinInput) error {
	_ = ctx
	_ = input
	return ierror.Error(ierror.ErrRuntimeUnsupportedOS, errCliLoginUnsupportedOnWindows)
}

// ResizeLoginCli 在 Windows 上返回平台不支持调整登录会话尺寸的错误
// 接收调用上下文和请求参数，返回操作错误
func (p *Plugin) ResizeLoginCli(ctx context.Context, input plugin_dto.ResizeLoginCliInput) error {
	_ = ctx
	_ = input
	return ierror.Error(ierror.ErrRuntimeUnsupportedOS, errCliLoginUnsupportedOnWindows)
}

// CancelLoginCli 在 Windows 上直接返回成功，因为此平台不会启动 CLI 登录流程
// 接收调用上下文和请求参数，返回操作错误
func (p *Plugin) CancelLoginCli(ctx context.Context, input plugin_dto.CancelLoginCliInput) error {
	_ = ctx
	_ = input
	return nil
}
