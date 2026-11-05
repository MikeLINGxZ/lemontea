//go:build windows

package cli

import "os/exec"

// applyProcessGroup 在 Windows 上无需额外配置进程组
// exec.CommandContext 的默认取消处理会通过 Process.Kill 终止直接子进程
// 若还需管理后代进程，可扩展为使用 Job Objects
// 接收命令，无返回值
func applyProcessGroup(cmd *exec.Cmd) {}
