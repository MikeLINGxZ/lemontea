package storage

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gorm.io/gorm"
)

const (
	// 默认记忆类型
	defaultMemoryType = "information"
	// 默认记忆目标
	defaultMemoryTarget = "user"
	// 默认记忆来源
	defaultMemorySource = "agent"
	// 默认记忆重要度
	defaultMemoryImportance = 50
	// 默认记忆置信度
	defaultMemoryConfidence = 80
)

// MemoryListFilter 定义记忆管理界面的列表筛选条件
type MemoryListFilter struct {
	// 搜索词
	Query string
	// 类型
	Type string
	// 目标
	Target string
	// 包含已遗忘
	IncludeForgotten bool
	// 偏移
	Offset int
	// 上限
	Limit int
}

// MemoryUpdate 包含需要局部更新的记忆字段
type MemoryUpdate struct {
	// 摘要
	Summary *string
	// 内容
	Content *string
	// 类型
	Type *string
	// 目标
	Target *string
	// 来源
	Source *string
	// 重要度
	Importance *int
	// 置信度
	Confidence *int
	// 已置顶
	Pinned *bool
	// 是否已遗忘
	IsForgotten *bool
}

// MemoryStatsSummary 包含设置界面使用的记忆数量统计
type MemoryStatsSummary struct {
	// 总量
	Total int64 `json:"total"`
	// 当前
	Active int64 `json:"active"`
	// 已遗忘
	Forgotten int64 `json:"forgotten"`
	// 按目标
	ByTarget map[string]int64 `json:"by_target"`
	// 按类型
	ByType map[string]int64 `json:"by_type"`
}

// CreateMemory 补齐默认值并计算字符数后保存记忆
// 接收记忆，返回记忆及可能发生的错误
func (s *Storage) CreateMemory(memory data_models.Memory) (*data_models.Memory, error) {
	normalizeMemory(&memory)
	err := s.sqliteDB.Create(&memory).Error // 创建当前数据时的错误
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

// GetMemory 按 ID 加载单条记忆
// 接收唯一标识，返回记忆及可能发生的错误
func (s *Storage) GetMemory(id uint) (*data_models.Memory, error) {
	// 记忆
	var memory data_models.Memory
	err := s.sqliteDB.First(&memory, id).Error // 处理第一项时的错误
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

// ListMemories 返回分页后的记忆及符合条件的记录总数
// 接收筛选条件，返回记忆列表、整数及可能发生的错误
func (s *Storage) ListMemories(filter MemoryListFilter) ([]data_models.Memory, int64, error) {
	limit := filter.Limit // 上限
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset // 偏移
	if offset < 0 {
		offset = 0
	}

	query := s.applyMemoryFilter(s.sqliteDB.Model(&data_models.Memory{}), filter) // 搜索词
	// 总量
	var total int64
	checkErr := query.Count(&total).Error // 处理数量时的错误
	if checkErr != nil {
		return nil, 0, checkErr
	}

	// 记忆列表
	var memories []data_models.Memory
	// 本次操作的错误
	err := query.
		Order("pinned DESC").
		Order("importance DESC").
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&memories).Error
	if err != nil {
		return nil, 0, err
	}
	return memories, total, nil
}

// UpdateMemory 局部更新记忆并重新计算派生字段
// 接收唯一标识、更新，返回记忆及可能发生的错误
// 获取记忆
// 保存存储
func (s *Storage) UpdateMemory(id uint, update MemoryUpdate) (*data_models.Memory, error) {
	memory, err := s.GetMemory(id) // 记忆、获取记忆时的错误
	if err != nil {
		return nil, err
	}
	if update.Summary != nil {
		memory.Summary = strings.TrimSpace(*update.Summary)
	}
	if update.Content != nil {
		memory.Content = strings.TrimSpace(*update.Content)
	}
	if update.Type != nil {
		memory.Type = normalizeMemoryType(*update.Type)
	}
	if update.Target != nil {
		memory.Target = normalizeMemoryTarget(*update.Target)
	}
	if update.Source != nil {
		memory.Source = normalizeMemorySource(*update.Source)
	}
	if update.Importance != nil {
		memory.Importance = clampScore(*update.Importance)
	}
	if update.Confidence != nil {
		memory.Confidence = clampScore(*update.Confidence)
	}
	if update.Pinned != nil {
		memory.Pinned = *update.Pinned
	}
	if update.IsForgotten != nil {
		memory.IsForgotten = *update.IsForgotten
	}
	normalizeMemory(memory)
	checkErr := s.sqliteDB.Save(memory).Error // 保存当前数据时的错误
	if checkErr != nil {
		return nil, checkErr
	}
	return memory, nil
}

// ForgetMemory 将记忆从注入和检索范围中隐藏
// 接收唯一标识，返回操作错误
func (s *Storage) ForgetMemory(id uint) error {
	return s.sqliteDB.Model(&data_models.Memory{}).Where("id = ?", id).Update("is_forgotten", true).Error
}

// RestoreMemory 将已遗忘的记忆恢复为启用状态
// 接收唯一标识，返回操作错误
func (s *Storage) RestoreMemory(id uint) error {
	return s.sqliteDB.Model(&data_models.Memory{}).Where("id = ?", id).Update("is_forgotten", false).Error
}

// SearchMemories 使用 LIKE 搜索记忆并更新召回元数据
// 接收搜索词文本、上限，返回记忆列表及可能发生的错误
func (s *Storage) SearchMemories(queryText string, limit int) ([]data_models.Memory, error) {
	queryText = strings.TrimSpace(queryText)
	if limit <= 0 || limit > 20 {
		limit = 8
	}

	db := s.sqliteDB.Model(&data_models.Memory{}).Where("is_forgotten = ?", false) // 数据库连接
	if queryText != "" {
		db = applyMemoryQuery(db, queryText)
	}

	// 记忆列表
	var memories []data_models.Memory
	// check 对应的数据错误
	checkErr := db.
		Order("pinned DESC").
		Order("importance DESC").
		Order("updated_at DESC").
		Limit(limit).
		Find(&memories).Error
	if checkErr != nil {
		return nil, checkErr
	}
	if len(memories) == 0 && queryText != "" {
		return s.SearchMemories("", limit)
	}
	if len(memories) > 0 {
		now := time.Now()                     // 操作发生时的时间
		ids := make([]uint, 0, len(memories)) // 编号列表
		// 遍历记忆列表，逐项取得循环索引
		for i := range memories {
			ids = append(ids, memories[i].ID)
			memories[i].RecallCount++
			memories[i].LastRecalledAt = &now
		}
		// check错误
		checkErr2 := s.sqliteDB.Model(&data_models.Memory{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"recall_count":     gorm.Expr("recall_count + 1"),
				"last_recalled_at": now,
			}).Error
		if checkErr2 != nil {
			return nil, checkErr2
		}
	}
	return memories, nil
}

// RenderCoreMemory 将启用的记忆格式化为顺序稳定的提示词文本
// 接收用户字符上限、助手字符上限，返回文本及可能发生的错误
func (s *Storage) RenderCoreMemory(userCharLimit int, assistantCharLimit int) (string, error) {
	userMemories, err := s.coreMemoriesForTarget("user", userCharLimit) // 用户记忆列表、检索指定目标下可用于提示词的核心记忆时的错误
	if err != nil {
		return "", err
	}
	assistantMemories, err := s.coreMemoriesForTarget("memory", assistantCharLimit) // 助手记忆列表
	if err != nil {
		return "", err
	}
	if len(userMemories) == 0 && len(assistantMemories) == 0 {
		return "", nil
	}

	// 构建器
	var builder strings.Builder
	if len(userMemories) > 0 {
		builder.WriteString("User memories:\n")
		// 遍历用户记忆列表，逐项取得记忆
		for _, memory := range userMemories {
			builder.WriteString("- ")
			builder.WriteString(memory.Content)
			builder.WriteString("\n")
		}
	}
	if len(assistantMemories) > 0 {
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("Assistant memories:\n")
		// 遍历助手记忆列表，逐项取得记忆
		for _, memory := range assistantMemories {
			builder.WriteString("- ")
			builder.WriteString(memory.Content)
			builder.WriteString("\n")
		}
	}
	return strings.TrimSpace(builder.String()), nil
}

// MemoryStats 返回记忆数量统计
func (s *Storage) MemoryStats() (MemoryStatsSummary, error) {
	// 统计
	stats := MemoryStatsSummary{
		ByTarget: map[string]int64{},
		ByType:   map[string]int64{},
	}
	checkErr := s.sqliteDB.Model(&data_models.Memory{}).Count(&stats.Total).Error // 处理模型时的错误
	if checkErr != nil {
		return stats, checkErr
	}
	// check错误
	checkErr2 := s.sqliteDB.Model(&data_models.Memory{}).Where("is_forgotten = ?", false).Count(&stats.Active).Error
	if checkErr2 != nil {
		return stats, checkErr2
	}
	// check错误
	checkErr3 := s.sqliteDB.Model(&data_models.Memory{}).Where("is_forgotten = ?", true).Count(&stats.Forgotten).Error
	if checkErr3 != nil {
		return stats, checkErr3
	}

	// 目标行数
	var targetRows []struct {
		// 目标
		Target string
		// 数量
		Count int64
	}
	// check错误
	checkErr4 := s.sqliteDB.Model(&data_models.Memory{}).Select("target, count(*) as count").Group("target").Scan(&targetRows).Error
	if checkErr4 != nil {
		return stats, checkErr4
	}
	// 遍历目标行数，逐项取得数据行
	for _, row := range targetRows {
		stats.ByTarget[row.Target] = row.Count
	}

	// 类型行数
	var typeRows []struct {
		// 类型
		Type string
		// 数量
		Count int64
	}
	// check错误
	checkErr5 := s.sqliteDB.Model(&data_models.Memory{}).Select("type, count(*) as count").Group("type").Scan(&typeRows).Error
	if checkErr5 != nil {
		return stats, checkErr5
	}
	// 遍历类型行数，逐项取得数据行
	for _, row := range typeRows {
		stats.ByType[row.Type] = row.Count
	}
	return stats, nil
}

// 把筛选参数转换为记忆查询条件，接收数据库连接、筛选条件，返回数据库
func (s *Storage) applyMemoryFilter(db *gorm.DB, filter MemoryListFilter) *gorm.DB {
	if !filter.IncludeForgotten {
		db = db.Where("is_forgotten = ?", false)
	}
	if strings.TrimSpace(filter.Type) != "" {
		db = db.Where("type = ?", normalizeMemoryType(filter.Type))
	}
	if strings.TrimSpace(filter.Target) != "" {
		db = db.Where("target = ?", normalizeMemoryTarget(filter.Target))
	}
	if strings.TrimSpace(filter.Query) != "" {
		db = applyMemoryQuery(db, filter.Query)
	}
	return db
}

// 把文本检索词加入记忆数据库查询，接收数据库连接、搜索词文本，返回数据库
func applyMemoryQuery(db *gorm.DB, queryText string) *gorm.DB {
	terms := memorySearchTerms(queryText) // 检索词
	if len(terms) == 0 {
		return db
	}
	conditions := make([]string, 0, len(terms)) // conditions 对应的数据
	args := make([]any, 0, len(terms)*2)        // 参数
	// 遍历检索词，逐项取得终端
	for _, term := range terms {
		conditions = append(conditions, "(summary LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\')")
		like := "%" + escapeLike(term) + "%" // 模糊查询
		args = append(args, like, like)
	}
	return db.Where(strings.Join(conditions, " OR "), args...)
}

// 拆分并整理可用于记忆检索的词项，接收搜索词文本，返回文本列表
func memorySearchTerms(queryText string) []string {
	// 字段
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(queryText)), func(r rune) bool {
		// 执行字段工具时调用的回调，接收读取源，返回是否满足条件
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ',' || r == '.' || r == ';' || r == ':' || r == '，' || r == '。' || r == '；' || r == '：'
	})
	seen := map[string]struct{}{}           // seen 对应的数据
	terms := make([]string, 0, len(fields)) // 检索词
	// 遍历字段，逐项取得field 对应的数据
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if /* 是否成功取得对应值 */ _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		terms = append(terms, field)
	}
	if len(terms) == 0 && strings.TrimSpace(queryText) != "" {
		terms = append(terms, strings.TrimSpace(queryText))
	}
	return terms
}

// 检索指定目标下可用于提示词的核心记忆，接收目标、上限，返回记忆列表及可能发生的错误
func (s *Storage) coreMemoriesForTarget(target string, limit int) ([]data_models.Memory, error) {
	// 全部
	var all []data_models.Memory
	// 处理Where 对应的数据时的错误
	err := s.sqliteDB.Where("target = ? AND is_forgotten = ?", target, false).
		Order("pinned DESC").
		Order("importance DESC").
		Order("updated_at DESC").
		Find(&all).Error
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		return all, nil
	}
	selected := make([]data_models.Memory, 0, len(all)) // 选中项
	used := 0                                           // 已使用
	// 遍历全部，逐项取得记忆
	for _, memory := range all {
		next := used + memory.CharCount // 下一项
		if next > limit && len(selected) > 0 {
			continue
		}
		selected = append(selected, memory)
		used = next
	}
	return selected, nil
}

// 规范化记忆，接收记忆，无返回值
func normalizeMemory(memory *data_models.Memory) {
	memory.Summary = strings.TrimSpace(memory.Summary)
	memory.Content = strings.TrimSpace(memory.Content)
	memory.Type = normalizeMemoryType(memory.Type)
	memory.Target = normalizeMemoryTarget(memory.Target)
	memory.Source = normalizeMemorySource(memory.Source)
	memory.CharCount = utf8.RuneCountInString(memory.Content)
	if memory.Importance == 0 {
		memory.Importance = defaultMemoryImportance
	}
	memory.Importance = clampScore(memory.Importance)
	if memory.Confidence == 0 {
		memory.Confidence = defaultMemoryConfidence
	}
	memory.Confidence = clampScore(memory.Confidence)
}

// 规范化记忆类型，接收文本，返回文本
func normalizeMemoryType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fact", "information", "event":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return defaultMemoryType
	}
}

// 规范化记忆目标，接收文本，返回文本
func normalizeMemoryTarget(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "user", "memory":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return defaultMemoryTarget
	}
}

// 规范化记忆来源，接收文本，返回文本
func normalizeMemorySource(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultMemorySource
	}
	return value
}

// 将评分限制在有效范围内，接收整数，返回整数
func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

// 转义模糊查询参数中的通配符，接收文本，返回文本
func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`) // replacer 对应的数据
	return replacer.Replace(strings.TrimSpace(value))
}

// 将相关记忆拼成可附加到提示词的上下文，接收记忆列表，返回文本
func formatMemoryContext(memories []data_models.Memory) string {
	lines := make([]string, 0, len(memories)) // 行列表
	// 遍历记忆列表，逐项取得记忆
	for _, memory := range memories {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", memory.UpdatedAt.Format("2006-01-02"), memory.Summary, memory.Content))
	}
	return strings.Join(lines, "\n")
}
