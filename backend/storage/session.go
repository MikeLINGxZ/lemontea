package storage

import (
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gorm.io/gorm"
)

// CreateSession 插入一条会话记录
// 接收会话，返回会话及可能发生的错误
func (s *Storage) CreateSession(session data_models.Session) (*data_models.Session, error) {
	err := s.sqliteDB.Create(&session).Error // 创建当前数据时的错误
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSession 按主键加载一条会话记录
// 接收唯一标识，返回会话及可能发生的错误
func (s *Storage) GetSession(id uint) (*data_models.Session, error) {
	// 会话
	var session data_models.Session
	err := s.sqliteDB.First(&session, id).Error // 处理第一项时的错误
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessions 按时间从新到旧返回会话，可选择包含已隐藏的旧版后台会话
// 接收游标、上限、已收藏仅、包含隐藏，返回会话列表及可能发生的错误
func (s *Storage) ListSessions(cursor uint, limit int, starredOnly bool, includeHidden bool) ([]data_models.Session, error) {
	// 会话列表
	var sessions []data_models.Session
	q := s.sqliteDB.Order("updated_at DESC") // q 对应的数据
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	if starredOnly {
		q = q.Where("starred = ?", true)
	}
	if !includeHidden {
		q = q.Where("kind <> ? OR kind = ''", "background")
	}
	err := q.Limit(limit).Find(&sessions).Error // 处理上限时的错误
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// UpdateSessionTitle 更新已保存的会话标题
// 接收唯一标识、标题，返回操作错误
func (s *Storage) UpdateSessionTitle(id uint, title string) error {
	return s.sqliteDB.Model(&data_models.Session{}).Where("id = ?", id).Update("title", title).Error
}

// UpdateSessionStatus 更新已保存的会话状态
// 接收唯一标识、状态，返回操作错误
func (s *Storage) UpdateSessionStatus(id uint, status string) error {
	return s.sqliteDB.Model(&data_models.Session{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateSessionStarred 更新已保存的会话收藏标记
// 接收唯一标识、已收藏，返回操作错误
func (s *Storage) UpdateSessionStarred(id uint, starred bool) error {
	return s.sqliteDB.Model(&data_models.Session{}).Where("id = ?", id).Update("starred", starred).Error
}

// DeleteSession 在同一事务中删除会话及其所有消息
// 接收唯一标识，返回操作错误
func (s *Storage) DeleteSession(id uint) error {
	return s.sqliteDB.Transaction( /* 在同一个数据库事务中执行写入，失败时返回错误以回滚 */ func(tx *gorm.DB) error {
		err := tx.Where("session_id = ?", id).Delete(&data_models.Message{}).Error // 处理Where 对应的数据时的错误
		if err != nil {
			return err
		}
		return tx.Delete(&data_models.Session{}, id).Error
	})
}

// TouchSession 刷新 updated_at，保证近期会话按最新活动时间排序
// 接收唯一标识，返回操作错误
func (s *Storage) TouchSession(id uint) error {
	return s.sqliteDB.Model(&data_models.Session{}).Where("id = ?", id).Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
