package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// DefaultStdoutLimit 是 RunParams.StdoutLimit 或 StderrLimit 为零时对应输出流的字节上限
const DefaultStdoutLimit = 32 * 1024

// DefaultTimeoutSeconds 是 RunParams.TimeoutSec 为零时单次调用的默认实际耗时上限
const DefaultTimeoutSeconds = 60

// MaxTimeoutSeconds 是超时硬上限，清单配置无法突破该值
const MaxTimeoutSeconds = 600

// RunParams 描述一次 CLI 子进程调用
type RunParams struct {
	// 可执行文件
	Executable string
	// 命令参数
	Argv []string
	// 环境变量
	Env []string
	// 当前工作项目录
	WorkingDir string
	// 输出模式
	OutputMode OutputMode
	// 超时秒
	TimeoutSec int
	// 标准输出上限
	StdoutLimit int
	// 标准错误上限
	StderrLimit int
	// UsePTY 对应的是否满足条件
	UsePTY bool
}

// RunResult 保存 CLI 调用结果，供转发给模型
type RunResult struct {
	// 退出代码
	ExitCode int
	// 标准输出
	Stdout string
	// 标准错误
	Stderr      string
	Parsed      json.RawMessage // OutputMode 为 OutputJSON 且标准输出为有效 JSON 时填充
	ParsedLines []string        // OutputMode 为 OutputLines 时填充
	// 已截断标准输出
	TruncatedStdout bool
	// 已截断标准错误
	TruncatedStderr bool
	// 时长毫秒
	DurationMS int64
}

// RunProgress 表示子进程流式输出的累计快照
type RunProgress struct {
	// 标准输出
	Stdout string
	// 标准错误
	Stderr string
	// 已截断标准输出
	TruncatedStdout bool
	// 已截断标准错误
	TruncatedStderr bool
}

// ErrRunTimeout 表示 Run 中的子进程在实际耗时达到上限前未退出
var ErrRunTimeout = errors.New("cli: run timed out")

// Run 执行一次 CLI 子进程，应用超时和输出截断，并按 OutputMode 解析输出
// 非零退出码通过 RunResult.ExitCode 返回，Go 错误保留给执行过程中的故障
// 接收本次调用的上下文、运行参数，返回运行结果及可能发生的错误
func Run(ctx context.Context, p RunParams) (RunResult, error) {
	return RunWithProgress(ctx, p, nil)
}

// RunWithProgress 执行一次 CLI 子进程，随标准输出和标准错误到达持续发送累计快照
// 接收本次调用的上下文、运行参数、on进度，返回运行结果及可能发生的错误
func RunWithProgress(ctx context.Context, p RunParams, onProgress func(RunProgress)) (RunResult, error) {
	if p.UsePTY {
		return runWithProgressPTY(ctx, p, onProgress)
	}
	return runWithProgressPipe(ctx, p, onProgress)
}

// 通过标准输出和标准错误管道运行 CLI，接收上下文、运行参数与进度回调
// 并发采集两路输出并发布累计快照，等待进程和读管道结束后返回结果及错误
func runWithProgressPipe(ctx context.Context, p RunParams, onProgress func(RunProgress)) (RunResult, error) {
	stdoutLimit := p.StdoutLimit // 标准输出上限
	if stdoutLimit <= 0 {
		stdoutLimit = DefaultStdoutLimit
	}
	stderrLimit := p.StderrLimit // 标准错误上限
	if stderrLimit <= 0 {
		stderrLimit = DefaultStdoutLimit
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
	// 让子进程在独立进程组中运行，并在取消时向整个进程组发送信号
	// 使 Shell 启动的进程（如 `bash -c "sleep 5"` 中的 sleep）随直接子进程一同退出
	applyProcessGroup(cmd)
	// WaitDelay 限制调用 Cancel 后 Wait 的等待时长
	// 即使后代进程收到取消信号后仍保持标准输出管道打开，也会在该时限内返回
	cmd.WaitDelay = 2 * time.Second

	stdoutBuf := &cappedBuffer{limit: stdoutLimit} // 标准输出缓冲区
	stderrBuf := &cappedBuffer{limit: stderrLimit} // 标准错误缓冲区
	stdoutPipe, err := cmd.StdoutPipe()            // 标准输出管道、处理标准输出管道时的错误
	if err != nil {
		return RunResult{}, err
	}
	stderrPipe, err := cmd.StderrPipe() // 标准错误管道
	if err != nil {
		return RunResult{}, err
	}

	start := time.Now()     // 开始
	startErr := cmd.Start() // 启动任务时的错误
	if startErr != nil {
		return RunResult{}, startErr
	}

	chunks := make(chan cliOutputChunk, 32) // 分片列表
	// 等待所有并发任务结束的等待组
	var wg sync.WaitGroup
	wg.Add(2)
	go readCLIOutputPipe(&wg, stdoutPipe, stdoutBuf, true, chunks)
	go readCLIOutputPipe(&wg, stderrPipe, stderrBuf, false, chunks)
	go /* 等待关闭分片列表时调用的回调，无返回值 */ func() {
		wg.Wait()
		close(chunks)
	}()

	for range chunks {
		if onProgress != nil {
			onProgress(RunProgress{
				Stdout:          stdoutBuf.String(),
				Stderr:          stderrBuf.String(),
				TruncatedStdout: stdoutBuf.truncated,
				TruncatedStderr: stderrBuf.truncated,
			})
		}
	}

	err = cmd.Wait()
	// 结果
	res := RunResult{
		Stdout:          stdoutBuf.String(),
		Stderr:          stderrBuf.String(),
		TruncatedStdout: stdoutBuf.truncated,
		TruncatedStderr: stderrBuf.truncated,
		DurationMS:      time.Since(start).Milliseconds(),
		ExitCode:        cmd.ProcessState.ExitCode(),
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

// 描述CLI输出分片使用的数据结构
type cliOutputChunk struct{}

// 读取CLI输出管道，接收等待所有并发任务结束的等待组、读取源、缓冲区、发布、分片列表，无返回值
func readCLIOutputPipe(wg *sync.WaitGroup, r io.ReadCloser, buf *cappedBuffer, emit bool, chunks chan<- cliOutputChunk) {
	defer wg.Done()
	defer r.Close()

	tmp := make([]byte, 2048) // 临时
	for {
		n, err := r.Read(tmp) // 数量、读取当前数据时的错误
		if n > 0 {
			_, _ = buf.Write(tmp[:n])
			if emit {
				chunks <- cliOutputChunk{}
			} else {
				chunks <- cliOutputChunk{}
			}
		}
		if err != nil {
			return
		}
	}
}

// parseOutputs 根据 OutputMode 和标准输出填充 Parsed 或 ParsedLines
// 接收结果、模式，无返回值
func parseOutputs(res *RunResult, mode OutputMode) {
	switch mode {
	case OutputJSON:
		trimmed := strings.TrimSpace(res.Stdout) // 已去除空白
		if trimmed == "" {
			return
		}
		// 探测
		var probe interface{}
		err := json.Unmarshal([]byte(trimmed), &probe) // 解析序列化内容时的错误
		if err == nil {
			res.Parsed = json.RawMessage(trimmed)
		}
	case OutputLines:
		if res.Stdout == "" {
			res.ParsedLines = []string{}
			return
		}
		res.ParsedLines = strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n")
	}
}

// cappedBuffer 实现 io.Writer，写满 limit 字节后停止追加，并记录是否发生截断
type cappedBuffer struct {
	// 缓冲区
	buf bytes.Buffer
	// 上限
	limit int
	// 已截断
	truncated bool
}

// Write 最多累计 limit 字节，后续数据直接丢弃并设置 truncated
// 接收数据字节，返回整数及可能发生的错误
func (c *cappedBuffer) Write(b []byte) (int, error) {
	if c.buf.Len() >= c.limit {
		c.truncated = true
		return len(b), nil
	}
	remaining := c.limit - c.buf.Len() // 剩余
	if len(b) > remaining {
		c.buf.Write(b[:remaining])
		c.truncated = true
		return len(b), nil
	}
	return c.buf.Write(b)
}

// String 返回累计字节对应的字符串
func (c *cappedBuffer) String() string { return c.buf.String() }
