package skills

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// loadConfig 从数据目录中的 config.json 读取应用配置
// 配置文件尚不存在时返回默认配置
func (s *Skills) loadConfig() (*data_models.Config, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(dataDir, dir.ConfigFileName) // 配置路径
	bytes, err := os.ReadFile(configPath)                    // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			return &data_models.Config{DataDir: dataDir}, nil
		}
		return nil, err
	}
	cfg := &data_models.Config{}            // 配置
	decodeErr := json.Unmarshal(bytes, cfg) // 解析序列化内容时的错误
	if decodeErr != nil {
		return nil, decodeErr
	}
	if cfg.DataDir == "" {
		cfg.DataDir = dataDir
	}
	return cfg, nil
}

// saveConfig 将应用配置写入数据目录下的 config.json
// 接收配置，返回操作错误
func (s *Skills) saveConfig(cfg *data_models.Config) error {
	mkdirErr := os.MkdirAll(cfg.DataDir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return mkdirErr
	}
	content, err := json.MarshalIndent(cfg, "", "  ") // 内容、序列化Indent 对应的数据时的错误
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfg.DataDir, dir.ConfigFileName), content, 0o644)
}

// renameDisabledSkill 在用户技能改名时同步更新禁用名单中的条目
// 接收旧名称、新名称，返回操作错误
func (s *Skills) renameDisabledSkill(oldName string, newName string) error {
	cfg, err := s.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return err
	}
	changed := false // 是否有变化
	// 遍历禁用状态技能列表，逐项取得循环索引、当前名称
	for i, name := range cfg.DisabledSkills {
		if name == oldName {
			cfg.DisabledSkills[i] = newName
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.saveConfig(cfg)
}
