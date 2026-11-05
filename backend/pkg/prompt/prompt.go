// 提示词定义位于 backend/pkg/prompt/prompt.go
package prompt

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
)

// 注册表
var registry = make(map[string]string)

// 注册当前数据，接收提示词编号、默认内容，无返回值
func Register(promptId string, defaultContent string) {
	registry[promptId] = defaultContent
}

// 加载当前数据，接收提示词编号，返回文本及可能发生的错误
func Load(promptId string) (string, error) {
	defaultContent, ok := registry[promptId] // 默认内容、是否成功取得对应值
	if !ok {
		return "", fmt.Errorf("prompt %q is not registered", promptId)
	}

	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return defaultContent, nil
	}

	customPath := filepath.Join(dataDir, "prompt", promptId, "index.md") // 自定义路径
	content, err := os.ReadFile(customPath)                              // 内容
	if err != nil {
		return defaultContent, nil
	}

	return string(content), nil
}
