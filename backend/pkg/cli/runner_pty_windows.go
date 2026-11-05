//go:build windows

package cli

import "context"

// 使用伪终端运行命令并推送可见输出，接收本次调用的上下文、运行参数、on进度，返回运行结果及可能发生的错误
func runWithProgressPTY(ctx context.Context, p RunParams, onProgress func(RunProgress)) (RunResult, error) {
	p.UsePTY = false
	return runWithProgressPipe(ctx, p, onProgress)
}
