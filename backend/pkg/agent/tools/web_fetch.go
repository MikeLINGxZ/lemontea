package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/text/transform"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

const (
	// 拉取超时
	fetchTimeout = 60 * time.Second

	// maxHTMLBodySize 限制读取的 HTTP 响应体大小
	// 防止读取超大网页时卡住，沿用 Claude Code 的 10 MB 上限
	maxHTMLBodySize = 10 * 1024 * 1024

	// maxMarkdownLength 限制 HTML 转换为 Markdown 后的字符数
	// 控制 token 用量，沿用 Claude Code 的 10 万字符上限
	maxMarkdownLength = 50_000

	// maxTextLength 限制提取的纯文本字符数
	maxTextLength = 20_000
)

// 为便于测试提供类型别名
type htmlNode = html.Node

// 元素节点
const elementNode = html.ElementNode

// 拉取HTTP客户端
var fetchHTTPClient = newFetchHTTPClient()

// 构建拉取HTTP客户端，返回客户端
func newFetchHTTPClient() *http.Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List}) // Cookie容器
	return &http.Client{
		Timeout: fetchTimeout,
		Jar:     jar,
		CheckRedirect: /* 执行当前步骤的回调，接收待发送的请求、via 对应的数据，返回操作错误 */ func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// 封装处理网页拉取所需的请求参数
type webFetchInput struct {
	// 地址
	URL string `json:"url"    jsonschema:"description=The URL to fetch (must start with http:// or https://),required"`
	// 格式化
	Format string `json:"format" jsonschema:"description=Output format: text (default) or markdown"`
}

// 封装处理网页拉取返回给调用方的数据
type webFetchOutput struct {
	// 地址
	URL string `json:"url"`
	// 内容
	Content string `json:"content"`
	// 错误
	Error string `json:"error,omitempty"`
}

// fetchContent 请求指定 URL，返回解码后的响应体和 Content-Type
// 最多读取 maxHTMLBodySize 字节，并将非 UTF-8 编码转换为 UTF-8
// 接收本次调用的上下文、客户端、目标地址，返回正文、内容类型、操作错误
// 构造带上下文的网络请求
// 结束时关闭当前资源
// 读取全部内容
func fetchContent(ctx context.Context, client *http.Client, targetURL string) (body string, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil) // 待发送的请求
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/markdown, text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req) // 远程服务响应
	if err != nil {
		return "", "", fmt.Errorf("fetch %s: %w", targetURL, err)
	}

	// 结束时关闭当前资源
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("status %d from %s", resp.StatusCode, targetURL)
	}

	ct := resp.Header.Get("Content-Type")                                   // ct 对应的数据
	rawBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBodySize)) // 原始值字节
	if err != nil {
		return "", "", fmt.Errorf("read body: %w", err)
	}

	// 按字符集解码 HTML 响应，兼容 GBK、GB2312 等编码
	if strings.Contains(ct, "text/html") {
		enc, _, _ := charset.DetermineEncoding(rawBytes, ct)              // 编码器
		decoded, _, decErr := transform.Bytes(enc.NewDecoder(), rawBytes) // 解码结果、处理字节时的错误
		if decErr == nil {
			return string(decoded), ct, nil
		}
	}

	return string(rawBytes), ct, nil
}

// HTML 解析辅助逻辑

// 将输入文本解析成 HTML 节点树，接收HTML文本，返回节点及可能发生的错误
func parseHTML(htmlStr string) (*html.Node, error) {
	return html.Parse(strings.NewReader(htmlStr))
}

// 遍历 HTML 节点并调用传入的访问函数，接收数量、fn 对应的数据，无返回值
func walkHTML(n *html.Node, fn func(*html.Node) bool) {
	if !fn(n) {
		return
	}
	for /* 当前条目 */ c := n.FirstChild; c != nil; c = c.NextSibling {
		walkHTML(c, fn)
	}
}

// 提取文本，接收数量，返回文本
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	// 文本构建器
	var sb strings.Builder
	for /* 当前条目 */ c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}

// isSkippedElement 判断是否应跳过该元素及其子树
// 接收当前标签，返回是否满足条件
func isSkippedElement(tag string) bool {
	switch tag {
	case "script", "style", "nav", "header", "footer", "aside", "noscript":
		return true
	}
	return false
}

// extractMainText 将 HTML 页面中的可见文本提取为纯文本
// 提取标题、段落、列表项和表格单元格
// 找不到结构化元素时，回退到整个 body 的文本
// 接收HTML文本，返回文本
func extractMainText(htmlStr string) string {
	doc, err := parseHTML(htmlStr) // 文档、将输入文本解析成 HTML 节点树时的错误
	if err != nil {
		return ""
	}

	// 组成部分
	var parts []string
	walkHTML(doc /* 遍历 HTML 节点并调用传入的访问函数时调用的回调，接收数量，返回是否满足条件 */, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		if isSkippedElement(n.Data) {
			return false
		}
		switch n.Data {
		case "p", "li", "td", "th", "dt", "dd", "h1", "h2", "h3", "h4", "h5", "h6":
			text := strings.TrimSpace(extractText(n)) // 文本
			if text != "" {
				parts = append(parts, text)
			}
			return false
		}
		return true
	})

	content := strings.Join(parts, "\n") // 内容

	// 找不到结构化元素时返回整个 body 的文本
	if content == "" {
		content = strings.TrimSpace(extractText(doc))
	}

	if len(content) > maxTextLength {
		content = content[:maxTextLength] + "..."
	}
	return content
}

// Markdown 转换

// toMarkdown 将 HTML 字符串转换为 Markdown
// 保留标题、段落、列表和行内链接
// 跳过 script、style、nav、header、footer 和 aside 元素及其子树
// 接收HTML文本，返回文本
func toMarkdown(htmlStr string) string {
	doc, err := parseHTML(htmlStr) // 文档、将输入文本解析成 HTML 节点树时的错误
	if err != nil {
		return ""
	}
	// 文本构建器
	var sb strings.Builder
	renderMarkdownNode(doc, &sb)
	result := strings.TrimSpace(sb.String()) // 去掉首尾空白得到的结果
	if len(result) > maxMarkdownLength {
		result = result[:maxMarkdownLength] + "..."
	}
	return result
}

// 按 HTML 节点类型向构建器写入 Markdown，无返回值
// 标题、段落、列表和链接按对应语法转换，脚本与导航等无关节点在遍历时跳过
func renderMarkdownNode(n *htmlNode, sb *strings.Builder) {
	if n.Type == elementNode {
		if isSkippedElement(n.Data) {
			return
		}
		switch n.Data {
		case "h1":
			if /* 文本 */ text := strings.TrimSpace(extractText(n)); text != "" {
				sb.WriteString("# " + text + "\n\n")
			}
			return
		case "h2":
			if /* 文本 */ text := strings.TrimSpace(extractText(n)); text != "" {
				sb.WriteString("## " + text + "\n\n")
			}
			return
		case "h3":
			if /* 文本 */ text := strings.TrimSpace(extractText(n)); text != "" {
				sb.WriteString("### " + text + "\n\n")
			}
			return
		case "h4", "h5", "h6":
			if /* 文本 */ text := strings.TrimSpace(extractText(n)); text != "" {
				sb.WriteString("#### " + text + "\n\n")
			}
			return
		case "p":
			// 行内
			var inline strings.Builder
			renderInlineNode(n, &inline)
			if /* 文本 */ text := strings.TrimSpace(inline.String()); text != "" {
				sb.WriteString(text + "\n\n")
			}
			return
		case "li":
			// 行内
			var inline strings.Builder
			renderInlineNode(n, &inline)
			if /* 文本 */ text := strings.TrimSpace(inline.String()); text != "" {
				sb.WriteString("- " + text + "\n")
			}
			return
		case "ul", "ol":
			sb.WriteString("\n")
			for /* 当前条目 */ c := n.FirstChild; c != nil; c = c.NextSibling {
				renderMarkdownNode(c, sb)
			}
			sb.WriteString("\n")
			return
		case "a":
			// 行内
			var inline strings.Builder
			renderInlineNode(n, &inline)
			if /* 文本 */ text := strings.TrimSpace(inline.String()); text != "" {
				sb.WriteString(text + "\n\n")
			}
			return
		case "br":
			sb.WriteString("\n")
			return
		}
	}
	for /* 当前条目 */ c := n.FirstChild; c != nil; c = c.NextSibling {
		renderMarkdownNode(c, sb)
	}
}

// renderInlineNode 将节点内容渲染为行内文本，并将 <a> 转换为 Markdown 链接
// 接收数量、文本构建器，无返回值
func renderInlineNode(n *htmlNode, sb *strings.Builder) {
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
		return
	}
	if n.Type == elementNode {
		if isSkippedElement(n.Data) {
			return
		}
		if n.Data == "a" {
			// 链接地址
			var href string
			// 遍历属性，逐项取得属性
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}
			text := strings.TrimSpace(extractText(n)) // 文本
			if href != "" && text != "" {
				sb.WriteString("[" + text + "](" + href + ")")
			} else {
				sb.WriteString(text)
			}
			return
		}
		if n.Data == "br" {
			sb.WriteString("\n")
			return
		}
	}
	for /* 当前条目 */ c := n.FirstChild; c != nil; c = c.NextSibling {
		renderInlineNode(c, sb)
	}
}

// 网页内容获取入口

// 获取网页内容并按指定格式转换输出，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func webFetchFunc(ctx context.Context, input webFetchInput) (webFetchOutput, error) {
	if !strings.HasPrefix(input.URL, "http://") && !strings.HasPrefix(input.URL, "https://") {
		return webFetchOutput{
			URL:   input.URL,
			Error: "URL must start with http:// or https://",
		}, nil
	}

	// 正文、内容类型、拉取内容时的错误
	body, contentType, err := fetchContent(ctx, fetchHTTPClient, input.URL)
	if err != nil {
		return webFetchOutput{URL: input.URL, Error: err.Error()}, nil
	}

	// 直接返回 JSON、纯文本和 Markdown 等非 HTML 响应
	if !strings.Contains(contentType, "text/html") {
		if len(body) > maxMarkdownLength {
			body = body[:maxMarkdownLength] + "..."
		}
		return webFetchOutput{URL: input.URL, Content: body}, nil
	}

	// 内容
	var content string
	if strings.ToLower(input.Format) == "markdown" {
		content = toMarkdown(body)
	} else {
		content = extractMainText(body)
	}

	return webFetchOutput{URL: input.URL, Content: content}, nil
}

// 构建网页拉取工具，返回Function工具网页拉取输入网页拉取输出
func NewWebFetchTool() *function.FunctionTool[webFetchInput, webFetchOutput] {
	return function.NewFunctionTool(
		webFetchFunc,
		function.WithName("web_fetch"),
		function.WithDescription("Fetch a URL and return its content. HTML is converted to readable text (default) or markdown. JSON and plain text are returned as-is. The model decides which URL to visit."),
	)
}

// 构造网页拉取工具的展示信息，返回工具元信息
func WebFetchMeta() ToolMeta {
	return ToolMeta{
		Name:            "web_fetch",
		Description:     "Fetch any URL and read its content",
		Category:        CategoryBuiltin,
		RequiresConfirm: false,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input webFetchInput
			_ = json.Unmarshal(args, &input)
			return fmt.Sprintf("Fetch: %s", input.URL)
		},
	}
}
