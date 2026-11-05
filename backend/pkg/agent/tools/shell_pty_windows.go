//go:build windows

package tools

import "context"

// 处理外框函数PTY，接收本次调用的上下文、事件发布器、会话编号、命令执行参数，返回命令执行结果及可能发生的错误
func shellFuncPTY(ctx context.Context, emitter ShellProgressEmitter, sessionID uint, input shellInput) (shellOutput, error) {
	return shellFuncPipe(ctx, emitter, sessionID, input)
}
