package runtime_dto

// 封装获取当前状态所需的请求参数
type GetStatusInput struct{}

// 封装获取当前状态返回给调用方的数据
type GetStatusOutput struct {
	// 当前状态
	State string `json:"state"`
	// 版本
	Version string `json:"version"`
	// 安装目录
	InstallDir string `json:"install_dir"`
	// Node.js 可执行文件路径
	NodePath string `json:"node_path"`
	// npm 可执行文件路径
	NpmPath string `json:"npm_path"`
	// 错误消息
	ErrorMsg string `json:"error_msg"`
}
