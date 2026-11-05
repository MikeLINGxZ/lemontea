package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/runtime/runtime_dto"
)

// Runtime 管理本地 Node.js 运行时的下载和生命周期
type Runtime struct {
	// Wails 应用实例
	wailsApp *application.App

	// 保护共享状态的互斥锁
	mu sync.Mutex
	// 取消函数
	cancelFunc context.CancelFunc
	// 运行中
	running bool
}

// NewRuntime 创建运行时管理服务
// 返回运行环境
func NewRuntime() *Runtime {
	return &Runtime{}
}

// GetStatus 返回已持久化的 Node.js 运行时状态
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Runtime) GetStatus(ctx context.Context, input runtime_dto.GetStatusInput) (*runtime_dto.GetStatusOutput, error) {
	state, err := LoadPersistedState() // 状态、加载持久化状态时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrRuntimeReadState, err)
	}
	return &runtime_dto.GetStatusOutput{
		State:      state.State,
		Version:    state.Version,
		InstallDir: state.InstallDir,
		NodePath:   state.NodePath,
		NpmPath:    state.NpmPath,
		ErrorMsg:   state.ErrorMsg,
	}, nil
}

// progressPayload 携带运行时下载进度，通过 runtime.node.progress 事件发送
type progressPayload struct {
	// 阶段
	Phase string `json:"phase"`
	// 已接收数量
	Received int64 `json:"received"`
	// 总量
	Total int64 `json:"total"`
	// 百分比
	Percent int `json:"percent"`
}

// emitProgress 向前端发布运行时下载进度
// 接收阶段、已接收数量、总量，无返回值
func (s *Runtime) emitProgress(phase string, received, total int64) {
	percent := 0 // 百分比
	if total > 0 {
		percent = int(received * 100 / total)
	}
	if s.wailsApp != nil {
		s.wailsApp.Event.Emit("runtime.node.progress", progressPayload{
			Phase:    phase,
			Received: received,
			Total:    total,
			Percent:  percent,
		})
	}
}

// failState 将失败状态写入 state.json，并发送最后一条进度事件
// 接收消息，无返回值
func (s *Runtime) failState(msg string) {
	_ = saveState(RuntimeState{
		State:    StateFailed,
		Version:  NodeLTSVersion,
		ErrorMsg: msg,
	})
	s.emitProgress("verify", 0, 0)
}

// DownloadNode 异步启动 Node.js 下载后立即返回
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Runtime) DownloadNode(ctx context.Context, input runtime_dto.DownloadNodeInput) (*runtime_dto.DownloadNodeOutput, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return &runtime_dto.DownloadNodeOutput{}, nil
	}
	runCtx, cancel := context.WithCancel(context.Background()) // 运行上下文、用于取消本次操作的回调
	s.cancelFunc = cancel
	s.running = true
	s.mu.Unlock()

	go s.runDownload(runCtx)

	return &runtime_dto.DownloadNodeOutput{}, nil
}

// 下载安装应用内置的 Node.js 运行环境，接收用于取消的上下文，无返回值
// 先选定平台归档并校验下载内容，再解包并定位 Node.js 与 npm
// 解包后保存可执行文件路径和就绪状态，失败或取消时更新对应状态
func (s *Runtime) runDownload(ctx context.Context) {
	// 无论下载成功、失败还是取消，都清理本次任务的运行标志
	defer func() {
		s.mu.Lock()
		s.running = false
		s.cancelFunc = nil
		s.mu.Unlock()
	}()

	_ = saveState(RuntimeState{State: StateDownloading, Version: NodeLTSVersion})

	url, err := archiveURL(NodeLTSVersion) // 地址、处理压缩包地址时的错误
	if err != nil {
		s.failState(err.Error())
		return
	}
	archiveName := filepath.Base(url) // 压缩包名称

	// 从版本清单读取期望的校验和，供下载后核验
	expectedSum, err := fetchSha256(ctx, sha256SumsURL(NodeLTSVersion), archiveName)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			_ = saveState(RuntimeState{State: StateMissing, Version: NodeLTSVersion})
			return
		}
		s.failState(err.Error())
		return
	}

	base, err := runtimeBaseDir() // 基础
	if err != nil {
		s.failState(err.Error())
		return
	}
	archivePath := filepath.Join(base, archiveName) // 压缩包路径

	// 下载内容的实际校验和
	actualSum, err := downloadArchive(ctx, url, archivePath, func(received, total int64) {
		// 处理下载压缩包时调用的回调，接收已接收数量、总量，无返回值
		s.emitProgress("download", received, total)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			_ = saveState(RuntimeState{State: StateMissing, Version: NodeLTSVersion})
			_ = os.Remove(archivePath)
			return
		}
		s.failState(err.Error())
		return
	}
	if !strings.EqualFold(actualSum, expectedSum) {
		s.failState(fmt.Sprintf("checksum mismatch: expected %s got %s", expectedSum, actualSum))
		return
	}

	installRoot := filepath.Join(base, NodeLTSVersion) // 安装根目录
	_ = os.RemoveAll(installRoot)
	mkdirErr := os.MkdirAll(installRoot, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		s.failState(mkdirErr.Error())
		return
	}

	// 压缩包内的顶层目录名
	var topDir string
	ext, _ := archiveExt() // 扩展名
	switch ext {
	case "tar.gz":
		topDir, err = extractTarGz(ctx, archivePath, installRoot, func(received, total int64) {
			// 提取TarGz 对应的数据时调用的回调，接收已接收数量、总量，无返回值
			s.emitProgress("extract", received, total)
		})
	case "zip":
		topDir, err = extractZip(ctx, archivePath, installRoot, func(received, total int64) {
			// 提取Zip 对应的数据时调用的回调，接收已接收数量、总量，无返回值
			s.emitProgress("extract", received, total)
		})
	default:
		err = fmt.Errorf("unsupported ext: %s", ext)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			_ = saveState(RuntimeState{State: StateMissing, Version: NodeLTSVersion})
			return
		}
		s.failState(err.Error())
		return
	}

	s.emitProgress("verify", 1, 1)

	extractedRoot := filepath.Join(installRoot, topDir) // 已提取根目录
	// Node.js 可执行文件路径
	var nodePath string
	// npm 可执行文件路径
	var npmPath string
	if goRuntime.GOOS == "windows" {
		nodePath = filepath.Join(extractedRoot, "node.exe")
		npmPath = filepath.Join(extractedRoot, "npm.cmd")
	} else {
		nodePath = filepath.Join(extractedRoot, "bin", "node")
		npmPath = filepath.Join(extractedRoot, "bin", "npm")
	}

	_ = os.Remove(archivePath)

	// 保存当前状态时的错误
	saveStateErr := saveState(RuntimeState{
		State:       StateReady,
		Version:     NodeLTSVersion,
		InstallDir:  extractedRoot,
		NodePath:    nodePath,
		NpmPath:     npmPath,
		InstalledAt: time.Now(),
	})
	if saveStateErr != nil {
		s.failState(saveStateErr.Error())
		return
	}
}

// CancelDownload 取消正在进行的下载，没有下载任务时直接返回
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Runtime) CancelDownload(ctx context.Context, input runtime_dto.CancelDownloadInput) (*runtime_dto.CancelDownloadOutput, error) {
	s.mu.Lock()
	cancel := s.cancelFunc // 用于取消本次操作的回调
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return &runtime_dto.CancelDownloadOutput{}, nil
}

// MarkDownloadLater 记录用户稍后下载运行时的选择
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Runtime) MarkDownloadLater(ctx context.Context, input runtime_dto.MarkDownloadLaterInput) (*runtime_dto.MarkDownloadLaterOutput, error) {
	current, err := LoadPersistedState() // 当前、加载持久化状态时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrRuntimeReadState, err)
	}
	current.State = StatePendingLater
	current.Version = NodeLTSVersion
	current.ErrorMsg = ""
	saveStateErr := saveState(current) // 保存当前状态时的错误
	if saveStateErr != nil {
		return nil, ierror.Error(ierror.ErrRuntimeWriteState, saveStateErr)
	}
	return &runtime_dto.MarkDownloadLaterOutput{}, nil
}
