package terminal

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	pkgterminal "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/terminal"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/terminal/terminal_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

// Terminal 向前端提供交互式终端操作
type Terminal struct {
	// 管理器
	manager *pkgterminal.Manager
	// Wails 应用实例
	wailsApp *application.App
}

// NewTerminal 使用共享应用存储创建终端服务
// 接收状态仓库，返回终端
func NewTerminal(store *storage.Storage) *Terminal {
	return NewTerminalWithManager(pkgterminal.NewManager(store, nil))
}

// NewTerminalWithManager 使用已有的终端管理器创建终端服务
// 接收管理器，返回终端
func NewTerminalWithManager(manager *pkgterminal.Manager) *Terminal {
	return &Terminal{manager: manager}
}

// ListTerminals 返回指定聊天会话中可见且已持久化的终端会话
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (t *Terminal) ListTerminals(ctx context.Context, input terminal_dto.ListTerminalsInput) (*terminal_dto.ListTerminalsOutput, error) {
	_ = ctx
	items, err := t.manager.ListBySession(input.SessionID) // 条目列表、列出按会话时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrTerminalList, err)
	}
	return &terminal_dto.ListTerminalsOutput{Items: items}, nil
}

// ReadTerminalOutput 返回给定字节游标之后的终端输出片段
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (t *Terminal) ReadTerminalOutput(ctx context.Context, input terminal_dto.ReadTerminalOutputInput) (*terminal_dto.ReadTerminalOutputOutput, error) {
	_ = ctx
	chunks, err := t.manager.ReadOutput(input.TerminalID, input.Cursor) // 分片列表、读取输出时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrTerminalReadOutput, err)
	}
	return &terminal_dto.ReadTerminalOutputOutput{Chunks: chunks}, nil
}

// WriteTerminalInput 将用户键入或粘贴的内容发送到活动终端
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (t *Terminal) WriteTerminalInput(ctx context.Context, input terminal_dto.WriteTerminalInputInput) (*terminal_dto.WriteTerminalInputOutput, error) {
	_ = ctx
	err := t.manager.Write(input.TerminalID, input.Data) // 写入当前数据时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrTerminalWriteInput, err)
	}
	return &terminal_dto.WriteTerminalInputOutput{}, nil
}

// ResizeTerminal 更新 PTY 尺寸，使其与前端终端视口一致
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (t *Terminal) ResizeTerminal(ctx context.Context, input terminal_dto.ResizeTerminalInput) (*terminal_dto.ResizeTerminalOutput, error) {
	_ = ctx
	err := t.manager.Resize(input.TerminalID, input.Rows, input.Cols) // 处理调整尺寸时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrTerminalResize, err)
	}
	return &terminal_dto.ResizeTerminalOutput{}, nil
}
