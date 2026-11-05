package storage

import (
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
)

// CreateNotification 插入一条通知记录
// 接收通知，返回通知及可能发生的错误
func (s *Storage) CreateNotification(notification data_models.Notification) (*data_models.Notification, error) {
	err := s.sqliteDB.Create(&notification).Error // 创建当前数据时的错误
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// ListNotifications 返回近期通知，可选择是否包含已处理的通知
// 接收包含已处理，返回通知列表及可能发生的错误
func (s *Storage) ListNotifications(includeResolved bool) ([]data_models.Notification, error) {
	// 条目列表
	var items []data_models.Notification
	query := s.sqliteDB.Order("created_at DESC") // 搜索词
	if !includeResolved {
		query = query.Where("resolved_at IS NULL")
	}
	err := query.Limit(200).Find(&items).Error // 处理上限时的错误
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ResolveNotification 记录通知的处理时间
// 接收唯一标识，返回操作错误
func (s *Storage) ResolveNotification(id uint) error {
	now := time.Now() // 操作发生时的时间
	return s.sqliteDB.Model(&data_models.Notification{}).Where("id = ?", id).Update("resolved_at", &now).Error
}

// DeleteNotification 永久删除一条通知
// 接收唯一标识，返回操作错误
func (s *Storage) DeleteNotification(id uint) error {
	return s.sqliteDB.Delete(&data_models.Notification{}, id).Error
}
