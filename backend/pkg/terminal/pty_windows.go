//go:build windows

package terminal

import (
	"context"
	"errors"
)

// startPTYProcess 在 Windows 上返回 PTY 会话暂不可用的错误
// 接收本次调用的上下文、参数，返回PTY进程、整数及可能发生的错误
func startPTYProcess(ctx context.Context, params CreateParams) (ptyProcess, int, error) {
	_ = ctx
	_ = params
	return nil, 0, errors.New("terminal: PTY sessions are not implemented on Windows yet")
}
