package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// InstallParams 描述来自 npm 或本地目录的单次安装请求
// NpmPackage 和 LocalPath 必须恰好有一项非空
type InstallParams struct {
	NpmPackage string // npm 包名，如 "lark-cli" 或 "@scope/pkg"
	LocalPath  string // 包含 package.json 的本地目录绝对路径
	Name       string // 用户指定的插件名称，安装目录为 plugins/cli/<Name>/
}

// InstallResult 汇总安装成功后生成的磁盘产物
type InstallResult struct {
	// 名称
	Name string
	// 版本
	Version string
	// 描述
	Description string
	// 作者
	Author string
	// 安装目录
	InstallDir string
	// CLI数据目录
	CLIDataDir string
	// 清单路径
	ManifestPath string
	// 可执行文件
	Executable string
}

// Manager 提供 CLI 插件的安装、卸载、重置和运行入口
// 构造时接收数据目录及内置 Node.js 和 npm 可执行文件的绝对路径
// 因此编译时无需依赖运行时服务
type Manager struct {
	// 应用数据目录
	dataDir string
	// Node.js 可执行文件路径
	nodePath string
	// npm 可执行文件路径
	npmPath string
}

// NewManager 使用指定数据目录和内置 Node.js、npm 可执行文件构造管理器
// 接收应用数据目录、节点路径、npm路径，返回管理器
func NewManager(dataDir, nodePath, npmPath string) *Manager {
	return &Manager{dataDir: dataDir, nodePath: nodePath, npmPath: npmPath}
}

// InstallFromNpm 将已发布的 npm 包作为 CLI 插件安装到 plugins/cli/<Name>/
// 接收本次调用的上下文、安装参数，返回安装结果及可能发生的错误
func (m *Manager) InstallFromNpm(ctx context.Context, p InstallParams) (InstallResult, error) {
	if strings.TrimSpace(p.NpmPackage) == "" {
		return InstallResult{}, errors.New("cli: NpmPackage required")
	}
	return m.install(ctx, p.Name, p.NpmPackage)
}

// InstallFromLocal 将包含 package.json 的本地目录作为 CLI 插件安装
// 接收本次调用的上下文、安装参数，返回安装结果及可能发生的错误
func (m *Manager) InstallFromLocal(ctx context.Context, p InstallParams) (InstallResult, error) {
	if strings.TrimSpace(p.LocalPath) == "" {
		return InstallResult{}, errors.New("cli: LocalPath required")
	}
	_, err := os.Stat(filepath.Join(p.LocalPath, "package.json")) // 读取文件元数据时的错误
	if err != nil {
		return InstallResult{}, errors.New("cli: source dir has no package.json")
	}
	return m.install(ctx, p.Name, p.LocalPath)
}

// install 依次执行 npm 安装、探测和空清单生成，最后返回安装结果
// 接收本次调用的上下文、当前名称、来源，返回安装结果及可能发生的错误
// 创建所需目录
// 加载 CLI 扩展清单
// 保存 CLI 扩展清单
func (m *Manager) install(ctx context.Context, name, source string) (InstallResult, error) {
	if strings.TrimSpace(name) == "" {
		return InstallResult{}, errors.New("cli: Name required")
	}
	installDir := filepath.Join(dir.CLIRoot(m.dataDir), name) // 安装目录
	mkdirErr := os.MkdirAll(installDir, 0o755)                // 创建所需目录时的错误
	if mkdirErr != nil {
		return InstallResult{}, mkdirErr
	}
	// 处理npm安装时的错误
	npmInstallErr := NpmInstall(ctx, NpmInstallParams{
		NodePath:  m.nodePath,
		NpmPath:   m.npmPath,
		Source:    source,
		TargetDir: installDir,
		ParentEnv: os.Environ(),
	})
	if npmInstallErr != nil {
		return InstallResult{}, npmInstallErr
	}

	pkg, err := ReadPackageJSON(installDir) // 包名、读取包名JSON时的错误
	if err != nil {
		return InstallResult{}, err
	}
	executable, err := SelectExecutable(installDir, pkg) // 可执行文件
	if err != nil {
		return InstallResult{}, err
	}

	cliDataDir := filepath.Join(dir.CLIDataRoot(m.dataDir), name) // CLI数据目录
	manifestPath := filepath.Join(cliDataDir, "manifest.json")    // 清单路径

	existing, _ := LoadManifest(manifestPath) // 已存在
	// 清单
	manifest := Manifest{
		Name:        pkg.Name,
		Version:     pkg.Version,
		Description: pkg.Description,
		Executable:  executable,
		Isolation:   existing.Isolation, // 重新安装时保留用户选择的隔离或共享模式
		Tools:       existing.Tools,     // 重新安装时保留 AI 或用户编辑过的工具
	}
	if manifest.Isolation == "" {
		manifest.Isolation = IsolationIsolated
	}
	saveErr := SaveManifest(manifestPath, manifest) // 保存 CLI 扩展清单时的错误
	if saveErr != nil {
		return InstallResult{}, saveErr
	}

	return InstallResult{
		Name:         name,
		Version:      pkg.Version,
		Description:  pkg.Description,
		Author:       pkg.Author,
		InstallDir:   installDir,
		CLIDataDir:   cliDataDir,
		ManifestPath: manifestPath,
		Executable:   executable,
	}, nil
}

// Uninstall 删除 plugins/cli/<name>/，保留 plugins/cli_data/<name>/
// 接收本次调用的上下文、当前名称，返回操作错误
func (m *Manager) Uninstall(ctx context.Context, name string) error {
	_ = ctx
	if strings.TrimSpace(name) == "" {
		return errors.New("cli: name required")
	}
	installDir := filepath.Join(dir.CLIRoot(m.dataDir), name) // 安装目录
	return os.RemoveAll(installDir)
}

// ResetData 删除 plugins/cli_data/<name>/，清除登录令牌、生成的清单和 XDG 状态
// 保留插件安装目录
// 接收本次调用的上下文、当前名称，返回操作错误
func (m *Manager) ResetData(ctx context.Context, name string) error {
	_ = ctx
	if strings.TrimSpace(name) == "" {
		return errors.New("cli: name required")
	}
	cliDataDir := filepath.Join(dir.CLIDataRoot(m.dataDir), name) // CLI数据目录
	return os.RemoveAll(cliDataDir)
}

// RunTool 从插件清单中查找指定工具，并将输入代入参数模板生成 argv
// 按清单配置的隔离模式执行 CLI 子进程
// 接收本次调用的上下文、当前名称、工具名称、按键组织的数据映射，返回运行结果及可能发生的错误
// 加载 CLI 扩展清单
// 构造环境变量
func (m *Manager) RunTool(ctx context.Context, name string, toolName string, input map[string]any) (RunResult, error) {
	cliDataDir := filepath.Join(dir.CLIDataRoot(m.dataDir), name) // CLI数据目录
	installDir := filepath.Join(dir.CLIRoot(m.dataDir), name)     // 安装目录
	manifestPath := filepath.Join(cliDataDir, "manifest.json")    // 清单路径

	manifest, err := LoadManifest(manifestPath) // 清单、加载 CLI 扩展清单时的错误
	if err != nil {
		return RunResult{}, err
	}
	if manifest.Executable == "" {
		manifest, err = RepairManifestExecutable(manifestPath, installDir, manifest)
		if err != nil {
			return RunResult{}, errors.New("cli: manifest has no executable; reinstall the plugin")
		}
	}

	// 工具
	var tool Tool
	found := false // 是否找到
	// 遍历工具列表，逐项取得当前条目
	for _, t := range manifest.Tools {
		if t.Name == toolName {
			tool = t
			found = true
			break
		}
	}
	if !found {
		return RunResult{}, errors.New("cli: tool not found in manifest: " + toolName)
	}

	argv, err := renderArgv(tool, input) // 命令参数
	if err != nil {
		return RunResult{}, err
	}

	// 环境变量
	env := BuildEnv(EnvParams{
		Isolation:  manifest.Isolation,
		CLIDataDir: cliDataDir,
		RuntimeBin: filepath.Dir(m.nodePath),
		PluginBin:  filepath.Join(installDir, "node_modules", ".bin"),
		ParentEnv:  os.Environ(),
	})

	return Run(ctx, RunParams{
		Executable: manifest.Executable,
		Argv:       argv,
		Env:        env,
		OutputMode: tool.OutputMode,
		TimeoutSec: tool.TimeoutSeconds,
		UsePTY:     true,
	})
}

// RunCommand 使用托管运行环境，向已安装的 CLI 传入任意 argv 并执行
// 供清单中的专用工具可用之前的设置、登录和初始化流程使用
// 接收本次调用的上下文、当前名称、命令参数、输出模式、超时秒，返回运行结果及可能发生的错误
func (m *Manager) RunCommand(ctx context.Context, name string, argv []string, outputMode OutputMode, timeoutSeconds int) (RunResult, error) {
	return m.runCommand(ctx, name, argv, outputMode, timeoutSeconds, nil)
}

// RunCommandStreaming 执行任意 argv，并在运行期间持续发送累计的标准输出和标准错误快照
// 接收本次调用的上下文、当前名称、命令参数、输出模式、超时秒、on进度，返回运行结果及可能发生的错误
func (m *Manager) RunCommandStreaming(ctx context.Context, name string, argv []string, outputMode OutputMode, timeoutSeconds int, onProgress func(RunProgress)) (RunResult, error) {
	return m.runCommand(ctx, name, argv, outputMode, timeoutSeconds, onProgress)
}

// 运行命令，接收本次调用的上下文、当前名称、命令参数、输出模式、超时秒、on进度，返回运行结果及可能发生的错误
// 加载 CLI 扩展清单
// 构造环境变量
func (m *Manager) runCommand(ctx context.Context, name string, argv []string, outputMode OutputMode, timeoutSeconds int, onProgress func(RunProgress)) (RunResult, error) {
	if len(argv) == 0 {
		return RunResult{}, errors.New("cli: argv required")
	}

	cliDataDir := filepath.Join(dir.CLIDataRoot(m.dataDir), name) // CLI数据目录
	installDir := filepath.Join(dir.CLIRoot(m.dataDir), name)     // 安装目录
	manifestPath := filepath.Join(cliDataDir, "manifest.json")    // 清单路径

	manifest, err := LoadManifest(manifestPath) // 清单、加载 CLI 扩展清单时的错误
	if err != nil {
		return RunResult{}, err
	}
	if manifest.Executable == "" {
		manifest, err = RepairManifestExecutable(manifestPath, installDir, manifest)
		if err != nil {
			return RunResult{}, errors.New("cli: manifest has no executable; reinstall the plugin")
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

	return RunWithProgress(ctx, RunParams{
		Executable: manifest.Executable,
		Argv:       argv,
		Env:        env,
		OutputMode: outputMode,
		TimeoutSec: timeoutSeconds,
		WorkingDir: installDir,
		UsePTY:     onProgress != nil,
	}, onProgress)
}

// renderArgv 将 argv_template 中各段的 {placeholder} 替换为对应输入字段的字符串值
// 缺少输入字段时按以下规则处理
//   - 若字段列在 input_schema.required 中，则返回错误，由调用方补齐
//   - 其他情况移除该段，若该段仅含 {placeholder} 且前一段是以 '-' 开头的选项
//     则一并移除该选项，以完整省略可选的 `--flag {value}` 参数对
func renderArgv(t Tool, input map[string]any) ([]string, error) {
	required, err := schemaRequired(t.InputSchema) // 必填项、处理模式必填项时的错误
	if err != nil {
		return nil, fmt.Errorf("cli: tool %q: parse input_schema.required: %w", t.Name, err)
	}

	out := make([]string, 0, len(t.ArgvTemplate)) // 本次处理的输出
	// 遍历命令参数模板，逐项取得片段
	for _, segment := range t.ArgvTemplate {
		matches := placeholderRegex.FindAllStringSubmatchIndex(segment, -1) // 匹配项
		if len(matches) == 0 {
			out = append(out, segment)
			continue
		}

		// 缺失可选
		var missingOptional bool
		// 遍历匹配项，逐项取得当前条目
		for _, m := range matches {
			key := segment[m[2]:m[3]] // 键
			if /* 是否存在 */ _, present := input[key]; present {
				continue
			}
			if /* 待发送的请求 */ _, req := required[key]; req {
				return nil, errors.New("cli: argv_template references missing input field: " + key)
			}
			missingOptional = true
		}

		if missingOptional {
			if isPureValuePlaceholder(segment) && len(out) > 0 && isFlagSegment(out[len(out)-1]) {
				out = out[:len(out)-1]
			}
			continue
		}

		// 构建器
		var builder strings.Builder
		cursor := 0 // 游标
		// 遍历匹配项，逐项取得当前条目
		for _, m := range matches {
			start, end := m[0], m[1] // 开始、结束
			builder.WriteString(segment[cursor:start])
			builder.WriteString(stringifyInput(input[segment[m[2]:m[3]]]))
			cursor = end
		}
		builder.WriteString(segment[cursor:])
		out = append(out, builder.String())
	}
	return out, nil
}

// schemaRequired 提取 input_schema 的 "required" 数组并返回集合，便于 O(1) 查询
// 接收尚未解析的原始数据，返回按键组织的数据映射及可能发生的错误
func schemaRequired(raw json.RawMessage) (map[string]struct{}, error) {
	if len(raw) == 0 {
		return map[string]struct{}{}, nil
	}
	// 文档
	var doc struct {
		// 必填项
		Required []string `json:"required"`
	}
	err := json.Unmarshal(raw, &doc) // 解析序列化内容时的错误
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(doc.Required)) // 本次处理的输出
	// 遍历必填项，逐项取得当前名称
	for _, name := range doc.Required {
		out[name] = struct{}{}
	}
	return out, nil
}

// isPureValuePlaceholder 判断参数段是否仅含一个 {placeholder}，且前后无字面文本
// 移除时只有纯占位符段会带走前一个选项，"--key={val}" 等混合参数段只移除自身
// 接收片段，返回是否满足条件
func isPureValuePlaceholder(segment string) bool {
	loc := placeholderRegex.FindStringIndex(segment) // 时区位置
	return loc != nil && loc[0] == 0 && loc[1] == len(segment)
}

// isFlagSegment 判断参数段是否形似 CLI 选项，如 "-v" 或 "--data"
// 接收片段，返回是否满足条件
func isFlagSegment(segment string) bool {
	return len(segment) >= 2 && segment[0] == '-'
}

// stringifyInput 将字符串、数值、布尔值等输入转为 argv 使用的字符串
// 接收待检查的动态数据，返回文本
func stringifyInput(value any) string {
	switch /* 当前值 */ v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		bytes, err := json.Marshal(v) // 原始字节内容、将数据序列化时的错误
		if err != nil {
			return ""
		}
		return string(bytes)
	}
}
