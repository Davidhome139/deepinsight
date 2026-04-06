package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"backend/internal/config"
)

// UniversalAIService 通用的 AI 服务，支持多种提供商
type UniversalAIService struct {
	provider config.ModelProvider
	adapter  ProviderAdapter
}

// NewUniversalAIService 创建通用 AI 服务
func NewUniversalAIService(providerName string, provider config.ModelProvider) AIService {
	adapter := GetAdapter(providerName)

	// 如果 BaseURL 包含 compatible-mode，使用 OpenAI 适配器（兼容模式）
	if strings.Contains(provider.BaseURL, "compatible-mode") {
		adapter = &OpenAIAdapter{}
	}

	return &UniversalAIService{
		provider: provider,
		adapter:  adapter,
	}
}

func (s *UniversalAIService) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error) {
	// 验证配置
	if s.provider.BaseURL == "" {
		return nil, fmt.Errorf("provider base URL is empty")
	}
	if s.provider.APIKey == "" || s.provider.APIKey == "YOUR_API_KEY" {
		return nil, fmt.Errorf("API key is not configured")
	}

	ch := make(chan ChatChunk)

	// 使用适配器构建请求
	payload, err := s.adapter.BuildRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 获取完整的 endpoint
	endpoint := s.adapter.GetEndpoint(s.provider)

	fmt.Printf("[Universal AI] Request to %s with model: %s\n", endpoint, req.Model)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 使用适配器构建请求头
	headers := s.adapter.BuildHeaders(s.provider.APIKey, s.provider)
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Universal AI] Error response (%d): %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("provider returned error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[Universal AI] Successfully connected, streaming response\n")

	go s.streamResponse(resp, ch)

	return ch, nil
}

func (s *UniversalAIService) streamResponse(resp *http.Response, ch chan ChatChunk) {
	defer resp.Body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		// fmt.Printf("[Universal AI] Received line: %s\n", line)

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			ch <- ChatChunk{Done: true}
			return
		}

		// 解析不同格式的响应
		content, done := s.parseResponse(data)
		if done {
			ch <- ChatChunk{Done: true}
			return
		}

		if content != "" {
			// fmt.Printf("[Universal AI] Parsed content: %s\n", content)
			ch <- ChatChunk{Content: content}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("[Universal AI] Scanner error: %v\n", err)
	}

	fmt.Printf("[Universal AI] Stream ended\n")
}

func (s *UniversalAIService) parseResponse(data string) (string, bool) {
	// 尝试解析 OpenAI/MiniMax 流式格式
	var openAIResp struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
			Message      *struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &openAIResp); err == nil {
		if len(openAIResp.Choices) > 0 {
			choice := openAIResp.Choices[0]

			// 如果有 finish_reason 为 "stop"，表示结束
			if choice.FinishReason == "stop" {
				// 如果还有 delta.content，先返回内容
				if choice.Delta.Content != "" {
					return choice.Delta.Content, false
				}
				// 如果有 message.content（最后的汇总），跳过（因为已经流式输出过了）
				return "", true
			}

			// 返回 delta 内容
			return choice.Delta.Content, false
		}
	}

	// 尝试解析 Claude 格式
	var claudeResp struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}

	if err := json.Unmarshal([]byte(data), &claudeResp); err == nil {
		if claudeResp.Type == "message_stop" || claudeResp.Delta.Type == "message_stop" {
			return "", true
		}
		if claudeResp.Delta.Type == "text_delta" {
			return claudeResp.Delta.Text, false
		}
	}

	// 尝试解析阿里云格式
	var aliyunResp struct {
		Output struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		} `json:"output"`
	}

	if err := json.Unmarshal([]byte(data), &aliyunResp); err == nil {
		if len(aliyunResp.Output.Choices) > 0 {
			choice := aliyunResp.Output.Choices[0]
			if choice.FinishReason == "stop" {
				return "", true
			}
			return choice.Message.Content, false
		}
	}

	return "", false
}

func (s *UniversalAIService) GetAvailableModels() []string {
	return s.provider.Models
}
