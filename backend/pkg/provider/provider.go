package provider

// 定义类型的类型
type Type string

const (
	// Deepseek 对应的数据
	Deepseek Type = "deepseek"
	// 阿里云
	Aliyun Type = "aliyun"
	// Openrouter 对应的数据
	Openrouter Type = "openrouter"
	// Ollama 对应的数据
	Ollama Type = "ollama"
	// 展开AICompatibility
	OpenAiCompatibility Type = "openai_compatibility"
)

// 返回当前值的文本表示，返回文本
func (p Type) String() string {
	return string(p)
}
