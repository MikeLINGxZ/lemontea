package notification

import (
	"context"
	"errors"
	"sync"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/notification/notification_dto"
)

// 描述等待结果使用的数据结构
type waitResult struct {
	// 操作错误
	err error
}

var (
	// 等待项互斥锁
	waitersMu sync.Mutex
	// 等待项
	waiters = map[uint]chan waitResult{}
)

// NotifyAttention 创建 needs_attention 通知并登记等待方，等待用户操作
// 接收本次调用的上下文、会话编号、标题、消息，返回无符号整数及可能发生的错误
func (n *Notification) NotifyAttention(ctx context.Context, sessionID uint, title, message string) (uint, error) {
	// 本次处理的输出、通知当前数据时的错误
	out, err := n.Notify(ctx, notification_dto.NotifyInput{
		SessionID: sessionID,
		Kind:      "needs_attention",
		Title:     title,
		Message:   message,
	})
	if err != nil {
		return 0, err
	}

	waitersMu.Lock()
	waiters[out.NotificationID] = make(chan waitResult, 1)
	waitersMu.Unlock()
	return out.NotificationID, nil
}

// WaitForResolution 阻塞等待通知被接受、拒绝或删除，也可因超时退出
// 接收本次调用的上下文、通知编号，返回操作错误
func (n *Notification) WaitForResolution(ctx context.Context, notificationID uint) error {
	waitersMu.Lock()
	ch, ok := waiters[notificationID] // 字符、是否成功取得对应值
	waitersMu.Unlock()
	if !ok {
		return errors.New("no waiter registered")
	}

	select {
	case /* 本次操作的结果 */ result := <-ch:
		return result.err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Minute):
		return errors.New("attention request timed out")
	}
}

// emit 在 Wails 应用句柄可用时发送全局事件
// 接收当前名称、载荷，无返回值
func (n *Notification) emit(name string, payload any) {
	if n.wailsApp != nil {
		n.wailsApp.Event.Emit(name, payload)
	}
}

// dtoFromModel 将通知存储模型转换为对外返回的数据
// 接收模型，返回通知条目
func dtoFromModel(model data_models.Notification) notification_dto.NotificationItem {
	return notification_dto.NotificationItem{
		ID:         model.ID,
		SessionID:  model.SessionID,
		Kind:       model.Kind,
		Title:      model.Title,
		Message:    model.Message,
		CreatedAt:  model.CreatedAt,
		ResolvedAt: model.ResolvedAt,
	}
}

// wakeWaiter 将处理结果交给等待方并移除其登记项
// 接收通知编号、操作错误，无返回值
func wakeWaiter(notificationID uint, err error) {
	waitersMu.Lock()
	ch, ok := waiters[notificationID] // 字符、是否成功取得对应值
	if ok {
		delete(waiters, notificationID)
	}
	waitersMu.Unlock()
	if ok {
		ch <- waitResult{err: err}
		close(ch)
	}
}
