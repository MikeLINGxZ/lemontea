package file_dto

// 封装以指定模式打开文件所需的请求参数
type OpenFileInput struct {
	// 路径
	Path string `json:"path"`
}

// 封装以指定模式打开文件返回给调用方的数据
type OpenFileOutput struct {
}
