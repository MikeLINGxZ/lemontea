package skills

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// Manager 在内存中合并内置技能、磁盘技能和禁用标记，
// 并提供将变更保存到磁盘的方法
type Manager struct {
	// 保护共享状态的互斥锁
	mu sync.RWMutex
	// 根目录目录
	rootDir string
	skills  map[string]Skill // 按名称索引
}

// NewManager 创建从 rootDir 读取用户技能的管理器
// 调用 Refresh 填充内存缓存
// 接收根目录目录，返回管理器
func NewManager(rootDir string) *Manager {
	return &Manager{
		rootDir: rootDir,
		skills:  make(map[string]Skill),
	}
}

// Refresh 从内嵌资源和磁盘目录重新加载技能，并根据禁用名称更新内存状态
// 磁盘技能优先于同名内置技能
// 接收禁用状态名称列表，返回操作错误
func (m *Manager) Refresh(disabledNames []string) error {
	disabled := make(map[string]bool, len(disabledNames)) // 是否禁用
	// 遍历禁用状态名称列表，逐项取得数量
	for _, n := range disabledNames {
		disabled[n] = true
	}

	builtin, err := LoadBuiltin() // 内置、加载内置时的错误
	if err != nil {
		return err
	}
	disk, err := LoadFromDir(m.rootDir) // disk 对应的数据
	if err != nil {
		return err
	}

	// 先合并内置技能，再用磁盘技能覆盖同名条目
	merged := make(map[string]Skill, len(builtin)+len(disk))
	// 遍历内置，逐项取得当前值
	for _, s := range builtin {
		s.Disabled = disabled[s.Name]
		merged[s.Name] = s
	}
	// 遍历disk 对应的数据，逐项取得当前值
	for _, s := range disk {
		s.Disabled = disabled[s.Name]
		merged[s.Name] = s
	}

	m.mu.Lock()
	m.skills = merged
	m.mu.Unlock()
	return nil
}

// List 按名称排序返回全部技能
func (m *Manager) List() []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Skill, 0, len(m.skills)) // 本次处理的输出
	// 遍历技能列表，逐项取得当前值
	for _, s := range m.skills {
		out = append(out, s)
	}
	sort.Slice(out /* 比较两个索引对应的条目，返回前一项是否应排在前面 */, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Enabled 只返回启用的技能
func (m *Manager) Enabled() []Skill {
	all := m.List()                   // 全部
	out := make([]Skill, 0, len(all)) // 本次处理的输出
	// 遍历全部，逐项取得当前值
	for _, s := range all {
		if !s.Disabled {
			out = append(out, s)
		}
	}
	return out
}

// Get 按名称返回一项技能
// 接收当前名称，返回技能、是否满足条件
func (m *Manager) Get(name string) (Skill, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.skills[name] // 当前值、是否成功取得对应值
	return s, ok
}

// Create 将新技能保存到磁盘并刷新缓存
// 新技能不能与内置技能同名
// 接收技能，返回技能及可能发生的错误
func (m *Manager) Create(s Skill) (Skill, error) {
	validateErr := Validate(s) // 校验当前数据时的错误
	if validateErr != nil {
		return Skill{}, validateErr
	}
	if s.Source == SourceBuiltin {
		return Skill{}, errors.New("cannot create builtin skill")
	}
	m.mu.RLock()
	if /* 已存在、是否成功取得对应值 */ existing, ok := m.skills[s.Name]; ok && existing.Source == SourceBuiltin {
		m.mu.RUnlock()
		return Skill{}, errors.New("builtin skill with this name already exists")
	}
	m.mu.RUnlock()

	if s.Source == "" {
		s.Source = SourceUser
	}
	writeErr := WriteSkill(m.rootDir, s) // 写入技能时的错误
	if writeErr != nil {
		return Skill{}, writeErr
	}
	// 重新从磁盘加载，取得已填充元数据的技能
	_ = m.Refresh(nil)
	got, ok := m.Get(s.Name) // got 对应的数据、是否成功取得对应值
	if !ok {
		return Skill{}, errors.New("skill not found after create")
	}
	return got, nil
}

// Update 覆盖磁盘上的现有技能，内置技能不可更新
// 接收当前名称、技能，返回技能及可能发生的错误
// 写入技能
// 获取管理器
func (m *Manager) Update(name string, s Skill) (Skill, error) {
	m.mu.RLock()
	existing, ok := m.skills[name] // 已存在、是否成功取得对应值
	if !ok {
		m.mu.RUnlock()
		return Skill{}, errors.New("skill not found")
	}
	targetName := s.Name // 目标名称
	if targetName == "" {
		targetName = name
	}
	if targetName != name {
		if /* 是否存在 */ _, exists := m.skills[targetName]; exists {
			m.mu.RUnlock()
			return Skill{}, ErrSkillNameTaken
		}
	}
	m.mu.RUnlock()
	if existing.Source == SourceBuiltin {
		return Skill{}, errors.New("builtin skill is read-only")
	}
	s.Name = targetName
	s.Source = existing.Source
	writeErr := WriteSkill(m.rootDir, s) // 写入技能时的错误
	if writeErr != nil {
		return Skill{}, writeErr
	}
	if targetName != name {
		deleteErr := DeleteSkill(m.rootDir, name) // 删除技能时的错误
		if deleteErr != nil {
			return Skill{}, deleteErr
		}
	}
	_ = m.Refresh(nil)
	got, ok := m.Get(targetName) // got 对应的数据
	if !ok {
		return Skill{}, errors.New("skill not found after update")
	}
	return got, nil
}

// Delete 从磁盘和缓存移除非内置技能
// 接收当前名称，返回操作错误
func (m *Manager) Delete(name string) error {
	m.mu.RLock()
	existing, ok := m.skills[name] // 已存在、是否成功取得对应值
	m.mu.RUnlock()
	if !ok {
		return errors.New("skill not found")
	}
	if existing.Source == SourceBuiltin {
		return errors.New("builtin skill cannot be deleted")
	}
	err := DeleteSkill(m.rootDir, name) // 删除技能时的错误
	if err != nil {
		return err
	}
	_ = m.Refresh(nil)
	return nil
}

// SetDisabled 更新内存中的技能禁用标记
// 调用方需要单独持久化 DisabledNames
// 接收当前名称、是否禁用，返回操作错误
func (m *Manager) SetDisabled(name string, disabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.skills[name] // 当前值、是否成功取得对应值
	if !ok {
		return errors.New("skill not found")
	}
	s.Disabled = disabled
	s.UpdatedAt = time.Now()
	m.skills[name] = s
	return nil
}

// RootDir 返回磁盘上的技能根目录
func (m *Manager) RootDir() string {
	return m.rootDir
}
