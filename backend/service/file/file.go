package file

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

// ReadMediaPreview 读取指定路径的本地图片或视频，返回供对话预览使用的数据 URL
// 文件不可用、格式不支持或超出大小上限时返回空字符串，读取异常时返回错误
func (f *File) ReadMediaPreview(ctx context.Context, path string) (string, error) {
	const maxPreviewBytes = 20 * 1024 * 1024 // 媒体预览大小上限，与聊天附件上限一致
	info, err := os.Stat(path)               // 预览文件信息及查询错误
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxPreviewBytes {
		return "", nil
	}

	mediaFile, err := os.Open(path) // 本地媒体文件及打开错误
	if err != nil {
		slog.WarnContext(ctx, "打开媒体附件失败", "error", err)
		return "", err
	}
	defer mediaFile.Close()

	data, err := io.ReadAll(io.LimitReader(mediaFile, maxPreviewBytes+1)) // 有大小限制的媒体字节及读取错误
	if err != nil {
		slog.WarnContext(ctx, "读取媒体附件失败", "error", err)
		return "", err
	}
	if len(data) > maxPreviewBytes {
		return "", nil
	}

	// 根据文件内容识别可在应用内预览的媒体
	mimeType := http.DetectContentType(data) // 根据文件字节检测出的媒体类型
	if len(data) >= 12 && string(data[4:12]) == "ftypqt  " {
		mimeType = "video/quicktime"
	} else if len(data) >= 12 && string(data[4:12]) == "ftypM4V " {
		mimeType = "video/mp4"
	}
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "video/mp4", "video/webm", "video/quicktime":
		return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
	}
	return "", nil
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
