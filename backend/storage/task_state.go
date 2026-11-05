package storage

import (
	"errors"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gorm.io/gorm"
)

// SaveTaskState 新增或更新任务会话的一组键值
// 接收会话编号、键、文本，返回操作错误
func (s *Storage) SaveTaskState(sessionID uint, key, value string) error {
	// 状态
	var state data_models.TaskState
	// 处理Where 对应的数据时的错误
	err := s.sqliteDB.Where("session_id = ? AND key = ?", sessionID, key).First(&state).Error
	if err == nil {
		return s.sqliteDB.Model(&state).Update("value", value).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return s.sqliteDB.Create(&data_models.TaskState{
		SessionID: sessionID,
		Key:       key,
		Value:     value,
	}).Error
}

// LoadTaskState 读取任务会话中已保存的一组键值
// 接收会话编号、键，返回文本、是否满足条件及可能发生的错误
func (s *Storage) LoadTaskState(sessionID uint, key string) (string, bool, error) {
	// 状态
	var state data_models.TaskState
	// 处理Where 对应的数据时的错误
	err := s.sqliteDB.Where("session_id = ? AND key = ?", sessionID, key).First(&state).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return state.Value, true, nil
}

// DeleteTaskState 删除任务会话中已保存的指定键
// 接收会话编号、键，返回操作错误
func (s *Storage) DeleteTaskState(sessionID uint, key string) error {
	return s.sqliteDB.Where("session_id = ? AND key = ?", sessionID, key).Delete(&data_models.TaskState{}).Error
}
