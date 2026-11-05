package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

// 承载输出事件的数据
type OutputEvent struct {
	// 终端编号
	TerminalID string `json:"terminal_id"`
	// 会话编号
	SessionID uint `json:"session_id"`
	// 游标
	Cursor int64 `json:"cursor"`
	// 数据
	Data string `json:"data"`
	// 当前状态
	Status string `json:"status"`
	// 可见
	Visible bool `json:"visible"`
}

// 承载当前状态事件的数据
type StatusEvent struct {
	// 终端编号
	TerminalID string `json:"terminal_id"`
	// 会话编号
	SessionID uint `json:"session_id"`
	// 当前状态
	Status string `json:"status"`
	// 退出代码
	ExitCode int `json:"exit_code"`
	// 可见
	Visible *bool `json:"visible,omitempty"`
	// 错误
	Error string `json:"error,omitempty"`
}

// 定义终端输出分片的类型
type TerminalOutputChunk = data_models.TerminalOutputChunk

// 描述信息使用的数据结构
type Info struct {
	// 唯一标识
	ID string `json:"id"`
	// 会话编号
	SessionID uint `json:"session_id"`
	// 消息编号
	MessageID *uint `json:"message_id,omitempty"`
	// 工具调用编号
	ToolCallID string `json:"tool_call_id,omitempty"`
	// 标题
	Title string `json:"title"`
	// 命令
	Command string `json:"command"`
	// 参数
	Args string `json:"args"`
	// 工作目录
	Cwd string `json:"cwd"`
	// 当前状态
	Status string `json:"status"`
	// 可见
	Visible bool `json:"visible"`
	// 进程号
	PID int `json:"pid"`
	// 退出代码
	ExitCode *int `json:"exit_code,omitempty"`
	// 当前游标
	CurrentCursor int64 `json:"current_cursor"`
}

// 描述创建需要的参数
type CreateParams struct {
	// 会话编号
	SessionID uint
	// 消息编号
	MessageID *uint
	// 工具调用编号
	ToolCallID string
	// 标题
	Title string
	// 命令
	Command string
	// 参数
	Args []string
	// 环境变量
	Env []string
	// 工作目录
	Cwd string
	// 可见
	Visible bool
	// 行数
	Rows uint16
	// 列数
	Cols uint16
}

// 描述会话使用的数据结构
type session struct {
	// 信息
	info Info
	// 进程
	proc ptyProcess
	// 保护共享状态的互斥锁
	mu sync.Mutex
}

// 描述管理器使用的数据结构
type Manager struct {
	// 状态仓库
	store *storage.Storage
	// 发布输出，接收输出事件，无返回值
	emitOutput func(OutputEvent)
	// 发布当前状态，接收状态事件，无返回值
	emitStatus func(StatusEvent)
	// 保护共享状态的互斥锁
	mu sync.RWMutex
	// 会话列表
	sessions map[string]*session
	// 处理下一项编号，返回文本
	nextID func() string
	// 启动进程，接收调用上下文、创建参数，返回PTY进程、整数及可能发生的错误
	startProcess func(context.Context, CreateParams) (ptyProcess, int, error)
}

// NewManager 创建使用持久化存储的 PTY 终端管理器
// 接收状态仓库、发布输出，返回管理器
func NewManager(store *storage.Storage, emitOutput func(OutputEvent)) *Manager {
	return &Manager{
		store:        store,
		emitOutput:   emitOutput,
		sessions:     make(map[string]*session),
		nextID:       defaultID,
		startProcess: startPTYProcess,
	}
}

// SetOutputEmitter 设置向界面流式发送终端输出块的回调
// 接收发布，无返回值
func (m *Manager) SetOutputEmitter(emit func(OutputEvent)) {
	m.emitOutput = emit
}

// SetStatusEmitter 设置报告终端状态变化的回调
// 接收发布，无返回值
func (m *Manager) SetStatusEmitter(emit func(StatusEvent)) {
	m.emitStatus = emit
}

// Create 启动新的 PTY 进程并保存终端元数据
// 接收本次调用的上下文、参数，返回信息及可能发生的错误
// 将数据序列化
// 创建终端
func (m *Manager) Create(ctx context.Context, params CreateParams) (Info, error) {
	if params.Command == "" {
		return Info{}, errors.New("terminal: command required")
	}
	id := m.nextID() // 唯一标识
	// 进程、进程号、启动进程时的错误
	proc, pid, err := m.startProcess(ctx, params)
	if err != nil {
		return Info{}, err
	}
	argsJSON, _ := json.Marshal(params.Args) // 参数JSON
	title := params.Title                    // 标题
	if title == "" {
		title = params.Command
	}
	// 记录
	record, err := m.store.CreateTerminal(data_models.Terminal{
		TerminalID: id,
		SessionID:  params.SessionID,
		MessageID:  params.MessageID,
		ToolCallID: params.ToolCallID,
		Title:      title,
		Command:    params.Command,
		Args:       string(argsJSON),
		Cwd:        params.Cwd,
		Status:     "active",
		Visible:    params.Visible,
		PID:        pid,
		StartedAt:  time.Now(),
	})
	if err != nil {
		_ = proc.Close()
		return Info{}, err
	}
	info := terminalInfo(*record)            // 信息
	sess := &session{info: info, proc: proc} // 会话

	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	go m.drain(id, sess)
	return info, nil
}

// CreateTerminal 以智能体工具约定的方法名暴露 Create
// 接收本次调用的上下文、参数，返回信息及可能发生的错误
func (m *Manager) CreateTerminal(ctx context.Context, params CreateParams) (Info, error) {
	return m.Create(ctx, params)
}

// Write 向运行中的 PTY 会话写入字节
// 接收终端编号、待处理的数据，返回操作错误
func (m *Manager) Write(terminalID string, data string) error {
	m.mu.RLock()
	sess := m.sessions[terminalID] // 会话
	m.mu.RUnlock()
	if sess == nil {
		return errors.New("terminal: session not active")
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.proc.Write([]byte(data))
}

// WriteTerminal 以 Wails 服务约定的方法名暴露 Write
// 接收终端编号、待处理的数据，返回操作错误
func (m *Manager) WriteTerminal(terminalID string, data string) error {
	return m.Write(terminalID, data)
}

// Resize 调整运行中的 PTY 的行列数
// 接收终端编号、行数、列数，返回操作错误
func (m *Manager) Resize(terminalID string, rows, cols uint16) error {
	m.mu.RLock()
	sess := m.sessions[terminalID] // 会话
	m.mu.RUnlock()
	if sess == nil {
		return nil
	}
	return sess.proc.Resize(rows, cols)
}

// SetVisible 保存终端可见性，不发送运行时事件
// 接收终端编号、可见、标题，返回操作错误
func (m *Manager) SetVisible(terminalID string, visible bool, title string) error {
	return m.store.UpdateTerminalVisibility(terminalID, visible, title)
}

// SetTerminalVisible 更新终端可见性并发送对应的状态事件
// 接收本次调用的上下文、终端编号、可见、标题，返回操作错误
// 更新终端可见状态
// 获取终端
func (m *Manager) SetTerminalVisible(ctx context.Context, terminalID string, visible bool, title string) error {
	_ = ctx
	updateErr := m.store.UpdateTerminalVisibility(terminalID, visible, title) // 更新终端可见状态时的错误
	if updateErr != nil {
		return updateErr
	}
	m.mu.RLock()
	sess := m.sessions[terminalID] // 会话
	m.mu.RUnlock()
	if sess != nil {
		sess.mu.Lock()
		sess.info.Visible = visible
		if title != "" {
			sess.info.Title = title
		}
		sess.mu.Unlock()
	}
	record, err := m.store.GetTerminal(terminalID) // 记录、获取终端时的错误
	if err != nil {
		return err
	}
	if m.emitStatus != nil {
		exitCode := 0 // 退出代码
		if record.ExitCode != nil {
			exitCode = *record.ExitCode
		}
		m.emitStatus(StatusEvent{
			TerminalID: terminalID,
			SessionID:  record.SessionID,
			Status:     record.Status,
			ExitCode:   exitCode,
			Visible:    &visible,
		})
	}
	return nil
}

// ListBySession 返回指定聊天会话下所有已保存的终端
// 接收会话编号，返回信息列表及可能发生的错误
func (m *Manager) ListBySession(sessionID uint) ([]Info, error) {
	records, err := m.store.ListTerminalsForSession(sessionID) // 记录列表、列出终端列表对应会话时的错误
	if err != nil {
		return nil, err
	}
	out := make([]Info, 0, len(records)) // 本次处理的输出
	// 遍历记录列表，逐项取得记录
	for _, record := range records {
		out = append(out, terminalInfo(record))
	}
	return out, nil
}

// ReadOutput 从字节游标之后读取已保存的终端输出
// 接收终端编号、游标，返回终端输出分片列表及可能发生的错误
func (m *Manager) ReadOutput(terminalID string, cursor int64) ([]data_models.TerminalOutputChunk, error) {
	return m.store.ReadTerminalOutput(terminalID, cursor)
}

// ReadTerminalOutput 以智能体工具约定的方法名暴露 ReadOutput
// 接收本次调用的上下文、终端编号、游标，返回终端输出分片列表及可能发生的错误
func (m *Manager) ReadTerminalOutput(ctx context.Context, terminalID string, cursor int64) ([]data_models.TerminalOutputChunk, error) {
	_ = ctx
	return m.ReadOutput(terminalID, cursor)
}

// Wait 阻塞到终端退出，并返回最后保存的元数据
// 接收终端编号，返回信息及可能发生的错误
func (m *Manager) Wait(terminalID string) (Info, error) {
	m.mu.RLock()
	sess := m.sessions[terminalID] // 会话
	m.mu.RUnlock()
	if sess == nil {
		record, err := m.store.GetTerminal(terminalID) // 记录、获取终端时的错误
		if err != nil {
			return Info{}, err
		}
		return terminalInfo(*record), nil
	}
	<-sess.proc.Done()
	record, err := m.store.GetTerminal(terminalID) // 记录、获取终端时的错误
	if err != nil {
		return sess.info, err
	}
	return terminalInfo(*record), nil
}

// WaitTerminal 以智能体工具约定的方法名暴露 Wait
// 接收本次调用的上下文、终端编号，返回信息及可能发生的错误
func (m *Manager) WaitTerminal(ctx context.Context, terminalID string) (Info, error) {
	_ = ctx
	return m.Wait(terminalID)
}

// drain 持久化 PTY 输出块，并发送终端生命周期事件
// 接收唯一标识、会话，无返回值
func (m *Manager) drain(id string, sess *session) {
	// 遍历输出，逐项取得分片
	for chunk := range sess.proc.Output() {
		stored, err := m.store.AppendTerminalOutput(id, string(chunk)) // 已持久化、追加终端输出时的错误
		if err != nil {
			continue
		}
		if m.emitOutput != nil {
			// 持有会话锁时读取可变的会话字段，确保事件中的可见性
			// 与 SetTerminalVisible 的更新保持一致
			sess.mu.Lock()
			sessionID := sess.info.SessionID // 会话编号
			visible := sess.info.Visible     // 可见
			sess.mu.Unlock()
			m.emitOutput(OutputEvent{
				TerminalID: id,
				SessionID:  sessionID,
				Cursor:     stored.CursorEnd,
				Data:       stored.Data,
				Status:     "active",
				Visible:    visible,
			})
		}
	}
	exitCode, waitErr := sess.proc.Wait() // 退出代码、等待任务结束时的错误
	status := "done"                      // 状态
	if waitErr != nil || exitCode != 0 {
		status = "error"
	}
	_ = m.store.FinishTerminal(id, status, exitCode)

	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()

	if m.emitStatus != nil {
		errString := "" // 错误String
		if waitErr != nil {
			errString = waitErr.Error()
		}
		m.emitStatus(StatusEvent{
			TerminalID: id,
			SessionID:  sess.info.SessionID,
			Status:     status,
			ExitCode:   exitCode,
			Error:      errString,
		})
	}
}

// terminalInfo 将存储模型转换为 API 使用的终端元数据
// 接收记录，返回信息
func terminalInfo(record data_models.Terminal) Info {
	return Info{
		ID:            record.TerminalID,
		SessionID:     record.SessionID,
		MessageID:     record.MessageID,
		ToolCallID:    record.ToolCallID,
		Title:         record.Title,
		Command:       record.Command,
		Args:          record.Args,
		Cwd:           record.Cwd,
		Status:        record.Status,
		Visible:       record.Visible,
		PID:           record.PID,
		ExitCode:      record.ExitCode,
		CurrentCursor: record.CurrentCursor,
	}
}

// defaultID 为终端存储记录生成低碰撞概率的 ID
// 返回文本
func defaultID() string {
	return fmt.Sprintf("term_%d", time.Now().UnixNano())
}

// 定义PTY进程的调用契约
type ptyProcess interface {
	// 处理输出，返回字节数据通道
	Output() <-chan []byte
	// 写入PTY进程，接收字节数据，返回操作错误
	Write([]byte) error
	// 处理调整尺寸，接收行数、列数，返回操作错误
	Resize(rows, cols uint16) error
	// 关闭当前资源，返回操作错误
	Close() error
	// 等待任务结束，返回整数及可能发生的错误
	Wait() (int, error)
	// 处理完成，返回struct{} 对应的数据通道
	Done() <-chan struct{}
}

// buildCommand 创建将在 PTY 中运行的 exec.Cmd
// 接收调用上下文、参数，返回命令
func buildCommand(_ context.Context, params CreateParams) *exec.Cmd {
	cmd := exec.Command(params.Command, params.Args...) // 命令
	if params.Env != nil {
		cmd.Env = params.Env
	}
	if params.Cwd != "" {
		cmd.Dir = params.Cwd
	}
	return cmd
}
