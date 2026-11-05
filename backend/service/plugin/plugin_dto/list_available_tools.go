package plugin_dto

// 描述工具条目使用的数据结构
type ToolItem struct {
	// 唯一标识
	ID string `json:"id"`
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 分类
	Category string `json:"category"`
}

// 封装列出可用项工具列表所需的请求参数
type ListAvailableToolsInput struct{}

// 封装列出可用项工具列表返回给调用方的数据
type ListAvailableToolsOutput struct {
	// 工具列表
	Tools []ToolItem `json:"tools"`
}
