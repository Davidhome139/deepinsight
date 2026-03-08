package search

import (
	"context"
	"fmt"
	"strings"

	"backend/internal/models"
	"backend/internal/services/ai"
)

// AIProcessor AI预处理器，用于优化搜索查询
type AIProcessor struct {
	aiService ai.AIService
}

// NewAIProcessor 创建AI预处理器
func NewAIProcessor(aiService ai.AIService) *AIProcessor {
	return &AIProcessor{
		aiService: aiService,
	}
}

// ProcessQuery 使用AI处理和优化搜索查询
func (p *AIProcessor) ProcessQuery(ctx context.Context, originalQuery string, history []models.Message, userID uint, model string) (string, error) {
	// 构建AI处理提示
	prompt := p.buildProcessingPrompt(originalQuery, history)

	// 创建AI请求
	req := &ai.ChatRequest{
		UserID: userID,
		Model:  model,
		Messages: []models.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	// 调用AI服务处理
	ch, err := p.aiService.ChatStream(ctx, req)
	if err != nil {
		return originalQuery, fmt.Errorf("AI processing failed: %v", err)
	}

	// 收集AI处理结果
	var processedQuery string
	for chunk := range ch {
		if !chunk.Done {
			processedQuery += chunk.Content
		}
	}

	processedQuery = strings.TrimSpace(processedQuery)

	// 如果AI处理失败，返回原始查询
	if processedQuery == "" {
		return originalQuery, nil
	}

	fmt.Printf("[AI Processor] Original: '%s' -> Processed: '%s'\n", originalQuery, processedQuery)
	return processedQuery, nil
}

// buildProcessingPrompt 构建AI处理提示
func (p *AIProcessor) buildProcessingPrompt(originalQuery string, history []models.Message) string {
	var prompt strings.Builder

	prompt.WriteString("你是一个智能搜索查询优化助手。请根据对话历史优化用户的搜索查询，确保包含所有必要的上下文信息。\n\n")

	// 添加对话历史（如果有）
	if len(history) > 0 {
		prompt.WriteString("对话历史：\n")
		for i := len(history) - 1; i >= 0 && i >= len(history)-3; i-- {
			if history[i].Role == "user" {
				prompt.WriteString(fmt.Sprintf("用户：%s\n", history[i].Content))
			}
		}
		prompt.WriteString("\n")
	}

	// 当前查询
	prompt.WriteString(fmt.Sprintf("当前用户查询：%s\n\n", originalQuery))

	// 指令
	prompt.WriteString("请分析以上内容，输出一个优化后的搜索查询，要求：\n")
	prompt.WriteString("1. 保留所有关键信息（地名、时间、主题等）\n")
	prompt.WriteString("2. 保持查询的简洁性\n")
	prompt.WriteString("3. 确保包含必要的上下文信息\n")
	prompt.WriteString("4. 只返回优化后的查询，不要其他说明\n\n")
	prompt.WriteString("优化后的查询：")

	return prompt.String()
}

// ExtractKeyContext 从对话历史中提取关键上下文
func (p *AIProcessor) ExtractKeyContext(history []models.Message) map[string]string {
	context := make(map[string]string)

	// 从最近的对话中提取关键信息
	for i := len(history) - 1; i >= 0 && i >= len(history)-3; i-- {
		if history[i].Role == "user" {
			content := history[i].Content

			// 简单的关键信息提取（可以根据需要扩展）
			if strings.Contains(content, "天气") {
				context["topic"] = "天气"
			}
			if strings.Contains(content, "旅游") {
				context["topic"] = "旅游"
			}

			// 提取可能的地名（简单实现）
			locations := []string{"北京", "上海", "广州", "深圳", "杭州", "成都", "西安", "重庆",
				"泰国", "日本", "韩国", "菲律宾", "德国", "法国", "英国", "美国"}

			for _, location := range locations {
				if strings.Contains(content, location) {
					context["location"] = location
					break
				}
			}
		}
	}

	return context
}
