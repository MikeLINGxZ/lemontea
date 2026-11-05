package mcp

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Manifest 描述一个 MCP 包目录的导入元数据
type Manifest struct {
	// 名称
	Name string `json:"name"`
	// 版本
	Version string `json:"version"`
	// 描述
	Description string `json:"description"`
	// 作者
	Author string `json:"author"`
	// 配置文件
	ConfigFile string `json:"config_file"`
}

// loadManifest 读取扩展包清单，缺失字段使用根据目录推导的默认值
// 接收根目录目录，返回清单及可能发生的错误
func loadManifest(rootDir string) (Manifest, error) {
	manifestPath := filepath.Join(rootDir, "manifest.json") // 清单路径
	bytes, err := os.ReadFile(manifestPath)                 // 原始字节内容、读取文件内容时的错误
	if err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{}                        // 清单
	decodeErr := json.Unmarshal(bytes, &manifest) // 解析序列化内容时的错误
	if decodeErr != nil {
		return Manifest{}, decodeErr
	}
	if manifest.Name == "" {
		manifest.Name = filepath.Base(rootDir)
	}
	if manifest.ConfigFile == "" {
		manifest.ConfigFile = "mcp.json"
	}
	return manifest, nil
}

// loadManifestOptional 在 manifest.json 存在时读取该文件，否则使用根据目录推导的默认值
// 接收根目录目录，返回清单及可能发生的错误
func loadManifestOptional(rootDir string) (Manifest, error) {
	manifest, err := loadManifest(rootDir) // 清单、加载清单时的错误
	if err == nil {
		return manifest, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	return Manifest{
		Name: filepath.Base(rootDir),
	}, nil
}

// writeManifest 将规范化后的清单写入复制后的扩展目录
// 接收根目录目录、清单，返回操作错误
func writeManifest(rootDir string, manifest Manifest) error {
	bytes, err := json.MarshalIndent(manifest, "", "  ") // 原始字节内容、序列化Indent 对应的数据时的错误
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(rootDir, "manifest.json"), bytes, 0o644)
}
