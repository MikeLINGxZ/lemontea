package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

const (
	// 默认外框超时
	defaultShellTimeout = 10 * time.Minute
	// 上限外框超时
	maxShellTimeout = time.Hour
)

// 定义外框进度事件发布器的调用契约
type ShellProgressEmitter interface {
	// 将工具执行结果发布给对应会话，接收会话编号、工具名称、本次操作的结果，无返回值
	EmitToolResult(sessionID uint, toolName, result string)
}

// 封装处理外框所需的请求参数
type shellInput struct {
	// 命令
	Command string `json:"command" jsonschema:"description=Shell command to execute,required"`
	// 工作目录目录
	WorkDir string `json:"work_dir" jsonschema:"description=Working directory (default: user home)"`
	// 超时秒
	TimeoutSeconds int `json:"timeout_seconds" jsonschema:"description=Optional timeout in seconds; defaults to 600 and is capped at 3600"`
	// UsePty 对应的是否满足条件
	UsePty bool `json:"use_pty" jsonschema:"description=Run in a pseudo-terminal for interactive commands that draw terminal UI, print QR codes, or wait for browser/scan login"`
	// 显示终端
	ShowTerminal bool `json:"show_terminal" jsonschema:"description=Show the terminal panel to the user. Defaults to false for normal commands and true when use_pty is true"`
	// 保留终端
	KeepTerminal bool `json:"keep_terminal" jsonschema:"description=Keep a shown terminal visible after the command exits. Defaults to false"`
}

// 封装处理外框返回给调用方的数据
type shellOutput struct {
	// 标准输出
	Stdout string `json:"stdout"`
	// 标准错误
	Stderr string `json:"stderr"`
	// 退出代码
	ExitCode int `json:"exit_code"`
	// 终端编号
	TerminalID string `json:"terminal_id,omitempty"`
}

// 描述外框分片使用的数据结构
type shellChunk struct {
	// 流式输出
	stream string
	// 文本
	text string
}

// resolveShellTimeout 为 Shell 命令应用默认超时和超时上限
// 接收秒，返回时长
func resolveShellTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultShellTimeout
	}
	timeout := time.Duration(seconds) * time.Second // 本次操作的最长等待时长
	if timeout > maxShellTimeout {
		return maxShellTimeout
	}
	return timeout
}

// 解析外框命令，接收命令，返回文本、文本列表
func resolveShellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		shell := os.Getenv("ComSpec") // 外框
		if shell == "" {
			shell = "cmd.exe"
		}
		return shell, []string{"/d", "/s", "/c", command}
	}
	if /* 外框 */ shell := os.Getenv("SHELL"); shell != "" {
		return shell, []string{"-lc", command}
	}
	// 遍历stringbashzshshusr可执行目录bash可执行目录bashusr可执行目录zsh可执行目录zshusr可执行目录sh可执行目录sh，逐项取得候选值
	for _, candidate := range []string{
		"bash",
		"zsh",
		"sh",
		"/usr/bin/bash",
		"/bin/bash",
		"/usr/bin/zsh",
		"/bin/zsh",
		"/usr/bin/sh",
		"/bin/sh",
	} {
		if candidate == "" {
			continue
		}
		if shellCandidateAvailable(candidate) {
			return candidate, []string{"-lc", command}
		}
	}
	return "sh", []string{"-lc", command}
}

// 处理外框候选值可用项，接收候选值，返回是否满足条件
func shellCandidateAvailable(candidate string) bool {
	if strings.Contains(candidate, "/") {
		info, err := os.Stat(candidate) // 信息、读取文件元数据时的错误
		return err == nil && !info.IsDir()
	}
	_, err := exec.LookPath(candidate) // 处理Look路径时的错误
	return err == nil
}

// 解析外框工作目录目录，接收工作目录目录，返回文本
func resolveShellWorkDir(workDir string) string {
	if isDir(workDir) {
		return workDir
	}
	home, err := os.UserHomeDir() // 主界面、处理用户主界面目录时的错误
	if err == nil && isDir(home) {
		return home
	}
	return ""
}

// 判断是否为目录，接收文件路径，返回是否满足条件
func isDir(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path) // 信息、读取文件元数据时的错误
	return err == nil && info.IsDir()
}

// shellFunc 在无共享终端管理器的情况下执行 Shell 命令
// 接收本次调用的上下文、事件发布器、会话编号、命令执行参数，返回命令执行结果及可能发生的错误
func shellFunc(ctx context.Context, emitter ShellProgressEmitter, sessionID uint, input shellInput) (shellOutput, error) {
	return shellFuncWithTerminal(ctx, emitter, sessionID, input, nil)
}

// 描述终端创建需要的参数
type terminalCreateParams = pkgterminal.CreateParams

// 定义终端信息的类型
type terminalInfo = pkgterminal.Info

// 定义终端输出分片的类型
type terminalOutputChunk = pkgterminal.TerminalOutputChunk

// 定义外框终端Runner的调用契约
type ShellTerminalRunner interface {
	// 创建终端，接收本次调用的上下文、参数，返回终端信息及可能发生的错误
	CreateTerminal(ctx context.Context, params terminalCreateParams) (terminalInfo, error)
	// 等待终端，接收本次调用的上下文、终端编号，返回终端信息及可能发生的错误
	WaitTerminal(ctx context.Context, terminalID string) (terminalInfo, error)
	// 读取终端输出，接收本次调用的上下文、终端编号、游标，返回终端输出分片列表及可能发生的错误
	ReadTerminalOutput(ctx context.Context, terminalID string, cursor int64) ([]terminalOutputChunk, error)
	// 设置终端可见，接收本次调用的上下文、终端编号、可见、标题，返回操作错误
	SetTerminalVisible(ctx context.Context, terminalID string, visible bool, title string) error
}

// shellFuncWithTerminal 在配置共享终端管理器时通过它执行 Shell 命令
// 接收本次调用的上下文、事件发布器、会话编号、命令执行参数、终端Runner，返回命令执行结果及可能发生的错误
func shellFuncWithTerminal(ctx context.Context, emitter ShellProgressEmitter, sessionID uint, input shellInput, terminalRunner ShellTerminalRunner) (shellOutput, error) {
	timeout := resolveShellTimeout(input.TimeoutSeconds) // 本次操作的最长等待时长
	ctx, cancel := context.WithTimeout(ctx, timeout)     // 用于取消本次操作的回调
	defer cancel()

	if terminalRunner != nil {
		return shellFuncTerminal(ctx, emitter, sessionID, input, terminalRunner)
	}
	if input.UsePty {
		return shellFuncPTY(ctx, emitter, sessionID, input)
	}
	return shellFuncPipe(ctx, emitter, sessionID, input)
}

// shellFuncTerminal 通过持久化交互终端执行 Shell 命令
// 接收本次调用的上下文、事件发布器、会话编号、命令执行参数、终端Runner，返回命令执行结果及可能发生的错误
// 创建终端
// 等待终端
// 读取终端输出
func shellFuncTerminal(ctx context.Context, emitter ShellProgressEmitter, sessionID uint, input shellInput, terminalRunner ShellTerminalRunner) (shellOutput, error) {
	visible := input.ShowTerminal || input.UsePty     // 可见
	shell, args := resolveShellCommand(input.Command) // 外框、参数
	// 信息、创建终端时的错误
	info, err := terminalRunner.CreateTerminal(ctx, terminalCreateParams{
		SessionID: sessionID,
		Title:     "Shell command",
		Command:   shell,
		Args:      args,
		Cwd:       resolveShellWorkDir(input.WorkDir),
		Visible:   visible,
	})
	if err != nil {
		return shellOutput{}, err
	}
	finalInfo, err := terminalRunner.WaitTerminal(ctx, info.ID) // 最终信息
	// 分片列表、读取终端输出时的错误
	chunks, readErr := terminalRunner.ReadTerminalOutput(context.Background(), info.ID, 0)
	if readErr != nil && err == nil {
		err = readErr
	}
	if visible && !input.KeepTerminal {
		hideErr := terminalRunner.SetTerminalVisible(context.Background(), info.ID, false, "") // 设置终端可见时的错误
		if hideErr != nil && err == nil {
			err = hideErr
		}
	}
	// 标准输出
	var stdout strings.Builder
	// 遍历分片列表，逐项取得分片
	for _, chunk := range chunks {
		stdout.WriteString(chunk.Data)
	}
	output := stdout.String() // 输出数据
	if emitter != nil && sessionID != 0 {
		emitter.EmitToolResult(sessionID, "shell", output)
	}
	exitCode := 0 // 退出代码
	if finalInfo.ExitCode != nil {
		exitCode = *finalInfo.ExitCode
	}
	return shellOutput{
		Stdout:     output,
		Stderr:     "",
		ExitCode:   exitCode,
		TerminalID: info.ID,
	}, err
}

// 通过标准输出和标准错误管道执行命令，接收上下文、事件发布器、会话编号和命令参数
// 启动后并发收集两路输出并发送进度，结束时返回命令结果或执行错误
func shellFuncPipe(ctx context.Context, emitter ShellProgressEmitter, sessionID uint, input shellInput) (shellOutput, error) {
	shell, args := resolveShellCommand(input.Command) // 外框、参数
	cmd := exec.CommandContext(ctx, shell, args...)   // 命令
	cmd.Dir = resolveShellWorkDir(input.WorkDir)

	stdoutPipe, err := cmd.StdoutPipe() // 标准输出管道、处理标准输出管道时的错误
	if err != nil {
		return shellOutput{}, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe() // 标准错误管道
	if err != nil {
		return shellOutput{}, fmt.Errorf("stderr pipe: %w", err)
	}

	startErr := cmd.Start() // 启动任务时的错误
	if startErr != nil {
		return shellOutput{}, fmt.Errorf("start command: %w", startErr)
	}

	chunks := make(chan shellChunk, 32) // 分片列表
	// 等待所有并发任务结束的等待组
	var wg sync.WaitGroup
	wg.Add(2)
	go readShellPipe(&wg, stdoutPipe, "stdout", chunks)
	go readShellPipe(&wg, stderrPipe, "stderr", chunks)
	go /* 等待关闭分片列表时调用的回调，无返回值 */ func() {
		wg.Wait()
		close(chunks)
	}()

	// 标准输出
	var stdout strings.Builder
	// 标准错误输出
	var stderr strings.Builder
	// 遍历分片列表，逐项取得分片
	for chunk := range chunks {
		switch chunk.stream {
		case "stdout":
			stdout.WriteString(chunk.text)
		case "stderr":
			stderr.WriteString(chunk.text)
		}
		if emitter != nil && sessionID != 0 {
			emitter.EmitToolResult(sessionID, "shell", formatShellProgress(stdout.String(), stderr.String()))
		}
	}

	err = cmd.Wait()
	exitCode := 0 // 退出代码
	if err != nil {
		if /* 退出错误、是否成功取得对应值 */ exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return shellOutput{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: -1,
			}, fmt.Errorf("exec: %w", ctx.Err())
		} else {
			return shellOutput{}, fmt.Errorf("exec: %w", err)
		}
	}

	return shellOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// readShellPipe 读取一路命令输出并转换为 shellChunk 消息
// 接收等待所有并发任务结束的等待组、读取源、流式输出、分片列表，无返回值
func readShellPipe(wg *sync.WaitGroup, r io.ReadCloser, stream string, chunks chan<- shellChunk) {
	defer wg.Done()
	defer r.Close()

	buf := make([]byte, 2048) // 缓冲区
	for {
		n, err := r.Read(buf) // 数量、读取当前数据时的错误
		if n > 0 {
			chunks <- shellChunk{stream: stream, text: string(buf[:n])}
		}
		if err != nil {
			return
		}
	}
}

// formatShellProgress 合并标准输出和标准错误，形成流式工具进度
// 接收标准输出、标准错误输出，返回文本
func formatShellProgress(stdout, stderr string) string {
	switch {
	case stdout == "" && stderr == "":
		return ""
	case stderr == "":
		return stdout
	case stdout == "":
		return "[stderr]\n" + stderr
	default:
		return stdout + "\n[stderr]\n" + stderr
	}
}

// NewShellTool 创建执行 Shell 命令的智能体工具
// 接收事件发布器、会话编号、终端Runner，返回Function工具命令执行参数命令执行结果
func NewShellTool(emitter ShellProgressEmitter, sessionID uint, terminalRunner ...ShellTerminalRunner) *function.FunctionTool[shellInput, shellOutput] {
	// runner 对应的数据
	var runner ShellTerminalRunner
	if len(terminalRunner) > 0 {
		runner = terminalRunner[0]
	}
	return function.NewFunctionTool(
		func(ctx context.Context, input shellInput) (shellOutput, error) {
			// 构建Function工具时调用的回调，接收本次调用的上下文、命令执行参数，返回命令执行结果及可能发生的错误
			return shellFuncWithTerminal(ctx, emitter, sessionID, input, runner)
		},
		function.WithName("shell"),
		function.WithDescription("Internal system execution tool for complex user tasks such as organizing folders, searching local files, summarizing local materials, or calling fixed local utilities. Do not ask the user to write shell commands. Prefer safe read-only commands for inspection, and explain user-facing results in plain language. Commands run in the shared interactive terminal backend when available. Set use_pty=true for commands that render terminal UI, print QR codes, or wait for login confirmation. Set show_terminal=true only when the terminal should be visible to the user, and keep_terminal=true only when it should remain visible after exit."),
	)
}

// ShellMeta 返回确认界面展示 Shell 工具所需的元数据
func ShellMeta() ToolMeta {
	return ToolMeta{
		Name:            "shell",
		Description:     "Execute shell commands",
		Category:        CategoryBuiltin,
		RequiresConfirm: true,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input shellInput
			_ = json.Unmarshal(args, &input)
			return fmt.Sprintf("Execute command: %s", input.Command)
		},
	}
}
