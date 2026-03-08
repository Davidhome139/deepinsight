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

type openAIService struct {
	config config.AIProviderConfig
}

func NewOpenAIService(cfg config.AIProviderConfig) AIService {
	return &openAIService{config: cfg}
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	// Tencent Hunyuan specific search parameter
	SearchOptions map[string]interface{} `json:"search_options,omitempty"`
	// Aliyun/OpenAI standard or other variants
	EnableSearch bool `json:"enable_search,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *openAIService) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error) {
	if s.config.BaseURL == "" {
		return nil, fmt.Errorf("AI provider base URL is empty. Please check your configuration.")
	}

	if s.config.APIKey == "" || s.config.APIKey == "YOUR_API_KEY" {
		return nil, fmt.Errorf("AI provider API key is not configured. Please set your API key in settings.")
	}

	ch := make(chan ChatChunk)

	messages := make([]openAIMessage, 0, len(req.Messages))

	// Note: Custom system prompt is already merged into message history at chat service level
	// No need to prepend it separately here

	for _, m := range req.Messages {
		messages = append(messages, openAIMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	payload := openAIChatRequest{
		Model:        req.Model,
		Messages:     messages,
		Stream:       true,
		EnableSearch: req.WebSearch,
	}

	// Add Tencent specific search options if it's a Hunyuan model
	if req.WebSearch && strings.Contains(strings.ToLower(req.Model), "hunyuan") {
		payload.SearchOptions = map[string]interface{}{
			"enable_search": true,
		}
		fmt.Printf("[Tencent] Enabling native search for model: %s\n", req.Model)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[AI Service] Request to %s/chat/completions with model: %s, search: %v\n",
		strings.TrimRight(s.config.BaseURL, "/"), payload.Model, req.WebSearch)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", strings.TrimRight(s.config.BaseURL, "/")), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[AI Service] Error response (%d): %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("LLM provider returned error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[AI Service] Successfully connected, starting to stream response\n")

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- ChatChunk{Done: true}
				return
			}

			var streamResp struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					ch <- ChatChunk{Content: content}
				}
			}
		}
	}()

	return ch, nil
}

func (s *openAIService) GetAvailableModels() []string {
	// This would ideally come from the config or an API call
	return []string{s.config.BaseURL}
}
