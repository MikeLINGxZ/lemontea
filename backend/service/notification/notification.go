package notification

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/notification/notification_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

const (
	// 事件通知创建
	eventNotificationCreated = "notification.created"
	// 事件通知已处理
	eventNotificationResolved = "notification.resolved"
)

// Notification 为 Wails 提供全局通知的增删改查及等待机制
type Notification struct {
	// 状态仓库
	store *storage.Storage
	// Wails 应用实例
	wailsApp *application.App
}

// NewNotification 使用共享存储创建通知服务
// 接收状态仓库，返回通知
func NewNotification(store *storage.Storage) *Notification {
	return &Notification{store: store}
}

// Notify 创建通知记录并广播创建事件
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (n *Notification) Notify(ctx context.Context, input notification_dto.NotifyInput) (*notification_dto.NotifyOutput, error) {
	// 创建、创建通知时的错误
	created, err := n.store.CreateNotification(data_models.Notification{
		SessionID: input.SessionID,
		Kind:      input.Kind,
		Title:     input.Title,
		Message:   input.Message,
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrNotificationCreate, err)
	}
	n.emit(eventNotificationCreated, dtoFromModel(*created))
	return &notification_dto.NotifyOutput{NotificationID: created.ID}, nil
}

// ListNotifications 按创建时间顺序返回当前通知列表
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (n *Notification) ListNotifications(ctx context.Context, input notification_dto.ListNotificationsInput) (*notification_dto.ListNotificationsOutput, error) {
	items, err := n.store.ListNotifications(input.IncludeResolved) // 条目列表、列出通知列表时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrNotificationList, err)
	}

	result := make([]notification_dto.NotificationItem, 0, len(items)) // 处理make 对应的数据得到的结果
	// 遍历条目列表，逐项取得当前条目
	for _, item := range items {
		result = append(result, dtoFromModel(item))
	}
	return &notification_dto.ListNotificationsOutput{Items: result}, nil
}

// ResolveNotification 将通知标记为已处理，发送事件并通知等待方处理成功
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (n *Notification) ResolveNotification(ctx context.Context, input notification_dto.ResolveNotificationInput) (*notification_dto.ResolveNotificationOutput, error) {
	err := n.store.ResolveNotification(input.ID) // 解析通知时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrNotificationResolve, err)
	}
	wakeWaiter(input.ID, nil)
	n.emit(eventNotificationResolved, map[string]uint{"id": input.ID})
	return &notification_dto.ResolveNotificationOutput{}, nil
}

// RejectNotification 将通知标记为已处理，并向等待方返回拒绝错误
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (n *Notification) RejectNotification(ctx context.Context, input notification_dto.ResolveNotificationInput) (*notification_dto.ResolveNotificationOutput, error) {
	err := n.store.ResolveNotification(input.ID) // 解析通知时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrNotificationResolve, err)
	}
	wakeWaiter(input.ID, errors.New("user rejected"))
	n.emit(eventNotificationResolved, map[string]uint{"id": input.ID})
	return &notification_dto.ResolveNotificationOutput{}, nil
}

// DismissNotification 永久删除指定通知记录
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (n *Notification) DismissNotification(ctx context.Context, input notification_dto.DismissNotificationInput) (*notification_dto.DismissNotificationOutput, error) {
	err := n.store.DeleteNotification(input.ID) // 删除通知时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrNotificationDismiss, err)
	}
	wakeWaiter(input.ID, errors.New("notification dismissed"))
	n.emit(eventNotificationResolved, map[string]uint{"id": input.ID})
	return &notification_dto.DismissNotificationOutput{}, nil
}
