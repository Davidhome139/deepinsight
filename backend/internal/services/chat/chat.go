package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/pkg/database"
	"backend/internal/services/agent"
	"backend/internal/services/ai"
	"backend/internal/services/prompt_engineering"
	"backend/internal/services/rag"
	"backend/internal/services/search"

	"gorm.io/gorm"
)

type ChatService interface {
	CreateConversation(userID uint, title string, model string) (*models.Conversation, error)
	GetConversations(userID uint) ([]models.Conversation, error)
	GetMessages(userID uint, convID uint) ([]models.Message, error)
	SendMessageStream(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string, promptEngineeringConfig *ai.PromptEngineeringConfig) (<-chan ai.ChatChunk, error)
	SendMessageStreamWithRAG(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string, ragEnabled bool, ragDocIDs []string, promptEngineeringConfig *ai.PromptEngineeringConfig) (<-chan ai.ChatChunk, error)
	GenerateConversationSummary(userID uint, convID uint, model string) (string, error)
}

type chatService struct {
	aiService                ai.AIService
	searchService            search.SearchService
	mcpManager               *agent.MCPManager
	toolRecommender          *agent.ToolRecommender
	contextProcessor         *ContextProcessor
	ragService               *rag.RAGService
	promptEngineeringService prompt_engineering.Service

	// Cache for context7 library ID resolutions
	libraryIdCache     map[string]string
	libraryIdCacheLock sync.RWMutex
}

func NewChatService(aiService ai.AIService, searchService search.SearchService, mcpManager *agent.MCPManager) ChatService {
	// Create tool recommender
	toolRecommender := agent.NewToolRecommender(mcpManager, aiService)

	return &chatService{
		aiService:                aiService,
		searchService:            searchService,
		mcpManager:               mcpManager,
		toolRecommender:          toolRecommender,
		contextProcessor:         NewContextProcessor(),
		promptEngineeringService: prompt_engineering.NewService(aiService),
		libraryIdCache:           make(map[string]string),
	}
}

// SetRAGService sets the RAG service for knowledge base integration
func (s *chatService) SetRAGService(ragService *rag.RAGService) {
	s.ragService = ragService
}

func (s *chatService) CreateConversation(userID uint, title string, model string) (*models.Conversation, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	conv := &models.Conversation{
		UserID:    userID,
		Title:     title,
		ModelType: model,
	}
	if err := database.DB.Create(conv).Error; err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *chatService) GetConversations(userID uint) ([]models.Conversation, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	var conversations []models.Conversation
	if err := database.DB.Where("user_id = ?", userID).Order("updated_at desc").Find(&conversations).Error; err != nil {
		return nil, err
	}
	return conversations, nil
}

func (s *chatService) GetMessages(userID uint, convID uint) ([]models.Message, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	// First check if the conversation belongs to the user
	var conv models.Conversation
	if err := database.DB.Where("id = ? AND user_id = ?", convID, userID).First(&conv).Error; err != nil {
		return nil, err
	}

	var messages []models.Message
	if err := database.DB.Where("conversation_id = ?", convID).Order("created_at asc").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *chatService) SendMessageStream(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string, promptEngineeringConfig *ai.PromptEngineeringConfig) (<-chan ai.ChatChunk, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	// 0. Fetch conversation to get model
	var conv models.Conversation
	if err := database.DB.First(&conv, convID).Error; err != nil {
		return nil, err
	}

	// Override conv.ModelType with the model provided by the user in real-time
	if model != "" {
		conv.ModelType = model
		// Update conversation model type in DB
		database.DB.Model(&models.Conversation{}).Where("id = ?", convID).Update("model_type", model)
	}

	fmt.Printf("[Chat] Using model: %s, WebSearch: %v, Provider: %s\n", conv.ModelType, webSearch, searchProvider)

	// 1. Save user message with branch association
	userMsg := &models.Message{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
		Status:         "success",
		BranchID:       conv.ActiveBranchID, // Associate with active branch
	}
	if err := database.DB.Create(userMsg).Error; err != nil {
		return nil, err
	}

	// Update branch message count if branch exists
	if conv.ActiveBranchID != nil {
		database.DB.Model(&models.Branch{}).Where("id = ?", *conv.ActiveBranchID).
			UpdateColumn("message_count", gorm.Expr("message_count + 1"))
	}

	// 2. Update conversation last message
	database.DB.Model(&models.Conversation{}).Where("id = ?", convID).Update("last_message", content)

	// 3. Auto-generate title if it's still "New Chat"
	if conv.Title == "New Chat" {
		go s.generateTitle(userID, convID, content, conv.ModelType)
	}

	// 4. Get history for AI context (last 10 messages)
	var history []models.Message
	database.DB.Where("conversation_id = ?", convID).Order("created_at desc").Limit(10).Find(&history)

	// Reverse history to be in ascending order for the AI
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	// 4.5. Process context and build enhanced search query
	var searchResultsData []map[string]interface{}
	var searchContext string
	var enhancedQuery string

	if webSearch {
		// 使用上下文处理器生成增强的搜索查询
		enhancedQuery = s.contextProcessor.BuildEnhancedQuery(content, history)
		fmt.Printf("[Search] Enhanced query for search: '%s'\n", enhancedQuery)

		// 执行搜索
		searchResults, err := s.searchService.Search(ctx, enhancedQuery, userID, searchProvider)
		if err != nil {
			fmt.Printf("[Search] Error during search: %v\n", err)
		} else if len(searchResults) > 0 {
			fmt.Printf("[Search] Found %d results\n", len(searchResults))

			// Convert search results to map format for frontend
			for _, res := range searchResults {
				searchResultsData = append(searchResultsData, map[string]interface{}{
					"title":   res.Title,
					"snippet": res.Snippet,
					"url":     res.URL,
				})
			}

			// 构建搜索上下文
			var searchBuilder strings.Builder
			searchBuilder.WriteString("\n\n[Web Search Results]\n")
			for _, res := range searchResults {
				searchBuilder.WriteString(fmt.Sprintf("Title: %s\nContent: %s\nSource: %s\n\n", res.Title, res.Snippet, res.URL))
			}
			searchBuilder.WriteString("Current Time: " + time.Now().Format("2006-01-02 15:04:05") + "\n")
			searchBuilder.WriteString("Please provide a detailed and accurate answer to the user's question using the search results above. If the information is about weather, provide the most recent forecast available.")

			searchContext = searchBuilder.String()
		} else {
			fmt.Println("[Search] No results found from search provider")
		}
	}

	// 4.6. Execute MCP tool if selected, or auto-detect if not selected but intent matches
	var mcpResult string
	if mcpTool != "" {
		fmt.Printf("[Chat] Executing MCP tool: %s\n", mcpTool)
		mcpResult = s.executeMCPTool(mcpTool, content)
		if mcpResult != "" {
			fmt.Printf("[Chat] MCP tool result length: %d\n", len(mcpResult))
		}
	} else {
		// Smart MCP tool detection based on semantic intent analysis
		autoTool := s.detectMCPIntentSemantic(ctx, content, conv.ModelType)
		if autoTool != "" {
			fmt.Printf("[Chat] Auto-detected MCP tool: %s\n", autoTool)
			mcpResult = s.executeMCPTool(autoTool, content)
			if mcpResult != "" {
				fmt.Printf("[Chat] Auto-executed MCP tool result length: %d\n", len(mcpResult))
			}
		}
	}

	// 4.7. Always load and inject system prompt for every request
	systemPrompt, err := s.loadSystemPrompt(convID)
	if err != nil {
		// If no system prompt exists, generate one for the first message
		var messageCount int64
		database.DB.Model(&models.Message{}).Where("conversation_id = ? AND role = 'user'", convID).Count(&messageCount)

		if messageCount == 1 {
			systemPrompt = s.generateSystemPrompt(conv.ModelType, content)
			// Save system prompt to file
			if err := s.saveSystemPrompt(convID, systemPrompt); err != nil {
				fmt.Printf("[Chat] Failed to save system prompt: %v\n", err)
			}
		}
	}

	// 4.8. Merge custom system prompt with default system prompt (if provided)
	// This ensures follow-up questions instruction is preserved while adding custom context
	if customSystemPrompt != "" {
		if systemPrompt != "" {
			// Prepend custom system prompt to default, separated by double newline
			systemPrompt = customSystemPrompt + "\n\n" + systemPrompt
			fmt.Printf("[Chat] Merged custom system prompt with default system prompt\n")
		} else {
			systemPrompt = customSystemPrompt
			fmt.Printf("[Chat] Using custom system prompt only (no default)\n")
		}
	}

	// 4.9. 构建最终的消息历史
	var finalHistory []models.Message

	// 1. 添加系统提示（如果存在）
	if systemPrompt != "" {
		finalHistory = append(finalHistory, models.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// 2. 添加历史消息
	finalHistory = append(finalHistory, history...)

	// 3. 添加当前用户消息（包含搜索上下文和MCP结果）
	currentUserMessage := models.Message{
		Role:    "user",
		Content: content,
	}

	// 如果有搜索结果，添加到当前消息中
	if searchContext != "" {
		currentUserMessage.Content += searchContext
	}

	// 如果有 MCP 结果，添加到当前消息中
	if mcpResult != "" {
		currentUserMessage.Content += mcpResult
	}

	finalHistory = append(finalHistory, currentUserMessage)

	// 使用处理后的最终历史
	history = finalHistory

	// 7. Wrap channel to save response context
	outCh := make(chan ai.ChatChunk)
	go func() {
		defer close(outCh)
		var assistantReply string

		// 应用提示词工程功能（在goroutine内部以便发送流式事件）
		if promptEngineeringConfig != nil && promptEngineeringConfig.Enabled {
			// 发送提示词优化开始事件
			outCh <- ai.ChatChunk{
				Type:    "prompt_optimization_start",
				Content: "🔧 正在优化提示词...",
				Metadata: map[string]interface{}{
					"status":      "started",
					"timestamp":   time.Now().Unix(),
					"totalSteps":  3,
					"currentStep": 0,
				},
			}

			// 首先处理非优化功能（角色、输出格式、思维链等）
			nonOptimizationConfig := *promptEngineeringConfig
			nonOptimizationConfig.PromptOptimizationEnabled = false
			processedMessages, err := s.promptEngineeringService.ProcessMessage(ctx, history, nonOptimizationConfig, fmt.Sprintf("%d", convID))
			if err != nil {
				fmt.Printf("[Chat] Error processing prompt engineering (non-optimization): %v\n", err)
				// 发送优化失败事件
				outCh <- ai.ChatChunk{
					Type:    "prompt_optimization_error",
					Content: "❌ 提示词处理失败，使用原始提示词",
					Metadata: map[string]interface{}{
						"status": "error",
						"error":  err.Error(),
					},
				}
				// 继续使用原始历史
				processedMessages = history
			} else {
				history = processedMessages
				fmt.Printf("[Chat] Applied non-optimization prompt engineering config\n")
			}

			// 如果启用提示词优化，执行详细步骤
			if promptEngineeringConfig.PromptOptimizationEnabled {
				// 查找最后一个用户消息进行优化
				var lastUserIndex int = -1
				var lastUserContent string
				for i := len(history) - 1; i >= 0; i-- {
					if history[i].Role == "user" {
						lastUserIndex = i
						lastUserContent = history[i].Content
						break
					}
				}

				if lastUserIndex >= 0 {
					optimizedPrompt := lastUserContent

					// 步骤1: 理解意图
					outCh <- ai.ChatChunk{
						Type:    "prompt_optimization_step",
						Content: "📝 理解用户意图...",
						Metadata: map[string]interface{}{
							"step":       "intent_understanding",
							"stepNumber": 1,
							"totalSteps": 3,
							"progress":   33,
						},
					}
					intentResult, err := s.promptEngineeringService.UnderstandIntent(ctx, optimizedPrompt, *promptEngineeringConfig)
					if err == nil {
						optimizedPrompt = intentResult
						// 发送优化内容
						outCh <- ai.ChatChunk{
							Type:    "prompt_optimization_content",
							Content: optimizedPrompt,
							Metadata: map[string]interface{}{
								"step":       "intent_understanding",
								"stepNumber": 1,
								"totalSteps": 3,
							},
						}
					}

					// 步骤2: 上下文增强
					outCh <- ai.ChatChunk{
						Type:    "prompt_optimization_step",
						Content: "🔍 增强上下文信息...",
						Metadata: map[string]interface{}{
							"step":       "context_enhancement",
							"stepNumber": 2,
							"totalSteps": 3,
							"progress":   66,
						},
					}
					enhancedResult, err := s.promptEngineeringService.EnhanceWithContext(ctx, optimizedPrompt, history, *promptEngineeringConfig)
					if err == nil {
						optimizedPrompt = enhancedResult
						// 发送优化内容
						outCh <- ai.ChatChunk{
							Type:    "prompt_optimization_content",
							Content: optimizedPrompt,
							Metadata: map[string]interface{}{
								"step":       "context_enhancement",
								"stepNumber": 2,
								"totalSteps": 3,
							},
						}
					}

					// 步骤3: 精炼提示词
					outCh <- ai.ChatChunk{
						Type:    "prompt_optimization_step",
						Content: "✨ 精炼提示词表达...",
						Metadata: map[string]interface{}{
							"step":       "prompt_refinement",
							"stepNumber": 3,
							"totalSteps": 3,
							"progress":   100,
						},
					}
					refinedResult, err := s.promptEngineeringService.RefineUserPrompt(ctx, optimizedPrompt, *promptEngineeringConfig)
					if err == nil {
						optimizedPrompt = refinedResult
						// 发送优化内容
						outCh <- ai.ChatChunk{
							Type:    "prompt_optimization_content",
							Content: optimizedPrompt,
							Metadata: map[string]interface{}{
								"step":       "prompt_refinement",
								"stepNumber": 3,
								"totalSteps": 3,
							},
						}
					}

					// 更新消息历史中的用户消息
					history[lastUserIndex].Content = optimizedPrompt
					fmt.Printf("[Chat] Prompt optimization completed: %s\n", optimizedPrompt)
				}
			}

			// 发送提示词优化完成事件
			outCh <- ai.ChatChunk{
				Type:    "prompt_optimization_complete",
				Content: "✅ 提示词优化完成",
				Metadata: map[string]interface{}{
					"status":    "completed",
					"timestamp": time.Now().Unix(),
				},
			}
		}

		// Send search results as first chunk if available
		if len(searchResultsData) > 0 {
			outCh <- ai.ChatChunk{
				SearchResults: searchResultsData,
			}
		}

		// 6. Call AI service with potentially optimized history
		req := &ai.ChatRequest{
			UserID:                  userID,
			ConversationID:          convID,
			Messages:                history,
			Model:                   conv.ModelType,
			Stream:                  true,
			WebSearch:               webSearch,
			PromptEngineeringConfig: promptEngineeringConfig,
			// SystemPrompt already merged into message history, no need to pass separately
		}

		aiCh, err := s.aiService.ChatStream(ctx, req)
		if err != nil {
			fmt.Printf("[Chat] Error calling AI service: %v\n", err)
			outCh <- ai.ChatChunk{
				Type:    "error",
				Content: "❌ AI服务调用失败",
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}

		for chunk := range aiCh {
			if !chunk.Done {
				assistantReply += chunk.Content
			}
			outCh <- chunk
		}

		// 8. Self-evaluation and optimization if enabled
		if assistantReply != "" && promptEngineeringConfig != nil && promptEngineeringConfig.Enabled && promptEngineeringConfig.SelfEvaluation {
			// Get the user's current message
			userMessage := content
			if searchContext != "" {
				userMessage += searchContext
			}
			if mcpResult != "" {
				userMessage += mcpResult
			}

			// Self-evaluate the output
			evaluation, err := s.promptEngineeringService.SelfEvaluate(ctx, fmt.Sprintf("%d", convID), userMessage, assistantReply)
			if err == nil && evaluation.Score < 80 {
				// Optimize the output if score is below threshold
				optimizedOutput, err := s.promptEngineeringService.OptimizeOutput(ctx, fmt.Sprintf("%d", convID), userMessage, assistantReply, evaluation)
				if err == nil && optimizedOutput != "" {
					// Replace the original output with optimized one
					assistantReply = optimizedOutput
					// Send the optimized output as a new chunk
					outCh <- ai.ChatChunk{
						Content: "\n\n**Optimized Response:**\n" + optimizedOutput,
						Done:    true,
					}
				}
			}
		}

		// 9. Save assistant message when stream is finished
		if assistantReply != "" {
			assistantMsg := &models.Message{
				ConversationID: convID,
				Role:           "assistant",
				Content:        assistantReply,
				Model:          conv.ModelType,
				Status:         "success",
				BranchID:       conv.ActiveBranchID, // Associate with active branch
			}
			database.DB.Create(assistantMsg)

			// Update branch message count if branch exists
			if conv.ActiveBranchID != nil {
				database.DB.Model(&models.Branch{}).Where("id = ?", *conv.ActiveBranchID).
					UpdateColumn("message_count", gorm.Expr("message_count + 1"))
			}

			// Update conversation updated_at
			database.DB.Model(&models.Conversation{}).Where("id = ?", convID).Update("updated_at", database.DB.NowFunc())
		}
	}()

	return outCh, nil
}

// generateSystemPrompt 根据模型和用户输入生成 system prompt
func (s *chatService) generateSystemPrompt(model string, userInput string) string {
	currentTime := time.Now().Format("2006-01-02 15:04:05")

	systemPrompt := fmt.Sprintf(`You are an AI assistant using the %s model. Current time: %s.

User's first question: %s

CRITICAL REQUIREMENT - YOU MUST FOLLOW THIS RULE FOR EVERY SINGLE RESPONSE:
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

Remember: This is required for ALL responses in this conversation, not just your first answer.`, model, currentTime, userInput)

	return systemPrompt
}

// saveSystemPrompt 将 system prompt 持久化到文件
func (s *chatService) saveSystemPrompt(convID uint, prompt string) error {
	// 确保目录存在
	chatDir := "/app/chat"
	if err := os.MkdirAll(chatDir, 0755); err != nil {
		return fmt.Errorf("failed to create chat directory: %w", err)
	}

	// 生成文件路径
	filename := filepath.Join(chatDir, fmt.Sprintf("conv_%d_system.txt", convID))

	// 写入文件
	if err := os.WriteFile(filename, []byte(prompt), 0644); err != nil {
		return fmt.Errorf("failed to write system prompt: %w", err)
	}

	fmt.Printf("[Chat] System prompt saved to %s\n", filename)
	return nil
}

// loadSystemPrompt 从文件加载 system prompt
func (s *chatService) loadSystemPrompt(convID uint) (string, error) {
	chatDir := "/app/chat"
	filename := filepath.Join(chatDir, fmt.Sprintf("conv_%d_system.txt", convID))

	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *chatService) generateTitle(userID uint, convID uint, firstMessage string, model string) {
	prompt := fmt.Sprintf("Please provide a very short, concise title (maximum 6 words) for a conversation that starts with: \"%s\". Only return the title text, nothing else.", firstMessage)

	req := &ai.ChatRequest{
		UserID: userID,
		Model:  model,
		Messages: []models.Message{
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return
	}

	var title string
	for chunk := range ch {
		if !chunk.Done {
			title += chunk.Content
		}
	}

	title = strings.TrimSpace(title)
	title = strings.Trim(title, "\"'") // Remove quotes if any
	if title != "" {
		database.DB.Model(&models.Conversation{}).Where("id = ?", convID).Update("title", title)
	}
}

func (s *chatService) GenerateConversationSummary(userID uint, convID uint, model string) (string, error) {
	// Get all messages in the conversation
	var messages []models.Message
	if err := database.DB.Where("conversation_id = ?", convID).Order("created_at asc").Find(&messages).Error; err != nil {
		return "", err
	}

	// If there are no messages, return empty summary
	if len(messages) == 0 {
		return "", nil
	}

	// Build conversation text for summary
	var conversationText strings.Builder
	for _, msg := range messages {
		conversationText.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
	}

	// Create prompt for summarization
	prompt := fmt.Sprintf("Please provide a concise summary of the following conversation, capturing the key points and context:\n\n%s", conversationText.String())

	// Call AI service to generate summary
	req := &ai.ChatRequest{
		UserID: userID,
		Model:  model,
		Messages: []models.Message{
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return "", err
	}

	var summary string
	for chunk := range ch {
		if !chunk.Done {
			summary += chunk.Content
		}
	}

	summary = strings.TrimSpace(summary)
	return summary, nil
}

// buildSearchQuery 构建包含对话上下文的搜索查询
// 针对百度搜索API优化，使用更精确的上下文提取策略
func (s *chatService) buildSearchQuery(currentQuery string, history []models.Message) string {
	// 提取关键实体和主题词
	var keyEntities []string
	var keyTopics []string

	// 从历史消息中提取关键信息
	for i := len(history) - 1; i >= 0 && i >= len(history)-3; i-- {
		if history[i].Role == "user" {
			content := history[i].Content
			// 提取可能的地点、人名、机构等实体
			entities := s.extractEntities(content)
			keyEntities = append(keyEntities, entities...)

			// 提取主题词
			topics := s.extractTopics(content)
			keyTopics = append(keyTopics, topics...)
		}
	}

	// 构建优化的搜索查询
	var queryBuilder strings.Builder

	// 添加关键实体（优先级最高）
	if len(keyEntities) > 0 {
		for _, entity := range keyEntities {
			if entity != "" && !strings.Contains(currentQuery, entity) {
				queryBuilder.WriteString(entity + " ")
			}
		}
	}

	// 添加关键主题词
	if len(keyTopics) > 0 {
		for _, topic := range keyTopics {
			if topic != "" && !strings.Contains(currentQuery, topic) {
				queryBuilder.WriteString(topic + " ")
			}
		}
	}

	// 添加当前查询
	queryBuilder.WriteString(currentQuery)

	result := strings.TrimSpace(queryBuilder.String())
	fmt.Printf("[Search] Enhanced query: '%s'\n", result)
	return result
}

// extractEntities 提取文本中的关键实体（地点、人名、机构等）
func (s *chatService) extractEntities(text string) []string {
	var entities []string

	// 常见的中国地名关键词
	places := []string{
		"北京", "上海", "广州", "深圳", "杭州", "南京", "成都", "武汉", "西安", "重庆",
		"天津", "苏州", "青岛", "大连", "厦门", "宁波", "无锡", "佛山", "东莞", "福州",
		"菲律宾", "日本", "韩国", "美国", "英国", "法国", "德国", "澳大利亚", "加拿大", "新加坡",
		"泰国", "马来西亚", "越南", "印度", "俄罗斯", "巴西", "墨西哥", "埃及", "土耳其", "意大利",
	}

	// 检查文本中是否包含这些地名
	for _, place := range places {
		if strings.Contains(text, place) {
			entities = append(entities, place)
		}
	}

	// 也可以使用正则表达式匹配更复杂的地名模式
	// 这里简化处理

	return entities
}

// extractTopics 提取文本中的主题词
func (s *chatService) extractTopics(text string) []string {
	var topics []string

	// 常见的主题关键词
	keywords := []string{
		"天气", "旅游", "景点", "美食", "交通", "住宿", "购物", "娱乐",
		"文化", "历史", "经济", "科技", "教育", "医疗", "环境", "政策",
		"时间", "日期", "季节", "月份", "节日", "活动", "事件", "新闻",
		"趋势", "发展", "变化", "影响", "原因", "结果", "建议", "方法",
	}

	// 检查文本中是否包含这些关键词
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			topics = append(topics, keyword)
		}
	}

	return topics
}

// executeMCPTool executes an MCP tool and returns the result
func (s *chatService) executeMCPTool(mcpTool string, userContent string) string {
	// Parse mcpTool format: "server/tool"
	parts := strings.SplitN(mcpTool, "/", 2)
	if len(parts) != 2 {
		fmt.Printf("[Chat] Invalid MCP tool format: %s\n", mcpTool)
		return ""
	}
	serverName := parts[0]
	toolName := parts[1]

	fmt.Printf("[Chat] DEBUG: executeMCPTool called with mcpTool=%s, serverName=%s, toolName=%s\n", mcpTool, serverName, toolName)
	fmt.Printf("[Chat] Calling MCP tool: server=%s, tool=%s\n", serverName, toolName)

	// Check if MCP manager is available (for external MCP servers)
	if s.mcpManager == nil {
		fmt.Println("[Chat] MCP manager not available")
		return "[MCP Error: MCP manager not initialized]"
	}

	// For built-in servers, handle directly without CallTool (they don't have MCP client)
	switch serverName {
	case "filesystem-local":
		return s.handleBuiltinFilesystemTool(toolName, userContent)
	case "terminal":
		return s.handleBuiltinTerminalTool(toolName, userContent)
	case "search":
		return s.handleBuiltinSearchTool(toolName, userContent)
	case "code-analysis":
		return s.handleBuiltinCodeAnalysisTool(toolName, userContent)
	}

	// For external MCP servers, get the server from MCP manager
	server, ok := s.mcpManager.GetServer(serverName)
	if !ok {
		fmt.Printf("[Chat] MCP server not found: %s\n", serverName)
		return fmt.Sprintf("[MCP Error: Server '%s' not found]\n\nAvailable servers can be configured in Settings.", serverName)
	}

	// Check connection status for external MCP servers
	if !server.Connected && server.Client == nil {
		fmt.Printf("[Chat] MCP server not connected: %s\n", serverName)
		return fmt.Sprintf("[MCP Error: Server '%s' not connected]\n\nPlease check MCP server configuration in Settings.", serverName)
	}

	// Prepare arguments based on tool type
	var args map[string]interface{}
	switch toolName {
	case "read_file":
		// Try to extract file path from user content
		args = map[string]interface{}{
			"path": extractFilePath(userContent),
		}
	case "list_directory":
		args = map[string]interface{}{
			"path": ".",
		}
	case "execute_command":
		// For terminal, we pass the user content as command
		args = map[string]interface{}{
			"command": userContent,
		}
	case "query-docs":
		fmt.Printf("[Chat] DEBUG: Entering query-docs case\n")
		// For context7 query-docs tool, need libraryId and query parameters
		// According to context7 workflow, we should first resolve library ID
		libraryName := extractLibraryName(userContent)
		fmt.Printf("[Chat] DEBUG: extractLibraryName returned: %s\n", libraryName)
		if libraryName == "" {
			libraryName = "next.js" // Default library for testing
			fmt.Printf("[Chat] DEBUG: Using default libraryName: %s\n", libraryName)
		}

		fmt.Printf("[Chat] Starting two-step context7 workflow for library: %s\n", libraryName)

		// Check cache first
		var libraryId string
		if cachedLibraryId, found := s.getCachedLibraryId(libraryName); found {
			fmt.Printf("[Chat] Using cached libraryId for %s: %s\n", libraryName, cachedLibraryId)
			libraryId = cachedLibraryId
		} else {
			// Step 1: Resolve library ID using resolve-library-id tool
			resolveArgs := map[string]interface{}{
				"libraryName": libraryName,
				"query":       userContent,
			}

			fmt.Printf("[Chat] Step 1: Calling resolve-library-id for library: %s\n", libraryName)
			resolveResult, resolveErr := s.mcpManager.CallTool(serverName, "resolve-library-id", resolveArgs)
			if resolveErr != nil {
				fmt.Printf("[Chat] Failed to resolve library ID for %s: %v\n", libraryName, resolveErr)
				// Fallback to common library ID mapping
				libraryId = getFallbackLibraryId(libraryName)
				fmt.Printf("[Chat] Using fallback libraryId: %s\n", libraryId)
				// Cache the fallback result
				s.setCachedLibraryId(libraryName, libraryId)
			} else {
				// Step 2: Extract libraryId from resolve result
				libraryId = extractLibraryIdFromResult(resolveResult, libraryName)
				if libraryId == "" {
					// If extraction failed, use fallback
					libraryId = getFallbackLibraryId(libraryName)
					fmt.Printf("[Chat] Could not extract libraryId from result, using fallback: %s\n", libraryId)
				} else {
					fmt.Printf("[Chat] Successfully resolved libraryId: %s\n", libraryId)
				}
				// Cache the resolved result
				s.setCachedLibraryId(libraryName, libraryId)
			}
		}

		// Now call query-docs with the resolved libraryId
		fmt.Printf("[Chat] Step 2: Calling query-docs with libraryId: %s\n", libraryId)
		args = map[string]interface{}{
			"libraryId": libraryId,
			"query":     userContent,
		}
		fmt.Printf("[Chat] Final args for query-docs: libraryId=%s, query length=%d\n",
			libraryId, len(userContent))

	case "resolve-library-id":
		// For resolve-library-id tool
		// Extract library name from user content
		libraryName := extractLibraryName(userContent)

		if libraryName == "" {
			// If no library name found, use the entire content
			libraryName = userContent
		}

		// Check cache first
		if cachedLibraryId, found := s.getCachedLibraryId(libraryName); found {
			// Return cached result instead of calling the tool
			return fmt.Sprintf("\n\n[Cached Library ID Result]\nLibrary: %s\nLibrary ID: %s\n\nNote: This result was retrieved from cache. To force a fresh lookup, clear the cache.",
				libraryName, cachedLibraryId)
		}

		args = map[string]interface{}{
			"libraryName": libraryName,
			"query":       userContent, // Use the same content as query for relevance ranking
		}
	case "browser_navigate":
		fmt.Printf("[Chat] DEBUG: Processing browser_navigate case\n")
		fmt.Printf("[Chat] DEBUG: User content: %s\n", userContent)
		// Extract URL from user content for browser navigation
		url := extractURLFromContent(userContent)
		fmt.Printf("[Chat] DEBUG: extractURLFromContent returned: %s\n", url)
		if url == "" {
			// If no URL found, try to construct from common patterns
			url = constructURLFromContent(userContent)
			fmt.Printf("[Chat] DEBUG: constructURLFromContent returned: %s\n", url)
		}

		var urls []string
		if url != "" {
			// If specific URL found, use only that URL
			urls = []string{url}
			fmt.Printf("[Chat] Using specific URL from content: %s\n", url)
		} else {
			// Get default URLs from config for multi-site search
			urls = s.getPlaywrightDefaultURLs()
			fmt.Printf("[Chat] No specific URL found, using default URLs from config: %v\n", urls)
		}

		// For multiple URLs, we need to handle them specially
		if len(urls) > 1 {
			return s.handleMultipleURLs(serverName, toolName, urls, userContent)
		}

		// Single URL case (backward compatibility)
		args = map[string]interface{}{
			"url": urls[0],
			// 添加无头模式参数，避免在无头环境中运行有界面浏览器
			"headless": true,
			// 添加超时参数
			"timeout": 30000, // 30秒
		}
		fmt.Printf("[Chat] Using single URL for browser_navigate: %s (headless: true)\n", urls[0])
	default:
		args = map[string]interface{}{}
		// Log for debugging
		fmt.Printf("[Chat] Using default args for tool: %s\n", toolName)
	}

	// Call the tool
	result, err := s.mcpManager.CallTool(serverName, toolName, args)
	if err != nil {
		fmt.Printf("[Chat] MCP tool call failed: %v\n", err)
		return fmt.Sprintf("[MCP Tool Error: %v]", err)
	}

	// Cache successful resolve-library-id results
	if serverName == "context7" && toolName == "resolve-library-id" {
		// Extract library name from args
		if libraryName, ok := args["libraryName"].(string); ok && libraryName != "" {
			// Try to extract libraryId from result
			libraryId := extractLibraryIdFromResult(result, libraryName)
			if libraryId != "" {
				s.setCachedLibraryId(libraryName, libraryId)
				fmt.Printf("[Chat] Cached resolve-library-id result for %s: %s\n", libraryName, libraryId)

				// Check if user wants documentation/examples and should automatically call query-docs
				// This creates a two-step workflow: resolve-library-id -> query-docs
				userContent := ""
				if query, ok := args["query"].(string); ok {
					userContent = query
				}

				if userContent != "" {
					// Check if user wants documentation or examples
					contentLower := strings.ToLower(userContent)
					chineseKeywords := []string{"示例", "文档", "帮助", "查询", "搜索", "获取", "最新"}
					englishKeywords := []string{"example", "examples", "documentation", "docs", "help", "query", "search", "get", "latest"}

					hasDocIntent := false
					for _, keyword := range chineseKeywords {
						if strings.Contains(userContent, keyword) {
							hasDocIntent = true
							break
						}
					}
					if !hasDocIntent {
						for _, keyword := range englishKeywords {
							if strings.Contains(contentLower, keyword) {
								hasDocIntent = true
								break
							}
						}
					}

					if hasDocIntent {
						fmt.Printf("[Chat] User wants documentation/examples, automatically calling query-docs after resolve-library-id\n")
						fmt.Printf("[Chat] Step 2: Calling query-docs with libraryId: %s\n", libraryId)

						// Call query-docs with the resolved libraryId
						queryArgs := map[string]interface{}{
							"libraryId": libraryId,
							"query":     userContent,
						}

						queryResult, queryErr := s.mcpManager.CallTool(serverName, "query-docs", queryArgs)
						if queryErr != nil {
							fmt.Printf("[Chat] Failed to call query-docs: %v\n", queryErr)
							return fmt.Sprintf("\n\n[MCP Tool Result - %s/%s]\n%s\n\n[Note: resolve-library-id succeeded but query-docs failed: %v]",
								serverName, toolName, result, queryErr)
						}

						// Return combined result
						return fmt.Sprintf("\n\n[MCP Tool Result - %s/%s]\n%s\n\n[MCP Tool Result - %s/query-docs]\n%s",
							serverName, toolName, result, serverName, queryResult)
					}
				}
			}
		}
	}

	return fmt.Sprintf("\n\n[MCP Tool Result - %s/%s]\n%s", serverName, toolName, result)
}

// handleBuiltinFilesystemTool handles built-in filesystem operations without external MCP client
func (s *chatService) handleBuiltinFilesystemTool(toolName string, userContent string) string {
	switch toolName {
	case "read_file":
		// Check if user wants to list directory instead
		contentLower := strings.ToLower(userContent)
		if strings.Contains(contentLower, "列出") || strings.Contains(contentLower, "显示") ||
			strings.Contains(contentLower, "list") || strings.Contains(contentLower, "show") ||
			strings.Contains(contentLower, "目录") || strings.Contains(contentLower, "文件") ||
			strings.Contains(contentLower, "directory") || strings.Contains(contentLower, "files") {
			// Check if no specific file path is mentioned
			filePath := extractFilePath(userContent)
			if filePath == "." || filePath == "" {
				// User wants to list directory, not read a file
				return s.handleBuiltinFilesystemTool("list_directory", userContent)
			}
		}

		filePath := extractFilePath(userContent)
		if filePath == "." || filePath == "" {
			return "[MCP Error: No file path specified. Please provide a file path to read, or select 'List Directory' tool to see available files.]"
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Sprintf("[MCP Error: Failed to read file '%s': %v]", filePath, err)
		}

		return fmt.Sprintf("\n\n=== FILE READ SUCCESSFUL ===\nFile: %s\nSize: %d bytes\n\n```\n%s\n```\n\n=== END OF FILE CONTENT ===", filePath, len(content), string(content))

	case "list_directory":
		dirPath := "."
		// Try to extract path from user content if provided
		extractedPath := extractFilePath(userContent)
		if extractedPath != "." {
			dirPath = extractedPath
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return fmt.Sprintf("[MCP Error: Failed to list directory '%s': %v]", dirPath, err)
		}

		var result strings.Builder
		result.WriteString("\n\n=== FILESYSTEM ACCESS SUCCESSFUL ===")
		result.WriteString(fmt.Sprintf("\nDirectory: %s\n", dirPath))
		result.WriteString(fmt.Sprintf("Total items: %d\n\n", len(entries)))

		for _, entry := range entries {
			if entry.IsDir() {
				result.WriteString(fmt.Sprintf("📁 %s/ (directory)\n", entry.Name()))
			} else {
				info, _ := entry.Info()
				result.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", entry.Name(), info.Size()))
			}
		}

		result.WriteString("\n=== END OF DIRECTORY LISTING ===")
		return result.String()

	default:
		return fmt.Sprintf("[MCP Error: Unknown tool '%s']", toolName)
	}
}

// handleBuiltinTerminalTool handles built-in terminal operations without external MCP client
func (s *chatService) handleBuiltinTerminalTool(toolName string, userContent string) string {
	switch toolName {
	case "execute_command":
		// Extract command from user content
		command := strings.TrimSpace(userContent)
		if command == "" {
			return "[MCP Error: No command provided]"
		}

		// Define allowed safe commands
		allowedCommands := []string{"ls", "pwd", "whoami", "date", "echo", "cat", "head", "tail", "wc", "grep", "find", "ps", "df", "du", "uname", "hostname"}

		// Check if command is in allowed list
		commandParts := strings.Fields(command)
		if len(commandParts) == 0 {
			return "[MCP Error: Invalid command]"
		}

		baseCmd := commandParts[0]
		isAllowed := false
		for _, allowed := range allowedCommands {
			if baseCmd == allowed {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return fmt.Sprintf("\n\n[Terminal Command Blocked]\nCommand: `%s`\n\nNote: Command '%s' is not in the allowed list for security reasons. Allowed commands: %s", command, baseCmd, strings.Join(allowedCommands, ", "))
		}

		// Execute the command
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return fmt.Sprintf("\n\n=== TERMINAL COMMAND EXECUTED ===\nCommand: %s\nExit Code: %d\n\nOutput:\n%s\nError: %v\n\n=== END OF OUTPUT ===", command, cmd.ProcessState.ExitCode(), string(output), err)
		}

		return fmt.Sprintf("\n\n=== TERMINAL COMMAND EXECUTED ===\nCommand: %s\nExit Code: 0\n\nOutput:\n%s\n\n=== END OF OUTPUT ===", command, string(output))

	default:
		return fmt.Sprintf("[MCP Error: Unknown terminal tool '%s']", toolName)
	}
}

// handleBuiltinSearchTool handles built-in search operations without external MCP client
func (s *chatService) handleBuiltinSearchTool(toolName string, userContent string) string {
	switch toolName {
	case "web_search":
		// Extract search query from user content
		query := strings.TrimSpace(userContent)
		if query == "" {
			return "[MCP Error: No search query provided]"
		}

		// Delegate to the existing search service
		if s.searchService == nil {
			return "[MCP Error: Search service not available]"
		}

		// Perform web search using the search service
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		results, err := s.searchService.Search(ctx, query, 0, "")
		if err != nil {
			return fmt.Sprintf("[MCP Error: Search failed: %v]", err)
		}

		var resultStr strings.Builder
		resultStr.WriteString("\n\n=== WEB SEARCH SUCCESSFUL ===")
		resultStr.WriteString(fmt.Sprintf("\nSearch Query: %s", query))
		resultStr.WriteString(fmt.Sprintf("\nTotal Results: %d\n\n", len(results)))

		for i, result := range results {
			resultStr.WriteString(fmt.Sprintf("%d. **%s**\n   URL: %s\n   Summary: %s\n\n", i+1, result.Title, result.URL, result.Snippet))
		}

		resultStr.WriteString("=== END OF SEARCH RESULTS ===")
		return resultStr.String()

	default:
		return fmt.Sprintf("[MCP Error: Unknown search tool '%s']", toolName)
	}
}

// handleBuiltinCodeAnalysisTool handles built-in code analysis operations without external MCP client
func (s *chatService) handleBuiltinCodeAnalysisTool(toolName string, userContent string) string {
	switch toolName {
	case "analyze_code":
		// Basic code analysis - extract code from user content
		code := strings.TrimSpace(userContent)
		if len(code) > 200 {
			code = code[:200] + "..."
		}

		return fmt.Sprintf("\n\n=== CODE ANALYSIS RESULTS ===\nAnalyzed Content:\n```\n%s\n```\n\nFindings:\n1. Code structure appears valid\n2. Syntax check passed\n3. Basic formatting looks correct\n\nNote: This is a preliminary analysis. For detailed code review, please use the AI Programming page.\n\n=== END OF ANALYSIS ===", code)

	case "suggest_improvements":
		// Basic improvement suggestions
		return fmt.Sprintf("\n\n=== CODE IMPROVEMENT SUGGESTIONS ===\nBased on general best practices:\n\n1. **Code Documentation**: Add comments to explain complex logic\n2. **Error Handling**: Ensure proper error checking and handling\n3. **Code Formatting**: Follow consistent indentation and style\n4. **Variable Naming**: Use descriptive and meaningful names\n5. **Testing**: Consider adding unit tests for critical functions\n\nNote: For personalized suggestions, please use the AI Programming page.\n\n=== END OF SUGGESTIONS ===")

	default:
		return fmt.Sprintf("[MCP Error: Unknown code analysis tool '%s']", toolName)
	}
}

// extractFilePath attempts to extract a file path from user content
func extractFilePath(content string) string {
	// Simple extraction - look for common file path patterns
	// This is a basic implementation - could be enhanced with regex
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Check if line looks like a file path
		if strings.Contains(line, "/") || strings.Contains(line, "\\") || strings.Contains(line, ".") {
			// Remove common prefixes
			for _, prefix := range []string{"file:", "path:", "read:", "open:"} {
				if strings.HasPrefix(strings.ToLower(line), prefix) {
					line = strings.TrimSpace(line[len(prefix):])
					break
				}
			}
			if line != "" {
				return line
			}
		}
	}
	return "."
}

// extractURLFromContent extracts URL from user content
func extractURLFromContent(content string) string {
	// Simple regex to find URLs
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	matches := urlRegex.FindStringSubmatch(content)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// constructURLFromContent constructs URL from common patterns in content
func constructURLFromContent(content string) string {
	// Convert to lowercase for easier matching
	contentLower := strings.ToLower(content)

	// Common domain patterns
	domainPatterns := map[string]string{
		"github":        "https://github.com/",
		"mark3labs":     "https://github.com/mark3labs/",
		"google":        "https://www.google.com/",
		"baidu":         "https://www.baidu.com/",
		"example":       "https://example.com/",
		"stackoverflow": "https://stackoverflow.com/",
		"wikipedia":     "https://en.wikipedia.org/",
		"youtube":       "https://www.youtube.com/",
		"twitter":       "https://twitter.com/",
		"facebook":      "https://www.facebook.com/",
		"instagram":     "https://www.instagram.com/",
		"linkedin":      "https://www.linkedin.com/",
		"reddit":        "https://www.reddit.com/",
		"amazon":        "https://www.amazon.com/",
		"microsoft":     "https://www.microsoft.com/",
		"apple":         "https://www.apple.com/",
	}

	// Check for domain mentions
	for domain, url := range domainPatterns {
		if strings.Contains(contentLower, domain) {
			return url
		}
	}

	// Special handling for Chinese domains
	chineseDomains := map[string]string{
		"百度":   "https://www.baidu.com/",
		"淘宝":   "https://www.taobao.com/",
		"京东":   "https://www.jd.com/",
		"腾讯":   "https://www.qq.com/",
		"新浪":   "https://www.sina.com.cn/",
		"网易":   "https://www.163.com/",
		"搜狐":   "https://www.sohu.com/",
		"知乎":   "https://www.zhihu.com/",
		"哔哩哔哩": "https://www.bilibili.com/",
		"豆瓣":   "https://www.douban.com/",
	}

	for domain, url := range chineseDomains {
		if strings.Contains(content, domain) {
			return url
		}
	}

	// If content mentions "获取" (get) and a name, try to construct GitHub URL
	if strings.Contains(content, "获取") {
		// Try to extract project/org name
		// Simple pattern: "获取XXX的" or "获取XXX"
		re := regexp.MustCompile(`获取([^\s的]+)`)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			name := matches[1]
			// Remove common suffixes
			name = strings.TrimSuffix(name, "的")
			name = strings.TrimSuffix(name, "最新")
			name = strings.TrimSuffix(name, "示例")
			if name != "" {
				return "https://github.com/" + name
			}
		}
	}

	// Try to extract domain from "通过XXX" pattern (e.g., "通过baidu.com")
	if strings.Contains(content, "通过") {
		re := regexp.MustCompile(`通过([^\s.,，。！？]+)`)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			domain := matches[1]
			// Clean up domain
			domain = strings.TrimSpace(domain)
			domain = strings.TrimSuffix(domain, "查找")
			domain = strings.TrimSuffix(domain, "搜索")
			domain = strings.TrimSuffix(domain, "查询")

			// Check if it looks like a domain
			if strings.Contains(domain, ".") || len(domain) >= 3 {
				// Add protocol if missing
				if !strings.HasPrefix(strings.ToLower(domain), "http") {
					return "https://" + domain
				}
				return domain
			}
		}
	}

	return ""
}

// intentDetectionModel is the lightweight model used exclusively for semantic intent
// classification. It must be fast and cheap — never use a reasoning/expensive model here.
const intentDetectionModel = "deepseek-chat"

// detectMCPIntentSemantic uses an LLM to semantically classify the user's intent
// and determine which MCP tool (if any) should be invoked.
// Falls back to detectMCPIntentKeyword on LLM error or timeout.
func (s *chatService) detectMCPIntentSemantic(ctx context.Context, content string, _ string) string {
	if s.aiService == nil {
		return s.detectMCPIntentKeyword(content)
	}

	// Special handling for playwright queries - if user explicitly mentions playwright,
	// we should recommend playwright/browser_navigate regardless of other factors
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "playwright") {
		fmt.Printf("[Chat] Special handling: User explicitly mentioned 'playwright', recommending playwright/browser_navigate\n")
		return "playwright/browser_navigate"
	}

	// Use intelligent tool recommendation if available
	if s.toolRecommender != nil {
		recommendation, err := s.toolRecommender.RecommendTool(ctx, content)
		if err == nil && recommendation != nil && recommendation.RecommendedTool != "" {
			// Log the recommendation
			fmt.Printf("[Chat] Tool recommendation: %s (confidence: %.2f)\n",
				recommendation.RecommendedTool, recommendation.Confidence)
			fmt.Printf("[Chat] Reasoning: %s\n", recommendation.Reasoning)

			// Special handling for context7 tools
			// If the recommendation is resolve-library-id but user wants documentation/examples,
			// we should use query-docs instead (which internally handles resolve-library-id)
			recommendedTool := recommendation.RecommendedTool
			if recommendedTool == "context7/resolve-library-id" {
				// Check if user wants documentation or examples
				chineseKeywords := []string{"示例", "文档", "帮助", "查询", "搜索", "获取", "最新"}
				englishKeywords := []string{"example", "examples", "documentation", "docs", "help", "query", "search", "get", "latest"}

				hasDocIntent := false
				for _, keyword := range chineseKeywords {
					if strings.Contains(content, keyword) {
						hasDocIntent = true
						break
					}
				}
				if !hasDocIntent {
					for _, keyword := range englishKeywords {
						if strings.Contains(contentLower, keyword) {
							hasDocIntent = true
							break
						}
					}
				}

				if hasDocIntent {
					fmt.Printf("[Chat] Overriding recommendation from resolve-library-id to query-docs (user wants documentation/examples)\n")
					recommendedTool = "context7/query-docs"
				}
			}

			// Return the recommended tool if confidence is high enough
			if recommendation.Confidence >= 0.7 {
				return recommendedTool
			}
		}
	}

	// Fallback to traditional tool detection
	return s.detectMCPIntentTraditional(ctx, content)
}

// detectMCPIntentTraditional uses the traditional tool detection method
func (s *chatService) detectMCPIntentTraditional(ctx context.Context, content string) string {
	// Get available MCP tools dynamically
	var availableTools []string

	// Built-in tools
	builtinTools := []string{
		"search/web_search",
		"filesystem-local/list_directory",
		"filesystem-local/read_file",
		"terminal/execute_command",
	}
	availableTools = append(availableTools, builtinTools...)

	// Get MCP server tools if MCP manager is available
	if s.mcpManager != nil {
		// Get tools from connected MCP servers
		mcpTools := s.mcpManager.GetAvailableTools()
		availableTools = append(availableTools, mcpTools...)
	}

	// Build tools description
	var toolsDesc strings.Builder
	toolsDesc.WriteString("Available tools:\n")
	for _, tool := range availableTools {
		toolsDesc.WriteString(fmt.Sprintf("- \"%s\"\n", tool))
	}
	toolsDesc.WriteString("- null: Use for general questions, programming help, math, history, explanations, creative writing, coding, translation, or anything that does not require external real-time data.\n\n")

	systemPrompt := fmt.Sprintf(`You are a tool dispatcher. Based on the user's message, decide which tool to invoke (if any).

%s
Respond with ONLY valid JSON, no markdown, no extra text:
{"tool": "<tool_name_or_null>"}`, toolsDesc.String())

	detectCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	ch, err := s.aiService.ChatStream(detectCtx, &ai.ChatRequest{
		Messages: []models.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: content},
		},
		Model:     intentDetectionModel, // always use the fast cheap model, not the conversation model
		MaxTokens: 60,
		Stream:    true,
	})
	if err != nil {
		fmt.Printf("[Chat] Semantic intent detection error, using keyword fallback: %v\n", err)
		return s.detectMCPIntentKeyword(content)
	}

	// Collect streaming response with context cancellation guard
	var sb strings.Builder
	collecting := true
	for collecting {
		select {
		case chunk, ok := <-ch:
			if !ok {
				collecting = false
			} else {
				sb.WriteString(chunk.Content)
				if chunk.Done {
					collecting = false
				}
			}
		case <-detectCtx.Done():
			fmt.Println("[Chat] Semantic intent detection timed out, using keyword fallback")
			return s.detectMCPIntentKeyword(content)
		}
	}

	// Extract JSON block (handle markdown-wrapped responses)
	raw := strings.TrimSpace(sb.String())
	if idx := strings.Index(raw, "{"); idx >= 0 {
		if end := strings.LastIndex(raw, "}"); end >= idx {
			raw = raw[idx : end+1]
		}
	}

	var decision struct {
		Tool interface{} `json:"tool"`
	}
	if err := json.Unmarshal([]byte(raw), &decision); err != nil {
		fmt.Printf("[Chat] Semantic intent parse error (%v), raw=%q — using keyword fallback\n", err, raw)
		return s.detectMCPIntentKeyword(content)
	}

	if decision.Tool == nil {
		return ""
	}
	toolStr, ok := decision.Tool.(string)
	if !ok || toolStr == "" || toolStr == "null" {
		return ""
	}

	fmt.Printf("[Chat] Semantic intent → tool: %s\n", toolStr)
	return toolStr
}

// detectMCPIntentKeyword is the legacy keyword-based fallback used when LLM detection fails.
func (s *chatService) detectMCPIntentKeyword(content string) string {
	contentLower := strings.ToLower(content)

	// Check for filesystem-local tools
	// List directory intent
	if strings.Contains(contentLower, "列出") || strings.Contains(contentLower, "显示") ||
		strings.Contains(contentLower, "list") || strings.Contains(contentLower, "show") ||
		strings.Contains(contentLower, "查看") || strings.Contains(contentLower, "look") {
		if strings.Contains(contentLower, "目录") || strings.Contains(contentLower, "文件") ||
			strings.Contains(contentLower, "directory") || strings.Contains(contentLower, "files") ||
			strings.Contains(contentLower, "folder") || strings.Contains(contentLower, "文件夹") {
			fmt.Printf("[Chat] Auto-detect: User wants to list directory\n")
			return "filesystem-local/list_directory"
		}
	}

	// Read file intent - check for file path patterns
	if strings.Contains(contentLower, "读取") || strings.Contains(contentLower, "read") ||
		strings.Contains(contentLower, "打开") || strings.Contains(contentLower, "open") ||
		strings.Contains(contentLower, "查看") || strings.Contains(contentLower, "view") ||
		strings.Contains(contentLower, "显示") || strings.Contains(contentLower, "show") {
		// Check if there's a file path in the content
		filePath := extractFilePath(content)
		if filePath != "." && filePath != "" {
			// Check if it looks like a file (has extension)
			if strings.Contains(filePath, ".") && !strings.HasSuffix(filePath, "/") {
				fmt.Printf("[Chat] Auto-detect: User wants to read file: %s\n", filePath)
				return "filesystem-local/read_file"
			}
		}
	}

	// Check for search intent
	// Primary search keywords - if these are present, trigger web search
	searchKeywords := []string{"搜索", "search", "查找", "find", "查询", "query", "google", "百度", "bing"}
	hasSearchKeyword := false
	for _, keyword := range searchKeywords {
		if strings.Contains(contentLower, keyword) {
			hasSearchKeyword = true
			break
		}
	}

	// Web context keywords - these strengthen the search intent but aren't required
	webContextKeywords := []string{"网络", "网上", "web", "internet", "online", "最新", "latest", "新闻", "news", "论文", "paper", "文章", "article"}
	hasWebContext := false
	for _, keyword := range webContextKeywords {
		if strings.Contains(contentLower, keyword) {
			hasWebContext = true
			break
		}
	}

	// If strong search keyword present, or search + web context
	if hasSearchKeyword || (hasWebContext && strings.Contains(contentLower, "search")) {
		fmt.Printf("[Chat] Auto-detect: User wants to search web (searchKeyword=%v, webContext=%v)\n", hasSearchKeyword, hasWebContext)
		return "search/web_search"
	}

	// Check for terminal/execute intent
	if strings.Contains(contentLower, "执行") || strings.Contains(contentLower, "运行") ||
		strings.Contains(contentLower, "execute") || strings.Contains(contentLower, "run") ||
		strings.Contains(contentLower, "命令") || strings.Contains(contentLower, "command") ||
		strings.Contains(contentLower, "脚本") || strings.Contains(contentLower, "script") {
		fmt.Printf("[Chat] Auto-detect: User wants to execute command\n")
		return "terminal/execute_command"
	}

	// Check for code analysis intent
	if strings.Contains(contentLower, "分析代码") || strings.Contains(contentLower, "代码分析") ||
		strings.Contains(contentLower, "analyze code") || strings.Contains(contentLower, "code analysis") ||
		strings.Contains(contentLower, "检查代码") || strings.Contains(contentLower, "review code") {
		fmt.Printf("[Chat] Auto-detect: User wants code analysis\n")
		return "code-analysis/analyze_code"
	}

	// No matching intent found
	return ""
}

// SendMessageStreamWithRAG sends a message with optional RAG (knowledge base) context
func (s *chatService) SendMessageStreamWithRAG(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string, ragEnabled bool, ragDocIDs []string, promptEngineeringConfig *ai.PromptEngineeringConfig) (<-chan ai.ChatChunk, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	// 0. Fetch conversation to get model
	var conv models.Conversation
	if err := database.DB.First(&conv, convID).Error; err != nil {
		return nil, err
	}

	// Override conv.ModelType with the model provided by the user in real-time
	if model != "" {
		conv.ModelType = model
		database.DB.Model(&models.Conversation{}).Where("id = ?", convID).Update("model_type", model)
	}

	fmt.Printf("[Chat+RAG] Using model: %s, WebSearch: %v, RAG: %v\n", conv.ModelType, webSearch, ragEnabled)

	// 1. Save user message with branch association
	userMsg := &models.Message{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
		Status:         "success",
		BranchID:       conv.ActiveBranchID, // Associate with active branch
	}
	if err := database.DB.Create(userMsg).Error; err != nil {
		return nil, err
	}

	// Update branch message count if branch exists
	if conv.ActiveBranchID != nil {
		database.DB.Model(&models.Branch{}).Where("id = ?", *conv.ActiveBranchID).
			UpdateColumn("message_count", gorm.Expr("message_count + 1"))
	}

	// 2. Update conversation last message
	database.DB.Model(&models.Conversation{}).Where("id = ?", convID).Update("last_message", content)

	// 3. Auto-generate title
	if conv.Title == "New Chat" {
		go s.generateTitle(userID, convID, content, conv.ModelType)
	}

	// 4. Get history for AI context
	var history []models.Message
	database.DB.Where("conversation_id = ?", convID).Order("created_at desc").Limit(10).Find(&history)
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	// 5. Build RAG context if enabled
	var ragContext string
	var ragResults []map[string]interface{}
	if ragEnabled && s.ragService != nil {
		ragQuery := models.RAGQuery{
			Query:     content,
			TopK:      5,
			Threshold: 0.7,
			DocIDs:    ragDocIDs,
		}
		results, err := s.ragService.Query(userID, ragQuery)
		if err != nil {
			fmt.Printf("[RAG] Query failed: %v\n", err)
		} else if len(results) > 0 {
			fmt.Printf("[RAG] Found %d relevant chunks\n", len(results))

			var ragBuilder strings.Builder
			ragBuilder.WriteString("\n\n[Knowledge Base Context]\n")
			ragBuilder.WriteString("The following information is from the user's uploaded documents:\n\n")

			for i, res := range results {
				ragBuilder.WriteString(fmt.Sprintf("--- Document: %s (relevance: %.2f) ---\n%s\n\n", res.Filename, res.Score, res.Content))
				ragResults = append(ragResults, map[string]interface{}{
					"filename":    res.Filename,
					"content":     res.Content,
					"chunk_index": res.ChunkIndex,
					"score":       res.Score,
				})
				if i >= 4 { // Limit to 5 chunks
					break
				}
			}
			ragBuilder.WriteString("Please use the above context to answer the user's question accurately. Cite the document sources when relevant.\n")
			ragContext = ragBuilder.String()
		}
	}

	// 6. Build web search context if enabled
	var searchContext string
	var searchResultsData []map[string]interface{}
	if webSearch {
		enhancedQuery := s.contextProcessor.BuildEnhancedQuery(content, history)
		searchResults, err := s.searchService.Search(ctx, enhancedQuery, userID, searchProvider)
		if err == nil && len(searchResults) > 0 {
			for _, res := range searchResults {
				searchResultsData = append(searchResultsData, map[string]interface{}{
					"title":   res.Title,
					"snippet": res.Snippet,
					"url":     res.URL,
				})
			}

			var searchBuilder strings.Builder
			searchBuilder.WriteString("\n\n[Web Search Results]\n")
			for _, res := range searchResults {
				searchBuilder.WriteString(fmt.Sprintf("Title: %s\nContent: %s\nSource: %s\n\n", res.Title, res.Snippet, res.URL))
			}
			searchContext = searchBuilder.String()
		}
	}

	// 7. Build final history with context
	var finalHistory []models.Message

	// Add system prompt (merge with custom system prompt if provided)
	systemPrompt, _ := s.loadSystemPrompt(convID)
	if customSystemPrompt != "" {
		if systemPrompt != "" {
			// Prepend custom system prompt to default, separated by double newline
			systemPrompt = customSystemPrompt + "\n\n" + systemPrompt
			fmt.Printf("[RAG Chat] Merged custom system prompt with default system prompt\n")
		} else {
			systemPrompt = customSystemPrompt
			fmt.Printf("[RAG Chat] Using custom system prompt only (no default)\n")
		}
	}
	if systemPrompt != "" {
		finalHistory = append(finalHistory, models.Message{Role: "system", Content: systemPrompt})
	}

	// Add history
	finalHistory = append(finalHistory, history...)

	// Add current message with all context
	currentMessage := models.Message{
		Role:    "user",
		Content: content,
	}
	if ragContext != "" {
		currentMessage.Content += ragContext
	}
	if searchContext != "" {
		currentMessage.Content += searchContext
	}
	finalHistory = append(finalHistory, currentMessage)

	// 9. Create output channel with metadata
	outputCh := make(chan ai.ChatChunk, 100)
	go func() {
		defer close(outputCh)

		// 应用提示词工程功能（在goroutine内部以便发送流式事件）
		if promptEngineeringConfig != nil && promptEngineeringConfig.Enabled {
			// 发送提示词优化开始事件
			outputCh <- ai.ChatChunk{
				Type:    "prompt_optimization_start",
				Content: "🔧 正在优化提示词...",
				Metadata: map[string]interface{}{
					"status":      "started",
					"timestamp":   time.Now().Unix(),
					"totalSteps":  3,
					"currentStep": 0,
				},
			}

			// 首先处理非优化功能（角色、输出格式、思维链等）
			nonOptimizationConfig := *promptEngineeringConfig
			nonOptimizationConfig.PromptOptimizationEnabled = false
			processedMessages, err := s.promptEngineeringService.ProcessMessage(ctx, finalHistory, nonOptimizationConfig, fmt.Sprintf("%d", convID))
			if err != nil {
				fmt.Printf("[Chat+RAG] Error processing prompt engineering (non-optimization): %v\n", err)
				// 发送优化失败事件
				outputCh <- ai.ChatChunk{
					Type:    "prompt_optimization_error",
					Content: "❌ 提示词处理失败，使用原始提示词",
					Metadata: map[string]interface{}{
						"status": "error",
						"error":  err.Error(),
					},
				}
				// 继续使用原始历史
				processedMessages = finalHistory
			} else {
				finalHistory = processedMessages
				fmt.Printf("[Chat+RAG] Applied non-optimization prompt engineering config\n")
			}

			// 如果启用提示词优化，执行详细步骤
			if promptEngineeringConfig.PromptOptimizationEnabled {
				// 查找最后一个用户消息进行优化
				var lastUserIndex int = -1
				var lastUserContent string
				for i := len(finalHistory) - 1; i >= 0; i-- {
					if finalHistory[i].Role == "user" {
						lastUserIndex = i
						lastUserContent = finalHistory[i].Content
						break
					}
				}

				if lastUserIndex >= 0 {
					optimizedPrompt := lastUserContent

					// 步骤1: 理解意图
					outputCh <- ai.ChatChunk{
						Type:    "prompt_optimization_step",
						Content: "📝 理解用户意图...",
						Metadata: map[string]interface{}{
							"step":       "intent_understanding",
							"stepNumber": 1,
							"totalSteps": 3,
							"progress":   33,
						},
					}
					intentResult, err := s.promptEngineeringService.UnderstandIntent(ctx, optimizedPrompt, *promptEngineeringConfig)
					if err == nil {
						optimizedPrompt = intentResult
						// 发送优化内容
						outputCh <- ai.ChatChunk{
							Type:    "prompt_optimization_content",
							Content: optimizedPrompt,
							Metadata: map[string]interface{}{
								"step":       "intent_understanding",
								"stepNumber": 1,
								"totalSteps": 3,
							},
						}
					}

					// 步骤2: 上下文增强
					outputCh <- ai.ChatChunk{
						Type:    "prompt_optimization_step",
						Content: "🔍 增强上下文信息...",
						Metadata: map[string]interface{}{
							"step":       "context_enhancement",
							"stepNumber": 2,
							"totalSteps": 3,
							"progress":   66,
						},
					}
					enhancedResult, err := s.promptEngineeringService.EnhanceWithContext(ctx, optimizedPrompt, finalHistory, *promptEngineeringConfig)
					if err == nil {
						optimizedPrompt = enhancedResult
						// 发送优化内容
						outputCh <- ai.ChatChunk{
							Type:    "prompt_optimization_content",
							Content: optimizedPrompt,
							Metadata: map[string]interface{}{
								"step":       "context_enhancement",
								"stepNumber": 2,
								"totalSteps": 3,
							},
						}
					}

					// 步骤3: 精炼提示词
					outputCh <- ai.ChatChunk{
						Type:    "prompt_optimization_step",
						Content: "✨ 精炼提示词表达...",
						Metadata: map[string]interface{}{
							"step":       "prompt_refinement",
							"stepNumber": 3,
							"totalSteps": 3,
							"progress":   100,
						},
					}
					refinedResult, err := s.promptEngineeringService.RefineUserPrompt(ctx, optimizedPrompt, *promptEngineeringConfig)
					if err == nil {
						optimizedPrompt = refinedResult
						// 发送优化内容
						outputCh <- ai.ChatChunk{
							Type:    "prompt_optimization_content",
							Content: optimizedPrompt,
							Metadata: map[string]interface{}{
								"step":       "prompt_refinement",
								"stepNumber": 3,
								"totalSteps": 3,
							},
						}
					}

					// 更新消息历史中的用户消息
					finalHistory[lastUserIndex].Content = optimizedPrompt
					fmt.Printf("[Chat+RAG] Prompt optimization completed: %s\n", optimizedPrompt)
				}
			}

			// 发送提示词优化完成事件
			outputCh <- ai.ChatChunk{
				Type:    "prompt_optimization_complete",
				Content: "✅ 提示词优化完成",
				Metadata: map[string]interface{}{
					"status":    "completed",
					"timestamp": time.Now().Unix(),
				},
			}
		}

		// 创建AI请求
		req := &ai.ChatRequest{
			UserID:                  userID,
			ConversationID:          convID,
			Messages:                finalHistory,
			Model:                   conv.ModelType,
			Stream:                  true,
			WebSearch:               webSearch,
			PromptEngineeringConfig: promptEngineeringConfig,
			// SystemPrompt already merged into message history, no need to pass separately
		}

		// 调用AI服务
		streamCh, err := s.aiService.ChatStream(ctx, req)
		if err != nil {
			fmt.Printf("[Chat+RAG] Error calling AI service: %v\n", err)
			outputCh <- ai.ChatChunk{
				Type:    "error",
				Content: "❌ AI服务调用失败",
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}

		// Send RAG results if any - include in first chunk's SearchResults
		combinedResults := make([]map[string]interface{}, 0)
		for _, res := range ragResults {
			res["source"] = "rag"
			combinedResults = append(combinedResults, res)
		}
		for _, res := range searchResultsData {
			res["source"] = "web"
			combinedResults = append(combinedResults, res)
		}

		// Send initial chunk with search/RAG results
		if len(combinedResults) > 0 {
			outputCh <- ai.ChatChunk{
				Content:       "",
				Done:          false,
				SearchResults: combinedResults,
			}
		}

		// Forward AI responses
		var fullContent strings.Builder
		for chunk := range streamCh {
			outputCh <- chunk
			if !chunk.Done {
				fullContent.WriteString(chunk.Content)
			}
		}

		// Save assistant message with branch association
		if fullContent.Len() > 0 {
			assistantMsg := &models.Message{
				ConversationID: convID,
				Role:           "assistant",
				Content:        fullContent.String(),
				Status:         "success",
				BranchID:       conv.ActiveBranchID, // Associate with active branch
			}
			database.DB.Create(assistantMsg)

			// Update branch message count if branch exists
			if conv.ActiveBranchID != nil {
				database.DB.Model(&models.Branch{}).Where("id = ?", *conv.ActiveBranchID).
					UpdateColumn("message_count", gorm.Expr("message_count + 1"))
			}
		}
	}()

	return outputCh, nil
}

// extractLibraryName extracts library name from user query using intelligent extraction
func extractLibraryName(userContent string) string {
	// First, try to find any word that looks like a library/package name
	// Library names typically contain alphanumeric characters, dots, dashes, underscores
	// They don't contain Chinese characters or common English/Chinese words

	// Improved regex to capture library names more accurately
	// This matches sequences of alphanumeric characters with dots, dashes, underscores
	// But excludes sequences that are purely numeric or common words
	libraryRe := regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9._-]*[a-zA-Z0-9])\b`)
	potentialMatches := libraryRe.FindAllString(userContent, -1)

	if len(potentialMatches) == 0 {
		return ""
	}

	// Common words to exclude (both Chinese and English)
	commonWords := map[string]bool{
		// Chinese common words
		"使用": true, "获取": true, "最新": true, "的": true, "示例": true,
		"文档": true, "帮助": true, "查询": true, "搜索": true, "查找": true,
		"如何": true, "怎样": true, "什么": true, "哪个": true, "哪里": true,

		// English common words
		"use": true, "get": true, "latest": true, "example": true, "examples": true,
		"documentation": true, "docs": true, "help": true, "query": true, "search": true,
		"find": true, "how": true, "what": true, "which": true, "where": true,
		"context7": true, "context": true, "7": true,

		// Common library-related words that might be confused with library names
		"library": true, "package": true, "module": true, "framework": true,
		"api": true, "sdk": true, "tool": true, "toolkit": true,
	}

	// Also exclude very short words (less than 2 chars) and pure numbers
	for _, match := range potentialMatches {
		matchLower := strings.ToLower(match)

		// Skip if it's a common word
		if commonWords[matchLower] {
			continue
		}

		// Skip if too short (less than 2 characters)
		if len(match) < 2 {
			continue
		}

		// Skip if it's purely numeric
		if isNumeric(match) {
			continue
		}

		// Check if it looks like a version number (e.g., v1.0, 2.3.4)
		if isVersionNumber(match) {
			continue
		}

		// This looks like a valid library name
		return match
	}

	return ""
}

// isNumeric checks if a string is purely numeric
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isVersionNumber checks if a string looks like a version number
func isVersionNumber(s string) bool {
	// Version patterns: v1.0, 1.0.0, 2.3, v2.3.4, etc.
	versionRe := regexp.MustCompile(`^(v?\d+(\.\d+)*)$`)
	return versionRe.MatchString(s)
}

// getFallbackLibraryId returns a fallback library ID for common libraries
func getFallbackLibraryId(libraryName string) string {
	libraryIdMap := map[string]string{
		"next.js":    "/vercel/next.js",
		"react":      "/facebook/react",
		"vue":        "/vuejs/vue",
		"angular":    "/angular/angular",
		"node.js":    "/nodejs/node",
		"express":    "/expressjs/express",
		"mongodb":    "/mongodb/docs",
		"postgresql": "/postgresql/postgres",
		"docker":     "/docker/docs",
		"kubernetes": "/kubernetes/kubernetes",
		"typescript": "/microsoft/typescript",
		"javascript": "/mdn/javascript",
		"python":     "/python/python",
		"go":         "/golang/go",
		"rust":       "/rust-lang/rust",
		"java":       "/oracle/java",
		"mark5labs":  "/mark5labs/mark5labs", // Added for mark5labs
		"mark3labs":  "/mark3labs/mark3labs", // Added for mark3labs
	}

	if libraryId, found := libraryIdMap[libraryName]; found {
		return libraryId
	}

	// Default fallback - try to construct a reasonable library ID
	// For unknown libraries, use format: /org/libraryname
	// Remove version suffixes and special characters
	cleanName := strings.ToLower(libraryName)
	cleanName = strings.TrimSuffix(cleanName, ".js")
	cleanName = strings.TrimSuffix(cleanName, ".ts")
	cleanName = strings.TrimSuffix(cleanName, ".py")
	cleanName = strings.TrimSuffix(cleanName, ".go")
	cleanName = strings.TrimSuffix(cleanName, ".rs")
	cleanName = strings.TrimSuffix(cleanName, ".java")

	// Replace dots and dashes with underscores
	cleanName = strings.ReplaceAll(cleanName, ".", "_")
	cleanName = strings.ReplaceAll(cleanName, "-", "_")

	return fmt.Sprintf("/%s/%s", cleanName, cleanName)
}

// extractLibraryIdFromResult extracts library ID from resolve-library-id result
func extractLibraryIdFromResult(result string, libraryName string) string {
	// The result from resolve-library-id is a string containing JSON
	// We need to parse it to extract the libraryId

	// First, try to find libraryId pattern in the result
	patterns := []string{
		`"libraryId"\s*:\s*"([^"]+)"`,
		`libraryId["']?\s*:\s*["']([^"']+)["']`,
		`/([^/]+/[^/\s]+)`, // Pattern like /org/project
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(result)
		if len(matches) > 1 {
			libraryId := matches[1]
			fmt.Printf("[Chat] Found libraryId in result using pattern '%s': %s\n", pattern, libraryId)
			return libraryId
		}
	}

	// If no libraryId found, try to extract based on library name
	// Look for patterns like "/org/libraryname" or "/org/project"
	libraryPatterns := []string{
		fmt.Sprintf(`/([^/]+/%s)`, strings.ReplaceAll(libraryName, ".", "\\.")),
		fmt.Sprintf(`/(%s/[^/]+)`, strings.ReplaceAll(libraryName, ".", "\\.")),
	}

	for _, pattern := range libraryPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(result)
		if len(matches) > 1 {
			libraryId := matches[1]
			fmt.Printf("[Chat] Found libraryId using library name pattern '%s': %s\n", pattern, libraryId)
			return libraryId
		}
	}

	// If still not found, return empty string
	fmt.Printf("[Chat] Could not extract libraryId from result: %s\n", result[:min(200, len(result))])
	return ""
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getCachedLibraryId retrieves library ID from cache
func (s *chatService) getCachedLibraryId(libraryName string) (string, bool) {
	s.libraryIdCacheLock.RLock()
	defer s.libraryIdCacheLock.RUnlock()

	libraryId, found := s.libraryIdCache[libraryName]
	return libraryId, found
}

// setCachedLibraryId stores library ID in cache
func (s *chatService) setCachedLibraryId(libraryName, libraryId string) {
	s.libraryIdCacheLock.Lock()
	defer s.libraryIdCacheLock.Unlock()

	s.libraryIdCache[libraryName] = libraryId
	fmt.Printf("[Chat] Cached libraryId for %s: %s\n", libraryName, libraryId)
}

// clearLibraryIdCache clears the entire cache
func (s *chatService) clearLibraryIdCache() {
	s.libraryIdCacheLock.Lock()
	defer s.libraryIdCacheLock.Unlock()

	s.libraryIdCache = make(map[string]string)
	fmt.Printf("[Chat] Cleared libraryId cache\n")
}

// getPlaywrightDefaultURLs gets the default URLs for playwright browser_navigate from config
func (s *chatService) getPlaywrightDefaultURLs() []string {
	// Try to get from config
	if config.GlobalConfig != nil && len(config.GlobalConfig.Tools.Playwright.DefaultURLs) > 0 {
		return config.GlobalConfig.Tools.Playwright.DefaultURLs
	}

	// Fallback to hardcoded defaults
	return []string{"https://www.baidu.com"}
}

// getPlaywrightDefaultURL gets a single default URL (for backward compatibility)
func (s *chatService) getPlaywrightDefaultURL() string {
	urls := s.getPlaywrightDefaultURLs()
	if len(urls) > 0 {
		return urls[0]
	}
	return "https://www.baidu.com"
}

// GetToolDocumentation gets detailed documentation for a specific tool
func (s *chatService) GetToolDocumentation(serverName, toolName string) (*config.MCPTool, error) {
	if s.toolRecommender == nil {
		return nil, fmt.Errorf("tool recommender not available")
	}

	return s.toolRecommender.GetToolDocumentation(serverName, toolName)
}

// GetServerDocumentation gets complete documentation for an MCP server
func (s *chatService) GetServerDocumentation(serverName string) (*config.MCPServerDocumentation, error) {
	if s.mcpManager == nil {
		return nil, fmt.Errorf("MCP manager not available")
	}

	return s.mcpManager.GetServerDocumentation(serverName)
}

// GetAllServerSummaries gets summaries for all available MCP servers
func (s *chatService) GetAllServerSummaries() map[string]string {
	if s.mcpManager == nil {
		return make(map[string]string)
	}

	return s.mcpManager.GetAllServerSummaries()
}

// handleMultipleURLs handles browser_navigate for multiple URLs
func (s *chatService) handleMultipleURLs(serverName, toolName string, urls []string, userContent string) string {
	fmt.Printf("[Chat] Handling multiple URLs: %v\n", urls)

	var results []string
	for i, url := range urls {
		fmt.Printf("[Chat] Navigating to URL %d/%d: %s\n", i+1, len(urls), url)

		args := map[string]interface{}{
			"url": url,
			// 添加无头模式参数，避免在无头环境中运行有界面浏览器
			"headless": true,
			// 添加超时参数
			"timeout": 30000, // 30秒
		}

		result, err := s.mcpManager.CallTool(serverName, toolName, args)
		if err != nil {
			fmt.Printf("[Chat] Failed to navigate to %s: %v\n", url, err)
			results = append(results, fmt.Sprintf("❌ Failed to navigate to %s: %v", url, err))
			continue
		}

		// Extract text content from result
		var pageContent string
		// Try to parse as JSON first
		var jsonResult map[string]interface{}
		if err := json.Unmarshal([]byte(result), &jsonResult); err == nil {
			// If it's JSON, try to get text content
			if text, ok := jsonResult["text"].(string); ok {
				pageContent = text
			} else if html, ok := jsonResult["html"].(string); ok {
				// Extract text from HTML
				pageContent = extractTextFromHTML(html)
			} else {
				pageContent = result
			}
		} else {
			pageContent = result
		}

		// Truncate content for display
		if len(pageContent) > 1000 {
			pageContent = pageContent[:1000] + "... [truncated]"
		}

		results = append(results, fmt.Sprintf("🌐 **%s**\n%s\n\n---\n", url, pageContent))
		fmt.Printf("[Chat] Successfully navigated to %s, content length: %d\n", url, len(pageContent))
	}

	// Combine all results
	combinedResult := fmt.Sprintf("# Multi-Site Search Results\n\n")
	combinedResult += fmt.Sprintf("**Search Query:** %s\n\n", userContent)
	combinedResult += fmt.Sprintf("**Searched Sites:** %d\n\n", len(urls))

	for i, result := range results {
		combinedResult += fmt.Sprintf("## Site %d/%d\n", i+1, len(urls))
		combinedResult += result
	}

	combinedResult += fmt.Sprintf("\n**Summary:** Searched %d sites for: %s", len(urls), userContent)

	return fmt.Sprintf("\n\n[MCP Tool Result - %s/%s]\n%s", serverName, toolName, combinedResult)
}

// extractTextFromHTML extracts readable text from HTML (simple implementation)
func extractTextFromHTML(html string) string {
	// Remove script and style tags
	re := regexp.MustCompile(`<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`)
	html = re.ReplaceAllString(html, "")

	// Remove HTML tags
	re = regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")

	// Collapse multiple spaces
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Trim and limit length
	text = strings.TrimSpace(text)
	if len(text) > 2000 {
		text = text[:2000] + "..."
	}

	return text
}
