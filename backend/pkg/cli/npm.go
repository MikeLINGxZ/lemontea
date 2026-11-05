package cli

import (
	"context"
	"errors"
	"path/filepath"
)

// NpmInstallParams 描述一次 npm 安装调用
type NpmInstallParams struct {
	NodePath  string   // 内置 Node.js 可执行文件的绝对路径，留待直接调用时使用
	NpmPath   string   // 内置 npm 可执行文件的绝对路径
	Source    string   // npm 包名（如 "lark-cli" 或 "@scope/pkg"）或本地目录绝对路径
	TargetDir string   // 安装目录 {data_dir}/plugins/cli/{name}，用作 --prefix 的值
	ParentEnv []string // 通常来自 os.Environ()，会按 BuildEnv 规则清理环境变量
}

// NpmInstall 使用内置 npm 执行 `npm install <source> --prefix <targetDir> --no-audit --no-fund`
// 子进程继承清除了 NPM_CONFIG_PREFIX 和 NODE_PATH 的 ParentEnv
// 以免用户级 npm 配置改变安装位置
// 接收本次调用的上下文、npm安装参数，返回操作错误
func NpmInstall(ctx context.Context, p NpmInstallParams) error {
	if p.NodePath == "" {
		return errors.New("npm: NodePath required")
	}
	if p.NpmPath == "" {
		return errors.New("npm: NpmPath required")
	}
	if p.Source == "" {
		return errors.New("npm: Source required")
	}
	if p.TargetDir == "" {
		return errors.New("npm: TargetDir required")
	}

	// 环境变量
	env := BuildEnv(EnvParams{
		Isolation:  IsolationShared, // 保留 npm 访问网络和读取证书所需的用户环境，仅清理指定变量
		CLIDataDir: "",
		RuntimeBin: filepath.Dir(p.NpmPath),
		PluginBin:  filepath.Join(p.TargetDir, "node_modules", ".bin"),
		ParentEnv:  p.ParentEnv,
	})

	// 参数
	args := []string{
		"install",
		p.Source,
		"--prefix", p.TargetDir,
		"--no-audit",
		"--no-fund",
	}

	// 结果、运行当前数据时的错误
	res, err := Run(ctx, RunParams{
		Executable: p.NodePath,
		Argv:       append([]string{p.NpmPath}, args...),
		Env:        env,
		OutputMode: OutputText,
		TimeoutSec: MaxTimeoutSeconds, // 安装较大的 npm 包可能需要几分钟
	})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return errors.New("npm install failed: " + res.Stderr)
	}
	return nil
}
