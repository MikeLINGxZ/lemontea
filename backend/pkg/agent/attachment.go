package agent

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/model"
)

// MIME类型类型列表
var mimeTypes = map[string]string{
	".c":        "text/x-c",
	".cpp":      "text/x-c++",
	".cs":       "text/x-csharp",
	".css":      "text/css",
	".doc":      "application/msword",
	".docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".gif":      "image/gif",
	".html":     "text/html",
	".java":     "text/x-java",
	".jpg":      "image/jpeg",
	".jpeg":     "image/jpeg",
	".js":       "text/javascript",
	".json":     "application/json",
	".log":      "text/plain",
	".markdown": "text/markdown",
	".md":       "text/markdown",
	".pdf":      "application/pdf",
	".php":      "text/x-php",
	".png":      "image/png",
	".pptx":     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".py":       "text/x-python",
	".rb":       "text/x-ruby",
	".sh":       "application/x-sh",
	".tex":      "text/x-tex",
	".ts":       "application/typescript",
	".txt":      "text/plain",
	".webp":     "image/webp",
}

// Attachment 表示一条用户消息携带的本地文件附件元数据
// 只持久化路径与轻量元信息，文件本身不入库
type Attachment struct {
	// 名称
	Name string `json:"name"`
	// 路径
	Path string `json:"path"`
	// MIME类型
	Mime string `json:"mime"`
	Kind string `json:"kind"` // "image" 表示图片，"file" 表示其他文件
}

// MarshalAttachments 将附件切片序列化为存库的 JSON 字符串
// 空切片返回空串，便于在数据库中保持空状态
// 接收附件列表，返回文本及可能发生的错误
func MarshalAttachments(atts []Attachment) (string, error) {
	if len(atts) == 0 {
		return "", nil
	}
	b, err := json.Marshal(atts) // 数据字节、将数据序列化时的错误
	if err != nil {
		return "", fmt.Errorf("marshal attachments: %w", err)
	}
	return string(b), nil
}

// UnmarshalAttachments 从存库 JSON 字符串解析附件切片
// 空串或纯空白返回 nil 切片，表示无附件
// 接收文本，返回附件列表及可能发生的错误
func UnmarshalAttachments(s string) ([]Attachment, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	// 附件列表
	var atts []Attachment
	err := json.Unmarshal([]byte(s), &atts) // 解析序列化内容时的错误
	if err != nil {
		return nil, fmt.Errorf("unmarshal attachments: %w", err)
	}
	return atts, nil
}

// NormalizeAttachment 根据 Path 补齐缺失的 Name、Mime 和 Kind
// Name 取自 filepath.Base，Mime 先按扩展名推断，再以 application/octet-stream 兜底
// Kind 根据 MIME 类型前缀判定为 image 或 file
// 接收当前对象，返回附件
func NormalizeAttachment(a Attachment) Attachment {
	if a.Name == "" && a.Path != "" {
		a.Name = filepath.Base(a.Path)
	}
	if a.Mime == "" {
		a.Mime = inferMimeFromPath(a.Path)
	}
	if a.Kind == "" {
		a.Kind = kindFromMime(a.Mime)
	}
	return a
}

// 按文件扩展名推断 MIME 类型，未知扩展名使用通用二进制类型，接收文件路径，返回文本
func inferMimeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path)) // 扩展名
	if /* 当前条目 */ m := mime.TypeByExtension(ext); m != "" {
		// mime.TypeByExtension 可能返回带 charset 参数的类型，截取分号前的部分
		if /* 索引 */ idx := strings.Index(m, ";"); idx >= 0 {
			return strings.TrimSpace(m[:idx])
		}
		return m
	}

	mimeType, ok := mimeTypes[ext] // MIME类型类型、是否成功取得对应值
	if !ok {
		return "application/octet-stream"
	}

	return mimeType
}

// 根据 MIME 前缀区分图片附件和普通文件，接收文本，返回文本
func kindFromMime(m string) string {
	if strings.HasPrefix(m, "image/") {
		return "image"
	}
	return "file"
}

// BuildUserMessage 根据文本与附件构造 trpc model.Message
// 根据 Kind 调用 AddImageData 或 AddFileData
// 单个文件读取失败时，跳过对应的 ContentPart
// 在 Content 末尾追加 "[Missing attachment: <name>]"，提示模型该附件已不可用
// 接收内容、附件列表，返回消息
func BuildUserMessage(content string, atts []Attachment) model.Message {
	msg := model.NewUserMessage(content) // 消息
	if len(atts) == 0 {
		return msg
	}

	// 遍历附件列表，逐项取得尚未解析的原始数据
	for _, raw := range atts {
		a := NormalizeAttachment(raw)    // 当前对象
		data, err := os.ReadFile(a.Path) // 待处理的数据、读取文件内容时的错误
		if err != nil {
			msg.Content = appendMissingPlaceholder(msg.Content, a.Name)
			continue
		}
		if a.Kind == "image" {
			format := imageFormatFromMime(a.Mime) // 格式化
			msg.AddImageData(data, "auto", format)
			continue
		}
		// 声明的 MIME 类型为空或 application/octet-stream 时，用 http.DetectContentType 复核
		mimeType := a.Mime
		if mimeType == "" || mimeType == "application/octet-stream" {
			head := data // 开头
			if len(head) > 512 {
				head = head[:512]
			}
			detected := http.DetectContentType(head) // 检测到的
			if detected != "" {
				// 移除检测结果中可能附带的 "; charset=..." 参数
				if /* 索引 */ idx := strings.Index(detected, ";"); idx >= 0 {
					mimeType = strings.TrimSpace(detected[:idx])
				} else {
					mimeType = detected
				}
			}
		}
		if strings.HasPrefix(mimeType, "image/") {
			msg.AddImageData(data, "auto", imageFormatFromMime(mimeType))
			continue
		}
		msg.AddFileData(a.Name, data, mimeType)
	}
	return msg
}

// 将缺失附件的提示追加到用户消息正文，接收内容、当前名称，返回文本
func appendMissingPlaceholder(content, name string) string {
	placeholder := fmt.Sprintf("[Missing attachment: %s]", name) // 占位内容
	if content == "" {
		return placeholder
	}
	return content + "\n" + placeholder
}

// imageFormatFromMime 将 MIME 类型转换为提供方 SDK 使用的图片格式提示
// 格式仅供参考，SDK 接受任意非空字符串，也能处理空字符串
// 遇到未识别的图片 MIME 类型（如 "image/avif"）时返回空字符串
// 接收文本，返回文本
func imageFormatFromMime(m string) string {
	switch m {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpeg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	}
	return ""
}
