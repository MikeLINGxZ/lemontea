package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// 描述模型数据使用的数据结构
type ModelData struct {
	// 唯一标识
	ID string `json:"id"`
	// 创建
	Created int64 `json:"created"`
	// 所属按
	OwnedBy string `json:"owned_by"`
	// 对象
	Object string `json:"object"`
}

// 描述模型列表响应使用的数据结构
type modelsResponse struct {
	// 对象
	Object string `json:"object"`
	// 数据
	Data []ModelData `json:"data"`
}

// GetModels 获取大语言模型供应商提供的可用模型
// 接收服务基础地址、API 密钥，返回模型数据列表及可能发生的错误
func GetModels(baseURL, apiKey string) ([]ModelData, error) {
	client := &http.Client{}                 // 客户端
	url := fmt.Sprintf("%s/models", baseURL) // 地址

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil) // 待发送的请求、构造带上下文的网络请求时的错误
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := client.Do(req) // 远程服务响应
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body) // 正文
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 模型列表响应
	var modelsResp modelsResponse
	decodeErr := json.Unmarshal(body, &modelsResp) // 解析序列化内容时的错误
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to parse response: %w", decodeErr)
	}

	return modelsResp.Data, nil
}
