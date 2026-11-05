//go:build !windows

package cli

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

// 通过伪终端运行 CLI，接收上下文、运行参数与进度回调
// 启动进程后持续读取终端输出并发布快照，完成或超时后收集退出状态和已收到的数据
func runWithProgressPTY(ctx context.Context, p RunParams, onProgress func(RunProgress)) (RunResult, error) {
	stdoutLimit := p.StdoutLimit // 标准输出上限
	if stdoutLimit <= 0 {
		stdoutLimit = DefaultStdoutLimit
	}
	timeoutSec := p.TimeoutSec // 超时秒
	if timeoutSec <= 0 {
		timeoutSec = DefaultTimeoutSeconds
	}
	if timeoutSec > MaxTimeoutSeconds {
		timeoutSec = MaxTimeoutSeconds
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second) // 用于取消本次操作的回调
	defer cancel()

	cmd := exec.CommandContext(ctx, p.Executable, p.Argv...) // 命令
	if p.Env != nil {
		cmd.Env = p.Env
	}
	if p.WorkingDir != "" {
		cmd.Dir = p.WorkingDir
	}
	cmd.WaitDelay = 2 * time.Second

	start := time.Now()                                                    // 开始
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 120}) // 伪终端主端、启动带有尺寸时的错误
	if err != nil {
		return RunResult{}, err
	}

	// 结束时关闭当前资源
	defer ptmx.Close()

	stdoutBuf := &cappedBuffer{limit: stdoutLimit} // 标准输出缓冲区
	tmp := make([]byte, 4096)                      // 临时
	for {
		n, readErr := ptmx.Read(tmp) // 数量、读取当前数据时的错误
		if n > 0 {
			_, _ = stdoutBuf.Write(tmp[:n])
			if onProgress != nil {
				onProgress(RunProgress{
					Stdout:          stdoutBuf.String(),
					TruncatedStdout: stdoutBuf.truncated,
				})
			}
		}
		if readErr != nil {
			break
		}
	}

	err = cmd.Wait()
	exitCode := 0 // 退出代码
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	// 结果
	res := RunResult{
		Stdout:          stdoutBuf.String(),
		TruncatedStdout: stdoutBuf.truncated,
		DurationMS:      time.Since(start).Milliseconds(),
		ExitCode:        exitCode,
	}

	if ctx.Err() == context.DeadlineExceeded {
		return res, ErrRunTimeout
	}
	if err != nil {
		// 退出错误
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return res, err
		}
	}

	parseOutputs(&res, p.OutputMode)
	return res, nil
}
