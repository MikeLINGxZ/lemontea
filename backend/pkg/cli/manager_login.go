//go:build !windows

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// StartLoginCommand 读取指定插件的清单，并根据 manifest.Isolation 构造环境变量
// 启动执行 `manifest.Executable + argv` 的 LoginSession
// argv 由调用方提供，通常取自 manifest.ResolveLoginSteps() 的单个步骤
// 由调用方编排多个步骤，服务层负责检测并发冲突
// 接收本次调用的上下文、当前名称、命令参数，返回登录会话及可能发生的错误
func (m *Manager) StartLoginCommand(ctx context.Context, name string, argv []string) (*LoginSession, error) {
	if len(argv) == 0 {
		return nil, errors.New("cli: login step has empty argv")
	}
	cliDataDir := filepath.Join(dir.CLIDataRoot(m.dataDir), name) // CLI数据目录
	installDir := filepath.Join(dir.CLIRoot(m.dataDir), name)     // 安装目录
	manifestPath := filepath.Join(cliDataDir, "manifest.json")    // 清单路径

	manifest, err := LoadManifest(manifestPath) // 清单、加载 CLI 扩展清单时的错误
	if err != nil {
		return nil, err
	}
	if manifest.Executable == "" {
		manifest, err = RepairManifestExecutable(manifestPath, installDir, manifest)
		if err != nil {
			return nil, errors.New("cli: manifest has no executable; reinstall the plugin")
		}
	}

	// 环境变量
	env := BuildEnv(EnvParams{
		Isolation:  manifest.Isolation,
		CLIDataDir: cliDataDir,
		RuntimeBin: filepath.Dir(m.nodePath),
		PluginBin:  filepath.Join(installDir, "node_modules", ".bin"),
		ParentEnv:  os.Environ(),
	})

	// 将 Cwd 固定在 installDir，使登录脚本能稳定定位相对路径中的文件
	// 例如配置模板和随包附带的 Node.js 模块，与应用的启动目录无关
	// RunTool 的 argv 完全由清单解析，因此无需设置 Cwd
	return StartLoginSession(ctx, LoginSessionParams{
		Executable: manifest.Executable,
		Argv:       argv,
		Env:        env,
		Cwd:        installDir,
	})
}

// StartLogin 执行解析出的第一个登录步骤，供按单命令登录的既有调用方使用
// 多步骤流程应使用 manifest.ResolveLoginSteps() 获取步骤
// 并对每个步骤调用一次 StartLoginCommand
// 未配置步骤时返回 errors.New("cli: manifest has no login_command")
// 接收本次调用的上下文、当前名称，返回登录会话及可能发生的错误
func (m *Manager) StartLogin(ctx context.Context, name string) (*LoginSession, error) {
	cliDataDir := filepath.Join(dir.CLIDataRoot(m.dataDir), name) // CLI数据目录
	manifestPath := filepath.Join(cliDataDir, "manifest.json")    // 清单路径
	manifest, err := LoadManifest(manifestPath)                   // 清单、加载 CLI 扩展清单时的错误
	if err != nil {
		return nil, err
	}
	steps := manifest.ResolveLoginSteps() // 步骤列表
	if len(steps) == 0 {
		return nil, errors.New("cli: manifest has no login_command")
	}
	return m.StartLoginCommand(ctx, name, steps[0])
}
