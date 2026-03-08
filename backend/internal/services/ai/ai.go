package ai

import (
	"context"
	"fmt"
	"time"

	"backend/internal/config"
	"backend/internal/models"
)

type ChatChunk struct {
	Content       string                   `json:"content"`
	Done          bool                     `json:"done"`
	SearchResults []map[string]interface{} `json:"search_results,omitempty"`
}

type ChatRequest struct {
	UserID         uint             `json:"user_id,omitempty"`
	ConversationID uint             `json:"conversation_id,omitempty"`
	Messages       []models.Message `json:"messages"`
	Model          string           `json:"model"`
	Temperature    float64          `json:"temperature,omitempty"`
	MaxTokens      int              `json:"max_tokens,omitempty"`
	Stream         bool             `json:"stream,omitempty"`
	WebSearch      bool             `json:"web_search,omitempty"`
	SystemPrompt   string           `json:"system_prompt,omitempty"`
}

type AIService interface {
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error)
	GetAvailableModels() []string
}

type aiManager struct {
	providers    map[string]AIService
	modelsConfig *config.ModelsConfig
}

func NewAIManager() AIService {
	m := &aiManager{
		providers: make(map[string]AIService),
	}

	// 加载模型配置
	modelsConfig := config.GetModelsConfig()
	if modelsConfig != nil {
		m.modelsConfig = modelsConfig

		// 为每个启用的提供商创建服务
		for providerName, providerConfig := range modelsConfig.Providers {
			if providerConfig.Enabled {
				m.providers[providerName] = NewUniversalAIService(providerName, providerConfig)
				fmt.Printf("[AI Manager] Initialized provider: %s\n", providerName)
			}
		}
	}

	// 添加 mock 服务用于测试
	m.providers["mock"] = NewMockAIService()

	return m
}

func (m *aiManager) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error) {
	// 根据模型名称检测提供商
	providerName := DetectProviderFromModel(req.Model)

	if providerName == "unknown" {
		providerName = "mock"
		fmt.Printf("[AI Manager] Unknown model: %s, using mock service\n", req.Model)
	}

	fmt.Printf("[AI Manager] Detected provider: %s for model: %s\n", providerName, req.Model)

	// AI model settings are stored in config files, use system configured service directly
	if service, ok := m.providers[providerName]; ok {
		return service.ChatStream(ctx, req)
	}

	// 如果提供商未配置，使用 mock
	if providerName != "mock" {
		fmt.Printf("[AI Manager] Provider '%s' not configured, using mock\n", providerName)
		return m.providers["mock"].ChatStream(ctx, req)
	}

	return nil, fmt.Errorf("provider '%s' is not configured or enabled", providerName)
}

func (m *aiManager) GetAvailableModels() []string {
	models := []string{}

	if m.modelsConfig != nil {
		for providerName, provider := range m.modelsConfig.Providers {
			if provider.Enabled {
				for _, model := range provider.Models {
					models = append(models, fmt.Sprintf("%s:%s", providerName, model))
				}
			}
		}
	}

	if len(models) == 0 {
		models = append(models, "mock-gpt")
	}

	return models
}

type mockAIService struct{}

func NewMockAIService() AIService {
	return &mockAIService{}
}

func (s *mockAIService) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error) {
	ch := make(chan ChatChunk)
	go func() {
		defer close(ch)
		words := []string{"This", " is", " a", " mock", " response", " from", " the", " AI", " assistant.", " How", " can", " I", " help", " you", " today?"}
		for _, word := range words {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				ch <- ChatChunk{Content: word}
			}
		}
		ch <- ChatChunk{Done: true}
	}()
	return ch, nil
}

func (s *mockAIService) GetAvailableModels() []string {
	return []string{"mock-gpt"}
}
