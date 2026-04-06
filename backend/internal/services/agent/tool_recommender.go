package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/services/ai"
)

// ToolRecommender provides intelligent tool recommendations based on user queries
type ToolRecommender struct {
	mcpManager *MCPManager
	aiService  ai.AIService
	cache      map[string]*RecommendationResult
	cacheLock  sync.RWMutex
}

// RecommendationResult represents the result of a tool recommendation
type RecommendationResult struct {
	RecommendedServer string
	RecommendedTool   string
	Confidence        float64
	Reasoning         string
	ServerSummary     string
	ToolDescription   string
	AlternativeTools  []AlternativeTool
}

// AlternativeTool represents an alternative tool option
type AlternativeTool struct {
	Server      string
	Tool        string
	Confidence  float64
	Description string
}

// NewToolRecommender creates a new tool recommender
func NewToolRecommender(mcpManager *MCPManager, aiService ai.AIService) *ToolRecommender {
	return &ToolRecommender{
		mcpManager: mcpManager,
		aiService:  aiService,
		cache:      make(map[string]*RecommendationResult),
	}
}

// RecommendTool recommends the best tool for a user query
func (r *ToolRecommender) RecommendTool(ctx context.Context, userQuery string) (*RecommendationResult, error) {
	// Check cache first
	r.cacheLock.RLock()
	if cached, exists := r.cache[userQuery]; exists {
		r.cacheLock.RUnlock()
		return cached, nil
	}
	r.cacheLock.RUnlock()

	// Get server summaries
	serverSummaries := r.mcpManager.GetAllServerSummaries()
	if len(serverSummaries) == 0 {
		return &RecommendationResult{
			RecommendedServer: "",
			RecommendedTool:   "",
			Confidence:        0.0,
			Reasoning:         "No MCP servers available",
		}, nil
	}

	// Build prompt for AI recommendation
	prompt := r.buildRecommendationPrompt(userQuery, serverSummaries)

	// Get AI recommendation
	recommendation, err := r.getAIRecommendation(ctx, prompt)
	if err != nil {
		// Fallback to keyword-based recommendation
		return r.keywordBasedRecommendation(userQuery, serverSummaries), nil
	}

	// Parse AI response
	result := r.parseAIResponse(recommendation, serverSummaries)

	// Cache the result
	r.cacheLock.Lock()
	r.cache[userQuery] = result
	r.cacheLock.Unlock()

	return result, nil
}

// buildRecommendationPrompt builds the prompt for AI recommendation
func (r *ToolRecommender) buildRecommendationPrompt(userQuery string, serverSummaries map[string]string) string {
	var prompt strings.Builder

	prompt.WriteString("You are a tool recommendation system. Based on the user's query, recommend the most appropriate MCP server and tool.\n\n")
	prompt.WriteString("User query: " + userQuery + "\n\n")

	prompt.WriteString("Available MCP servers and their summaries:\n")
	for serverName, summary := range serverSummaries {
		prompt.WriteString(fmt.Sprintf("## Server: %s\n", serverName))
		prompt.WriteString(summary + "\n\n")
	}

	prompt.WriteString("Instructions:\n")
	prompt.WriteString("1. Analyze the user's query and match it to the most relevant MCP server\n")
	prompt.WriteString("2. Within that server, select the most appropriate tool\n")
	prompt.WriteString("3. Provide your reasoning\n")
	prompt.WriteString("4. Provide confidence score (0.0 to 1.0)\n")
	prompt.WriteString("5. Suggest alternative tools if applicable\n\n")

	prompt.WriteString("Respond in this exact format:\n")
	prompt.WriteString("RECOMMENDED_SERVER: <server_name>\n")
	prompt.WriteString("RECOMMENDED_TOOL: <server_name>/<tool_name>\n")
	prompt.WriteString("CONFIDENCE: <0.0-1.0>\n")
	prompt.WriteString("REASONING: <explanation>\n")
	prompt.WriteString("ALTERNATIVES: <server1>/<tool1> (confidence), <server2>/<tool2> (confidence)\n")

	return prompt.String()
}

// getAIRecommendation gets recommendation from AI
func (r *ToolRecommender) getAIRecommendation(ctx context.Context, prompt string) (string, error) {
	if r.aiService == nil {
		return "", fmt.Errorf("AI service not available")
	}

	req := &ai.ChatRequest{
		Messages: []models.Message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: "Please recommend the best tool for this query."},
		},
		Model:     "deepseek-chat", // Use a fast model
		MaxTokens: 500,
		Stream:    true,
	}

	ch, err := r.aiService.ChatStream(ctx, req)
	if err != nil {
		return "", err
	}

	// Collect streaming response
	var sb strings.Builder
	for chunk := range ch {
		sb.WriteString(chunk.Content)
		if chunk.Done {
			break
		}
	}

	return sb.String(), nil
}

// parseAIResponse parses the AI response into a RecommendationResult
func (r *ToolRecommender) parseAIResponse(aiResponse string, serverSummaries map[string]string) *RecommendationResult {
	result := &RecommendationResult{
		Confidence: 0.5, // Default confidence
	}

	lines := strings.Split(aiResponse, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "RECOMMENDED_SERVER:") {
			result.RecommendedServer = strings.TrimSpace(strings.TrimPrefix(line, "RECOMMENDED_SERVER:"))
			if summary, exists := serverSummaries[result.RecommendedServer]; exists {
				result.ServerSummary = summary
			}
		} else if strings.HasPrefix(line, "RECOMMENDED_TOOL:") {
			result.RecommendedTool = strings.TrimSpace(strings.TrimPrefix(line, "RECOMMENDED_TOOL:"))
			// Try to get tool description
			if result.RecommendedServer != "" && result.RecommendedTool != "" {
				if doc, err := r.mcpManager.GetServerDocumentation(result.RecommendedServer); err == nil {
					toolName := strings.TrimPrefix(result.RecommendedTool, result.RecommendedServer+"/")
					if tool, found := doc.GetToolByName(toolName); found {
						result.ToolDescription = tool.Description
					}
				}
			}
		} else if strings.HasPrefix(line, "CONFIDENCE:") {
			confStr := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			fmt.Sscanf(confStr, "%f", &result.Confidence)
		} else if strings.HasPrefix(line, "REASONING:") {
			result.Reasoning = strings.TrimSpace(strings.TrimPrefix(line, "REASONING:"))
		} else if strings.HasPrefix(line, "ALTERNATIVES:") {
			altStr := strings.TrimSpace(strings.TrimPrefix(line, "ALTERNATIVES:"))
			result.AlternativeTools = r.parseAlternatives(altStr)
		}
	}

	return result
}

// parseAlternatives parses alternative tools from string
func (r *ToolRecommender) parseAlternatives(altStr string) []AlternativeTool {
	var alternatives []AlternativeTool

	if altStr == "" || altStr == "none" {
		return alternatives
	}

	parts := strings.Split(altStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse format: "server/tool (confidence)"
		var serverTool string
		var confidence float64

		if idx := strings.Index(part, "("); idx != -1 {
			serverTool = strings.TrimSpace(part[:idx])
			confStr := strings.TrimSuffix(strings.TrimPrefix(part[idx:], "("), ")")
			fmt.Sscanf(confStr, "%f", &confidence)
		} else {
			serverTool = part
			confidence = 0.3 // Default confidence for alternatives
		}

		// Split server/tool
		serverToolParts := strings.Split(serverTool, "/")
		if len(serverToolParts) != 2 {
			continue
		}

		alt := AlternativeTool{
			Server:     serverToolParts[0],
			Tool:       serverToolParts[1],
			Confidence: confidence,
		}

		// Get tool description
		if doc, err := r.mcpManager.GetServerDocumentation(alt.Server); err == nil {
			if tool, found := doc.GetToolByName(alt.Tool); found {
				alt.Description = tool.Description
			}
		}

		alternatives = append(alternatives, alt)
	}

	return alternatives
}

// keywordBasedRecommendation provides a fallback recommendation based on keywords
func (r *ToolRecommender) keywordBasedRecommendation(userQuery string, serverSummaries map[string]string) *RecommendationResult {
	queryLower := strings.ToLower(userQuery)
	result := &RecommendationResult{
		Confidence: 0.6,
		Reasoning:  "Keyword-based recommendation",
	}

	// Enhanced keyword patterns with tool-specific mappings
	keywordPatterns := map[string]struct {
		Server string
		Tool   string
		Weight float64 // Weight for this pattern (higher = more specific)
	}{
		// Context7 patterns - documentation and library queries
		"documentation":   {"context7", "query-docs", 0.7},
		"library":         {"context7", "resolve-library-id", 0.7},
		"api docs":        {"context7", "query-docs", 0.8},
		"how to use":      {"context7", "query-docs", 0.8},
		"query docs":      {"context7", "query-docs", 0.9},
		"search docs":     {"context7", "query-docs", 0.9},
		"code example":    {"context7", "query-docs", 0.8},
		"code examples":   {"context7", "query-docs", 0.8},
		"latest example":  {"context7", "query-docs", 0.8},
		"latest examples": {"context7", "query-docs", 0.8},

		// Search patterns
		"search":  {"brave-search", "search", 0.6},
		"find":    {"brave-search", "search", 0.6},
		"look up": {"brave-search", "search", 0.7},
		"google":  {"brave-search", "search", 0.8},

		// Filesystem patterns
		"file":       {"filesystem", "read_file", 0.6},
		"read file":  {"filesystem", "read_file", 0.8},
		"list files": {"filesystem", "list_directory", 0.8},
		"directory":  {"filesystem", "list_directory", 0.7},

		// Terminal patterns
		"command": {"terminal", "execute_command", 0.7},
		"run":     {"terminal", "execute_command", 0.7},
		"execute": {"terminal", "execute_command", 0.7},
		"shell":   {"terminal", "execute_command", 0.8},

		// Playwright patterns - browser automation
		// Navigation and browsing
		"browser":      {"playwright", "browser_navigate", 0.7},
		"webpage":      {"playwright", "browser_navigate", 0.7},
		"website":      {"playwright", "browser_navigate", 0.7},
		"navigate":     {"playwright", "browser_navigate", 0.8},
		"open browser": {"playwright", "browser_navigate", 0.9},
		"open website": {"playwright", "browser_navigate", 0.9},
		"visit":        {"playwright", "browser_navigate", 0.8},
		"go to":        {"playwright", "browser_navigate", 0.8},

		// Interaction
		"click":         {"playwright", "browser_click", 0.9},
		"hover":         {"playwright", "browser_hover", 0.9},
		"drag":          {"playwright", "browser_drag", 0.9},
		"drop":          {"playwright", "browser_drag", 0.9},
		"drag and drop": {"playwright", "browser_drag", 1.0},

		// Screenshot and snapshot
		"screenshot":             {"playwright", "browser_take_screenshot", 0.9},
		"take screenshot":        {"playwright", "browser_take_screenshot", 1.0},
		"capture screenshot":     {"playwright", "browser_take_screenshot", 1.0},
		"snapshot":               {"playwright", "browser_snapshot", 0.9},
		"accessibility snapshot": {"playwright", "browser_snapshot", 1.0},

		// Form interaction
		"fill form":     {"playwright", "browser_fill_form", 1.0},
		"fill out form": {"playwright", "browser_fill_form", 1.0},
		"type":          {"playwright", "browser_type", 0.8},
		"type text":     {"playwright", "browser_type", 0.9},
		"enter text":    {"playwright", "browser_type", 0.9},
		"select option": {"playwright", "browser_select_option", 1.0},
		"choose option": {"playwright", "browser_select_option", 1.0},
		"dropdown":      {"playwright", "browser_select_option", 0.8},

		// File operations
		"upload file":  {"playwright", "browser_file_upload", 1.0},
		"upload files": {"playwright", "browser_file_upload", 1.0},

		// JavaScript and code execution
		"javascript":         {"playwright", "browser_evaluate", 0.8},
		"execute javascript": {"playwright", "browser_evaluate", 1.0},
		"run javascript":     {"playwright", "browser_evaluate", 1.0},
		"run code":           {"playwright", "browser_run_code", 1.0},
		"playwright code":    {"playwright", "browser_run_code", 1.0},

		// Debugging and monitoring
		"console messages": {"playwright", "browser_console_messages", 1.0},
		"console logs":     {"playwright", "browser_console_messages", 1.0},
		"network requests": {"playwright", "browser_network_requests", 1.0},
		"network traffic":  {"playwright", "browser_network_requests", 1.0},

		// Tab management
		"tab":        {"playwright", "browser_tabs", 0.7},
		"new tab":    {"playwright", "browser_tabs", 0.9},
		"close tab":  {"playwright", "browser_tabs", 0.9},
		"switch tab": {"playwright", "browser_tabs", 0.9},

		// Waiting
		"wait for":   {"playwright", "browser_wait_for", 0.9},
		"wait until": {"playwright", "browser_wait_for", 0.9},

		// Keyboard
		"press key": {"playwright", "browser_press_key", 0.9},
		"keyboard":  {"playwright", "browser_press_key", 0.7},

		// Window management
		"resize":        {"playwright", "browser_resize", 0.9},
		"resize window": {"playwright", "browser_resize", 1.0},
		"window size":   {"playwright", "browser_resize", 0.8},

		// Dialog handling
		"dialog":  {"playwright", "browser_handle_dialog", 0.8},
		"alert":   {"playwright", "browser_handle_dialog", 0.9},
		"confirm": {"playwright", "browser_handle_dialog", 0.9},

		// Navigation
		"go back": {"playwright", "browser_navigate_back", 0.9},
		"back":    {"playwright", "browser_navigate_back", 0.8},

		// General automation
		"scrape":     {"playwright", "browser_navigate", 0.8},
		"crawl":      {"playwright", "browser_navigate", 0.8},
		"automate":   {"playwright", "browser_navigate", 0.8},
		"playwright": {"playwright", "browser_navigate", 1.0},

		// Chinese keywords for playwright
		"浏览器":          {"playwright", "browser_navigate", 0.9},
		"网页":           {"playwright", "browser_navigate", 0.8},
		"网站":           {"playwright", "browser_navigate", 0.8},
		"打开浏览器":        {"playwright", "browser_navigate", 1.0},
		"打开网站":         {"playwright", "browser_navigate", 1.0},
		"点击":           {"playwright", "browser_click", 1.0},
		"悬停":           {"playwright", "browser_hover", 1.0},
		"截图":           {"playwright", "browser_take_screenshot", 1.0},
		"拖放":           {"playwright", "browser_drag", 1.0},
		"填写表单":         {"playwright", "browser_fill_form", 1.0},
		"输入文本":         {"playwright", "browser_type", 1.0},
		"选择选项":         {"playwright", "browser_select_option", 1.0},
		"上传文件":         {"playwright", "browser_file_upload", 1.0},
		"执行javascript": {"playwright", "browser_evaluate", 1.0},
		"运行代码":         {"playwright", "browser_run_code", 1.0},
		"控制台消息":        {"playwright", "browser_console_messages", 1.0},
		"网络请求":         {"playwright", "browser_network_requests", 1.0},
		"标签页":          {"playwright", "browser_tabs", 0.9},
		"等待":           {"playwright", "browser_wait_for", 1.0},
		"按键":           {"playwright", "browser_press_key", 1.0},
		"调整窗口大小":       {"playwright", "browser_resize", 1.0},
		"对话框":          {"playwright", "browser_handle_dialog", 1.0},
		"返回":           {"playwright", "browser_navigate_back", 1.0},
		"爬取":           {"playwright", "browser_navigate", 0.9},
		"爬虫":           {"playwright", "browser_navigate", 0.9},
		"自动化":          {"playwright", "browser_navigate", 0.9},

		// Chinese keywords for context7
		"查询文档":  {"context7", "query-docs", 1.0},
		"搜索文档":  {"context7", "query-docs", 1.0},
		"获取文档":  {"context7", "query-docs", 1.0},
		"文档查询":  {"context7", "query-docs", 1.0},
		"代码示例":  {"context7", "query-docs", 1.0},
		"最新示例":  {"context7", "query-docs", 1.0},
		"如何使用":  {"context7", "query-docs", 1.0},
		"api文档": {"context7", "query-docs", 1.0},
	}

	// Find all matching patterns
	var matches []struct {
		Keyword string
		Server  string
		Tool    string
		Weight  float64
	}

	for keyword, toolInfo := range keywordPatterns {
		if strings.Contains(queryLower, keyword) {
			matches = append(matches, struct {
				Keyword string
				Server  string
				Tool    string
				Weight  float64
			}{
				Keyword: keyword,
				Server:  toolInfo.Server,
				Tool:    toolInfo.Tool,
				Weight:  toolInfo.Weight,
			})
		}
	}

	// Select the best match (highest weight)
	if len(matches) > 0 {
		// Sort by weight (descending)
		sort.Slice(matches, func(i, j int) bool {
			if matches[i].Weight == matches[j].Weight {
				// If weights are equal, prefer longer keywords (more specific)
				return len(matches[i].Keyword) > len(matches[j].Keyword)
			}
			return matches[i].Weight > matches[j].Weight
		})

		bestMatch := matches[0]
		result.RecommendedServer = bestMatch.Server
		result.RecommendedTool = bestMatch.Server + "/" + bestMatch.Tool
		result.Confidence = bestMatch.Weight
		result.Reasoning = fmt.Sprintf("Matched keyword: '%s' (weight: %.2f)", bestMatch.Keyword, bestMatch.Weight)

		// Get server summary
		if summary, exists := serverSummaries[bestMatch.Server]; exists {
			result.ServerSummary = summary
		}

		// Get tool description
		if doc, err := r.mcpManager.GetServerDocumentation(bestMatch.Server); err == nil {
			if tool, found := doc.GetToolByName(bestMatch.Tool); found {
				result.ToolDescription = tool.Description
			}
		}
	} else {
		// If no match found, return empty result
		result.Reasoning = "No matching tool found for query"
		result.Confidence = 0.0
	}

	return result
}

// ClearCache clears the recommendation cache
func (r *ToolRecommender) ClearCache() {
	r.cacheLock.Lock()
	defer r.cacheLock.Unlock()

	r.cache = make(map[string]*RecommendationResult)
}

// GetToolDocumentation gets detailed documentation for a specific tool
func (r *ToolRecommender) GetToolDocumentation(serverName, toolName string) (*config.MCPTool, error) {
	doc, err := r.mcpManager.GetServerDocumentation(serverName)
	if err != nil {
		return nil, err
	}

	tool, found := doc.GetToolByName(toolName)
	if !found {
		return nil, fmt.Errorf("tool %s not found in server %s", toolName, serverName)
	}

	return tool, nil
}
