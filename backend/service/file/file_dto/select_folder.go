package file_dto

// 封装打开目录选择对话框所需的请求参数
type SelectFolderInput struct {
	// 文件夹路径
	FolderPath string `json:"folder_path"`
}

// 封装打开目录选择对话框返回给调用方的数据
type SelectFolderOutput struct {
	// 文件夹路径
	FolderPath string `json:"folder_path"`
}
