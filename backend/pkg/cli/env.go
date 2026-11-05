package cli

import (
	"path/filepath"
	"runtime"
	"strings"
)

// EnvParams 汇集构造隔离或共享环境所需的输入
type EnvParams struct {
	// 隔离
	Isolation  IsolationMode
	CLIDataDir string   // {data_dir}/plugins/cli_data/{name} 的绝对路径
	RuntimeBin string   // 内置运行时的 bin 目录绝对路径，优先从此处查找 npm 和 npx
	PluginBin  string   // {install_dir}/node_modules/.bin 的绝对路径
	ParentEnv  []string // 父进程环境变量，通常来自 os.Environ()
}

// strippedEnvKeys 列出从继承环境中删除的变量
// 避免宿主的 npm 配置或 NODE_PATH 被带入隔离子进程
var strippedEnvKeys = []string{"NPM_CONFIG_PREFIX", "NODE_PATH"}

// overriddenKeysIsolated 列出 BuildEnv 在隔离模式下一律替换的环境变量
var overriddenKeysIsolated = []string{
	"HOME",
	"USERPROFILE",
	"XDG_CONFIG_HOME",
	"XDG_DATA_HOME",
	"XDG_CACHE_HOME",
}

// BuildEnv 返回可赋给 exec.Cmd 的 CLI 子进程环境变量切片
// 隔离模式将 HOME、USERPROFILE 和 XDG_* 指向 CLIDataDir 的子目录
// 共享模式保留这些变量在父进程中的值
// 两种模式均将 RuntimeBin 和 PluginBin 添加到 PATH 开头，并移除 strippedEnvKeys 中的变量
// 接收环境变量参数，返回文本列表
func BuildEnv(p EnvParams) []string {
	out := make([]string, 0, len(p.ParentEnv)+8) // 本次处理的输出

	skipKeys := map[string]struct{}{} // 跳过键集合
	// 遍历已移除环境变量键集合，逐项取得当前键
	for _, k := range strippedEnvKeys {
		skipKeys[k] = struct{}{}
	}
	if p.Isolation == IsolationIsolated {
		// 遍历已覆盖键集合隔离，逐项取得当前键
		for _, k := range overriddenKeysIsolated {
			skipKeys[k] = struct{}{}
		}
	}

	// 遍历父级环境变量，逐项取得条目
	for _, entry := range p.ParentEnv {
		idx := strings.IndexByte(entry, '=') // 索引
		if idx <= 0 {
			continue
		}
		key := entry[:idx] // 键
		if key == "PATH" {
			out = append(out, "PATH="+prependPath(pathPrefix(p.RuntimeBin, p.PluginBin), entry[idx+1:]))
			continue
		}
		if /* 跳过 */ _, skip := skipKeys[key]; skip {
			continue
		}
		out = append(out, entry)
	}

	if hasNoPath(p.ParentEnv) {
		out = append(out, "PATH="+pathPrefix(p.RuntimeBin, p.PluginBin))
	}

	if p.Isolation == IsolationIsolated {
		out = append(out,
			"HOME="+filepath.Join(p.CLIDataDir, "home"),
			"USERPROFILE="+filepath.Join(p.CLIDataDir, "home"),
			"XDG_CONFIG_HOME="+filepath.Join(p.CLIDataDir, "config"),
			"XDG_DATA_HOME="+filepath.Join(p.CLIDataDir, "data"),
			"XDG_CACHE_HOME="+filepath.Join(p.CLIDataDir, "cache"),
		)
	}
	return out
}

// 处理路径前缀，接收组成部分，返回文本
func pathPrefix(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts)) // 非空
	// 遍历组成部分，逐项取得片段
	for _, part := range parts {
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, pathListSeparator())
}

// prependPath 返回 "<bin>{sep}<existing>"，bin 为空时直接返回 existing
// 接收可执行目录、已存在，返回文本
func prependPath(bin string, existing string) string {
	if bin == "" {
		return existing
	}
	return bin + pathListSeparator() + existing
}

// 获取当前操作系统用于分隔路径列表的字符，返回文本
func pathListSeparator() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}

// hasNoPath 判断父进程的环境变量中是否缺少 PATH
// 接收环境变量，返回是否满足条件
func hasNoPath(env []string) bool {
	// 遍历环境变量，逐项取得e 对应的数据
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			return false
		}
	}
	return true
}
