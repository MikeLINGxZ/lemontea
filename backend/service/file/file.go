package file

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/i18n"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file/file_dto"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// 描述文件使用的数据结构
type File struct {
	// Wails 应用实例
	wailsApp *application.App
}

// SelectFolder 打开目录选择对话框并返回所选目录
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (f *File) SelectFolder(ctx context.Context, input file_dto.SelectFolderInput) (*file_dto.SelectFolderOutput, error) {
	if input.FolderPath == "" {
		input.FolderPath = dir.HomeDir()
	}
	// 文件路径、以指定模式打开文件时的错误
	path, err := f.wailsApp.Dialog.OpenFile().
		SetTitle(i18n.TCurrent(i18n.TCurrent("select.folder", nil), nil)).
		SetDirectory(input.FolderPath).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()

	if err != nil || path == "" {
		return nil, ierror.Error(ierror.ErrFileSelectFolder, err)
	}

	return &file_dto.SelectFolderOutput{FolderPath: path}, nil
}

// SelectFile 打开文件选择对话框并返回所选文件
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (f *File) SelectFile(ctx context.Context, input file_dto.SelectFileInput) (*file_dto.SelectFileOutput, error) {
	if input.DefaultFolderPath == "" {
		input.DefaultFolderPath = dir.HomeDir()
	}
	// 文件路径、以指定模式打开文件时的错误
	path, err := f.wailsApp.Dialog.OpenFile().
		SetTitle(i18n.TCurrent("select.file", nil)).
		SetDirectory(input.DefaultFolderPath).
		CanChooseFiles(true).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, ierror.Error(ierror.ErrFileSelectFile, err)
	}

	return &file_dto.SelectFileOutput{FilePath: path}, nil
}

// OpenFile 使用系统默认应用打开指定文件
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (f *File) OpenFile(ctx context.Context, input file_dto.OpenFileInput) (*file_dto.OpenFileOutput, error) {
	err := f.wailsApp.Browser.OpenFile(input.Path) // 以指定模式打开文件时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrFileOpen, err)
	}
	return &file_dto.OpenFileOutput{}, nil
}

// SaveTempFile 解码 Base64 数据并写入应用的临时目录
// 写入成功时返回文件的绝对路径
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (f *File) SaveTempFile(ctx context.Context, input file_dto.SaveTempFileInput) (*file_dto.SaveTempFileOutput, error) {
	data, err := base64.StdEncoding.DecodeString(input.Data) // 待处理的数据、解码String 对应的数据时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrFileSaveTempFile, err)
	}

	tmpDir := filepath.Join(os.TempDir(), "lemontea") // 临时目录
	mkdirErr := os.MkdirAll(tmpDir, 0o755)            // 创建所需目录时的错误
	if mkdirErr != nil {
		return nil, ierror.Error(ierror.ErrFileSaveTempFile, mkdirErr)
	}

	safeName := filepath.Base(input.Name) // 安全名称
	if safeName == "" || safeName == "." {
		safeName = "file"
	}
	name := fmt.Sprintf("%d-%s", time.Now().UnixNano(), safeName) // 当前名称
	path := filepath.Join(tmpDir, name)                           // 文件路径

	writeErr := os.WriteFile(path, data, 0o644) // 写入文件内容时的错误
	if writeErr != nil {
		return nil, ierror.Error(ierror.ErrFileSaveTempFile, writeErr)
	}

	return &file_dto.SaveTempFileOutput{FilePath: path}, nil
}
