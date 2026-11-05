//go:build !windows

package terminal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// 描述UnixPTY进程使用的数据结构
type unixPTYProcess struct {
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
	// 单次执行锁
	once sync.Once
	// 保护共享状态的互斥锁
	mu sync.Mutex
	// 关闭
	closed bool
}

// startPTYProcess 在 Unix 伪终端中启动指定命令
// 接收本次调用的上下文、参数，返回PTY进程、整数及可能发生的错误
func startPTYProcess(ctx context.Context, params CreateParams) (ptyProcess, int, error) {
	cmd := buildCommand(ctx, params)       // 命令
	rows, cols := params.Rows, params.Cols // 行数、列数
	if rows == 0 {
		rows = 40
	}
	if cols == 0 {
		cols = 120
	}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols}) // 伪终端主端、启动带有尺寸时的错误
	if err != nil {
		return nil, 0, err
	}
	// 当前条目
	p := &unixPTYProcess{
		cmd:    cmd,
		ptmx:   ptmx,
		output: make(chan []byte, 64),
		done:   make(chan struct{}),
	}
	go p.readLoop()
	go /* 关闭开始PTY进程时调用的回调，无返回值 */ func() {
		<-ctx.Done()
		_ = p.Close()
	}()
	pid := 0 // 进程号
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	return p, pid, nil
}

// readLoop 将 PTY 输出转发到管理器的通道，并记录进程退出状态
func (p *unixPTYProcess) readLoop() {
	buf := make([]byte, 4096) // 缓冲区
	for {
		n, err := p.ptmx.Read(buf) // 数量、读取当前数据时的错误
		if n > 0 {
			chunk := make([]byte, n) // 分片
			copy(chunk, buf[:n])
			p.output <- chunk
		}
		if err != nil {
			break
		}
	}
	waitErr := p.cmd.Wait() // 等待任务结束时的错误
	exitCode := -1          // 退出代码
	if p.cmd.ProcessState != nil {
		exitCode = p.cmd.ProcessState.ExitCode()
	}
	p.once.Do( /* 发送请求并读取响应时调用的回调，无返回值 */ func() {
		p.mu.Lock()
		p.waitErr = waitErr
		p.exitCode = exitCode
		p.closed = true
		p.mu.Unlock()
		close(p.output)
		_ = p.ptmx.Close()
		close(p.done)
	})
}

// Output 返回传输原始 PTY 输出字节的通道
func (p *unixPTYProcess) Output() <-chan []byte { return p.output }

// Write 向 PTY 主端写入字节
// 接收待处理的数据，返回操作错误
func (p *unixPTYProcess) Write(data []byte) error {
	p.mu.Lock()
	closed := p.closed // 关闭
	p.mu.Unlock()
	if closed {
		return errors.New("terminal: session already closed")
	}
	_, err := p.ptmx.Write(data) // 写入当前数据时的错误
	return err
}

// Resize 更新 PTY 窗口大小
// 接收行数、列数，返回操作错误
func (p *unixPTYProcess) Resize(rows, cols uint16) error {
	p.mu.Lock()
	closed := p.closed // 关闭
	p.mu.Unlock()
	if closed {
		return errors.New("terminal: session already closed")
	}
	return pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

// Close 终止进程组并等待读取循环退出
// 返回操作错误
func (p *unixPTYProcess) Close() error {
	p.mu.Lock()
	closed := p.closed // 关闭
	p.mu.Unlock()
	if closed {
		return nil
	}
	if p.cmd.Process != nil {
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		_ = p.cmd.Process.Kill()
	}
	_ = p.ptmx.Close()
	<-p.done
	return nil
}

// Wait 等待进程结束，返回退出码和等待过程中产生的错误
func (p *unixPTYProcess) Wait() (int, error) {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode, p.waitErr
}

// Done 返回 PTY 进程退出后关闭的通道
func (p *unixPTYProcess) Done() <-chan struct{} { return p.done }
