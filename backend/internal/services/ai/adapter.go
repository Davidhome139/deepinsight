package ai

import (
	"fmt"
	"strings"

	"backend/internal/config"
)

// ProviderAdapter 定义提供商适配器接口
type ProviderAdapter interface {
	// BuildRequest 构建请求体
	BuildRequest(req *ChatRequest) (interface{}, error)
	// BuildHeaders 构建请求头
	BuildHeaders(apiKey string, provider config.ModelProvider) map[string]string
	// GetEndpoint 获取完整的请求URL
	GetEndpoint(provider config.ModelProvider) string
	// ParseStreamResponse 解析流式响应
	ParseStreamResponse(line string) (string, bool, error)
}

// OpenAIAdapter OpenAI 兼容的适配器（OpenAI, DeepSeek, 智谱, 月之暗面, 豆包等）
type OpenAIAdapter struct{}

func (a *OpenAIAdapter) BuildRequest(req *ChatRequest) (interface{}, error) {
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   true,
	}

	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}

	return payload, nil
}

func (a *OpenAIAdapter) BuildHeaders(apiKey string, provider config.ModelProvider) map[string]string {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
	}

	// 添加配置中的自定义 headers
	for k, v := range provider.Headers {
		headers[k] = v
	}

	return headers
}

func (a *OpenAIAdapter) GetEndpoint(provider config.ModelProvider) string {
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "/chat/completions"
	}
	return baseURL + endpoint
}

func (a *OpenAIAdapter) ParseStreamResponse(line string) (string, bool, error) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, nil
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, nil
	}

	// 简化解析，实际实现中需要完整的 JSON 解析
	return data, false, nil
}

// ClaudeAdapter Claude 适配器
type ClaudeAdapter struct{}

func (a *ClaudeAdapter) BuildRequest(req *ChatRequest) (interface{}, error) {
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	payload := map[string]interface{}{
		"model":      req.Model,
		"messages":   messages,
		"stream":     true,
		"max_tokens": 4096, // Claude requires max_tokens
	}

	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}

	return payload, nil
}

func (a *ClaudeAdapter) BuildHeaders(apiKey string, provider config.ModelProvider) map[string]string {
	headers := map[string]string{
		"Content-Type":      "application/json",
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	}

	// 添加配置中的自定义 headers
	for k, v := range provider.Headers {
		headers[k] = v
	}

	return headers
}

func (a *ClaudeAdapter) GetEndpoint(provider config.ModelProvider) string {
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "/v1/messages"
	}
	return baseURL + endpoint
}

func (a *ClaudeAdapter) ParseStreamResponse(line string) (string, bool, error) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, nil
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" || strings.Contains(data, "message_stop") {
		return "", true, nil
	}

	return data, false, nil
}

// AliyunAdapter 阿里云通义千问适配器
type AliyunAdapter struct{}

func (a *AliyunAdapter) BuildRequest(req *ChatRequest) (interface{}, error) {
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	input := map[string]interface{}{
		"messages": messages,
	}

	parameters := map[string]interface{}{
		"result_format": "message",
	}

	if req.Temperature > 0 {
		parameters["temperature"] = req.Temperature
	}

	payload := map[string]interface{}{
		"model":      req.Model,
		"input":      input,
		"parameters": parameters,
	}

	return payload, nil
}

func (a *AliyunAdapter) BuildHeaders(apiKey string, provider config.ModelProvider) map[string]string {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
	}

	for k, v := range provider.Headers {
		headers[k] = v
	}

	return headers
}

func (a *AliyunAdapter) GetEndpoint(provider config.ModelProvider) string {
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "/services/aigc/text-generation/generation"
	}
	return baseURL + endpoint
}

func (a *AliyunAdapter) ParseStreamResponse(line string) (string, bool, error) {
	// 阿里云的响应格式不同，需要特殊处理
	return "", false, nil
}

// MinimaxAdapter MiniMax 适配器
type MinimaxAdapter struct{}

func (a *MinimaxAdapter) BuildRequest(req *ChatRequest) (interface{}, error) {
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, m := range req.Messages {
		msg := map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		}

		// MiniMax 需要 name 字段
		if m.Role == "system" {
			msg["name"] = "MiniMax AI"
		} else if m.Role == "user" {
			msg["name"] = "用户"
		} else if m.Role == "assistant" {
			msg["name"] = "MiniMax AI"
		}

		messages = append(messages, msg)
	}

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   true,
	}

	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}

	return payload, nil
}

func (a *MinimaxAdapter) BuildHeaders(apiKey string, provider config.ModelProvider) map[string]string {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
	}

	for k, v := range provider.Headers {
		headers[k] = v
	}

	return headers
}

func (a *MinimaxAdapter) GetEndpoint(provider config.ModelProvider) string {
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	endpoint := provider.Endpoint
	if endpoint == "" {
		endpoint = "/text/chatcompletion_v2"
	}

	// 确保 endpoint 以 / 开头
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// MiniMax 需要在 URL 中添加 group_id 参数
	if provider.GroupID != "" {
		endpoint = fmt.Sprintf("%s?GroupId=%s", endpoint, provider.GroupID)
	}

	return baseURL + endpoint
}

func (a *MinimaxAdapter) ParseStreamResponse(line string) (string, bool, error) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, nil
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, nil
	}

	return data, false, nil
}

// GetAdapter 根据提供商名称获取对应的适配器
func GetAdapter(providerName string) ProviderAdapter {
	switch providerName {
	case "claude":
		return &ClaudeAdapter{}
	case "aliyun":
		return &AliyunAdapter{}
	case "minimax":
		return &MinimaxAdapter{}
	case "openai", "deepseek", "zhipu", "moonshot", "doubao":
		return &OpenAIAdapter{}
	default:
		return &OpenAIAdapter{} // 默认使用 OpenAI 适配器
	}
}

// DetectProviderFromModel 根据模型名称检测提供商
func DetectProviderFromModel(model string) string {
	model = strings.ToLower(model)

	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") {
		return "openai"
	} else if strings.HasPrefix(model, "claude-") {
		return "claude"
	} else if strings.HasPrefix(model, "qwen") || strings.HasPrefix(model, "qwq") {
		return "aliyun"
	} else if strings.HasPrefix(model, "deepseek") {
		return "deepseek"
	} else if strings.HasPrefix(model, "glm-") {
		return "zhipu"
	} else if strings.HasPrefix(model, "moonshot") {
		return "moonshot"
	} else if strings.HasPrefix(model, "doubao") {
		return "doubao"
	} else if strings.HasPrefix(model, "hunyuan") {
		return "tencent"
	} else if strings.HasPrefix(model, "ernie") {
		return "baidu"
	} else if strings.HasPrefix(model, "abab") || strings.HasPrefix(model, "m2-") || strings.HasPrefix(model, "minimax") {
		return "minimax"
	}

	return "unknown"
}
