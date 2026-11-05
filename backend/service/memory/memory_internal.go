package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory/memory_dto"
)

// 记忆时间格式化
const memoryTimeFormat = time.RFC3339

// 把一组记忆记录转换为前端响应条目，接收记忆列表，返回记忆条目列表
func toMemoryItems(memories []data_models.Memory) []memory_dto.MemoryItem {
	items := make([]memory_dto.MemoryItem, 0, len(memories)) // 条目列表
	// 遍历记忆列表，逐项取得记忆
	for _, memory := range memories {
		items = append(items, toMemoryItem(memory))
	}
	return items
}

// 把一条记忆记录转换为前端响应条目，接收记忆，返回记忆条目
func toMemoryItem(memory data_models.Memory) memory_dto.MemoryItem {
	return memory_dto.MemoryItem{
		ID:             memory.ID,
		Summary:        memory.Summary,
		Content:        memory.Content,
		Type:           memory.Type,
		Target:         memory.Target,
		Source:         memory.Source,
		CharCount:      memory.CharCount,
		EmbeddingID:    memory.EmbeddingID,
		IsForgotten:    memory.IsForgotten,
		RecallCount:    memory.RecallCount,
		LastRecalledAt: formatOptionalTime(memory.LastRecalledAt),
		LastUsedAt:     formatOptionalTime(memory.LastUsedAt),
		Importance:     memory.Importance,
		Confidence:     memory.Confidence,
		Pinned:         memory.Pinned,
		Created:        memory.CreatedAt.Format(memoryTimeFormat),
		Updated:        memory.UpdatedAt.Format(memoryTimeFormat),
	}
}

// 格式化可为空的时间值，未设置时保留空值，接收时间，返回文本
func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(memoryTimeFormat) // 格式化结果
	return &formatted
}

// 加载配置，返回配置及可能发生的错误
func (m *Memory) loadConfig() (*data_models.Config, error) {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return nil, err
	}
	config := &data_models.Config{}                                       // 配置
	bytes, err := os.ReadFile(filepath.Join(dataDir, dir.ConfigFileName)) // 原始字节内容
	if err != nil {
		if os.IsNotExist(err) {
			config.DataDir = dataDir
			config.Memory.Enabled = true
			return config, nil
		}
		return nil, err
	}
	decodeErr := json.Unmarshal(bytes, config) // 解析序列化内容时的错误
	if decodeErr != nil {
		return nil, decodeErr
	}
	if !strings.Contains(string(bytes), `"memory"`) {
		config.Memory.Enabled = true
	}
	if config.DataDir == "" {
		config.DataDir = dataDir
	}
	return config, nil
}

// 保存配置，接收配置，返回操作错误
func (m *Memory) saveConfig(config *data_models.Config) error {
	dataDir := config.DataDir // 应用数据目录
	if dataDir == "" {
		// 本次操作的错误
		var err error
		dataDir, err = dir.GetDataDir()
		if err != nil {
			return err
		}
		config.DataDir = dataDir
	}
	mkdirErr := os.MkdirAll(dataDir, 0o755) // 创建所需目录时的错误
	if mkdirErr != nil {
		return mkdirErr
	}
	content, err := json.MarshalIndent(config, "", "  ") // 内容、序列化Indent 对应的数据时的错误
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, dir.ConfigFileName), content, 0o644)
}
