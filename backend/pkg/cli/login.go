//go:build !windows

package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// LoginSessionParams 包含一次交互式登录会话的输入
type LoginSessionParams struct {
	Executable string   // 来自 manifest.Executable 的可执行文件绝对路径
	Argv       []string // 来自 manifest.LoginCommand 的参数列表，不含 Executable
	Env        []string // BuildEnv 生成的隔离或共享环境变量
	Cwd        string   // 工作目录，通常为 plugins/cli/<name>，可为空
}

// LoginSession 管理连接到 PTY 的子进程及其生命周期
// 所有公开方法均可由多个 goroutine 并发调用
type LoginSession struct {
	// 命令
	cmd *exec.Cmd
	// 伪终端主端
	ptmx *os.File
	// 输出数据
	output chan []byte
	// 完成
	done chan struct{}
	// 等待错误
	waitErr error
	// 退出代码
	exitCode int
	// 保护共享状态的互斥锁
	mu sync.Mutex
	// 关闭
	closed bool
	once   sync.Once // 保证关闭 ptmx 和 done 的清理操作只执行一次
}

// StartLoginSession 在 PTY 中启动子进程并开始读取输出
// 返回的会话由调用方管理，可调用 Wait 或 Cancel 释放资源
// 接收本次调用的上下文、登录会话参数，返回登录会话及可能发生的错误
func StartLoginSession(ctx context.Context, p LoginSessionParams) (*LoginSession, error) {
	checkErr := ctx.Err() // 上下文取消或超时的错误
	if checkErr != nil {
		return nil, checkErr
	}

	cmd := exec.Command(p.Executable, p.Argv...) // 命令
	if p.Env != nil {
		cmd.Env = p.Env
	}
	if p.Cwd != "" {
		cmd.Dir = p.Cwd
	}

	// 由 pty.Start 将 PTY 从端连接到 cmd.Stdin、cmd.Stdout 和 cmd.Stderr
	// 如果自行连接管道，exec.CommandContext 会在取消时启动自己的等待 goroutine
	// 这会和下方 readLoop 中的 cmd.Wait 发生竞争
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// 当前值
	s := &LoginSession{
		cmd:    cmd,
		ptmx:   ptmx,
		output: make(chan []byte, 64),
		done:   make(chan struct{}),
	}

	go s.readLoop()
	return s, nil
}

// readLoop 持续从 PTY 主端文件描述符读取数据并分块转发到输出通道
// 清理由此处统一执行，依次调用 cmd.Wait、关闭 ptmx 和输出通道，并通知 done
func (s *LoginSession) readLoop() {
	buf := make([]byte, 4096) // 缓冲区
	for {
		n, err := s.ptmx.Read(buf) // 数量、读取当前数据时的错误
		if n > 0 {
			chunk := make([]byte, n) // 分片
			copy(chunk, buf[:n])
			s.output <- chunk
		}
		if err != nil {
			// 读到 EOF 或发生读取错误都视为子进程已关闭 PTY 主端
			break
		}
	}

	// 释放资源前先收集退出状态
	waitErr := s.cmd.Wait()
	exitCode := -1 // 退出代码
	if s.cmd.ProcessState != nil {
		exitCode = s.cmd.ProcessState.ExitCode()
	}

	s.once.Do( /* 发送请求并读取响应时调用的回调，无返回值 */ func() {
		s.mu.Lock()
		s.waitErr = waitErr
		s.exitCode = exitCode
		s.closed = true
		s.mu.Unlock()

		close(s.output)
		_ = s.ptmx.Close()
		close(s.done)
	})
}

// Output 返回只读通道，用于接收 PTY 原始字节
// 子进程退出或会话取消时关闭通道
func (s *LoginSession) Output() <-chan []byte { return s.output }

// Write 将数据写入 PTY 主端，作为子进程的标准输入
// 会话已结束时返回 "already closed"
// 如果写入期间会话关闭，则可能返回底层系统调用错误
// 接收待处理的数据，返回操作错误
func (s *LoginSession) Write(data []byte) error {
	s.mu.Lock()
	closed := s.closed // 关闭
	s.mu.Unlock()
	if closed {
		return errors.New("cli: login session already closed")
	}
	_, err := s.ptmx.Write(data) // 写入当前数据时的错误
	return err
}

// Resize 向 PTY 主端发起 TIOCSWINSZ ioctl 调用，调整终端窗口大小
// 会话已结束时返回 "already closed"
// 如果调整窗口大小期间会话关闭，则可能返回底层系统调用错误
// 接收行数、列数，返回操作错误
func (s *LoginSession) Resize(rows, cols uint16) error {
	s.mu.Lock()
	closed := s.closed // 关闭
	s.mu.Unlock()
	if closed {
		return errors.New("cli: login session already closed")
	}
	return pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

// Cancel 向子进程发送 SIGTERM
// 若 1.5 秒内仍未退出，则发送 SIGKILL
// 返回操作错误
func (s *LoginSession) Cancel() error {
	s.mu.Lock()
	closed := s.closed // 关闭
	s.mu.Unlock()
	if closed {
		return errors.New("cli: login session already closed")
	}

	if s.cmd.Process == nil {
		return nil
	}
	signalErr := s.cmd.Process.Signal(syscall.SIGTERM) // 发送终止信号时的错误
	if signalErr != nil {
		if errors.Is(signalErr, os.ErrProcessDone) {
			return errors.New("cli: login session already closed")
		}
		return errors.Join(errors.New("cli: SIGTERM failed"), signalErr)
	}

	go /* 处理错误进程完成时调用的回调，无返回值 */ func() {
		timer := time.NewTimer(1500 * time.Millisecond) // 定时器
		defer timer.Stop()
		select {
		case <-s.done:
			// 进程收到 SIGTERM 后已正常退出
		case <-timer.C:
			err := s.cmd.Process.Kill() // 强制终止进程时的错误
			if err != nil && !errors.Is(err, os.ErrProcessDone) {
				// 此处没有可用的日志记录器，无法记录这个清理错误
				_ = err
			}
		}
	}()
	return nil
}

// Wait 阻塞直到会话彻底退出，返回退出码以及 cmd.Wait 产生的错误
// 可以安全地重复调用
func (s *LoginSession) Wait() (int, error) {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode, s.waitErr
}
