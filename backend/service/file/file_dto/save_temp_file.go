package file_dto

// 封装保存临时文件所需的请求参数
type SaveTempFileInput struct {
	Name string `json:"name"` // 原始文件名，例如 screenshot.png
	Data string `json:"data"` // Base64 编码的文件内容
	Mime string `json:"mime"` // 文件的 MIME 类型，例如 image/png
}

// 封装保存临时文件返回给调用方的数据
type SaveTempFileOutput struct {
	// 文件路径
	FilePath string `json:"file_path"`
}
