package agent

import (
	"context"
	"sync"
)

// 描述当前流式输出使用的数据结构
type activeStream struct {
	// 取消上下文的回调
	cancel context.CancelFunc
	// 确认字符
	confirmCh chan ConfirmResponse
}

// ConfirmResponse 保存用户对工具确认请求的回复
type ConfirmResponse struct {
	// 已同意
	Approved bool
	// 消息
	Message string
	// 操作
	Action string
}

// 描述流式输出管理器使用的数据结构
type StreamManager struct {
	// 保护共享状态的互斥锁
	mu sync.RWMutex
	// 流状态表
	streams map[uint]*activeStream
}

// 构建流式输出管理器，返回流式输出管理器
func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[uint]*activeStream),
	}
}

// 启动任务，接收父级上下文、会话编号，返回调用上下文、取消函数
func (sm *StreamManager) Start(parentCtx context.Context, sessionID uint) (context.Context, context.CancelFunc) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if /* 已存在、是否成功取得对应值 */ existing, ok := sm.streams[sessionID]; ok {
		existing.cancel()
	}

	ctx, cancel := context.WithCancel(parentCtx) // 本次调用的上下文、用于取消本次操作的回调
	sm.streams[sessionID] = &activeStream{
		cancel:    cancel,
		confirmCh: make(chan ConfirmResponse, 1),
	}
	return ctx, cancel
}

// 停止流式输出管理器，接收会话编号，无返回值
func (sm *StreamManager) Stop(sessionID uint) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if /* 当前值、是否成功取得对应值 */ s, ok := sm.streams[sessionID]; ok {
		s.cancel()
		close(s.confirmCh)
		delete(sm.streams, sessionID)
	}
}

// 移除流式输出管理器，接收会话编号，无返回值
func (sm *StreamManager) Remove(sessionID uint) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, sessionID)
}

// 判断是否为当前，接收会话编号，返回是否满足条件
func (sm *StreamManager) IsActive(sessionID uint) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, ok := sm.streams[sessionID] // 是否成功取得对应值
	return ok
}

// 发送确认响应，接收会话编号、远程服务响应，返回是否满足条件
func (sm *StreamManager) SendConfirmResponse(sessionID uint, resp ConfirmResponse) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.streams[sessionID] // 当前值、是否成功取得对应值
	if !ok {
		return false
	}
	select {
	case s.confirmCh <- resp:
		return true
	default:
		return false
	}
}

// 等待对应确认，接收本次调用的上下文、会话编号，返回确认响应及可能发生的错误
func (sm *StreamManager) WaitForConfirm(ctx context.Context, sessionID uint) (ConfirmResponse, error) {
	sm.mu.RLock()
	s, ok := sm.streams[sessionID] // 当前值、是否成功取得对应值
	sm.mu.RUnlock()

	if !ok {
		return ConfirmResponse{}, context.Canceled
	}

	select {
	case <-ctx.Done():
		return ConfirmResponse{}, ctx.Err()
	case /* 远程服务响应、通道展开 */ resp, chanOpen := <-s.confirmCh:
		if !chanOpen {
			return ConfirmResponse{}, context.Canceled
		}
		return resp, nil
	}
}
