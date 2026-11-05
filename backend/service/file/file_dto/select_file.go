package file_dto

// 封装打开文件选择对话框所需的请求参数
type SelectFileInput struct {
	// 默认文件夹路径
	DefaultFolderPath string `json:"default_folder_path"`
}

// 封装打开文件选择对话框返回给调用方的数据
type SelectFileOutput struct {
	// 文件路径
	FilePath string `json:"file_path"`
}
