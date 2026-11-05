package mcp

import (
	"io"
	"os"
	"path/filepath"
)

// copyDirectory 递归地将目录树复制到目标路径
// 接收来源目录、目标目录，返回操作错误
func copyDirectory(sourceDir string, targetDir string) error {
	removeErr := os.RemoveAll(targetDir) // 移除指定路径下的数据时的错误
	if removeErr != nil {
		return removeErr
	}
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		// 处理遍历时调用的回调，接收文件路径、信息、操作错误，返回操作错误
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path) // 相对路径路径
		if err != nil {
			return err
		}
		destPath := filepath.Join(targetDir, relPath) // 目标路径路径

		if info.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		mkdirErr := os.MkdirAll(filepath.Dir(destPath), 0o755) // 创建所需目录时的错误
		if mkdirErr != nil {
			return mkdirErr
		}

		sourceFile, err := os.Open(path) // 来源文件
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		targetFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644) // 目标文件
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, sourceFile)
		return err
	})
}
