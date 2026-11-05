package storage

import (
	"errors"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gorm.io/gorm"
)

// CreateTerminal 以默认的启动元数据保存一条终端会话记录
// 接收终端，返回终端及可能发生的错误
func (s *Storage) CreateTerminal(term data_models.Terminal) (*data_models.Terminal, error) {
	if term.StartedAt.IsZero() {
		term.StartedAt = time.Now()
	}
	if term.Status == "" {
		term.Status = "active"
	}
	err := s.sqliteDB.Create(&term).Error // 创建当前数据时的错误
	if err != nil {
		return nil, err
	}
	return &term, nil
}

// GetTerminal 按稳定的终端 ID 返回终端会话
// 接收终端编号，返回终端及可能发生的错误
func (s *Storage) GetTerminal(terminalID string) (*data_models.Terminal, error) {
	// 终端
	var term data_models.Terminal
	err := s.sqliteDB.Where("terminal_id = ?", terminalID).First(&term).Error // 处理Where 对应的数据时的错误
	if err != nil {
		return nil, err
	}
	return &term, nil
}

// ListTerminalsForSession 返回聊天会话下的所有终端会话
// 接收会话编号，返回终端列表及可能发生的错误
func (s *Storage) ListTerminalsForSession(sessionID uint) ([]data_models.Terminal, error) {
	// 检索词
	var terms []data_models.Terminal
	// 处理Where 对应的数据时的错误
	err := s.sqliteDB.Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&terms).Error
	if err != nil {
		return nil, err
	}
	return terms, nil
}

// UpdateTerminalVisibility 修改已保存的终端在聊天中是否可见
// 接收终端编号、可见、标题，返回操作错误
func (s *Storage) UpdateTerminalVisibility(terminalID string, visible bool, title string) error {
	updates := map[string]any{"visible": visible} // updates 对应的数据
	if title != "" {
		updates["title"] = title
	}
	return s.sqliteDB.Model(&data_models.Terminal{}).
		Where("terminal_id = ?", terminalID).
		Updates(updates).Error
}

// FinishTerminal 记录终端进程的结束状态
// 接收终端编号、状态、退出代码，返回操作错误
func (s *Storage) FinishTerminal(terminalID string, status string, exitCode int) error {
	now := time.Now() // 操作发生时的时间
	return s.sqliteDB.Model(&data_models.Terminal{}).
		Where("terminal_id = ?", terminalID).
		Updates(map[string]any{
			"status":    status,
			"exit_code": exitCode,
			"ended_at":  &now,
		}).Error
}

// AppendTerminalOutput 保存一段终端输出并推进字节游标
// 接收终端编号、待处理的数据，返回终端输出分片及可能发生的错误
func (s *Storage) AppendTerminalOutput(terminalID string, data string) (*data_models.TerminalOutputChunk, error) {
	// 分片
	var chunk data_models.TerminalOutputChunk
	// 处理Transaction 对应的数据时的错误
	err := s.sqliteDB.Transaction( /* 在同一个数据库事务中执行写入，失败时返回错误以回滚 */ func(tx *gorm.DB) error {
		// 终端
		var term data_models.Terminal
		checkErr := tx.Where("terminal_id = ?", terminalID).First(&term).Error // 处理Where 对应的数据时的错误
		if checkErr != nil {
			return checkErr
		}

		// 最后一个
		var last data_models.TerminalOutputChunk
		// 处理Where 对应的数据时的错误
		err := tx.Where("terminal_id = ?", terminalID).Order("seq DESC").First(&last).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 游标以字节为单位，即使输出块包含多字节文本，
		// 前端轮询也可以从准确的输出块边界继续读取
		nextSeq := last.Seq + 1
		start := term.CurrentCursor             // 开始
		end := start + int64(len([]byte(data))) // 结束
		chunk = data_models.TerminalOutputChunk{
			TerminalID:  terminalID,
			Seq:         nextSeq,
			CursorStart: start,
			CursorEnd:   end,
			Data:        data,
		}
		checkErr2 := tx.Create(&chunk).Error // check错误
		if checkErr2 != nil {
			return checkErr2
		}
		return tx.Model(&term).Update("current_cursor", end).Error
	})
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

// ReadTerminalOutput 返回游标之后的输出块，必要时裁剪首块
// 接收终端编号、游标，返回终端输出分片列表及可能发生的错误
func (s *Storage) ReadTerminalOutput(terminalID string, cursor int64) ([]data_models.TerminalOutputChunk, error) {
	// 分片列表
	var chunks []data_models.TerminalOutputChunk
	// 处理Where 对应的数据时的错误
	err := s.sqliteDB.Where("terminal_id = ? AND cursor_end > ?", terminalID, cursor).
		Order("seq ASC").
		Find(&chunks).Error
	if err != nil {
		return nil, err
	}
	// 遍历分片列表，逐项取得循环索引
	for i := range chunks {
		if cursor > chunks[i].CursorStart && cursor < chunks[i].CursorEnd {
			// 只裁掉第一个与游标重叠的输出块中已读的字节，
			// 后续输出块原样返回
			offset := cursor - chunks[i].CursorStart
			data := []byte(chunks[i].Data) // 待处理的数据
			if offset < int64(len(data)) {
				chunks[i].Data = string(data[offset:])
				chunks[i].CursorStart = cursor
			}
		}
	}
	return chunks, nil
}
