package storage

import "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"

// 将一条会话消息写入数据库，接收消息，返回消息及可能发生的错误
func (s *Storage) CreateMessage(msg data_models.Message) (*data_models.Message, error) {
	err := s.sqliteDB.Create(&msg).Error // 创建当前数据时的错误
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// 将多条会话消息写入数据库，接收消息列表，返回操作错误
func (s *Storage) CreateMessages(msgs *[]data_models.Message) error {
	if len(*msgs) == 0 {
		return nil
	}
	return s.sqliteDB.Create(msgs).Error
}

// 按指定会话与分页条件查询消息，接收会话编号、偏移、上限，返回消息列表及可能发生的错误
func (s *Storage) ListMessagesForSession(sessionID uint, offset int, limit int) ([]data_models.Message, error) {
	// 消息列表
	var msgs []data_models.Message
	// 处理Where 对应的数据时的错误
	err := s.sqliteDB.Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Offset(offset).Limit(limit).
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// 统计指定会话中的消息数量，接收会话编号，返回整数及可能发生的错误
func (s *Storage) CountMessagesForSession(sessionID uint) (int64, error) {
	// 数量
	var count int64
	// 处理模型时的错误
	err := s.sqliteDB.Model(&data_models.Message{}).Where("session_id = ?", sessionID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 删除指定会话关联的消息，接收会话编号，返回操作错误
func (s *Storage) DeleteMessagesForSession(sessionID uint) error {
	return s.sqliteDB.Where("session_id = ?", sessionID).Delete(&data_models.Message{}).Error
}
