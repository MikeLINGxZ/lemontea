package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	toolmcp "trpc.group/trpc-go/trpc-agent-go/tool/mcp"
)

// ServerConfig 保存应用使用的 MCP 服务器连接配置
type ServerConfig struct {
	// 传输层
	Transport string `json:"transport"`
	// 命令
	Command string `json:"command,omitempty"`
	// 参数
	Args []string `json:"args,omitempty"`
	// 环境变量
	Env map[string]string `json:"env,omitempty"`
	// 超时
	Timeout time.Duration `json:"timeout,omitempty"`
	// 当前工作项目录
	WorkingDir string `json:"working_dir,omitempty"`
	// 服务地址
	ServerURL string `json:"server_url,omitempty"`
	// 请求头
	Headers map[string]string `json:"headers,omitempty"`
	// 描述
	Description string `json:"description,omitempty"`
}

// 默认标准输入输出超时
const defaultStdioTimeout = 30 * time.Second

// 描述原始值服务列表配置使用的数据结构
type rawServerListConfig struct {
	// MCP服务列表
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// loadServerConfig 读取 JSON 格式的 MCP 配置文件
// 接收配置路径，返回服务配置及可能发生的错误
func loadServerConfig(configPath string) (ServerConfig, error) {
	bytes, err := os.ReadFile(configPath) // 原始字节内容、读取文件内容时的错误
	if err != nil {
		return ServerConfig{}, err
	}

	cfg := ServerConfig{}                    // 配置
	decodeErr := json.Unmarshal(bytes, &cfg) // 解析序列化内容时的错误
	if decodeErr != nil {
		return ServerConfig{}, decodeErr
	}

	if cfg.Command != "" || cfg.ServerURL != "" || cfg.Transport != "" {
		if cfg.Transport == "" {
			cfg.Transport = "stdio"
		}
		return cfg, nil
	}

	rawList := rawServerListConfig{}              // 原始值列表
	decodeErr2 := json.Unmarshal(bytes, &rawList) // 解码错误
	if decodeErr2 != nil {
		return ServerConfig{}, decodeErr2
	}
	// 遍历MCP服务列表，逐项取得服务
	for _, server := range rawList.MCPServers {
		if server.Transport == "" {
			server.Transport = "stdio"
		}
		return server, nil
	}

	return ServerConfig{}, fmt.Errorf("invalid mcp config: no server definition found")
}

// toToolSetConfig 将本地配置转换为 trpc-agent-go 的 MCP 配置
// 返回Connection配置
func (c ServerConfig) toToolSetConfig() toolmcp.ConnectionConfig {
	command := c.Command                     // 命令
	args := append([]string(nil), c.Args...) // 参数
	timeout := c.Timeout                     // 本次操作的最长等待时长

	if c.Transport == "stdio" && timeout <= 0 {
		timeout = defaultStdioTimeout
	}

	// trpc-agent-go v0.4.0 的 ConnectionConfig 没有暴露 stdio 的 env 和 working_dir 字段，
	// 因此通过 shell 命令传递本地 stdio MCP 服务器的环境变量
	if c.Transport == "stdio" && (len(c.Env) > 0 || c.WorkingDir != "") {
		parts := []string{} // 组成部分
		if c.WorkingDir != "" {
			parts = append(parts, "cd "+strconv.Quote(c.WorkingDir), "&&")
		}
		envKeys := make([]string, 0, len(c.Env)) // 环境变量键集合
		// 遍历环境变量，逐项取得键
		for key := range c.Env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)
		// 遍历环境变量键集合，逐项取得键
		for _, key := range envKeys {
			parts = append(parts, key+"="+strconv.Quote(c.Env[key]))
		}
		parts = append(parts, strconv.Quote(command))
		// 遍历参数，逐项取得参数
		for _, arg := range args {
			parts = append(parts, strconv.Quote(arg))
		}
		command = "sh"
		args = []string{"-c", strings.Join(parts, " ")}
	}

	return toolmcp.ConnectionConfig{
		Transport:   c.Transport,
		Command:     command,
		Args:        args,
		Timeout:     timeout,
		ServerURL:   c.ServerURL,
		Headers:     c.Headers,
		Description: c.Description,
	}
}
