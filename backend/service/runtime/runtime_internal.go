package runtime

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// RuntimeState 表示持久化的 Node.js 运行时安装状态
type RuntimeState struct {
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
	// 状态缺失
	StateMissing = "missing"
	// 状态Downloading
	StateDownloading = "downloading"
	// 状态Ready
	StateReady = "ready"
	// 状态失败
	StateFailed = "failed"
	// 状态待处理稍后
	StatePendingLater = "pending_later"
)

// runtimeBaseDir 返回保存 Node.js 运行时文件的目录绝对路径
func runtimeBaseDir() (string, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, NodeSubdir), nil
}

// runtimeStatePath 返回运行时状态文件 state.json 的绝对路径
func runtimeStatePath() (string, error) {
	base, err := runtimeBaseDir() // 基础、处理运行环境基础目录时的错误
	if err != nil {
		return "", err
	}
	return filepath.Join(base, RuntimeStateFileName), nil
}

// LoadPersistedState 读取 Node.js 运行时的 state.json，文件不存在时返回初始缺失状态
// 插件等其他服务可据此获取内置 Node.js 和 npm 的路径，无需依赖运行时服务实例
func LoadPersistedState() (RuntimeState, error) {
	path, err := runtimeStatePath() // 文件路径、处理运行环境状态路径时的错误
	if err != nil {
		return RuntimeState{}, err
	}
	bytes, err := os.ReadFile(path) // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			return RuntimeState{State: StateMissing}, nil
		}
		return RuntimeState{}, err
	}

	// 状态
	var state RuntimeState
	decodeErr := json.Unmarshal(bytes, &state) // 解析序列化内容时的错误
	if decodeErr != nil {
		return RuntimeState{}, decodeErr
	}
	if state.State == "" {
		state.State = StateMissing
	}
	return state, nil
}

// saveState 将运行时状态写入 state.json，必要时创建上级目录
// 接收状态，返回操作错误
func saveState(state RuntimeState) error {
	base, err := runtimeBaseDir() // 基础、处理运行环境基础目录时的错误
	if err != nil {
		return err
	}
	mkdirErr := os.MkdirAll(base, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return mkdirErr
	}

	state.UpdatedAt = time.Now()
	bytes, err := json.MarshalIndent(state, "", "  ") // 原始字节内容
	if err != nil {
		return err
	}
	path, err := runtimeStatePath() // 文件路径
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o644)
}

// platformSlug 返回当前操作系统在 Node.js 发行包文件名中使用的标识
func platformSlug() (string, error) {
	switch goRuntime.GOOS {
	case "darwin":
		return "darwin", nil
	case "linux":
		return "linux", nil
	case "windows":
		return "win", nil
	default:
		return "", fmt.Errorf("unsupported os: %s", goRuntime.GOOS)
	}
}

// archSlug 返回当前处理器架构在 Node.js 发行包文件名中使用的标识
func archSlug() (string, error) {
	switch goRuntime.GOARCH {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported arch: %s", goRuntime.GOARCH)
	}
}

// archiveExt 返回当前操作系统对应的发行包扩展名
func archiveExt() (string, error) {
	switch goRuntime.GOOS {
	case "darwin", "linux":
		return "tar.gz", nil
	case "windows":
		return "zip", nil
	default:
		return "", fmt.Errorf("unsupported os: %s", goRuntime.GOOS)
	}
}

// archiveURL 返回指定 Node.js 版本在当前平台的完整下载地址
// 接收版本，返回文本及可能发生的错误
func archiveURL(version string) (string, error) {
	osSlug, err := platformSlug() // 操作系统短标识、处理platform短标识时的错误
	if err != nil {
		return "", err
	}
	arch, err := archSlug() // 架构
	if err != nil {
		return "", err
	}
	ext, err := archiveExt() // 扩展名
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/node-%s-%s-%s.%s", NodeDistBaseURL, version, version, osSlug, arch, ext), nil
}

// sha256SumsURL 返回指定 Node.js 版本的 SHASUMS256.txt 下载地址
// 接收版本，返回文本
func sha256SumsURL(version string) string {
	return fmt.Sprintf("%s/%s/SHASUMS256.txt", NodeDistBaseURL, version)
}

// fetchSha256 下载指定地址的校验清单并返回目标发行包的 SHA-256 校验值
// 接收本次调用的上下文、校验和列表地址、压缩包名称，返回文本及可能发生的错误
func fetchSha256(ctx context.Context, sumsURL, archiveName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sumsURL, nil) // 待发送的请求、构造带上下文的网络请求时的错误
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req) // 远程服务响应
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body) // 扫描器
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text()) // 组成部分
		if len(parts) == 2 && parts[1] == archiveName {
			return parts[0], nil
		}
	}
	checkErr := scanner.Err() // 上下文取消或超时的错误
	if checkErr != nil {
		return "", checkErr
	}
	return "", fmt.Errorf("checksum not found for %s", archiveName)
}

// downloadArchive 将指定地址的发行包流式写入 dest，并返回十六进制 SHA-256 摘要
// 接收本次调用的上下文、地址、目标路径、报告，返回文本及可能发生的错误
// 构造带上下文的网络请求
// 创建所需目录
// 构造模型或应用实例
func downloadArchive(ctx context.Context, url, dest string, report func(received, total int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) // 待发送的请求、构造带上下文的网络请求时的错误
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req) // 远程服务响应
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	mkdirErr := os.MkdirAll(filepath.Dir(dest), 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return "", mkdirErr
	}
	out, err := os.Create(dest) // 本次处理的输出
	if err != nil {
		return "", err
	}
	defer out.Close()

	hasher := sha256.New()      // hasher 对应的数据
	total := resp.ContentLength // 总量
	// 已接收数量
	var received int64
	buf := make([]byte, 64*1024) // 缓冲区
	for {
		checkErr := ctx.Err() // 上下文取消或超时的错误
		if checkErr != nil {
			return "", checkErr
		}
		n, readErr := resp.Body.Read(buf) // 数量、读取当前数据时的错误
		if n > 0 {
			_, writeErr := out.Write(buf[:n]) // 写入当前数据时的错误
			if writeErr != nil {
				return "", writeErr
			}
			_, writeErr2 := hasher.Write(buf[:n]) // 写入错误
			if writeErr2 != nil {
				return "", writeErr2
			}
			received += int64(n)
			if report != nil {
				report(received, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// 将 tar.gz 归档解压到目标目录，接收上下文、源路径、目标目录和进度回调，返回顶层目录名或错误
// 按归档条目分别创建目录、普通文件和符号链接，并持续检查取消状态
func extractTarGz(ctx context.Context, src, destDir string, report func(received, total int64)) (string, error) {
	f, err := os.Open(src) // 当前文件、打开当前数据时的错误
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f) // gz 对应的数据
	if err != nil {
		return "", err
	}

	// 结束时关闭当前资源
	defer gz.Close()

	mkdirErr := os.MkdirAll(destDir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return "", mkdirErr
	}

	tr := tar.NewReader(gz) // 逐条读取 tar 归档内容的读取器
	// 压缩包内的顶层目录名
	var topDir string
	stat, _ := f.Stat()  // 文件信息
	total := stat.Size() // 总量

	for {
		checkErr := ctx.Err() // 上下文取消或超时的错误
		if checkErr != nil {
			return "", checkErr
		}
		header, err := tr.Next() // 页头、处理下一项时的错误
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if topDir == "" {
			topDir = strings.SplitN(header.Name, "/", 2)[0]
		}
		target := filepath.Join(destDir, header.Name) // 目标
		switch header.Typeflag {
		case tar.TypeDir:
			mkdirErr2 := os.MkdirAll(target, os.FileMode(header.Mode)) // 创建目录时的错误
			if mkdirErr2 != nil {
				return "", mkdirErr2
			}
		case tar.TypeReg:
			mkdirErr3 := os.MkdirAll(filepath.Dir(target), 0o755) // 创建目录时的错误
			if mkdirErr3 != nil {
				return "", mkdirErr3
			}
			// 本次处理的输出、以指定模式打开文件时的错误
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			_, copyErr := io.Copy(out, tr) // 复制当前数据时的错误
			if copyErr != nil {
				out.Close()
				return "", copyErr
			}
			closeErr := out.Close() // 关闭当前资源时的错误
			if closeErr != nil {
				return "", closeErr
			}
		case tar.TypeSymlink:
			_ = os.Symlink(header.Linkname, target)
		}
		if report != nil {
			pos, _ := f.Seek(0, io.SeekCurrent) // 位置
			report(pos, total)
		}
	}

	return topDir, nil
}

// 将 ZIP 归档解压到目标目录，接收上下文、源路径、目标目录和进度回调，返回顶层目录名或错误
// 按归档条目在目标目录下创建目录和文件，并通过回调报告进度
func extractZip(ctx context.Context, src, destDir string, report func(received, total int64)) (string, error) {
	r, err := zip.OpenReader(src) // 读取源、打开Reader 对应的数据时的错误
	if err != nil {
		return "", err
	}

	// 结束时关闭当前资源
	defer r.Close()

	mkdirErr := os.MkdirAll(destDir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return "", mkdirErr
	}

	// 压缩包内的顶层目录名
	var topDir string
	total := int64(len(r.File)) // 总量
	// 遍历文件，逐项取得循环索引、文件
	for i, file := range r.File {
		checkErr := ctx.Err() // 上下文取消或超时的错误
		if checkErr != nil {
			return "", checkErr
		}
		if topDir == "" {
			topDir = strings.SplitN(file.Name, "/", 2)[0]
		}
		target := filepath.Join(destDir, file.Name) // 目标
		if file.FileInfo().IsDir() {
			mkdirErr2 := os.MkdirAll(target, 0o755) // 创建目录时的错误
			if mkdirErr2 != nil {
				return "", mkdirErr2
			}
			continue
		}
		mkdirErr3 := os.MkdirAll(filepath.Dir(target), 0o755) // 创建目录时的错误
		if mkdirErr3 != nil {
			return "", mkdirErr3
		}
		rc, err := file.Open() // rc 对应的数据、打开当前数据时的错误
		if err != nil {
			return "", err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode()) // 本次处理的输出
		if err != nil {
			rc.Close()
			return "", err
		}
		_, copyErr := io.Copy(out, rc) // 复制当前数据时的错误
		if copyErr != nil {
			rc.Close()
			out.Close()
			return "", copyErr
		}
		closeErr := rc.Close() // 关闭当前资源时的错误
		if closeErr != nil {
			out.Close()
			return "", closeErr
		}
		closeErr2 := out.Close() // 关闭错误
		if closeErr2 != nil {
			return "", closeErr2
		}
		if report != nil {
			report(int64(i+1), total)
		}
	}

	return topDir, nil
}
