package runtime

// NodeLTSVersion 指定应用固定使用的 Node.js LTS 版本
const NodeLTSVersion = "v22.11.0"

// NodeDistBaseURL 是 Node.js 官方发行包的下载基础地址
const NodeDistBaseURL = "https://nodejs.org/dist"

// RuntimeStateFileName 指定保存运行时下载状态的文件名
const RuntimeStateFileName = "state.json"

// NodeSubdir 指定数据目录中用于保存 Node.js 运行时文件的子目录名
const NodeSubdir = "runtime/node"
