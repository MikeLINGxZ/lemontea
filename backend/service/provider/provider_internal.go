package provider

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_model"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/dir"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/ierror"
	pkgProvider "gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/provider"
)

// toProviderViewModel 将 data_models.Provider 存储记录转换为对应的视图模型
// 接收提供商、是否默认，返回提供商
func toProviderViewModel(p data_models.Provider, isDefault bool) view_model.Provider {
	return view_model.Provider{
		ID:        p.ID,
		Icon:      providerIconPath(p.ProviderType),
		Type:      p.ProviderType,
		Name:      p.ProviderName,
		BaseURL:   p.BaseUrl,
		Enabled:   p.Enable,
		ApiKey:    p.ApiKey,
		IsDefault: isDefault,
	}
}

// toModelViewModel 将 data_models.Model 存储记录转换为对应的视图模型
// 接收模型、是否默认，返回模型
func toModelViewModel(m data_models.Model, isDefault bool) view_model.Model {
	return view_model.Model{
		ID:         m.ID,
		ProviderId: m.ProviderId,
		Model:      m.Model,
		OwnedBy:    m.OwnedBy,
		Object:     m.Object,
		Enable:     m.Enable,
		Alias:      m.Alias,
		IsCustom:   m.IsCustom,
		IsDefault:  isDefault,
	}
}

// providerIconPath 返回指定提供商类型的内置图标路径
// 接收类型，返回文本
func providerIconPath(t pkgProvider.Type) string {
	// icons 对应的数据
	icons := map[pkgProvider.Type]string{
		pkgProvider.Deepseek:            "/providers/deepseek_icon.png",
		pkgProvider.Aliyun:              "/providers/qwen_icon.png",
		pkgProvider.Ollama:              "/providers/ollama_icon.png",
		pkgProvider.OpenAiCompatibility: "/providers/openai_icon.png",
		pkgProvider.Openrouter:          "/providers/openrouter_icon.png",
	}
	if /* 图标、是否成功取得对应值 */ icon, ok := icons[t]; ok {
		return icon
	}
	return ""
}

// readDefaultProviderID 从配置文件读取 DefaultProviderID，发生错误时返回 0
func readDefaultProviderID() uint {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return 0
	}
	bytes, err := os.ReadFile(filepath.Join(dataDir, dir.ConfigFileName)) // 原始字节内容
	if err != nil {
		return 0
	}
	// 配置
	var cfg data_models.Config
	decodeErr := json.Unmarshal(bytes, &cfg) // 解析序列化内容时的错误
	if decodeErr != nil {
		return 0
	}
	return cfg.DefaultProviderID
}

// writeDefaultProviderID 将 DefaultProviderID 写入配置文件，并保留其余配置字段
// 接收提供商编号，返回操作错误
func writeDefaultProviderID(providerID uint) error {
	dataDir, err := dir.GetDataDir() // 应用数据目录、获取数据目录时的错误
	if err != nil {
		return ierror.Error(ierror.ErrProviderSetDefault, err)
	}
	configPath := filepath.Join(dataDir, dir.ConfigFileName) // 配置路径

	// 配置
	var cfg data_models.Config
	raw, readErr := os.ReadFile(configPath) // 尚未解析的原始数据、读取文件内容时的错误
	if readErr == nil {
		_ = json.Unmarshal(raw, &cfg)
	}
	cfg.DefaultProviderID = providerID

	content, err := json.MarshalIndent(cfg, "", "  ") // 内容
	if err != nil {
		return ierror.Error(ierror.ErrProviderSetDefault, err)
	}
	return ierror.Error(ierror.ErrProviderSetDefault, os.WriteFile(configPath, content, 0o644))
}
