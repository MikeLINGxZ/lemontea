package skills

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/skills"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/skills/skills_dto"
)

// Skills 通过 RPC 向 Wails 前端提供技能管理操作
type Skills struct {
	// 管理器
	manager *skills.Manager
	// Wails 应用实例
	wailsApp *application.App
}

// NewSkills 使用指定的技能管理器创建技能服务
// 接收管理器，返回技能列表
func NewSkills(manager *skills.Manager) *Skills {
	return &Skills{manager: manager}
}

// Manager 返回底层技能管理器
// 智能体服务使用该管理器接入 Skill 元工具的提供接口
func (s *Skills) Manager() *skills.Manager {
	return s.manager
}

// ListSkills 按名称排序后返回所有已知技能
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Skills) ListSkills(ctx context.Context, input skills_dto.ListSkillsInput) (*skills_dto.ListSkillsOutput, error) {
	all := s.manager.List()                            // 全部
	items := make([]skills_dto.SkillItem, 0, len(all)) // 条目列表
	// 遍历全部，逐项取得技能
	for _, sk := range all {
		items = append(items, toSkillItem(sk))
	}
	return &skills_dto.ListSkillsOutput{Skills: items}, nil
}

// GetSkill 按名称查询并返回一项技能
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Skills) GetSkill(ctx context.Context, input skills_dto.GetSkillInput) (*skills_dto.GetSkillOutput, error) {
	sk, ok := s.manager.Get(input.Name) // 技能、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSkillsNotFound, os.ErrNotExist)
	}
	return &skills_dto.GetSkillOutput{Skill: toSkillItem(sk)}, nil
}

// CreateSkill 新建用户技能并保存到磁盘
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Skills) CreateSkill(ctx context.Context, input skills_dto.CreateSkillInput) (*skills_dto.CreateSkillOutput, error) {
	// 技能
	sk := skills.Skill{
		Name:        input.Name,
		Description: input.Description,
		Body:        input.Body,
		Source:      skills.SourceUser,
	}
	created, err := s.manager.Create(sk) // 创建、创建当前数据时的错误
	if err != nil {
		if errors.Is(err, skills.ErrInvalidSkillName) {
			return nil, ierror.Error(ierror.ErrSkillsInvalidName, err)
		}
		if errors.Is(err, skills.ErrInvalidSkillContent) {
			return nil, ierror.Error(ierror.ErrSkillsInvalidContent, err)
		}
		return nil, ierror.Error(ierror.ErrSkillsWriteFailed, err)
	}
	// 从持久化配置重新同步内存中的禁用状态
	s.refreshDisabledFromConfig()
	return &skills_dto.CreateSkillOutput{Skill: toSkillItem(created)}, nil
}

// UpdateSkill 更新已有非内置技能的描述和正文
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
// 获取技能列表
// 去掉首尾空白
// 更新技能列表
func (s *Skills) UpdateSkill(ctx context.Context, input skills_dto.UpdateSkillInput) (*skills_dto.UpdateSkillOutput, error) {
	existing, ok := s.manager.Get(input.Name) // 已存在、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSkillsNotFound, os.ErrNotExist)
	}
	if existing.Source == skills.SourceBuiltin {
		return nil, ierror.Error(ierror.ErrSkillsBuiltinLocked, os.ErrPermission)
	}
	targetName := strings.TrimSpace(input.NewName) // 目标名称
	if targetName == "" {
		targetName = input.Name
	}
	// 更新
	update := skills.Skill{
		Name:        targetName,
		Description: input.Description,
		Body:        input.Body,
	}
	updated, err := s.manager.Update(input.Name, update) // 更新、更新当前数据时的错误
	if err != nil {
		if errors.Is(err, skills.ErrInvalidSkillName) {
			return nil, ierror.Error(ierror.ErrSkillsInvalidName, err)
		}
		if errors.Is(err, skills.ErrInvalidSkillContent) {
			return nil, ierror.Error(ierror.ErrSkillsInvalidContent, err)
		}
		if errors.Is(err, skills.ErrSkillNameTaken) {
			return nil, ierror.Error(ierror.ErrSkillsNameTaken, err)
		}
		return nil, ierror.Error(ierror.ErrSkillsWriteFailed, err)
	}
	if updated.Name != input.Name {
		renameDisabledSkillErr := s.renameDisabledSkill(input.Name, updated.Name) // 重命名禁用状态技能时的错误
		if renameDisabledSkillErr != nil {
			return nil, ierror.Error(ierror.ErrSettingsSaveConfig, renameDisabledSkillErr)
		}
	}
	// 恢复技能更新前的禁用状态，管理器的 Refresh 会清除此状态
	_ = s.manager.SetDisabled(updated.Name, existing.Disabled)
	s.refreshDisabledFromConfig()
	return &skills_dto.UpdateSkillOutput{Skill: toSkillItem(updated)}, nil
}

// DeleteSkill 从磁盘和缓存中移除指定的非内置技能
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Skills) DeleteSkill(ctx context.Context, input skills_dto.DeleteSkillInput) (*skills_dto.DeleteSkillOutput, error) {
	existing, ok := s.manager.Get(input.Name) // 已存在、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSkillsNotFound, os.ErrNotExist)
	}
	if existing.Source == skills.SourceBuiltin {
		return nil, ierror.Error(ierror.ErrSkillsBuiltinLocked, os.ErrPermission)
	}
	err := s.manager.Delete(input.Name) // 删除当前数据时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSkillsDeleteFailed, err)
	}
	s.refreshDisabledFromConfig()
	return &skills_dto.DeleteSkillOutput{}, nil
}

// ToggleSkill 启用或禁用指定技能，并将变更保存到 config.json
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Skills) ToggleSkill(ctx context.Context, input skills_dto.ToggleSkillInput) (*skills_dto.ToggleSkillOutput, error) {
	cfg, err := s.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSettingsLoadConfig, err)
	}

	// 根据目标状态更新已禁用技能的名称列表
	disabled := make(map[string]bool, len(cfg.DisabledSkills))
	// 遍历禁用状态技能列表，逐项取得当前名称
	for _, name := range cfg.DisabledSkills {
		disabled[name] = true
	}
	if input.Disabled {
		disabled[input.Name] = true
	} else {
		delete(disabled, input.Name)
	}
	names := make([]string, 0, len(disabled)) // 名称列表
	// 遍历是否禁用，逐项取得当前名称
	for name := range disabled {
		names = append(names, name)
	}
	cfg.DisabledSkills = names

	// 先保存更新后的配置，再修改管理器中的状态
	saveConfigErr := s.saveConfig(cfg)
	if saveConfigErr != nil {
		return nil, ierror.Error(ierror.ErrSettingsSaveConfig, saveConfigErr)
	}

	// 将禁用状态同步到内存缓存
	setErr := s.manager.SetDisabled(input.Name, input.Disabled)
	if setErr != nil {
		return nil, ierror.Error(ierror.ErrSkillsNotFound, setErr)
	}

	sk, ok := s.manager.Get(input.Name) // 技能、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSkillsNotFound, os.ErrNotExist)
	}
	return &skills_dto.ToggleSkillOutput{Skill: toSkillItem(sk)}, nil
}

// ImportSkill 读取本地路径上的 SKILL.md，将其写入技能目录后重新加载技能
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func (s *Skills) ImportSkill(ctx context.Context, input skills_dto.ImportSkillInput) (*skills_dto.ImportSkillOutput, error) {
	raw, err := os.ReadFile(input.Path) // 尚未解析的原始数据、读取文件内容时的错误
	if err != nil {
		return nil, ierror.Error(ierror.ErrSkillsLoadFailed, err)
	}
	sk, err := skills.Parse(raw) // 技能
	if err != nil {
		return nil, ierror.Error(ierror.ErrSkillsInvalidContent, err)
	}
	sk.Source = skills.SourceUser

	// 将解析后的技能写入技能根目录
	dataDir, err := dir.GetDataDir()
	if err != nil {
		return nil, ierror.Error(ierror.ErrSkillsWriteFailed, err)
	}
	skillsRoot := dir.SkillsRoot(dataDir)         // 技能列表根目录
	writeErr := skills.WriteSkill(skillsRoot, sk) // 写入技能时的错误
	if writeErr != nil {
		return nil, ierror.Error(ierror.ErrSkillsWriteFailed, writeErr)
	}

	// 使用配置中当前的禁用技能名单重新加载管理器
	s.refreshDisabledFromConfig()

	imported, ok := s.manager.Get(sk.Name) // 已导入、是否成功取得对应值
	if !ok {
		return nil, ierror.Error(ierror.ErrSkillsNotFound, os.ErrNotExist)
	}
	return &skills_dto.ImportSkillOutput{Skill: toSkillItem(imported)}, nil
}

// toSkillItem 将内部 Skill 转换为供前端使用的 SkillItem
// 接收技能，返回技能条目
func toSkillItem(sk skills.Skill) skills_dto.SkillItem {
	return skills_dto.SkillItem{
		Name:        sk.Name,
		Description: sk.Description,
		Body:        sk.Body,
		Source:      string(sk.Source),
		Disabled:    sk.Disabled,
	}
}

// refreshDisabledFromConfig 根据 config.json 中的禁用技能名单重新加载管理器的内存缓存
// 使变更后的磁盘配置和管理器保持相同的技能禁用状态
func (s *Skills) refreshDisabledFromConfig() {
	cfg, err := s.loadConfig() // 配置、加载配置时的错误
	if err != nil {
		// 配置无法读取时，使用空的禁用技能名单
		_ = s.manager.Refresh(nil)
		return
	}
	_ = s.manager.Refresh(cfg.DisabledSkills)
}
