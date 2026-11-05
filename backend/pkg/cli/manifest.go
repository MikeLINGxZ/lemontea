// Package cli 提供安装、执行和管理 CLI 插件包的基础功能
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// OutputMode 控制 RunTool 向调用方返回结果前如何解析 CLI 的标准输出
type OutputMode string

const (
	// 输出JSON
	OutputJSON OutputMode = "json"
	// 输出文本
	OutputText OutputMode = "text"
	// 输出行列表
	OutputLines OutputMode = "lines"
)

// IsolationMode 控制运行器如何构造 CLI 子进程的环境变量
type IsolationMode string

const (
	// 隔离隔离
	IsolationIsolated IsolationMode = "isolated"
	// 隔离共享
	IsolationShared IsolationMode = "shared"
)

// Tool 描述 CLI 插件暴露的一个可由 AI 调用的工具
type Tool struct {
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 输入模式
	InputSchema json.RawMessage `json:"input_schema"`
	// 命令参数模板
	ArgvTemplate []string `json:"argv_template"`
	// 输出模式
	OutputMode OutputMode `json:"output_mode"`
	// 超时秒
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	// 需要确认
	RequiresConfirm bool `json:"requires_confirm"`
	// 启用状态
	Enabled bool `json:"enabled"`
}

// Manifest 描述已安装的 CLI 插件及其暴露的工具
type Manifest struct {
	// 名称
	Name string `json:"name"`
	// 版本
	Version string `json:"version"`
	// 描述
	Description string `json:"description"`
	// 可执行文件
	Executable string `json:"executable"`
	// 登录命令
	LoginCommand []string `json:"login_command,omitempty"`
	// 登录步骤列表
	LoginSteps [][]string `json:"login_steps,omitempty"`
	// 隔离
	Isolation IsolationMode `json:"isolation"`
	// 工具列表
	Tools []Tool `json:"tools"`
}

// ResolveLoginSteps 返回交互式登录各步骤的有序 argv 列表
// LoginSteps 非空时优先使用，否则将 LoginCommand 包装为单个步骤
// 结果为空表示清单未配置登录流程
func (m Manifest) ResolveLoginSteps() [][]string {
	if len(m.LoginSteps) > 0 {
		out := make([][]string, 0, len(m.LoginSteps)) // 本次处理的输出
		// 遍历登录步骤列表，逐项取得步骤
		for _, step := range m.LoginSteps {
			if len(step) == 0 {
				continue
			}
			out = append(out, step)
		}
		if len(out) > 0 {
			return out
		}
	}
	if len(m.LoginCommand) > 0 {
		return [][]string{m.LoginCommand}
	}
	return nil
}

// LoadManifest 读取 JSON 清单文件，文件不存在时返回零值清单和 nil 错误
// 接收文件路径，返回清单及可能发生的错误
func LoadManifest(path string) (Manifest, error) {
	bytes, err := os.ReadFile(path) // 原始字节内容、读取文件内容时的错误
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, nil
		}
		return Manifest{}, err
	}
	// 当前条目
	var m Manifest
	decodeErr := json.Unmarshal(bytes, &m) // 解析序列化内容时的错误
	if decodeErr != nil {
		return Manifest{}, decodeErr
	}
	return m, nil
}

// SaveManifest 先写入临时文件，再重命名到 path，原子保存 JSON 清单
// 接收文件路径、清单，返回操作错误
func SaveManifest(path string, m Manifest) error {
	mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return mkdirErr
	}
	bytes, err := json.MarshalIndent(m, "", "  ") // 原始字节内容、序列化Indent 对应的数据时的错误
	if err != nil {
		return err
	}
	tmp := path + ".tmp"                        // 临时
	writeErr := os.WriteFile(tmp, bytes, 0o644) // 写入文件内容时的错误
	if writeErr != nil {
		return writeErr
	}
	renameErr := os.Rename(tmp, path) // 替换目标路径上的文件时的错误
	if renameErr != nil {
		_ = os.Remove(tmp)
		return renameErr
	}
	return nil
}

// RepairManifestExecutable 从已安装包的元数据补齐缺失的可执行文件路径，并保存修复后的清单
// 接收文件路径、安装目录、清单，返回清单及可能发生的错误
func RepairManifestExecutable(path string, installDir string, manifest Manifest) (Manifest, error) {
	if strings.TrimSpace(manifest.Executable) != "" {
		return manifest, nil
	}

	pkg, err := ReadPackageJSON(installDir) // 包名、读取包名JSON时的错误
	if err != nil {
		return Manifest{}, err
	}
	executable, err := SelectExecutable(installDir, pkg) // 可执行文件
	if err != nil {
		return Manifest{}, err
	}
	manifest.Executable = executable
	saveErr := SaveManifest(path, manifest) // 保存 CLI 扩展清单时的错误
	if saveErr != nil {
		return Manifest{}, saveErr
	}
	return manifest, nil
}

// 占位内容正则匹配
var placeholderRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// Validate 检查清单结构
// 目前要求所有工具的 argv_template 中每个 {placeholder} 都对应 input_schema 声明的属性
// 接收清单，返回操作错误
func Validate(m Manifest) error {
	if strings.TrimSpace(m.Executable) == "" {
		return errors.New("manifest: executable is empty")
	}
	// 遍历工具列表，逐项取得当前条目
	for _, t := range m.Tools {
		props, err := schemaPropertyNames(t.InputSchema) // 组件属性、处理模式属性名称列表时的错误
		if err != nil {
			return fmt.Errorf("manifest tool %q: parse input_schema: %w", t.Name, err)
		}
		// 遍历命令参数模板，逐项取得片段
		for _, segment := range t.ArgvTemplate {
			matches := placeholderRegex.FindAllStringSubmatch(segment, -1) // 匹配项
			// 遍历匹配项，逐项取得当前条目
			for _, m := range matches {
				if /* 是否成功取得对应值 */ _, ok := props[m[1]]; !ok {
					return fmt.Errorf("manifest tool %q: argv_template uses {%s} but input_schema has no such property", t.Name, m[1])
				}
			}
		}
	}
	return nil
}

// schemaPropertyNames 提取 JSON Schema 对象顶层 "properties" 中的属性名
// 接收尚未解析的原始数据，返回按键组织的数据映射及可能发生的错误
func schemaPropertyNames(raw json.RawMessage) (map[string]struct{}, error) {
	if len(raw) == 0 {
		return map[string]struct{}{}, nil
	}
	// 文档
	var doc struct {
		// 属性列表
		Properties map[string]json.RawMessage `json:"properties"`
	}
	err := json.Unmarshal(raw, &doc) // 解析序列化内容时的错误
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(doc.Properties)) // 本次处理的输出
	// 遍历属性列表，逐项取得当前键
	for k := range doc.Properties {
		out[k] = struct{}{}
	}
	return out, nil
}
