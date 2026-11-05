package cli

import (
	"context"
	"encoding/json"
	"log"
	"strings"
)

// 上限生成器帮助字符列表
const maxGeneratorHelpChars = 12000

// 清单生成器系统提示词
const manifestGeneratorSystemPrompt = `You convert a CLI tool's --help output into a JSON manifest for the Lemontea CLI plugin system.

The manifest schema is:
{
  "name": "<package name>",
  "version": "<package version>",
  "description": "<short>",
  "executable": "<provided>",
  "isolation": "isolated",
  "tools": [
    {
      "name": "<snake_case_tool_name>",
      "description": "<what it does>",
      "input_schema": { "type": "object", "properties": { "<key>": {"type":"string"} }, "required": [] },
      "argv_template": ["<subcmd>", "--flag", "{key}"],
      "output_mode": "json" | "text" | "lines",
      "timeout_seconds": 60,
      "requires_confirm": true|false,
      "enabled": true
    }
  ]
}

Rules:
- Map each useful subcommand to a tool. Skip "help" / "version" / "completion".
- requires_confirm: true for verbs that mutate state (send/delete/create/update/upload). false for read verbs (list/get/show).
- argv_template must use {placeholder} tokens that match input_schema.properties keys.
- Any {placeholder} the CLI cannot accept as missing must be listed in input_schema.required. Placeholders left out of required are treated as optional: at runtime, if the caller omits the value, the engine drops that argv segment AND the immediately preceding "--flag" segment, so optional pairs vanish cleanly. Pair optional values with a "--flag" segment (e.g. ["--data", "{data}"]); never put an optional placeholder in a positional slot.
- output_mode: prefer "json" if the CLI has a --json flag for that subcommand, else "text".
- Return ONLY the JSON object, no prose.`

// OneshotCaller 是生成清单时注入的模型调用依赖
type OneshotCaller func(ctx context.Context, system, user string) (string, error)

// GenerateParams 包含生成 CLI 清单草稿所需的输入
type GenerateParams struct {
	// 帮助文本
	HelpText string
	// 包名名称
	PackageName string
	// 包名元信息
	PackageMeta PackageMeta
	// 可执行文件
	Executable string
	// 调用方
	Caller OneshotCaller
}

// Generate 请求模型起草清单，解析或校验失败时返回不含工具的最简清单
// 接收本次调用的上下文、生成参数，返回清单及可能发生的错误
func Generate(ctx context.Context, p GenerateParams) (Manifest, error) {
	userPrompt := buildGeneratorUserPrompt(p)                            // 用户提示词
	raw, err := p.Caller(ctx, manifestGeneratorSystemPrompt, userPrompt) // 尚未解析的原始数据、处理调用方时的错误
	if err != nil {
		log.Printf("cli manifest generator caller failed: %v", err)
		return stubManifest(p), nil
	}

	manifest, parseErr := parseGeneratedManifest(raw) // 清单、解析生成结果清单时的错误
	if parseErr != nil {
		log.Printf("cli manifest generator parse failed: %v", parseErr)
		return stubManifest(p), nil
	}
	manifest.Executable = p.Executable
	if manifest.Isolation == "" {
		manifest.Isolation = IsolationIsolated
	}
	validateErr := Validate(manifest) // 校验当前数据时的错误
	if validateErr != nil {
		log.Printf("cli manifest generator validation failed: %v", validateErr)
		return stubManifest(p), nil
	}
	return manifest, nil
}

// buildGeneratorUserPrompt 生成指定 CLI 包对应的用户提示词
// 接收生成参数，返回文本
func buildGeneratorUserPrompt(p GenerateParams) string {
	return "Package: " + p.PackageName + " v" + p.PackageMeta.Version + "\n" +
		"Description: " + p.PackageMeta.Description + "\n" +
		"Executable: " + p.Executable + "\n\n" +
		"--help output:\n```\n" + clampGeneratorHelpText(p.HelpText) + "\n```"
}

// clampGeneratorHelpText 截断过长的帮助输出，避免提示词过长拖慢模型响应
// 接收帮助文本，返回文本
func clampGeneratorHelpText(helpText string) string {
	trimmed := strings.TrimSpace(helpText) // 已去除空白
	if len(trimmed) <= maxGeneratorHelpChars {
		return trimmed
	}

	// 已省略标记
	const omittedMarker = "\n...\n[help output truncated]\n...\n"
	headLen := 8000                                                 // 开头长度
	tailLen := maxGeneratorHelpChars - headLen - len(omittedMarker) // 末尾长度
	if tailLen < 0 {
		tailLen = 0
	}
	if headLen > len(trimmed) {
		headLen = len(trimmed)
	}
	start := trimmed[:headLen] // 开始
	end := ""                  // 结束
	if tailLen > 0 && tailLen < len(trimmed)-headLen {
		end = trimmed[len(trimmed)-tailLen:]
	}
	return start + omittedMarker + end
}

// parseGeneratedManifest 解码模型输出中的原始 JSON 或代码围栏内的 JSON
// 接收尚未解析的原始数据，返回清单及可能发生的错误
func parseGeneratedManifest(raw string) (Manifest, error) {
	trimmed := strings.TrimSpace(raw) // 已去除空白
	// 清单
	var manifest Manifest
	decodeErr := json.Unmarshal([]byte(trimmed), &manifest) // 解析序列化内容时的错误
	if decodeErr == nil {
		return manifest, nil
	}

	start := strings.Index(trimmed, "{")   // 开始
	end := strings.LastIndex(trimmed, "}") // 结束
	if start < 0 || end < 0 || end < start {
		return Manifest{}, json.Unmarshal([]byte(trimmed), &manifest)
	}
	decodeErr2 := json.Unmarshal([]byte(trimmed[start:end+1]), &manifest) // 解码错误
	if decodeErr2 != nil {
		return Manifest{}, decodeErr2
	}
	return manifest, nil
}

// stubManifest 创建可安全保存并供后续编辑的最简清单
// 接收生成参数，返回清单
func stubManifest(p GenerateParams) Manifest {
	return Manifest{
		Name:        p.PackageName,
		Version:     p.PackageMeta.Version,
		Description: p.PackageMeta.Description,
		Executable:  p.Executable,
		Isolation:   IsolationIsolated,
		Tools:       []Tool{},
	}
}
