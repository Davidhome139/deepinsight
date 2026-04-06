package prompt_engineering

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/services/ai"
)

// Service 提示词工程服务接口
type Service interface {
	ProcessMessage(ctx context.Context, messages []models.Message, config ai.PromptEngineeringConfig, conversationID string) ([]models.Message, error)
	GenerateChainOfThought(ctx context.Context, prompt string, maxChains int) ([]ChainOfThought, error)
	GetRoleSystemPrompt(role string, customInfo string) string
	FormatOutput(content string, format string, schema string) (string, error)
	SelfEvaluate(ctx context.Context, conversationID string, prompt string, output string) (EvaluationResult, error)
	OptimizeOutput(ctx context.Context, conversationID string, prompt string, output string, evaluation EvaluationResult) (string, error)
	RefineUserPrompt(ctx context.Context, userPrompt string, config ai.PromptEngineeringConfig) (string, error)
	EnhanceWithContext(ctx context.Context, userPrompt string, messages []models.Message, config ai.PromptEngineeringConfig) (string, error)
	UnderstandIntent(ctx context.Context, userPrompt string, config ai.PromptEngineeringConfig) (string, error)
}

// ChainOfThought 思维链
type ChainOfThought struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// EvaluationResult 评估结果
type EvaluationResult struct {
	Score       float64   `json:"score"`
	Feedback    string    `json:"feedback"`
	Suggestions []string  `json:"suggestions"`
	Improved    bool      `json:"improved"`
}

// RoleTemplate 角色模板
type RoleTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
}

type service struct {
	aiService ai.AIService
}

func NewService(aiService ai.AIService) Service {
	return &service{
		aiService: aiService,
	}
}

func (s *service) ProcessMessage(ctx context.Context, messages []models.Message, config ai.PromptEngineeringConfig, conversationID string) ([]models.Message, error) {
	if !config.Enabled {
		return messages, nil
	}

	rootPrompt := `CRITICAL REQUIREMENT - YOU MUST FOLLOW THIS RULE FOR EVERY SINGLE RESPONSE:
After providing your answer, you MUST ALWAYS end with exactly three follow-up questions. This is MANDATORY for EVERY response, not just the first one.

Use this exact format at the end of your response:

**延续探讨：**
1. [第一个相关问题]
2. [第二个相关问题]
3. [第三个相关问题]

These questions should be:
- Relevant to the current discussion
- Help explore different aspects of the topic
- Encourage the user to continue the conversation

Remember: This is required for ALL responses in this conversation, not just your first answer.`

	if config.Role != "" && config.Role != "default" {
		systemPrompt := s.GetRoleSystemPrompt(config.Role, config.CustomRoleInfo)
		systemPrompt += "\n\n" + rootPrompt
		hasSystemPrompt := false
		for i, msg := range messages {
			if msg.Role == "system" {
				if messages[i].Content != systemPrompt {
					messages[i].Content = systemPrompt
				}
				hasSystemPrompt = true
				break
			}
		}
		if !hasSystemPrompt {
			messages = append([]models.Message{{
				Role:    "system",
				Content: systemPrompt,
			}}, messages...)
		}
		metadata := map[string]string{
			"Role":             config.Role,
			"Custom Role Info": config.CustomRoleInfo,
			"Output Format":    config.OutputFormat,
			"Chain of Thought": fmt.Sprintf("%v", config.ChainOfThought),
			"Tool Calls":       fmt.Sprintf("%v", config.ToolCalls),
			"Self Evaluation":  fmt.Sprintf("%v", config.SelfEvaluation),
		}
		s.savePromptToFile(conversationID, "system", systemPrompt, metadata)
	} else {
		hasSystemPrompt := false
		for i, msg := range messages {
			if msg.Role == "system" {
				if !strings.Contains(messages[i].Content, "延续探讨：") {
					messages[i].Content += "\n\n" + rootPrompt
				}
				hasSystemPrompt = true
				metadata := map[string]string{
					"Role":             "default",
					"Output Format":    config.OutputFormat,
					"Chain of Thought": fmt.Sprintf("%v", config.ChainOfThought),
					"Tool Calls":       fmt.Sprintf("%v", config.ToolCalls),
					"Self Evaluation":  fmt.Sprintf("%v", config.SelfEvaluation),
				}
				s.savePromptToFile(conversationID, "system", messages[i].Content, metadata)
				break
			}
		}
		if !hasSystemPrompt {
			systemPrompt := "You are a helpful assistant.\n\n" + rootPrompt
			messages = append([]models.Message{{
				Role:    "system",
				Content: systemPrompt,
			}}, messages...)
			metadata := map[string]string{
				"Role":             "default",
				"Output Format":    config.OutputFormat,
				"Chain of Thought": fmt.Sprintf("%v", config.ChainOfThought),
				"Tool Calls":       fmt.Sprintf("%v", config.ToolCalls),
				"Self Evaluation":  fmt.Sprintf("%v", config.SelfEvaluation),
			}
			s.savePromptToFile(conversationID, "system", systemPrompt, metadata)
		}
	}

	if config.PromptOptimizationEnabled {
		fmt.Printf("[Prompt Optimization] Optimization enabled, starting process\n")
		var lastUserIndex int
		var lastUserContent string
		foundUser := false
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				lastUserIndex = i
				lastUserContent = messages[i].Content
				foundUser = true
				break
			}
		}

		if foundUser {
			fmt.Printf("[Prompt Optimization] Found user message to optimize: %s\n", lastUserContent)
			optimizationMetadata := map[string]string{
				"Original Prompt": lastUserContent,
				"Optimization Level": config.PromptOptimizationLevel,
			}

			optimizedPrompt := lastUserContent

			intentResult, err := s.UnderstandIntent(ctx, optimizedPrompt, config)
			if err == nil {
				optimizationMetadata["After Intent Understanding"] = intentResult
				optimizedPrompt = intentResult
			}

			enhancedResult, err := s.EnhanceWithContext(ctx, optimizedPrompt, messages, config)
			if err == nil {
				optimizationMetadata["After Context Enhancement"] = enhancedResult
				optimizedPrompt = enhancedResult
			}

			refinedResult, err := s.RefineUserPrompt(ctx, optimizedPrompt, config)
			if err == nil {
				optimizationMetadata["After Refinement"] = refinedResult
				optimizedPrompt = refinedResult
			}

			messages[lastUserIndex].Content = optimizedPrompt
			fmt.Printf("[Prompt Optimization] Final optimized prompt: %s\n", optimizedPrompt)
			s.savePromptToFile(conversationID, "prompt_optimization", optimizedPrompt, optimizationMetadata)
		} else {
			fmt.Printf("[Prompt Optimization] No user message found to optimize\n")
		}
	} else {
		fmt.Printf("[Prompt Optimization] Optimization disabled, skipping\n")
	}

	if config.OutputFormat != "default" {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				messages[i].Content += fmt.Sprintf("\n\nPlease output your response in %s format.", config.OutputFormat)
				if config.Schema != "" {
					messages[i].Content += fmt.Sprintf("\n\nSchema: %s", config.Schema)
				}
				break
			}
		}
	}

	if config.ChainOfThought {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				messages[i].Content += "\n\nPlease use chain of thought reasoning in your response."
				break
			}
		}
	}

	if config.ToolCalls {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				messages[i].Content += "\n\nYou can use tools to help answer this question if needed."
				break
			}
		}
	}

	if config.SelfEvaluation {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				messages[i].Content += "\n\nAfter providing your answer, please evaluate your response and suggest improvements."
				break
			}
		}
	}

	if config.ChainOfThought {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				messages[i].Content += "\n\nLet me think step by step:"
				break
			}
		}
	}

	return messages, nil
}

func (s *service) GenerateChainOfThought(ctx context.Context, prompt string, maxChains int) ([]ChainOfThought, error) {
	if maxChains <= 0 {
		maxChains = 3
	}

	chains := make([]ChainOfThought, 0, maxChains)

	for i := 0; i < maxChains; i++ {
		req := &ai.ChatRequest{
			Messages: []models.Message{
				{
					Role:    "user",
					Content: prompt + "\n\nLet me think step by step:",
				},
			},
			Model:  "deepseek-chat",
			Stream: false,
		}

		ch, err := s.aiService.ChatStream(ctx, req)
		if err != nil {
			continue
		}

		var chainContent strings.Builder
		for chunk := range ch {
			if !chunk.Done {
				chainContent.WriteString(chunk.Content)
			}
		}

		chain := ChainOfThought{
			ID:      fmt.Sprintf("chain_%d", i),
			Content: chainContent.String(),
			Score:   1.0 - float64(i)*0.1,
		}

		chains = append(chains, chain)
	}

	return chains, nil
}

func (s *service) GetRoleSystemPrompt(role string, customInfo string) string {
	templates := map[string]string{
		"default":   "You are a helpful assistant.",
		"doctor":    "You are a senior medical doctor specializing in internal medicine. You provide accurate medical information and advice based on evidence-based medicine.",
		"engineer":  "You are a senior software engineer with 10+ years of experience. You provide expert advice on software development, architecture, and best practices.",
		"lawyer":    "You are a professional lawyer specializing in contract law. You provide legal advice and explanations based on current laws and regulations.",
		"teacher":   "You are an experienced teacher with expertise in education. You explain concepts clearly and provide helpful learning resources.",
		"writer":    "You are a professional writer with expertise in creative writing and content creation. You craft engaging and well-structured content.",
	}

	systemPrompt, ok := templates[role]
	if !ok {
		systemPrompt = templates["default"]
	}

	if customInfo != "" {
		systemPrompt += "\n\n" + customInfo
	}

	return systemPrompt
}

func (s *service) FormatOutput(content string, format string, schema string) (string, error) {
	switch format {
	case "json":
		if !strings.HasPrefix(content, "{") {
			content = "{" + content + "}"
		}
	case "xml":
		if !strings.HasPrefix(content, "<") {
			content = "<root>" + content + "</root>"
		}
	case "yaml":
	}

	return content, nil
}

func (s *service) SelfEvaluate(ctx context.Context, conversationID string, prompt string, output string) (EvaluationResult, error) {
	evaluationPrompt := fmt.Sprintf(`Please evaluate the quality of the following response to the prompt:

Prompt: %s

Response: %s

Evaluation criteria:
1. Accuracy: How accurate is the response to the prompt?
2. Completeness: How complete is the response?
3. Relevance: How relevant is the response to the prompt?
4. Clarity: How clear and well-structured is the response?
5. Depth: How deep and insightful is the response?

Please provide:
1. A score from 0-100
2. Detailed feedback on the strengths and weaknesses
3. Specific suggestions for improvement

Format your response as JSON with the following structure:
{
  "score": 85,
  "feedback": "The response is accurate but could be more complete.",
  "suggestions": ["Add more examples", "Provide more detailed explanations"]
}`, prompt, output)

	metadata := map[string]string{
		"Original Prompt": prompt,
		"Response Length": fmt.Sprintf("%d", len(output)),
	}
	s.savePromptToFile(conversationID, "evaluation", evaluationPrompt, metadata)

	req := &ai.ChatRequest{
		Messages: []models.Message{
			{
				Role:    "user",
				Content: evaluationPrompt,
			},
		},
		Model:  "deepseek-chat",
		Stream: false,
	}

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return EvaluationResult{}, err
	}

	var evaluationContent strings.Builder
	for chunk := range ch {
		evaluationContent.WriteString(chunk.Content)
	}

	var result struct {
		Score       float64   `json:"score"`
		Feedback    string    `json:"feedback"`
		Suggestions []string  `json:"suggestions"`
	}

	err = json.Unmarshal([]byte(evaluationContent.String()), &result)
	if err != nil {
		return EvaluationResult{
			Score:       70,
			Feedback:    "Failed to parse evaluation result",
			Suggestions: []string{"Improve response quality"},
			Improved:    false,
		}, nil
	}

	return EvaluationResult{
		Score:       result.Score,
		Feedback:    result.Feedback,
		Suggestions: result.Suggestions,
		Improved:    false,
	}, nil
}

func (s *service) savePromptToFile(conversationID string, promptType string, content string, metadata map[string]string) error {
	promptDir := filepath.Join("/app/chat", conversationID)
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		return fmt.Errorf("failed to create prompt directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	randomID := fmt.Sprintf("%d", time.Now().UnixNano())
	filename := filepath.Join(promptDir, fmt.Sprintf("%s_%s_%s.md", timestamp, promptType, randomID))

	var mdContent strings.Builder
	mdContent.WriteString(fmt.Sprintf("# %s Prompt\n\n", promptType))
	mdContent.WriteString(fmt.Sprintf("**Generated at:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if len(metadata) > 0 {
		mdContent.WriteString("## Metadata\n\n")
		for key, value := range metadata {
			mdContent.WriteString(fmt.Sprintf("- **%s:** %s\n", key, value))
		}
		mdContent.WriteString("\n")
	}

	if originalPrompt, ok := metadata["Original Prompt"]; ok {
		mdContent.WriteString("## 原始用户提示词\n\n")
		mdContent.WriteString("```\n")
		mdContent.WriteString(originalPrompt)
		mdContent.WriteString("\n```\n\n")
	}

	optimizationKeys := []string{"After Intent Understanding", "After Context Enhancement", "After Refinement"}
	hasOptimizationProcess := false
	for _, key := range optimizationKeys {
		if _, ok := metadata[key]; ok {
			hasOptimizationProcess = true
			break
		}
	}

	if hasOptimizationProcess {
		mdContent.WriteString("## 优化过程\n\n")
		for _, key := range optimizationKeys {
			if value, ok := metadata[key]; ok {
				mdContent.WriteString(fmt.Sprintf("### %s\n\n", key))
				mdContent.WriteString("```\n")
				mdContent.WriteString(value)
				mdContent.WriteString("\n```\n\n")
			}
		}
	}

	mdContent.WriteString("## Prompt Content\n\n")
	mdContent.WriteString("```\n")
	mdContent.WriteString(content)
	mdContent.WriteString("\n```\n")

	if err := os.WriteFile(filename, []byte(mdContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	fmt.Printf("[Prompt Engineering] Saved %s prompt to: %s\n", promptType, filename)
	return nil
}

func (s *service) OptimizeOutput(ctx context.Context, conversationID string, prompt string, output string, evaluation EvaluationResult) (string, error) {
	optimizationPrompt := fmt.Sprintf(`Please optimize the following response based on the evaluation feedback:

Prompt: %s

Original response: %s

Evaluation feedback: %s

Improvement suggestions: %s

Please provide an improved version of the response that addresses the feedback and suggestions.`, prompt, output, evaluation.Feedback, strings.Join(evaluation.Suggestions, "; "))

	metadata := map[string]string{
		"Original Prompt":   prompt,
		"Evaluation Score":  fmt.Sprintf("%.2f", evaluation.Score),
	}
	s.savePromptToFile(conversationID, "optimization", optimizationPrompt, metadata)

	req := &ai.ChatRequest{
		Messages: []models.Message{
			{
				Role:    "user",
				Content: optimizationPrompt,
			},
		},
		Model:  "deepseek-chat",
		Stream: false,
	}

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return output, err
	}

	var optimizedContent strings.Builder
	for chunk := range ch {
		optimizedContent.WriteString(chunk.Content)
	}

	return optimizedContent.String(), nil
}

func (s *service) RefineUserPrompt(ctx context.Context, userPrompt string, config ai.PromptEngineeringConfig) (string, error) {
	fmt.Printf("[Prompt Optimization] Starting RefineUserPrompt for: %s\n", userPrompt)
	
	systemPrompt := `你是一位专业的提示词工程师。请根据以下要求优化用户的提示词：

优化要求：
1. 补充缺失信息 - 如果提示词中有不明确的地方，请合理补充必要的上下文信息
2. 明确目标和期望 - 清晰定义用户想要达到的目标和期望结果
3. 设定期望输出 - 明确说明期望的输出格式、长度、风格等
4. 改进提示词结构 - 使提示词更加清晰、有逻辑、有条理
5. 保持用户原始意图 - 不要改变用户的核心需求和意图

请直接返回优化后的提示词，不要包含任何其他说明文字。`

	refinementPrompt := fmt.Sprintf(`原始用户提示词：
%s

请根据上述要求优化这个提示词。`, userPrompt)

	req := &ai.ChatRequest{
		Messages: []models.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: refinementPrompt,
			},
		},
		Model:  "deepseek-chat",
		Stream: false,
	}

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		fmt.Printf("[Prompt Optimization] AI service error: %v, using fallback optimization\n", err)
		return s.fallbackRefinePrompt(userPrompt), nil
	}

	var refinedPrompt strings.Builder
	for chunk := range ch {
		refinedPrompt.WriteString(chunk.Content)
	}

	result := refinedPrompt.String()
	if result == "" {
		fmt.Printf("[Prompt Optimization] AI returned empty, using fallback optimization\n")
		return s.fallbackRefinePrompt(userPrompt), nil
	}

	fmt.Printf("[Prompt Optimization] Refined prompt: %s\n", result)
	return result, nil
}

func (s *service) fallbackRefinePrompt(userPrompt string) string {
	if len(userPrompt) < 20 {
		return userPrompt + "\n\n请提供详细的回答，包括背景信息和具体例子。"
	}
	return userPrompt
}

func (s *service) EnhanceWithContext(ctx context.Context, userPrompt string, messages []models.Message, config ai.PromptEngineeringConfig) (string, error) {
	fmt.Printf("[Prompt Optimization] Starting EnhanceWithContext\n")
	
	if !config.EnableContextEnhancement || len(messages) == 0 {
		fmt.Printf("[Prompt Optimization] Context enhancement disabled or no messages, skipping\n")
		return userPrompt, nil
	}

	const maxHistoryMessages = 5
	var historyBuilder strings.Builder

	start := len(messages) - maxHistoryMessages
	if start < 0 {
		start = 0
	}

	for i := start; i < len(messages); i++ {
		msg := messages[i]
		if msg.Role != "system" && !(i == len(messages)-1 && msg.Role == "user") {
			historyBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
	}

	historyContext := historyBuilder.String()
	if historyContext == "" {
		fmt.Printf("[Prompt Optimization] No relevant history found, skipping\n")
		return userPrompt, nil
	}

	fmt.Printf("[Prompt Optimization] Found history context: %s\n", historyContext)

	systemPrompt := `你是一位专业的提示词工程师。请根据对话历史增强用户的当前提示词。

增强要求：
1. 从对话历史中提取与当前问题相关的背景信息
2. 避免重复对话历史中已经明确的信息
3. 只补充与当前问题相关的、有助于AI理解上下文的信息
4. 保持用户原始意图不变
5. 返回增强后的完整提示词，不要包含其他说明文字

请直接返回增强后的提示词。`

	enhancementPrompt := fmt.Sprintf(`对话历史：
%s

当前用户提示词：
%s

请根据对话历史增强当前提示词。`, historyContext, userPrompt)

	req := &ai.ChatRequest{
		Messages: []models.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: enhancementPrompt,
			},
		},
		Model:  "deepseek-chat",
		Stream: false,
	}

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		fmt.Printf("[Prompt Optimization] AI service error for context enhancement: %v, using fallback\n", err)
		return s.fallbackEnhanceWithContext(userPrompt, historyContext), nil
	}

	var enhancedPrompt strings.Builder
	for chunk := range ch {
		enhancedPrompt.WriteString(chunk.Content)
	}

	result := enhancedPrompt.String()
	if result == "" {
		fmt.Printf("[Prompt Optimization] AI returned empty for context enhancement, using fallback\n")
		return s.fallbackEnhanceWithContext(userPrompt, historyContext), nil
	}

	fmt.Printf("[Prompt Optimization] Enhanced with context: %s\n", result)
	return result, nil
}

func (s *service) fallbackEnhanceWithContext(userPrompt string, historyContext string) string {
	if historyContext != "" {
		return "基于之前的对话：\n" + historyContext + "\n\n" + userPrompt
	}
	return userPrompt
}

func (s *service) UnderstandIntent(ctx context.Context, userPrompt string, config ai.PromptEngineeringConfig) (string, error) {
	fmt.Printf("[Prompt Optimization] Starting UnderstandIntent for: %s\n", userPrompt)
	
	if !config.EnableIntentUnderstanding {
		fmt.Printf("[Prompt Optimization] Intent understanding disabled, skipping\n")
		return userPrompt, nil
	}

	systemPrompt := `你是一位专业的意图分析师和提示词工程师。请分析用户的提示词，理解其意图，并根据意图添加相关的指令和约束。

分析要求：
1. 识别任务类型 - 判断这是什么类型的任务（问答、创作、编程、分析等）
2. 理解详细程度要求 - 判断用户需要多详细的回答
3. 识别风格偏好 - 判断用户偏好的回答风格（正式、随意、专业等）
4. 建议输出格式 - 根据意图建议合适的输出格式
5. 添加相关约束 - 根据意图添加必要的约束条件

请返回增强后的完整提示词，将分析出的指令和约束自然地融入到提示词中，不要改变用户的原始意图。

请直接返回增强后的提示词，不要包含任何其他说明文字。`

	intentPrompt := fmt.Sprintf(`用户提示词：
%s

请分析意图并增强这个提示词。`, userPrompt)

	req := &ai.ChatRequest{
		Messages: []models.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: intentPrompt,
			},
		},
		Model:  "deepseek-chat",
		Stream: false,
	}

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		fmt.Printf("[Prompt Optimization] AI service error for intent understanding: %v, using fallback\n", err)
		return s.fallbackUnderstandIntent(userPrompt), nil
	}

	var intentPromptResult strings.Builder
	for chunk := range ch {
		intentPromptResult.WriteString(chunk.Content)
	}

	result := intentPromptResult.String()
	if result == "" {
		fmt.Printf("[Prompt Optimization] AI returned empty for intent understanding, using fallback\n")
		return s.fallbackUnderstandIntent(userPrompt), nil
	}

	fmt.Printf("[Prompt Optimization] Intent understood: %s\n", result)
	return result, nil
}

func (s *service) fallbackUnderstandIntent(userPrompt string) string {
	lowerPrompt := strings.ToLower(userPrompt)
	
	switch {
	case strings.Contains(lowerPrompt, "代码") || strings.Contains(lowerPrompt, "编程") || strings.Contains(lowerPrompt, "写"):
		return userPrompt + "\n\n请提供完整、可运行的代码，并添加详细的注释说明。"
	case strings.Contains(lowerPrompt, "解释") || strings.Contains(lowerPrompt, "说明"):
		return userPrompt + "\n\n请用通俗易懂的语言详细解释，并举例子说明。"
	case strings.Contains(lowerPrompt, "列表") || strings.Contains(lowerPrompt, "列举"):
		return userPrompt + "\n\n请用列表形式清晰地列出，每个点都要有简短说明。"
	default:
		return userPrompt + "\n\n请提供清晰、有条理的回答。"
	}
}
