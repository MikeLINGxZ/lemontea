package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 网页搜索工具名称
const WebSearchToolName = "web_search"

var (
	// 网页搜索接口地址
	webSearchEndpoint = "https://www.bing.com/search"
	// 网页搜索HTTP客户端
	webSearchHTTPClient = newFetchHTTPClient()
)

// 封装处理网页搜索所需的请求参数
type webSearchInput struct {
	// 搜索词
	Query string `json:"query" jsonschema:"description=Search query,required"`
	// 上限
	Limit int `json:"limit" jsonschema:"description=Maximum results to return, default 5, max 10"`
}

// 描述网页搜索结果使用的数据结构
type webSearchResult struct {
	// 标题
	Title string `json:"title"`
	// 地址
	URL string `json:"url"`
	// 摘要
	Snippet string `json:"snippet"`
}

// 封装处理网页搜索返回给调用方的数据
type webSearchOutput struct {
	// 搜索词
	Query string `json:"query"`
	// 结果列表
	Results []webSearchResult `json:"results"`
}

// BuildWebSearchTool 返回供注册表使用的网页搜索工具元数据
func BuildWebSearchTool() ToolMeta {
	return ToolMeta{
		Name:            WebSearchToolName,
		Description:     "Search the web for public information and return concise result titles, URLs, and snippets.",
		Category:        CategoryBuiltin,
		RequiresConfirm: false,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input webSearchInput
			_ = json.Unmarshal(args, &input)
			return "Search web: " + input.Query
		},
	}
}

// webSearchFunc 请求配置的搜索端点并解析页面上的可见结果
// 接收调用上下文和请求参数，返回操作结果及可能发生的错误
func webSearchFunc(ctx context.Context, input webSearchInput) (webSearchOutput, error) {
	query := strings.TrimSpace(input.Query) // 搜索词
	if query == "" {
		return webSearchOutput{}, errors.New("query is required")
	}
	limit := input.Limit // 上限
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}
	searchURL, err := buildWebSearchURL(webSearchEndpoint, query, limit) // 搜索地址、构造网页搜索地址时的错误
	if err != nil {
		return webSearchOutput{}, err
	}
	body, _, err := fetchContent(ctx, webSearchHTTPClient, searchURL) // 正文
	if err != nil {
		return webSearchOutput{}, err
	}
	results := parseWebSearchResults(body, limit) // 结果列表
	return webSearchOutput{Query: query, Results: results}, nil
}

// NewWebSearchTool 创建用于公开网页搜索的函数工具
// 返回Function工具网页搜索输入网页搜索输出
func NewWebSearchTool() *function.FunctionTool[webSearchInput, webSearchOutput] {
	meta := BuildWebSearchTool() // 元信息
	return function.NewFunctionTool(
		webSearchFunc,
		function.WithName(WebSearchToolName),
		function.WithDescription(meta.Description),
	)
}

// buildWebSearchURL 将查询参数追加到配置的搜索端点
// 接收接口地址、搜索词、上限，返回文本及可能发生的错误
func buildWebSearchURL(endpoint string, query string, limit int) (string, error) {
	parsed, err := url.Parse(endpoint) // 解析结果、解析当前数据时的错误
	if err != nil {
		return "", fmt.Errorf("parse search endpoint: %w", err)
	}
	q := parsed.Query() // q 对应的数据
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", limit))
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

// parseWebSearchResults 从 HTML 页面中提取 Bing 风格的搜索结果卡片
// 接收原始值HTML、上限，返回网页搜索结果列表
func parseWebSearchResults(rawHTML string, limit int) []webSearchResult {
	doc, err := html.Parse(strings.NewReader(rawHTML)) // 文档、解析当前数据时的错误
	if err != nil {
		return nil
	}
	results := make([]webSearchResult, 0, limit) // 结果列表
	walkHTML(doc /* 遍历 HTML 节点并调用传入的访问函数时调用的回调，接收数量，返回是否满足条件 */, func(n *html.Node) bool {
		if len(results) >= limit {
			return false
		}
		if n.Type == html.ElementNode && n.Data == "li" && hasClass(n, "b_algo") {
			if /* 解析搜索结果节点得到的结果、是否成功取得对应值 */ result, ok := parseSearchResultNode(n); ok {
				results = append(results, result)
			}
			return false
		}
		return true
	})
	return results
}

// parseSearchResultNode 从单个结果节点中提取标题、URL 和摘要
// 接收数量，返回网页搜索结果、是否满足条件
func parseSearchResultNode(n *html.Node) (webSearchResult, bool) {
	// 本次操作的结果
	var result webSearchResult
	walkHTML(n /* 遍历 HTML 节点并调用传入的访问函数时调用的回调，接收当前，返回是否满足条件 */, func(current *html.Node) bool {
		if current.Type == html.ElementNode && current.Data == "a" && result.URL == "" {
			result.Title = strings.TrimSpace(extractText(current))
			result.URL = attrValue(current, "href")
			return false
		}
		return true
	})
	walkHTML(n /* 遍历 HTML 节点并调用传入的访问函数时调用的回调，接收当前，返回是否满足条件 */, func(current *html.Node) bool {
		if current.Type == html.ElementNode && current.Data == "p" && result.Snippet == "" {
			result.Snippet = strings.TrimSpace(extractText(current))
			return false
		}
		return true
	})
	return result, result.Title != "" && result.URL != ""
}

// hasClass 判断 HTML 节点的 class 属性是否包含指定类名
// 接收数量、类名名称，返回是否满足条件
func hasClass(n *html.Node, className string) bool {
	classes := strings.Fields(attrValue(n, "class")) // 类名列表
	// 遍历类名列表，逐项取得当前
	for _, current := range classes {
		if current == className {
			return true
		}
	}
	return false
}

// attrValue 返回指定名称的 HTML 属性值
// 接收数量、当前名称，返回文本
func attrValue(n *html.Node, name string) string {
	// 遍历属性，逐项取得属性
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}
