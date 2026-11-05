package memory

import (
	"context"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/memory/memory_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

// Memory 向前端提供长期记忆管理操作
type Memory struct {
	// 存储
	stor *storage.Storage
}

// NewMemory 使用应用存储创建长期记忆服务
// 接收存储，返回记忆
func NewMemory(stor *storage.Storage) *Memory {
	return &Memory{stor: stor}
}

// ListMemories 返回管理页面所需的长期记忆分页结果
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) ListMemories(ctx context.Context, input memory_dto.ListMemoriesInput) (*memory_dto.ListMemoriesOutput, error) {
	// 条目列表、总量、列出记忆列表时的错误
	items, total, err := m.stor.ListMemories(storage.MemoryListFilter{
		Query:            input.Query,
		Type:             input.Type,
		Target:           input.Target,
		IncludeForgotten: input.IncludeForgotten,
		Offset:           input.Offset,
		Limit:            input.Limit,
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemoryList, err)
	}
	return &memory_dto.ListMemoriesOutput{
		Items: toMemoryItems(items),
		Total: total,
	}, nil
}

// GetMemory 根据标识返回一条长期记忆
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) GetMemory(ctx context.Context, input memory_dto.GetMemoryInput) (*memory_dto.GetMemoryOutput, error) {
	memory, err := m.stor.GetMemory(input.ID) // 记忆、获取记忆时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemoryGet, err)
	}
	return &memory_dto.GetMemoryOutput{Memory: toMemoryItem(*memory)}, nil
}

// CreateMemory 创建一条用户手动录入的长期记忆
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) CreateMemory(ctx context.Context, input memory_dto.CreateMemoryInput) (*memory_dto.CreateMemoryOutput, error) {
	// 创建、创建记忆时的错误
	created, err := m.stor.CreateMemory(data_models.Memory{
		Summary:    input.Summary,
		Content:    input.Content,
		Type:       input.Type,
		Target:     input.Target,
		Source:     input.Source,
		Importance: input.Importance,
		Confidence: input.Confidence,
		Pinned:     input.Pinned,
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemoryCreate, err)
	}
	return &memory_dto.CreateMemoryOutput{Memory: toMemoryItem(*created)}, nil
}

// UpdateMemory 更新长期记忆的可编辑字段
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) UpdateMemory(ctx context.Context, input memory_dto.UpdateMemoryInput) (*memory_dto.UpdateMemoryOutput, error) {
	// 更新、更新记忆时的错误
	updated, err := m.stor.UpdateMemory(input.ID, storage.MemoryUpdate{
		Summary:    &input.Summary,
		Content:    &input.Content,
		Type:       &input.Type,
		Target:     &input.Target,
		Source:     &input.Source,
		Importance: &input.Importance,
		Confidence: &input.Confidence,
		Pinned:     &input.Pinned,
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemoryUpdate, err)
	}
	return &memory_dto.UpdateMemoryOutput{Memory: toMemoryItem(*updated)}, nil
}

// ForgetMemory 保留记忆记录，并将其从后续检索结果中排除
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) ForgetMemory(ctx context.Context, input memory_dto.ForgetMemoryInput) (*memory_dto.ForgetMemoryOutput, error) {
	err := m.stor.ForgetMemory(input.ID) // 遗忘记忆时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemoryForget, err)
	}
	return &memory_dto.ForgetMemoryOutput{}, nil
}

// RestoreMemory 重新启用已遗忘的长期记忆
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) RestoreMemory(ctx context.Context, input memory_dto.RestoreMemoryInput) (*memory_dto.RestoreMemoryOutput, error) {
	err := m.stor.RestoreMemory(input.ID) // 恢复记忆时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemoryRestore, err)
	}
	return &memory_dto.RestoreMemoryOutput{}, nil
}

// GetMemoryStats 返回记忆设置页所需的汇总计数
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) GetMemoryStats(ctx context.Context, input memory_dto.GetMemoryStatsInput) (*memory_dto.GetMemoryStatsOutput, error) {
	stats, err := m.stor.MemoryStats() // 统计、处理记忆统计时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemoryStats, err)
	}
	return &memory_dto.GetMemoryStatsOutput{
		Stats: memory_dto.MemoryStatsItem{
			Total:     stats.Total,
			Active:    stats.Active,
			Forgotten: stats.Forgotten,
			ByTarget:  stats.ByTarget,
			ByType:    stats.ByType,
		},
	}, nil
}

// GetMemorySettings 读取已保存的记忆功能开关
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) GetMemorySettings(ctx context.Context, input memory_dto.GetMemorySettingsInput) (*memory_dto.GetMemorySettingsOutput, error) {
	config, err := m.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemorySettings, err)
	}
	return &memory_dto.GetMemorySettingsOutput{Enabled: config.Memory.Enabled}, nil
}

// SaveMemorySettings 保存记忆功能开关
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (m *Memory) SaveMemorySettings(ctx context.Context, input memory_dto.SaveMemorySettingsInput) (*memory_dto.SaveMemorySettingsOutput, error) {
	config, err := m.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrMemorySettings, err)
	}
	config.Memory.Enabled = input.Enabled
	saveConfigErr := m.saveConfig(config) // 保存配置时的错误
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrMemorySettings, saveConfigErr)
	}
	return &memory_dto.SaveMemorySettingsOutput{}, nil
}
