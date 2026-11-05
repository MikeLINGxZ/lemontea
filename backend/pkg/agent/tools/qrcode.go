package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 二维码代码工具名称
const QRCodeToolName = "QRCode"

// 封装处理二维码代码所需的请求参数
type qrCodeInput struct {
	// 文本
	Text string `json:"text" jsonschema:"description=text or URL to encode as a QR code,required"`
}

// 封装处理二维码代码返回给调用方的数据
type qrCodeOutput struct {
	// 文本
	Text string `json:"text"`
	// 二维码
	QR string `json:"qr"`
	// 交互式终端
	InteractiveTerminal bool `json:"interactive_terminal"`
	// 终端状态
	TerminalStatus string `json:"terminal_status"`
	// 终端输出
	TerminalOutput string `json:"terminal_output"`
}

// 构造二维码代码工具，返回工具元信息
func BuildQRCodeTool() ToolMeta {
	return ToolMeta{
		Name:        QRCodeToolName,
		Description: "Convert text or a URL into a scannable terminal QR code. Use this when the user needs to scan a code or open a URL from another device.",
		Category:    CategoryBuiltin,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			// 请求参数
			var input qrCodeInput
			_ = json.Unmarshal(args, &input)
			if strings.TrimSpace(input.Text) == "" {
				return "Generate QR code"
			}
			return "Generate QR code for text"
		},
	}
}

// 构建二维码代码工具，返回Function工具二维码代码输入二维码代码输出
func NewQRCodeTool() *function.FunctionTool[qrCodeInput, qrCodeOutput] {
	meta := BuildQRCodeTool() // 元信息
	return function.NewFunctionTool(
		func(ctx context.Context, input qrCodeInput) (qrCodeOutput, error) {
			// 构建Function工具时调用的回调，接收调用上下文和请求参数，返回操作结果及可能发生的错误
			text := strings.TrimSpace(input.Text) // 文本
			if text == "" {
				return qrCodeOutput{}, errors.New("text is required")
			}
			qr, err := terminalQRCode(text) // 二维码、处理终端二维码代码时的错误
			if err != nil {
				return qrCodeOutput{}, err
			}
			return qrCodeOutput{
				Text:                text,
				QR:                  qr,
				InteractiveTerminal: true,
				TerminalStatus:      "active",
				TerminalOutput:      qr,
			}, nil
		},
		function.WithName(QRCodeToolName),
		function.WithDescription(meta.Description),
	)
}

// 处理终端二维码代码，接收文本，返回文本及可能发生的错误
func terminalQRCode(text string) (string, error) {
	qr, err := qrcode.New(text, qrcode.Medium) // 二维码、构建当前数据时的错误
	if err != nil {
		return "", fmt.Errorf("encode qr: %w", err)
	}
	bitmap := qr.Bitmap() // 位图
	if len(bitmap) == 0 {
		return "", nil
	}

	// 静默
	const quiet = 2
	size := len(bitmap) // 尺寸
	// 本次处理的输出
	var out strings.Builder
	for /* 纵坐标 */ y := -quiet; y < size+quiet; y += 2 {
		for /* 横坐标 */ x := -quiet; x < size+quiet; x++ {
			top := qrCell(bitmap, x, y)      // 顶部
			bottom := qrCell(bitmap, x, y+1) // 底部
			switch {
			case top && bottom:
				out.WriteString("█")
			case top:
				out.WriteString("▀")
			case bottom:
				out.WriteString("▄")
			default:
				out.WriteString(" ")
			}
		}
		out.WriteByte('\n')
	}
	return out.String(), nil
}

// 处理二维码单元格，接收位图、横坐标、纵坐标，返回是否满足条件
func qrCell(bitmap [][]bool, x, y int) bool {
	if y < 0 || y >= len(bitmap) || x < 0 || x >= len(bitmap[y]) {
		return false
	}
	return bitmap[y][x]
}
