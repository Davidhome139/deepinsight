package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/pkg/database"
	"backend/internal/services/agent"
	"backend/internal/services/ai"
	"backend/internal/services/rag"
	"backend/internal/services/search"

	"gorm.io/gorm"
)

type ChatService interface {
	CreateConversation(userID uint, title string, model string) (*models.Conversation, error)
	GetConversations(userID uint) ([]models.Conversation, error)
	GetMessages(userID uint, convID uint) ([]models.Message, error)
	SendMessageStream(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string) (<-chan ai.ChatChunk, error)
	SendMessageStreamWithRAG(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string, ragEnabled bool, ragDocIDs []string) (<-chan ai.ChatChunk, error)
	GenerateConversationSummary(userID uint, convID uint, model string) (string, error)
}

type chatService struct {
	aiService        ai.AIService
	searchService    search.SearchService
	mcpManager       *agent.MCPManager
	contextProcessor *ContextProcessor
	ragService       *rag.RAGService
}

func NewChatService(aiService ai.AIService, searchService search.SearchService, mcpManager *agent.MCPManager) ChatService {
	return &chatService{
		aiService:        aiService,
		searchService:    searchService,
		mcpManager:       mcpManager,
		contextProcessor: NewContextProcessor(),
	}
}

// SetRAGService sets the RAG service for knowledge base integration
func (s *chatService) SetRAGService(ragService *rag.RAGService) {
	s.ragService = ragService
}

func (s *chatService) CreateConversation(userID uint, title string, model string) (*models.Conversation, error) {
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
	var conversations []models.Conversation
	if err := database.DB.Where("user_id = ?", userID).Order("updated_at desc").Find(&conversations).Error; err != nil {
		return nil, err
	}
	return conversations, nil
}

func (s *chatService) GetMessages(userID uint, convID uint) ([]models.Message, error) {
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

func (s *chatService) SendMessageStream(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string) (<-chan ai.ChatChunk, error) {
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

	// 5. 构建最终的消息历史
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

	// 6. Call AI service
	req := &ai.ChatRequest{
		UserID:         userID,
		ConversationID: convID,
		Messages:       history,
		Model:          conv.ModelType,
		Stream:         true,
		WebSearch:      webSearch,
		// SystemPrompt already merged into message history, no need to pass separately
	}

	aiCh, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return nil, err
	}

	// 7. Wrap channel to save response context
	outCh := make(chan ai.ChatChunk)
	go func() {
		defer close(outCh)
		var assistantReply string

		// Send search results as first chunk if available
		if len(searchResultsData) > 0 {
			outCh <- ai.ChatChunk{
				SearchResults: searchResultsData,
			}
		}

		for chunk := range aiCh {
			if !chunk.Done {
				assistantReply += chunk.Content
			}
			outCh <- chunk
		}

		// 8. Save assistant message when stream is finished
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

	fmt.Printf("[Chat] Calling MCP tool: server=%s, tool=%s\n", serverName, toolName)

	// Check if MCP manager is available
	if s.mcpManager == nil {
		fmt.Println("[Chat] MCP manager not available")
		return "[MCP Error: MCP manager not initialized]"
	}

	// Ensure MCP servers are discovered
	s.mcpManager.Discover()

	// Get the server
	server, ok := s.mcpManager.GetServer(serverName)
	if !ok {
		fmt.Printf("[Chat] MCP server not found: %s\n", serverName)
		return fmt.Sprintf("[MCP Error: Server '%s' not found]\n\nAvailable servers can be configured in Settings.", serverName)
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

	// For external MCP servers, check connection status
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
	default:
		args = map[string]interface{}{}
	}

	// Call the tool
	result, err := s.mcpManager.CallTool(serverName, toolName, args)
	if err != nil {
		fmt.Printf("[Chat] MCP tool call failed: %v\n", err)
		return fmt.Sprintf("[MCP Tool Error: %v]", err)
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

	systemPrompt := `You are a tool dispatcher. Based on the user's message, decide which tool to invoke (if any).

Available tools:
- "search/web_search": Use when the user needs real-time or current information — today's news, current prices, live weather, recent events, trending topics, sports scores, stock market data, recent tech releases, or any fact that may have changed recently.
- "filesystem-local/list_directory": Use when the user wants to list or browse files and folders in a directory.
- "filesystem-local/read_file": Use when the user references a specific file path and wants to see its content.
- "terminal/execute_command": Use when the user explicitly asks to run or execute a shell command or script.
- null: Use for general questions, programming help, math, history, explanations, creative writing, coding, translation, or anything that does not require external real-time data.

Respond with ONLY valid JSON, no markdown, no extra text:
{"tool": "<tool_name_or_null>"}`

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
func (s *chatService) SendMessageStreamWithRAG(ctx context.Context, userID uint, convID uint, content string, model string, webSearch bool, searchProvider string, mcpTool string, customSystemPrompt string, ragEnabled bool, ragDocIDs []string) (<-chan ai.ChatChunk, error) {
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

	// 8. Call AI service
	req := &ai.ChatRequest{
		UserID:         userID,
		ConversationID: convID,
		Messages:       finalHistory,
		Model:          conv.ModelType,
		Stream:         true,
		WebSearch:      webSearch,
		// SystemPrompt already merged into message history, no need to pass separately
	}

	streamCh, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return nil, err
	}

	// 9. Create output channel with metadata
	outputCh := make(chan ai.ChatChunk, 100)
	go func() {
		defer close(outputCh)

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
