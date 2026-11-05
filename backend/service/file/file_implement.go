package file

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// 在 Wails 服务启动时保存运行环境并初始化当前服务，接收本次调用的上下文、选项列表，返回操作错误
func (f *File) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	f.wailsApp = application.Get()
	cleanTempDir(filepath.Join(os.TempDir(), "lemontea"))
	return nil
}

// cleanTempDir 删除指定目录下的普通文件
// 应用启动时清理旧文件，避免临时文件持续占用磁盘空间
// 接收目录，无返回值
func cleanTempDir(dir string) {
	entries, err := os.ReadDir(dir) // 条目列表、读取目录时的错误
	if err != nil {
		return
	}
	// 遍历条目列表，逐项取得e 对应的数据
	for _, e := range entries {
		if !e.IsDir() {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
