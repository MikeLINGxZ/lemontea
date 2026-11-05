//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// applyProcessGroup 配置 cmd，使其子进程在新的进程组中运行
// 超时或上下文取消时，向整个进程组发送 SIGKILL
// 使 Shell 包装命令启动的后代进程也能随之终止
// 接收命令，无返回值
func applyProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true

	cmd.Cancel = /* 实现取消回调，返回操作错误 */ func() error {
		if cmd.Process == nil {
			return nil
		}
		// 负数 PID 表示向整个进程组发送信号
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err != nil {
			// 若未找到进程组，则回退到终止直接子进程
			return cmd.Process.Kill()
		}
		return nil
	}
}
