package storage

import (
	"errors"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gorm.io/gorm"
)

// CreateProvider 插入供应商记录并返回带有自动生成 ID 的记录
// 接收提供商，返回提供商及可能发生的错误
func (s *Storage) CreateProvider(p data_models.Provider) (*data_models.Provider, error) {
	err := s.sqliteDB.Create(&p).Error // 创建当前数据时的错误
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateModels 批量插入模型记录，GORM 会在传入的切片中直接回填 ID
// 接收模型列表，返回操作错误
func (s *Storage) CreateModels(models *[]data_models.Model) error {
	if len(*models) == 0 {
		return nil
	}
	return s.sqliteDB.Create(models).Error
}

// ListProviders 返回所有未被软删除的供应商记录
func (s *Storage) ListProviders() ([]data_models.Provider, error) {
	// 提供商列表
	var providers []data_models.Provider
	err := s.sqliteDB.Find(&providers).Error // 查找当前数据时的错误
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// ListModelsForProvider 返回 providerID 对应供应商的所有模型记录
// 接收提供商编号，返回模型列表及可能发生的错误
func (s *Storage) ListModelsForProvider(providerID uint) ([]data_models.Model, error) {
	// 模型列表
	var models []data_models.Model
	err := s.sqliteDB.Where("provider_id = ?", providerID).Find(&models).Error // 处理Where 对应的数据时的错误
	if err != nil {
		return nil, err
	}
	return models, nil
}

// UpsertDefaultModel 新增或更新 providerID 对应供应商的默认模型记录
// 接收提供商编号、模型编号，返回操作错误
func (s *Storage) UpsertDefaultModel(providerID, modelID uint) error {
	// 数据行
	var row data_models.ProviderDefaultModel
	err := s.sqliteDB.Where("provider_id = ?", providerID).First(&row).Error // 处理Where 对应的数据时的错误
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return s.sqliteDB.Create(&data_models.ProviderDefaultModel{
			ProviderID: providerID,
			ModelId:    modelID,
		}).Error
	}
	return s.sqliteDB.Model(&row).Update("model_id", modelID).Error
}

// GetDefaultModel 返回 providerID 对应供应商的默认模型记录，未设置时返回 nil
// 接收提供商编号，返回提供商默认模型及可能发生的错误
func (s *Storage) GetDefaultModel(providerID uint) (*data_models.ProviderDefaultModel, error) {
	// 数据行
	var row data_models.ProviderDefaultModel
	err := s.sqliteDB.Where("provider_id = ?", providerID).First(&row).Error // 处理Where 对应的数据时的错误
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// UpdateProvider 更新 p.ID 对应供应商的名称、基础 URL、API 密钥和启用标记
// 接收提供商，返回操作错误
func (s *Storage) UpdateProvider(p data_models.Provider) error {
	return s.sqliteDB.Model(&p).Updates(map[string]any{
		"provider_name": p.ProviderName,
		"base_url":      p.BaseUrl,
		"api_key":       p.ApiKey,
		"enable":        p.Enable,
	}).Error
}

// DeleteProvider 软删除 providerID 对应的供应商记录
// 接收提供商编号，返回操作错误
func (s *Storage) DeleteProvider(providerID uint) error {
	return s.sqliteDB.Delete(&data_models.Provider{}, providerID).Error
}

// DeleteModelsForProvider 软删除 providerID 对应供应商的所有模型记录
// 接收提供商编号，返回操作错误
func (s *Storage) DeleteModelsForProvider(providerID uint) error {
	return s.sqliteDB.Where("provider_id = ?", providerID).Delete(&data_models.Model{}).Error
}

// DeleteDefaultModelRecord 软删除 providerID 对应供应商的默认模型记录
// 接收提供商编号，返回操作错误
func (s *Storage) DeleteDefaultModelRecord(providerID uint) error {
	return s.sqliteDB.Where("provider_id = ?", providerID).Delete(&data_models.ProviderDefaultModel{}).Error
}

// DeleteModel 按 ID 软删除单条模型记录
// 接收模型编号，返回操作错误
func (s *Storage) DeleteModel(modelID uint) error {
	return s.sqliteDB.Delete(&data_models.Model{}, modelID).Error
}

// DeleteProviderCascade 在同一事务中删除供应商、其所有模型和默认模型记录
// 接收提供商编号，返回操作错误
func (s *Storage) DeleteProviderCascade(providerID uint) error {
	return s.sqliteDB.Transaction( /* 在同一个数据库事务中执行写入，失败时返回错误以回滚 */ func(tx *gorm.DB) error {
		// 处理Where 对应的数据时的错误
		checkErr := tx.Where("provider_id = ?", providerID).Delete(&data_models.Model{}).Error
		if checkErr != nil {
			return checkErr
		}
		// check错误
		checkErr2 := tx.Where("provider_id = ?", providerID).Delete(&data_models.ProviderDefaultModel{}).Error
		if checkErr2 != nil {
			return checkErr2
		}
		return tx.Delete(&data_models.Provider{}, providerID).Error
	})
}
