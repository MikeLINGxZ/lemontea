//go:build !windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

// 默认外框PTY等待延时
const defaultShellPTYWaitDelay = time.Second

// 处理外框函数PTY，接收本次调用的上下文、事件发布器、会话编号、命令执行参数，返回命令执行结果及可能发生的错误
// 启动带有尺寸
// 结束时关闭当前资源
// 等待任务结束
func shellFuncPTY(ctx context.Context, emitter ShellProgressEmitter, sessionID uint, input shellInput) (shellOutput, error) {
	shell, args := resolveShellCommand(input.Command) // 外框、参数
	cmd := exec.CommandContext(ctx, shell, args...)   // 命令
	cmd.Dir = resolveShellWorkDir(input.WorkDir)
	cmd.WaitDelay = 2 * defaultShellPTYWaitDelay

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 120}) // 伪终端主端、启动带有尺寸时的错误
	if err != nil {
		return shellOutput{}, fmt.Errorf("start pty command: %w", err)
	}

	// 结束时关闭当前资源
	defer ptmx.Close()

	// 标准输出
	var stdout strings.Builder
	buf := make([]byte, 4096) // 缓冲区
	for {
		n, readErr := ptmx.Read(buf) // 数量、读取当前数据时的错误
		if n > 0 {
			text := string(buf[:n]) // 文本
			stdout.WriteString(text)
			if emitter != nil && sessionID != 0 {
				emitter.EmitToolResult(sessionID, "shell", stdout.String())
			}
		}
		if readErr != nil {
			break
		}
	}

	waitErr := cmd.Wait() // 等待任务结束时的错误
	exitCode := 0         // 退出代码
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	// 本次处理的输出
	out := shellOutput{
		Stdout:   stdout.String(),
		Stderr:   "",
		ExitCode: exitCode,
	}
	if ctx.Err() != nil {
		return out, fmt.Errorf("exec: %w", ctx.Err())
	}
	if waitErr != nil {
		// 退出错误
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) {
			return out, fmt.Errorf("exec: %w", waitErr)
		}
	}
	return out, nil
}
