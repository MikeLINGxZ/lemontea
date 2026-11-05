package storage

import (
	"path/filepath"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 描述存储使用的数据结构
type Storage struct {
	// SQLite数据库
	sqliteDB *gorm.DB
}

// NewStorage 打开应用的 SQLite 数据库并自动迁移所有模型
// 返回存储及可能发生的错误
func NewStorage() (*Storage, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, dir.DataBaseFileName)        // 数据库文件路径
	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{}) // GORM数据库
	if err != nil {
		return nil, err
	}

	return NewStorageFromDB(gormDB)
}

// NewStorageFromDB 对现有的 gorm.DB 句柄执行自动迁移后将其封装为存储对象
// 测试可以通过此入口注入内存数据库
// 接收数据库连接，返回存储及可能发生的错误
func NewStorageFromDB(db *gorm.DB) (*Storage, error) {
	// 处理自动迁移时的错误
	err := db.AutoMigrate(
		&data_models.Provider{},
		&data_models.ProviderDefaultModel{},
		&data_models.Model{},
		&data_models.Session{},
		&data_models.TaskState{},
		&data_models.Notification{},
		&data_models.Message{},
		&data_models.Terminal{},
		&data_models.TerminalOutputChunk{},
		&data_models.Memory{},
		&data_models.MemoryEmbedding{},
	)
	if err != nil {
		return nil, err
	}
	return &Storage{sqliteDB: db}, nil
}
