package provider

import (
	"context"
	"fmt"
	"sort"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider/provider_dto"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

// Provider 管理模型提供商的持久化及视图模型转换
type Provider struct {
	// 存储接口
	istorage *storage.Storage
}

// NewProvider 使用指定存储创建提供商服务
// 接收存储接口，返回提供商
func NewProvider(istorage *storage.Storage) *Provider {
	return &Provider{istorage: istorage}
}

// 新建模型提供商，接收调用上下文和提供商配置，返回操作结果或错误
// 先保存提供商记录，再为其批量创建模型，最后按请求设置默认项
func (p *Provider) CreateProvider(ctx context.Context, input provider_dto.CreateProviderInput) (*provider_dto.CreateProviderOutput, error) {
	// 创建、创建模型提供商时的错误
	created, err := p.istorage.CreateProvider(data_models.Provider{
		ProviderName: input.ProviderName,
		ProviderType: input.ProviderType,
		BaseUrl:      input.BaseUrl,
		ApiKey:       input.ApiKey,
		Enable:       input.Enable,
	})
	if err != nil {
		return nil, ierror.Error(ierror.ErrProviderCreate, err)
	}

	// 优先使用传入的模型列表，未提供时通过提供商 API 获取
	modelsToSave := input.Models
	if len(modelsToSave) == 0 && input.BaseUrl != "" {
		apiModels, fetchErr := pkgProvider.GetModels(input.BaseUrl, input.ApiKey) // API模型列表、获取模型列表时的错误
		if fetchErr == nil {
			// 遍历API模型列表，逐项取得当前条目
			for _, m := range apiModels {
				modelsToSave = append(modelsToSave, view_model.Model{
					Model:   m.ID,
					OwnedBy: m.OwnedBy,
					Object:  m.Object,
					Enable:  true,
				})
			}
		}
	}

	dbModels := make([]data_models.Model, 0, len(modelsToSave)) // 数据库模型列表
	// 遍历模型列表终点保存，逐项取得当前条目
	for _, m := range modelsToSave {
		dbModels = append(dbModels, data_models.Model{
			ProviderId: created.ID,
			Model:      m.Model,
			OwnedBy:    m.OwnedBy,
			Object:     m.Object,
			Enable:     true,
			IsCustom:   m.IsCustom,
		})
	}
	createErr := p.istorage.CreateModels(&dbModels) // 创建模型列表时的错误
	if createErr != nil {
		return nil, ierror.Error(ierror.ErrProviderCreateModels, createErr)
	}

	if readDefaultProviderID() == 0 {
		writeDefaultProviderIDErr := writeDefaultProviderID(created.ID) // 写入默认提供商编号时的错误
		if writeDefaultProviderIDErr != nil {
			return nil, writeDefaultProviderIDErr
		}
	}

	// 按模型名称查找并保存选定的默认模型
	if input.DefaultModel != nil {
		// 遍历数据库模型列表，逐项取得当前条目
		for _, m := range dbModels {
			if m.Model == *input.DefaultModel {
				_ = p.istorage.UpsertDefaultModel(created.ID, m.ID)
				break
			}
		}
	} else if len(dbModels) > 0 {
		_ = p.istorage.UpsertDefaultModel(created.ID, dbModels[0].ID)
	}

	return &provider_dto.CreateProviderOutput{}, nil
}

// ListProviders 返回所有提供商的视图模型，包含各自的模型列表和默认状态
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Provider) ListProviders(ctx context.Context, input provider_dto.ListProvidersInput) (*provider_dto.ListProvidersOutput, error) {
	providers, err := p.istorage.ListProviders() // 提供商列表、列出提供商列表时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrProviderListProviders, err)
	}

	defaultProviderID := readDefaultProviderID() // 默认提供商编号

	wrappers := make([]provider_dto.ProviderWrapper, 0, len(providers)) // wrappers 对应的数据
	// 遍历提供商列表，逐项取得提供商
	for _, prov := range providers {
		models, err := p.istorage.ListModelsForProvider(prov.ID) // 模型列表、列出模型列表对应提供商时的错误
		if err != nil {
			return nil, ierror.Error(ierror.ErrProviderListModels, err)
		}

		defaultModel, _ := p.istorage.GetDefaultModel(prov.ID) // 默认模型

		vmModels := make([]view_model.Model, 0, len(models)) // 视图模型模型列表
		// 遍历模型列表，逐项取得当前条目
		for _, m := range models {
			isDefault := defaultModel != nil && defaultModel.ModelId == m.ID // 是否默认
			vmModels = append(vmModels, toModelViewModel(m, isDefault))
		}

		wrappers = append(wrappers, provider_dto.ProviderWrapper{
			Provider: toProviderViewModel(prov, prov.ID == defaultProviderID),
			Models:   vmModels,
		})
	}

	return &provider_dto.ListProvidersOutput{Providers: wrappers}, nil
}

// DeleteProvider 删除指定提供商、其所有模型和默认模型记录
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Provider) DeleteProvider(ctx context.Context, input provider_dto.DeleteProviderInput) (*provider_dto.DeleteProviderOutput, error) {
	err := p.istorage.DeleteProviderCascade(uint(input.ProviderId)) // 删除提供商Cascade时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrProviderDelete, err)
	}
	return &provider_dto.DeleteProviderOutput{}, nil
}

// 编辑指定模型提供商，接收调用上下文和变更参数，返回更新结果或错误
// 保存可编辑字段及新模型后重新加载该提供商的模型列表，供设置界面同步状态
func (p *Provider) EditProvider(ctx context.Context, input provider_dto.EditProviderInput) (*provider_dto.EditProviderOutput, error) {
	id := uint(input.ProviderId) // 唯一标识

	// 更新提供商时的错误
	updateErr := p.istorage.UpdateProvider(data_models.Provider{
		OrmModel:     data_models.OrmModel{ID: id},
		ProviderName: input.ProviderName,
		BaseUrl:      input.BaseUrl,
		ApiKey:       input.ApiKey,
		Enable:       input.Enable,
	})
	if updateErr != nil {
		return nil, ierror.Error(ierror.ErrProviderUpdate, updateErr)
	}

	// 收集已有模型的名称，避免重复添加
	existing, err := p.istorage.ListModelsForProvider(id)
	if err != nil {
		return nil, ierror.Error(ierror.ErrProviderListModels, err)
	}
	existingNames := make(map[string]struct{}, len(existing)) // 已存在名称列表
	// 遍历已存在，逐项取得当前条目
	for _, m := range existing {
		existingNames[m.Model] = struct{}{}
	}

	// 新模型列表
	var newModels []data_models.Model
	// 遍历模型列表，逐项取得当前条目
	for _, m := range input.Models {
		if m.ID == 0 {
			if /* dup 对应的数据 */ _, dup := existingNames[m.Model]; !dup {
				newModels = append(newModels, data_models.Model{
					ProviderId: id,
					Model:      m.Model,
					OwnedBy:    m.OwnedBy,
					Object:     m.Object,
					Enable:     true,
					IsCustom:   m.IsCustom,
				})
			}
		}
	}
	if len(newModels) > 0 {
		createErr := p.istorage.CreateModels(&newModels) // 创建模型列表时的错误
		if createErr != nil {
			return nil, ierror.Error(ierror.ErrProviderCreateModels, createErr)
		}
	}

	// 输入指定默认模型时更新默认模型记录
	if input.DefaultModel != nil {
		all, err := p.istorage.ListModelsForProvider(id) // 全部、列出模型列表对应提供商时的错误
		if err != nil {
			return nil, ierror.Error(ierror.ErrProviderListModels, err)
		}
		// 遍历全部，逐项取得当前条目
		for _, m := range all {
			if m.Model == *input.DefaultModel {
				_ = p.istorage.UpsertDefaultModel(id, m.ID)
				break
			}
		}
	}

	return &provider_dto.EditProviderOutput{}, nil
}

// DeleteModel 按标识软删除一条模型记录
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Provider) DeleteModel(ctx context.Context, input provider_dto.DeleteModelInput) (*provider_dto.DeleteModelOutput, error) {
	if input.ModelId <= 0 {
		return nil, ierror.Error(ierror.ErrProviderInvalidModel, fmt.Errorf("invalid model_id: %d", input.ModelId))
	}
	err := p.istorage.DeleteModel(uint(input.ModelId)) // 删除模型时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrProviderDeleteModel, err)
	}
	return &provider_dto.DeleteModelOutput{}, nil
}

// SetDefault 将指定提供商设为全局默认提供商，并设置其默认模型
// ModelId 为 nil 时，使用按名称排序后的第一个模型
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Provider) SetDefault(ctx context.Context, input provider_dto.SetDefaultInput) (*provider_dto.SetDefaultOutput, error) {
	id := uint(input.ProviderId) // 唯一标识

	writeDefaultProviderIDErr := writeDefaultProviderID(id) // 写入默认提供商编号时的错误
	if writeDefaultProviderIDErr != nil {
		return nil, ierror.Error(ierror.ErrProviderSetDefault, writeDefaultProviderIDErr)
	}

	if input.ModelId != nil {
		if *input.ModelId <= 0 {
			return nil, ierror.Error(ierror.ErrProviderInvalidModel, fmt.Errorf("invalid model_id: %d", *input.ModelId))
		}
		_ = p.istorage.UpsertDefaultModel(id, uint(*input.ModelId))
	} else {
		models, err := p.istorage.ListModelsForProvider(id) // 模型列表、列出模型列表对应提供商时的错误
		if err == nil && len(models) > 0 {
			sort.Slice(models /* 比较两个索引对应的条目，返回前一项是否应排在前面 */, func(i, j int) bool { return models[i].Model < models[j].Model })
			_ = p.istorage.UpsertDefaultModel(id, models[0].ID)
		}
	}

	return &provider_dto.SetDefaultOutput{}, nil
}

// RequestProviderModelList 从提供商 API 获取最新的模型列表
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Provider) RequestProviderModelList(ctx context.Context, input provider_dto.RequestProviderModelListInput) (*provider_dto.RequestProviderModelListOutput, error) {
	apiModels, err := pkgProvider.GetModels(input.BaseUrl, input.ApiKey) // API模型列表、获取模型列表时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrProviderFetchModels, err)
	}

	models := make([]view_model.Model, 0, len(apiModels)) // 模型列表
	// 遍历API模型列表，逐项取得当前条目
	for _, m := range apiModels {
		models = append(models, view_model.Model{
			Model:   m.ID,
			OwnedBy: m.OwnedBy,
			Object:  m.Object,
			Enable:  true,
		})
	}

	return &provider_dto.RequestProviderModelListOutput{Models: models}, nil
}

// ProviderAndModelList 返回全部提供商及各自的模型列表，两级列表均按名称升序排列
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (p *Provider) ProviderAndModelList(ctx context.Context, input provider_dto.ProviderAndModelListInput) (*provider_dto.ProviderAndModelListOutput, error) {
	providers, err := p.istorage.ListProviders() // 提供商列表、列出提供商列表时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrProviderListProviders, err)
	}

	sort.Slice(providers /* 比较两个索引对应的条目，返回前一项是否应排在前面 */, func(i, j int) bool {
		return providers[i].ProviderName < providers[j].ProviderName
	})

	defaultProviderID := readDefaultProviderID() // 默认提供商编号

	result := make([]view_model.ProviderModel, 0, len(providers)) // 处理make 对应的数据得到的结果
	// 遍历提供商列表，逐项取得提供商
	for _, prov := range providers {
		models, err := p.istorage.ListModelsForProvider(prov.ID) // 模型列表、列出模型列表对应提供商时的错误
		if err != nil {
			return nil, ierror.Error(ierror.ErrProviderListModels, err)
		}

		sort.Slice(models /* 比较两个索引对应的条目，返回前一项是否应排在前面 */, func(i, j int) bool {
			return models[i].Model < models[j].Model
		})

		defaultModel, _ := p.istorage.GetDefaultModel(prov.ID) // 默认模型

		vmModels := make([]view_model.Model, 0, len(models)) // 视图模型模型列表
		// 遍历模型列表，逐项取得当前条目
		for _, m := range models {
			vmModels = append(vmModels, toModelViewModel(m, defaultModel != nil && defaultModel.ModelId == m.ID))
		}

		result = append(result, view_model.ProviderModel{
			Provider: toProviderViewModel(prov, prov.ID == defaultProviderID),
			Models:   vmModels,
		})
	}

	return &provider_dto.ProviderAndModelListOutput{ProviderModels: result}, nil
}
