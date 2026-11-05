package runtime_state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// StateSnapshot 表示已保存的 Node 运行时安装状态
type StateSnapshot struct {
	// 当前状态
	State string `json:"state"`
	// 版本
	Version string `json:"version"`
	// 安装目录
	InstallDir string `json:"install_dir"`
	// Node.js 可执行文件路径
	NodePath string `json:"node_path"`
	// npm 可执行文件路径
	NpmPath string `json:"npm_path"`
	// 错误消息
	ErrorMsg string `json:"error_msg"`
	// 已安装时间
	InstalledAt time.Time `json:"installed_at"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

const (
	// StateMissing 表示运行时尚未下载
	StateMissing = "missing"
	// NodeSubdir 是数据目录下存放 Node 运行时文件的子目录名称
	NodeSubdir = "runtime/node"
	// RuntimeStateFileName 是保存运行时下载状态的文件名
	RuntimeStateFileName = "state.json"
)

// LoadPersistedState 读取 runtime/node/state.json，文件不存在时返回初始的未安装状态
func LoadPersistedState() (StateSnapshot, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return StateSnapshot{}, err
	}
	path := filepath.Join(dataDir, NodeSubdir, RuntimeStateFileName) // 文件路径
	bytes, err := os.ReadFile(path)                                  // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			return StateSnapshot{State: StateMissing}, nil
		}
		return StateSnapshot{}, err
	}

	// 状态
	var state StateSnapshot
	decodeErr := json.Unmarshal(bytes, &state) // 解析序列化内容时的错误
	if decodeErr != nil {
		return StateSnapshot{}, decodeErr
	}
	if state.State == "" {
		state.State = StateMissing
	}
	return state, nil
}
