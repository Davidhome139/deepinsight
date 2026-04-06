package prompt_engineering

import (
	"context"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/services/ai"
)

// MockAIService 是一个模拟的 AI 服务，用于测试
type MockAIService struct{}

func (m *MockAIService) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	ch := make(chan ai.ChatChunk)
	go func() {
		defer close(ch)
		var response string
		userMessage := req.Messages[len(req.Messages)-1].Content
		
		switch {
		case strings.Contains(userMessage, "用户提示词：\n写一个简单的 Go 程序"):
			response = "这是一个编程任务，请用 Go 语言编写一个简单的程序。程序应该包含 main 函数，能够编译运行并输出一些内容。请确保代码符合 Go 语言的最佳实践，包含适当的注释。"
		case strings.Contains(userMessage, "对话历史：\nuser: 你好，我想学习 Go 语言"):
			response = "根据我们的对话，你正在学习 Go 语言。请写一个简单的 Go 程序，包含 main 函数，能够输出 'Hello, World!'。请使用标准的 Go 语言语法，确保代码可以直接编译运行，并包含简单的注释说明。"
		case strings.Contains(userMessage, "原始用户提示词：\n写一个简单的 Go 程序"):
			response = "请用 Go 语言编写一个简洁、规范的示例程序。要求：1) 包含完整的 main 函数 2) 使用 fmt 包输出问候信息 3) 代码结构清晰，有适当的注释 4) 符合 Go 语言的编码规范 5) 可以直接编译运行。请提供完整的代码实现。"
		default:
			response = "这是一个 mock 响应"
		}
		ch <- ai.ChatChunk{Content: response}
		ch <- ai.ChatChunk{Done: true}
	}()
	return ch, nil
}

func (m *MockAIService) GetAvailableModels() []string {
	return []string{"mock-gpt"}
}

func TestPromptEngineeringCoreFeatures(t *testing.T) {
	ctx := context.Background()

	// 创建测试用的 AI 服务
	aiService := &MockAIService{}

	// 创建提示词工程服务
	promptService := NewService(aiService)

	// 配置
	config := ai.NewDefaultPromptEngineeringConfig()

	// 包含对话历史的测试消息
	messagesWithHistory := []models.Message{
		{
			Role:    "user",
			Content: "你好，我想学习 Go 语言",
		},
		{
			Role:    "assistant",
			Content: "好的！Go 语言是一门优秀的编程语言。你想学习什么内容呢？",
		},
		{
			Role:    "user",
			Content: "写一个简单的 Go 程序",
		},
	}

	t.Run("1. Test UnderstandIntent - 意图理解功能", func(t *testing.T) {
		result, err := promptService.UnderstandIntent(ctx, "写一个简单的 Go 程序", *config)
		if err != nil {
			t.Errorf("UnderstandIntent failed: %v", err)
			return
		}
		t.Logf("UnderstandIntent 结果: %s", result)
	})

	t.Run("2. Test EnhanceWithContext - 上下文增强功能", func(t *testing.T) {
		result, err := promptService.EnhanceWithContext(ctx, "写一个简单的 Go 程序", messagesWithHistory, *config)
		if err != nil {
			t.Errorf("EnhanceWithContext failed: %v", err)
			return
		}
		t.Logf("EnhanceWithContext 结果: %s", result)
	})

	t.Run("3. Test RefineUserPrompt - 提示词重构功能", func(t *testing.T) {
		result, err := promptService.RefineUserPrompt(ctx, "写一个简单的 Go 程序", *config)
		if err != nil {
			t.Errorf("RefineUserPrompt failed: %v", err)
			return
		}
		t.Logf("RefineUserPrompt 结果: %s", result)
	})

	t.Run("4. Test ProcessMessage - 完整流程集成", func(t *testing.T) {
		config.PromptOptimizationEnabled = true
		config.EnableIntentUnderstanding = true
		config.EnableContextEnhancement = true

		resultMessages, err := promptService.ProcessMessage(ctx, messagesWithHistory, *config, "test-conversation-123")
		if err != nil {
			t.Errorf("ProcessMessage failed: %v", err)
			return
		}

		if len(resultMessages) == 0 {
			t.Error("ProcessMessage returned no messages")
			return
		}

		for _, msg := range resultMessages {
			t.Logf("ProcessMessage 消息: [%s] %s", msg.Role, msg.Content)
		}
	})

	t.Log("所有测试完成！")
}
