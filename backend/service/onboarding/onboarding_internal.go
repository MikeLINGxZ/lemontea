package onboarding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// 初始化文件名称
const initFileName = "init.json"

// 描述初始化状态使用的数据结构
type initState struct {
	// 已初始化
	Initialized bool `json:"initialized"`
	// 已完成时间
	CompletedAt time.Time `json:"completed_at"`
}

// initFilePath 返回数据目录下初始化标记文件的绝对路径
func initFilePath() (string, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, initFileName), nil
}

// isInitialized 返回 init.json 标记文件是否存在
func isInitialized() (bool, error) {
	path, err := initFilePath() // 文件路径、初始化文件路径时的错误
	if err != nil {
		return false, err
	}
	_, statErr := os.Stat(path) // 读取文件元数据时的错误
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return false, nil
		}
		return false, statErr
	}
	return true, nil
}

// markInitialized 将初始化完成时间写入 init.json 标记文件
// 返回操作错误
func markInitialized() error {
	path, err := initFilePath() // 文件路径、初始化文件路径时的错误
	if err != nil {
		return err
	}
	// 载荷
	payload, err := json.MarshalIndent(initState{
		Initialized: true,
		CompletedAt: time.Now(),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}
