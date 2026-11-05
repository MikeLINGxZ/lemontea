package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackageMeta 包含 CLI 安装和探测使用的 package.json 字段
type PackageMeta struct {
	// 名称
	Name string `json:"name"`
	// 版本
	Version string `json:"version"`
	// 描述
	Description string `json:"description"`
	// 作者
	Author string            `json:"author"`
	Bin    map[string]string `json:"-"` // 将字符串或对象形式的 bin 字段统一为名称到脚本的映射
}

// ReadPackageJSON 读取 {installDir}/package.json，找不到时尝试 .../node_modules/<name>/package.json
// 返回所需的字段，并将 bin 规范化为名称到脚本的映射
// 接收安装目录，返回包名元信息及可能发生的错误
func ReadPackageJSON(installDir string) (PackageMeta, error) {
	// 候选列表
	candidates := []string{
		filepath.Join(installDir, "package.json"),
	}
	// npm install --prefix 将包安装在 node_modules/<name>/ 下
	entries, _ := os.ReadDir(filepath.Join(installDir, "node_modules"))
	// 遍历条目列表，逐项取得e 对应的数据
	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".bin" {
			continue
		}
		if strings.HasPrefix(e.Name(), "@") {
			scopedEntries, _ := os.ReadDir(filepath.Join(installDir, "node_modules", e.Name())) // 限定范围条目列表
			// 遍历限定范围条目列表，逐项取得限定范围
			for _, scoped := range scopedEntries {
				if !scoped.IsDir() {
					continue
				}
				candidates = append(candidates, filepath.Join(installDir, "node_modules", e.Name(), scoped.Name(), "package.json"))
			}
			continue
		}
		candidates = append(candidates, filepath.Join(installDir, "node_modules", e.Name(), "package.json"))
	}

	// 最后一个错误
	var lastErr error
	// 遍历候选列表，逐项取得文件路径
	for _, path := range candidates {
		meta, err := readOnePackageJSON(path) // 元信息、读取One包名JSON时的错误
		if err == nil {
			return meta, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no package.json found")
	}
	return PackageMeta{}, lastErr
}

// readOnePackageJSON 解析单个 package.json 文件并规范化 bin 字段
// 接收文件路径，返回包名元信息及可能发生的错误
// 读取文件内容
// 解析序列化内容
func readOnePackageJSON(path string) (PackageMeta, error) {
	bytes, err := os.ReadFile(path) // 原始字节内容、读取文件内容时的错误
	if err != nil {
		return PackageMeta{}, err
	}
	// 尚未解析的原始数据
	var raw struct {
		// 名称
		Name string `json:"name"`
		// 版本
		Version string `json:"version"`
		// 描述
		Description string `json:"description"`
		// 作者
		Author json.RawMessage `json:"author"`
		// 可执行目录
		Bin json.RawMessage `json:"bin"`
	}
	decodeErr := json.Unmarshal(bytes, &raw) // 解析序列化内容时的错误
	if decodeErr != nil {
		return PackageMeta{}, decodeErr
	}
	// 元信息
	meta := PackageMeta{
		Name:        raw.Name,
		Version:     raw.Version,
		Description: raw.Description,
		Bin:         map[string]string{},
	}
	if len(raw.Author) > 0 {
		// 当前值
		var s string
		decodeErr2 := json.Unmarshal(raw.Author, &s) // 解码错误
		if decodeErr2 == nil {
			meta.Author = s
		} else {
			// 对象
			var obj struct {
				// 名称
				Name string `json:"name"`
			}
			decodeErr3 := json.Unmarshal(raw.Author, &obj) // 解码错误
			if decodeErr3 == nil {
				meta.Author = obj.Name
			}
		}
	}
	if len(raw.Bin) > 0 {
		// 当前值
		var s string
		decodeErr4 := json.Unmarshal(raw.Bin, &s) // 解码错误
		if decodeErr4 == nil {
			meta.Bin[meta.Name] = s
		} else {
			// 对象
			var obj map[string]string
			decodeErr5 := json.Unmarshal(raw.Bin, &obj) // 解码错误
			if decodeErr5 == nil {
				meta.Bin = obj
			}
		}
	}
	return meta, nil
}

// SelectExecutable 为包选择适合的 bin 条目，并返回对应符号链接的绝对路径
// 符号链接位于 {installDir}/node_modules/.bin/{binName}
// 优先选择键名与包名相同的条目，否则仅在条目唯一时选用它，其余情况返回错误
// 接收安装目录、包名，返回文本及可能发生的错误
func SelectExecutable(installDir string, pkg PackageMeta) (string, error) {
	if len(pkg.Bin) == 0 {
		return discoverExecutableFromBinDir(installDir, pkg)
	}
	if /* 是否成功取得对应值 */ _, ok := pkg.Bin[pkg.Name]; ok {
		return filepath.Join(installDir, "node_modules", ".bin", pkg.Name), nil
	}
	if len(pkg.Bin) == 1 {
		// 遍历可执行目录，逐项取得可执行目录名称
		for binName := range pkg.Bin {
			return filepath.Join(installDir, "node_modules", ".bin", binName), nil
		}
	}
	return discoverExecutableFromBinDir(installDir, pkg)
}

// 处理发现可执行文件起点可执行目录目录，接收安装目录、包名，返回文本及可能发生的错误
func discoverExecutableFromBinDir(installDir string, pkg PackageMeta) (string, error) {
	binDir := filepath.Join(installDir, "node_modules", ".bin") // 可执行目录目录
	entries, err := os.ReadDir(binDir)                          // 条目列表、读取目录时的错误
	if err != nil {
		if len(pkg.Bin) == 0 {
			return "", errors.New("package has no bin entry; not a CLI")
		}
		return "", errors.New("package has multiple bin entries and none matches the package name")
	}

	candidates := make([]string, 0, len(entries)) // 候选列表
	// 遍历条目列表，逐项取得条目
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name() // 当前名称
		if strings.HasPrefix(name, ".") {
			continue
		}
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)
	if len(candidates) == 1 {
		return filepath.Join(binDir, candidates[0]), nil
	}

	leaf := packageLeafName(pkg.Name) // 最末段
	// 遍历候选列表，逐项取得当前名称
	for _, name := range candidates {
		if name == pkg.Name || name == leaf {
			return filepath.Join(binDir, name), nil
		}
	}
	if len(candidates) == 0 && len(pkg.Bin) == 0 {
		return "", errors.New("package has no bin entry; not a CLI")
	}
	if len(candidates) == 0 {
		return "", errors.New("package has multiple bin entries and none matches the package name")
	}
	return "", errors.New("unable to infer executable from node_modules/.bin")
}

// 取 npm 包标识的末尾名称作为候选命令名，接收当前名称，返回文本
func packageLeafName(name string) string {
	if /* 索引 */ idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// HelpVariants 列出 ProbeHelp 按顺序尝试的帮助参数
var HelpVariants = [][]string{
	{"--help"},
	{"-h"},
	{"help"},
}

// ProbeHelp 依次使用 HelpVariants 中的参数运行 <executable>，返回首个非空标准输出
// 每次尝试的超时均为 10 秒
func ProbeHelp(ctx context.Context, executable string, env []string) (string, error) {
	// 最后一个错误
	var lastErr error
	// 遍历帮助变体列表，逐项取得参数
	for _, args := range HelpVariants {
		// 结果、运行当前数据时的错误
		res, err := Run(ctx, RunParams{
			Executable: executable,
			Argv:       args,
			Env:        env,
			OutputMode: OutputText,
			TimeoutSec: 10,
		})
		if err != nil {
			lastErr = err
			continue
		}
		if res.ExitCode == 0 && res.Stdout != "" {
			return res.Stdout, nil
		}
		lastErr = errors.New("help command exited non-zero")
	}
	if lastErr == nil {
		lastErr = errors.New("all help variants failed")
	}
	return "", lastErr
}
