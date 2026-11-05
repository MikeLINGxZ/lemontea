package tools

import (
	"context"
	"encoding/json"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 封装处理日期时间所需的请求参数
type dateTimeInput struct {
	// 格式化
	Format string `json:"format" jsonschema:"description=Go time format string (default: RFC3339)"`
	// 时区
	Timezone string `json:"timezone" jsonschema:"description=IANA timezone name (default: Local)"`
}

// 封装处理日期时间返回给调用方的数据
type dateTimeOutput struct {
	// 日期时间
	DateTime string `json:"datetime"`
	// Unix 对应的整数
	Unix int64 `json:"unix"`
	// 时区
	Timezone string `json:"timezone"`
}

// 按用户指定时区返回当前日期和时间，接收调用上下文和请求参数，返回操作结果及可能发生的错误
func dateTimeFunc(ctx context.Context, input dateTimeInput) (dateTimeOutput, error) {
	loc := time.Local // 时区位置
	if input.Timezone != "" {
		parsed, err := time.LoadLocation(input.Timezone) // 解析结果、加载Location 对应的数据时的错误
		if err != nil {
			return dateTimeOutput{}, err
		}
		loc = parsed
	}

	now := time.Now().In(loc) // 操作发生时的时间
	format := time.RFC3339    // 格式化
	if input.Format != "" {
		format = input.Format
	}

	return dateTimeOutput{
		DateTime: now.Format(format),
		Unix:     now.Unix(),
		Timezone: loc.String(),
	}, nil
}

// 构建日期时间工具，返回Function工具日期时间输入日期时间输出
func NewDateTimeTool() *function.FunctionTool[dateTimeInput, dateTimeOutput] {
	return function.NewFunctionTool(
		dateTimeFunc,
		function.WithName("datetime"),
		function.WithDescription("Get the current date and time in a specified format and timezone"),
	)
}

// 构造日期时间工具的展示信息，返回工具元信息
func DateTimeMeta() ToolMeta {
	return ToolMeta{
		Name:            "datetime",
		Description:     "Get the current date and time",
		Category:        CategoryBuiltin,
		RequiresConfirm: false,
		FormatPurpose: /* 执行当前步骤的回调，接收参数，返回文本 */ func(args json.RawMessage) string {
			return "Get current date and time"
		},
	}
}
