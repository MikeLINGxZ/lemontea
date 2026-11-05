// migration 包负责 Lemontea 版本之间的一次性数据目录结构迁移
package migration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// markerFileName 是迁移成功后写入扩展根目录的幂等标记文件名
const markerFileName = ".migrated_v1"

// MigrationResult 汇总一次迁移的执行结果
type MigrationResult struct {
	// 已已迁移
	AlreadyMigrated bool
	// 已移动MCP
	MovedMCP bool
	// 已移动插件
	MovedPlugin bool
	// 是否改写配置
	RewroteConfig bool
	// 警告列表
	Warnings []string
}

// MigrateExtensionDirs 将 dataDir 下旧版 mcp/ 和 plugin/ 目录迁入 plugins/，返回迁移结果和错误
// 可在每次启动时调用，幂等标记可避免重复迁移
// 接收应用数据目录，返回迁移结果及可能发生的错误
// 创建所需目录
// 读取文件元数据
// 写入文件内容
func MigrateExtensionDirs(dataDir string) (MigrationResult, error) {
	result := MigrationResult{} // 本次操作的结果

	mkdirErr := os.MkdirAll(dir.ExtensionsRoot(dataDir), 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return result, mkdirErr
	}

	markerPath := filepath.Join(dir.ExtensionsRoot(dataDir), markerFileName) // 标记路径
	_, statErr := os.Stat(markerPath)                                        // 读取文件元数据时的错误
	if statErr == nil {
		result.AlreadyMigrated = true
		return result, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return result, statErr
	}

	// 第一步，移动存在的旧目录
	movedValue, warnValue, moveLegacyErr := moveLegacy(dir.LegacyMCPRoot(dataDir), dir.MCPRoot(dataDir))
	if moveLegacyErr != nil {
		return result, moveLegacyErr
	} else {
		result.MovedMCP = movedValue
		if warnValue != "" {
			result.Warnings = append(result.Warnings, warnValue)
		}
	}
	// 已移动值、警告值、移动旧版错误
	movedValue2, warnValue2, moveLegacyErr2 := moveLegacy(dir.LegacyPluginRoot(dataDir), dir.PluginRoot(dataDir))
	if moveLegacyErr2 != nil {
		return result, moveLegacyErr2
	} else {
		result.MovedPlugin = movedValue2
		if warnValue2 != "" {
			result.Warnings = append(result.Warnings, warnValue2)
		}
	}

	// 第二步，始终尝试重写配置中的路径，只处理带有旧版前缀的条目，
	// 因此重复执行是安全的，也能补齐上次移动文件后崩溃而
	// 未能更新 config.json 的迁移
	rewrote, err := rewriteConfigPaths(dataDir)
	if err != nil {
		return result, err
	}
	result.RewroteConfig = rewrote

	// 第三步，写入幂等标记
	writeErr := os.WriteFile(markerPath, []byte("v1"), 0o644)
	if writeErr != nil {
		return result, writeErr
	}
	return result, nil
}

// moveLegacy 在源目录 srcDir 存在且目标目录 dstDir 不存在时执行移动
// 返回是否移动、警告文本和错误，两个目录都存在时保留源目录并返回警告
// 接收源数据目录、dst目录，返回是否满足条件、文本及可能发生的错误
func moveLegacy(srcDir string, dstDir string) (bool, string, error) {
	srcInfo, err := os.Stat(srcDir) // 源数据信息、读取文件元数据时的错误
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	if !srcInfo.IsDir() {
		return false, "legacy path is not a directory: " + srcDir, nil
	}

	_, statErr := os.Stat(dstDir) // 读取文件元数据时的错误
	if statErr == nil {
		return false, "both legacy and new dirs exist, leaving as-is: " + srcDir, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return false, "", statErr
	}

	mkdirErr := os.MkdirAll(filepath.Dir(dstDir), 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return false, "", mkdirErr
	}
	renameErr := os.Rename(srcDir, dstDir) // 替换目标路径上的文件时的错误
	if renameErr != nil {
		return false, "", renameErr
	}
	return true, "", nil
}

// rewriteConfigPaths 将 config.json 解析为 data_models.Config，替换每个 ExtensionItem 的
// 扩展目录前缀，再通过临时文件和重命名原子写回
// 至少有一个 ExtensionItem 的路径被重写时返回 true
//
// 通过 data_models.Config 读写 config.json 会忽略未知的顶层字段，
// 这与应用其他部分使用的标准配置结构一致，
// 若以后磁盘上的配置结构和 Go 结构体不再一致，
// 则需要改为在 JSON 标记层仅重写受影响的路径
// 接收应用数据目录，返回是否满足条件及可能发生的错误
func rewriteConfigPaths(dataDir string) (bool, error) {
	configPath := filepath.Join(dataDir, dir.ConfigFileName) // 配置路径
	bytes, err := os.ReadFile(configPath)                    // 原始字节内容、读取文件内容时的错误
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	// 配置
	var cfg data_models.Config
	decodeErr := json.Unmarshal(bytes, &cfg) // 解析序列化内容时的错误
	if decodeErr != nil {
		return false, decodeErr
	}

	rewrote := false // 是否改写
	// 遍历扩展列表，逐项取得循环索引
	for i := range cfg.Extensions {
		if /* 新值、是否有变化 */ newValue, changed := rewriteLegacyPathPrefix(cfg.Extensions[i].RootDir, dataDir); changed {
			cfg.Extensions[i].RootDir = newValue
			rewrote = true
		}
		if /* 新值、是否有变化 */ newValue, changed := rewriteLegacyPathPrefix(cfg.Extensions[i].ConfigFilePath, dataDir); changed {
			cfg.Extensions[i].ConfigFilePath = newValue
			rewrote = true
		}
	}
	if !rewrote {
		return false, nil
	}

	updated, err := json.MarshalIndent(cfg, "", "  ") // 更新
	if err != nil {
		return false, err
	}

	tmpPath := configPath + ".tmp"                    // 临时路径
	writeErr := os.WriteFile(tmpPath, updated, 0o644) // 写入文件内容时的错误
	if writeErr != nil {
		return false, writeErr
	}
	renameErr := os.Rename(tmpPath, configPath) // 替换目标路径上的文件时的错误
	if renameErr != nil {
		// 尽量清理临时文件，忽略清理错误以保留真正的重命名错误
		_ = os.Remove(tmpPath)
		return false, renameErr
	}
	return true, nil
}

// rewriteLegacyPathPrefix 在 value 以旧版前缀
// "{dataDir}/mcp/" 或 "{dataDir}/plugin/" 开头时重写路径，并返回是否改动
// 前缀末尾保留路径分隔符，以免误匹配
// "{dataDir}/mcp_old/..." 等路径
// 接收文本、应用数据目录，返回文本、是否满足条件
func rewriteLegacyPathPrefix(value string, dataDir string) (string, bool) {
	if value == "" {
		return value, false
	}
	sep := string(filepath.Separator)                   // 分隔符
	legacyMCP := dir.LegacyMCPRoot(dataDir) + sep       // 旧版MCP
	legacyPlugin := dir.LegacyPluginRoot(dataDir) + sep // 旧版插件

	switch {
	case strings.HasPrefix(value, legacyMCP):
		return dir.MCPRoot(dataDir) + sep + value[len(legacyMCP):], true
	case strings.HasPrefix(value, legacyPlugin):
		return dir.PluginRoot(dataDir) + sep + value[len(legacyPlugin):], true
	}
	return value, false
}
