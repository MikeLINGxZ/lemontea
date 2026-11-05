//go:build !windows

package plugin

import (
	"context"
	"fmt"
	"os"
	"strings"

	pkgcli "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/cli"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin/plugin_dto"
)

// loginSessionIface 定义交互式登录会话需要提供的最小操作集
// 在非 Windows 平台上由 *pkgcli.LoginSession 实现
// 测试中由 fakeLoginSession 实现
type loginSessionIface interface {
	// 处理输出，返回字节数据通道
	Output() <-chan []byte
	// 写入登录会话Iface，接收待处理的数据，返回操作错误
	Write(data []byte) error
	// 处理调整尺寸，接收行数、列数，返回操作错误
	Resize(rows, cols uint16) error
	// 取消登录会话Iface，返回操作错误
	Cancel() error
	// 等待任务结束，返回整数及可能发生的错误
	Wait() (int, error)
}

// loginOutputPayload 携带每段 PTY 输出，通过 cli.login.output 事件发送
type loginOutputPayload struct {
	// 唯一标识
	ID string `json:"id"`
	// 数据
	Data string `json:"data"`
}

// loginDonePayload 携带会话退出结果，通过 cli.login.done 事件发送
type loginDonePayload struct {
	// 唯一标识
	ID string `json:"id"`
	// 退出代码
	ExitCode int `json:"exit_code"`
	// 错误
	Error string `json:"error"`
}

// LoginCli 启动指定 CLI 插件的交互式登录流程后立即返回
// 按 manifest.ResolveLoginSteps() 的顺序执行各步骤，并在同一个 PTY 对话框展示输出
// 执行期间通过 cli.login.output 流式发送输出
// 最后一步成功、任一步骤失败或用户取消时，仅发送一次 cli.login.done
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Plugin) LoginCli(ctx context.Context, input plugin_dto.LoginCliInput) (*plugin_dto.LoginCliOutput, error) {
	config, err := p.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}
	item, ok := findExtension(config.Extensions, input.ID) // 当前条目、是否成功取得对应值
	if !ok || item.Kind != "cli" || !item.Enabled {
		return nil, ierror.Error(ierror.ErrCliLoginNotFound, os.ErrNotExist)
	}
	if item.RootDir == "" {
		return nil, ierror.Error(ierror.ErrCliLoginNotFound, os.ErrNotExist)
	}
	_, statErr := os.Stat(item.RootDir) // 读取文件元数据时的错误
	if statErr != nil {
		return nil, ierror.Error(ierror.ErrCliLoginNotFound, statErr)
	}

	manifest, err := pkgcli.LoadManifest(item.ConfigFilePath) // 清单
	if err != nil {
		return nil, ierror.Error(ierror.ErrCliLoginStartFailed, err)
	}
	steps := manifest.ResolveLoginSteps() // 步骤列表
	if len(steps) == 0 {
		return nil, ierror.Error(ierror.ErrCliLoginNoCommand, fmt.Errorf("manifest has no login_command/login_steps"))
	}

	name, err := cliNameFromID(item.ID) // 当前名称
	if err != nil {
		return nil, err
	}

	id := item.ID // 唯一标识
	p.loginMu.Lock()
	if p.loginSessions[id] != nil {
		p.loginMu.Unlock()
		return nil, ierror.Error(ierror.ErrCliLoginSessionConflict, fmt.Errorf("session already active for %s", id))
	}

	// 同步启动第一步，使命令不存在或管理器报告 no login_command 等错误时立即返回
	// 前端可以直接呈现错误，避免先弹出对话框再等待后端事件将其关闭
	firstSession, startErr := p.startCliLoginCommand(ctx, name, steps[0])
	if startErr != nil {
		p.loginMu.Unlock()
		if strings.Contains(startErr.Error(), "no login_command") {
			return nil, ierror.Error(ierror.ErrCliLoginNoCommand, startErr)
		}
		return nil, ierror.Error(ierror.ErrCliLoginStartFailed, startErr)
	}
	p.loginSessions[id] = firstSession
	p.loginMu.Unlock()

	go p.runLoginFlow(ctx, id, name, steps, firstSession)
	return &plugin_dto.LoginCliOutput{}, nil
}

// runLoginFlow 在独立的 goroutine 中依次执行登录步骤
// 首步会话由 LoginCli 启动并已登记，后续会话由本函数启动
// 流程结束时仅发送一次 cli.login.done 事件
// 随后删除会话登记，允许发起新的登录
// 接收本次调用的上下文、唯一标识、当前名称、步骤列表、第一项，无返回值
func (p *Plugin) runLoginFlow(ctx context.Context, id, name string, steps [][]string, first loginSessionIface) {
	// 流程退出时移除会话登记和取消标记，允许发起后续登录
	defer func() {
		p.loginMu.Lock()
		delete(p.loginSessions, id)
		delete(p.loginCancelled, id)
		p.loginMu.Unlock()
	}()

	// 最后一个退出
	var lastExit int
	// 最后一个错误
	var lastErr error
	current := first // 当前
	// 遍历步骤列表，逐项取得循环索引、命令参数
	for i, argv := range steps {
		if i > 0 {
			p.loginMu.Lock()
			_, cancelled := p.loginCancelled[id] // 已取消
			p.loginMu.Unlock()
			if cancelled {
				lastExit = 130
				lastErr = fmt.Errorf("cli login cancelled by user")
				break
			}

			p.emitLoginBanner(id, i, len(steps), argv)

			sess, err := p.startCliLoginCommand(ctx, name, argv) // 会话、启动CLI登录命令时的错误
			if err != nil {
				lastExit = -1
				lastErr = fmt.Errorf("start step %d (%s): %w", i+1, strings.Join(argv, " "), err)
				break
			}
			p.loginMu.Lock()
			p.loginSessions[id] = sess
			p.loginMu.Unlock()
			current = sess
		}

		// 遍历输出，逐项取得分片
		for chunk := range current.Output() {
			p.emitLoginOutput(id, chunk)
		}
		code, werr := current.Wait() // 代码、w 操作的错误
		lastExit = code
		lastErr = werr

		if code != 0 || werr != nil {
			break
		}
	}

	errStr := "" // 错误文本
	if lastErr != nil {
		errStr = lastErr.Error()
	}
	if p.wailsApp != nil {
		p.wailsApp.Event.Emit("cli.login.done", loginDonePayload{
			ID:       id,
			ExitCode: lastExit,
			Error:    errStr,
		})
	}
}

// emitLoginBanner 将步骤切换提示发送到 PTY 输出所用的事件通道
// xterm 会在命令输出中直接展示该提示
// 接收唯一标识、步骤数值、总量、命令参数，无返回值
func (p *Plugin) emitLoginBanner(id string, stepNum, total int, argv []string) {
	if p.wailsApp == nil {
		return
	}
	// 横幅
	banner := fmt.Sprintf("\r\n\x1b[36m=== step %d/%d: %s ===\x1b[0m\r\n", stepNum+1, total, strings.Join(argv, " "))
	p.wailsApp.Event.Emit("cli.login.output", loginOutputPayload{ID: id, Data: banner})
}

// emitLoginOutput 通过 cli.login.output 事件向前端转发一段 PTY 输出
// 接收唯一标识、分片，无返回值
func (p *Plugin) emitLoginOutput(id string, chunk []byte) {
	if p.wailsApp == nil {
		return
	}
	p.wailsApp.Event.Emit("cli.login.output", loginOutputPayload{ID: id, Data: string(chunk)})
}

// SendLoginStdin 将原始字节写入运行中登录会话的 PTY 标准输入
// 接收调用上下文和请求参数，返回操作错误
func (p *Plugin) SendLoginStdin(ctx context.Context, input plugin_dto.SendLoginStdinInput) error {
	_ = ctx
	p.loginMu.Lock()
	session := p.loginSessions[input.ID] // 会话
	p.loginMu.Unlock()
	if session == nil {
		return ierror.Error(ierror.ErrCliLoginNotFound, os.ErrNotExist)
	}
	err := session.Write([]byte(input.Data)) // 写入当前数据时的错误
	if err != nil {
		return fmt.Errorf("login stdin write: %w", err)
	}
	return nil
}

// ResizeLoginCli 调整运行中登录会话的 PTY 窗口尺寸
// 接收调用上下文和请求参数，返回操作错误
func (p *Plugin) ResizeLoginCli(ctx context.Context, input plugin_dto.ResizeLoginCliInput) error {
	_ = ctx
	p.loginMu.Lock()
	session := p.loginSessions[input.ID] // 会话
	p.loginMu.Unlock()
	if session == nil {
		// 前端发出的最后一次尺寸调整可能恰逢会话退出
		// 此时直接返回成功，避免将无效调整报告为错误
		return nil
	}
	err := session.Resize(input.Rows, input.Cols) // 处理调整尺寸时的错误
	if err != nil {
		if strings.Contains(err.Error(), "already closed") {
			return nil
		}
		return fmt.Errorf("login resize: %w", err)
	}
	return nil
}

// CancelLoginCli 终止正在运行的登录流程
// 设置取消标记以跳过剩余步骤，并通知当前会话终止
// 当前步骤的 PTY 随之关闭，使登录流程结束输出读取
// 接收调用上下文和请求参数，返回操作错误
func (p *Plugin) CancelLoginCli(ctx context.Context, input plugin_dto.CancelLoginCliInput) error {
	_ = ctx
	p.loginMu.Lock()
	session := p.loginSessions[input.ID] // 会话
	if /* 运行中 */ _, running := p.loginSessions[input.ID]; running || session != nil {
		p.loginCancelled[input.ID] = struct{}{}
	}
	p.loginMu.Unlock()
	if session == nil {
		// 对话框关闭时会话可能已自行退出，此时直接返回成功
		return nil
	}
	err := session.Cancel() // 取消当前数据时的错误
	if err != nil {
		if strings.Contains(err.Error(), "already closed") {
			return nil
		}
		return err
	}
	return nil
}
