// 本文件维护智能体工具元数据注册表
package tools

import (
	"encoding/json"
	"sync"
)

const (
	// 分类内置
	CategoryBuiltin = "builtin"
	// 分类用户
	CategoryUser = "user"
)

// 描述工具元信息使用的数据结构
type ToolMeta struct {
	// 名称
	Name string
	// 描述
	Description string
	// 分类
	Category string
	// 需要确认
	RequiresConfirm bool
	// 格式化用途，接收参数，返回文本
	FormatPurpose func(args json.RawMessage) string
}

// 描述注册表使用的数据结构
type Registry struct {
	// 保护共享状态的互斥锁
	mu sync.RWMutex
	// 工具列表
	tools map[string]ToolMeta
	// 顺序
	order []string
}

// 构建注册表，返回注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolMeta),
	}
}

// 注册注册表，接收元信息，无返回值
func (r *Registry) Register(meta ToolMeta) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if /* 是否存在 */ _, exists := r.tools[meta.Name]; !exists {
		r.order = append(r.order, meta.Name)
	}
	r.tools[meta.Name] = meta
}

// 获取注册表，接收当前名称，返回工具元信息、是否满足条件
func (r *Registry) Get(name string) (ToolMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	meta, ok := r.tools[name] // 元信息、是否成功取得对应值
	return meta, ok
}

// 处理内置工具列表，返回工具元信息列表
func (r *Registry) BuiltinTools() []ToolMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 本次操作的结果
	var result []ToolMeta
	// 遍历顺序，逐项取得当前名称
	for _, name := range r.order {
		if r.tools[name].Category == CategoryBuiltin {
			result = append(result, r.tools[name])
		}
	}
	return result
}

// 处理用户工具列表，返回工具元信息列表
func (r *Registry) UserTools() []ToolMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 本次操作的结果
	var result []ToolMeta
	// 遍历顺序，逐项取得当前名称
	for _, name := range r.order {
		if r.tools[name].Category == CategoryUser {
			result = append(result, r.tools[name])
		}
	}
	return result
}

// 处理启用状态工具列表，接收是否启用，返回工具元信息列表
func (r *Registry) EnabledTools(enabled []string) []ToolMeta {
	enabledSet := make(map[string]bool, len(enabled)) // 启用状态集合
	// 遍历是否启用，逐项取得当前名称
	for _, name := range enabled {
		enabledSet[name] = true
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	// 本次操作的结果
	var result []ToolMeta
	// 遍历顺序，逐项取得当前名称
	for _, name := range r.order {
		meta := r.tools[name] // 元信息
		if meta.Category == CategoryBuiltin || enabledSet[name] {
			result = append(result, meta)
		}
	}
	return result
}
