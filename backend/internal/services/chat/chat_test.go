package chat

import (
	"context"
	"testing"

	"backend/internal/services/ai"
	"backend/internal/services/search"
)

// MockAIService 是一个模拟的 AI 服务，用于测试
type MockAIService struct{}

func (m *MockAIService) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	ch := make(chan ai.ChatChunk)
	go func() {
		defer close(ch)
		ch <- ai.ChatChunk{Content: "Hello, I'm a mock AI assistant"}
		ch <- ai.ChatChunk{Done: true}
	}()
	return ch, nil
}

func (m *MockAIService) GetAvailableModels() []string {
	return []string{"mock-gpt"}
}

// MockSearchService 是一个模拟的搜索服务，用于测试
type MockSearchService struct{}

func (m *MockSearchService) Search(ctx context.Context, query string, userID uint, provider string) ([]search.SearchResult, error) {
	return []search.SearchResult{
		{Title: "Test Result", Snippet: "Test snippet", URL: "https://example.com"},
	}, nil
}

func TestSendMessageStreamWithPromptEngineering(t *testing.T) {
	// 创建测试服务
	aiService := &MockAIService{}
	searchService := &MockSearchService{}
	
	chatService := NewChatService(aiService, searchService, nil)

	// 测试用例 1: 启用提示词工程 - 角色专业化
	t.Run("WithPromptEngineeringRole", func(t *testing.T) {
		ctx := context.Background()
		userID := uint(1)
		convID := uint(1)
		content := "Hello"
		model := "mock-gpt"
		webSearch := false
		searchProvider := ""
		mcpTool := ""
		customSystemPrompt := ""
		promptEngineeringConfig := &ai.PromptEngineeringConfig{
			Enabled: true,
			Role:    "doctor",
		}

		_, err := chatService.SendMessageStream(ctx, userID, convID, content, model, webSearch, searchProvider, mcpTool, customSystemPrompt, promptEngineeringConfig)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	// 测试用例 2: 启用提示词工程 - 输出结构化
	t.Run("WithPromptEngineeringOutputFormat", func(t *testing.T) {
		ctx := context.Background()
		userID := uint(1)
		convID := uint(1)
		content := "Hello"
		model := "mock-gpt"
		webSearch := false
		searchProvider := ""
		mcpTool := ""
		customSystemPrompt := ""
		promptEngineeringConfig := &ai.PromptEngineeringConfig{
			Enabled:      true,
			OutputFormat: "json",
		}

		_, err := chatService.SendMessageStream(ctx, userID, convID, content, model, webSearch, searchProvider, mcpTool, customSystemPrompt, promptEngineeringConfig)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	// 测试用例 3: 启用提示词工程 - 思维链增强
	t.Run("WithPromptEngineeringChainOfThought", func(t *testing.T) {
		ctx := context.Background()
		userID := uint(1)
		convID := uint(1)
		content := "Hello"
		model := "mock-gpt"
		webSearch := false
		searchProvider := ""
		mcpTool := ""
		customSystemPrompt := ""
		promptEngineeringConfig := &ai.PromptEngineeringConfig{
			Enabled:        true,
			ChainOfThought: true,
		}

		_, err := chatService.SendMessageStream(ctx, userID, convID, content, model, webSearch, searchProvider, mcpTool, customSystemPrompt, promptEngineeringConfig)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	// 测试用例 4: 禁用提示词工程
	t.Run("WithoutPromptEngineering", func(t *testing.T) {
		ctx := context.Background()
		userID := uint(1)
		convID := uint(1)
		content := "Hello"
		model := "mock-gpt"
		webSearch := false
		searchProvider := ""
		mcpTool := ""
		customSystemPrompt := ""
		promptEngineeringConfig := &ai.PromptEngineeringConfig{
			Enabled: false,
		}

		_, err := chatService.SendMessageStream(ctx, userID, convID, content, model, webSearch, searchProvider, mcpTool, customSystemPrompt, promptEngineeringConfig)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestSendMessageStreamWithRAGWithPromptEngineering(t *testing.T) {
	// 创建测试服务
	aiService := &MockAIService{}
	searchService := &MockSearchService{}
	
	chatService := NewChatService(aiService, searchService, nil)

	// 测试用例: 启用 RAG 和提示词工程
	t.Run("WithRAGAndPromptEngineering", func(t *testing.T) {
		ctx := context.Background()
		userID := uint(1)
		convID := uint(1)
		content := "Hello"
		model := "mock-gpt"
		webSearch := false
		searchProvider := ""
		mcpTool := ""
		customSystemPrompt := ""
		ragEnabled := true
		ragDocIDs := []string{"1", "2"}
		promptEngineeringConfig := &ai.PromptEngineeringConfig{
			Enabled: true,
			Role:    "engineer",
		}

		_, err := chatService.SendMessageStreamWithRAG(ctx, userID, convID, content, model, webSearch, searchProvider, mcpTool, customSystemPrompt, ragEnabled, ragDocIDs, promptEngineeringConfig)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
