package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

const (
	// 地区中文中文
	LocaleZhCN = "zh-CN"
	// 地区英文美国
	LocaleEnUS = "en-US"
	// 默认地区
	DefaultLocale = LocaleZhCN
)

// 当前地区
var currentLocale atomic.Value

// 注册前端可订阅的自定义时间事件
func init() {
	currentLocale.Store(DefaultLocale)
}

// 资源
var resources = map[string]map[string]string{
	LocaleZhCN: zhCN,
	LocaleEnUS: enUS,
}

// 把语言区域标识转换为应用支持的标准值，接收地区，返回文本
func NormalizeLocale(locale string) string {
	trimmed := strings.TrimSpace(locale) // 已去除空白
	if /* 是否成功取得对应值 */ _, ok := resources[trimmed]; ok {
		return trimmed
	}
	return DefaultLocale
}

// 设置当前地区，接收地区，无返回值
func SetCurrentLocale(locale string) {
	currentLocale.Store(NormalizeLocale(locale))
}

// 处理当前地区，返回文本
func CurrentLocale() string {
	if /* 值、是否成功取得对应值 */ val, ok := currentLocale.Load().(string); ok && val != "" {
		return NormalizeLocale(val)
	}
	return DefaultLocale
}

// 在指定语言资源中查找消息并填入参数，接收地区、键、参数，返回文本
func T(locale, key string, params map[string]string) string {
	dict := resources[NormalizeLocale(locale)] // 字典
	if /* 文本、是否成功取得对应值 */ text, ok := dict[key]; ok {
		return applyParams(text, params)
	}
	if /* 回退、是否成功取得对应值 */ fallback, ok := resources[DefaultLocale][key]; ok {
		return applyParams(fallback, params)
	}
	return key
}

// 使用当前语言环境格式化消息，接收键、参数，返回文本
func TCurrent(key string, params map[string]string) string {
	return T(CurrentLocale(), key, params)
}

// 处理格式化，接收地区、键、参数，返回文本
func Sprintf(locale, key string, args ...interface{}) string {
	return fmt.Sprintf(T(locale, key, nil), args...)
}

// 将命名参数替换到本地化模板中，接收文本、参数，返回文本
func applyParams(text string, params map[string]string) string {
	if len(params) == 0 {
		return text
	}
	result := text // 本次操作的结果
	// 遍历参数，逐项取得键、待处理的值
	for key, value := range params {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
