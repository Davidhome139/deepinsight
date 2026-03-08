package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"backend/internal/config"
	"backend/internal/services/agent"
	"backend/internal/services/chat"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService chat.ChatService
	mcpManager  *agent.MCPManager
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewChatHandler(chatService chat.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// SetMCPManager sets the MCP manager for accessing external MCP tools
func (h *ChatHandler) SetMCPManager(mcpManager *agent.MCPManager) {
	h.mcpManager = mcpManager
}

// CreateConversation godoc
// @Summary Create a new conversation
// @Description Create a new conversation for the authenticated user
// @Tags chat
// @Accept json
// @Produce json
// @Param request body CreateConversationRequest true "Conversation creation request"
// @Success 201 {object} models.Conversation
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /chat/conversations [post]
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	conv, err := h.chatService.CreateConversation(userID.(uint), req.Title, req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusCreated, conv)
}

// GetConversations godoc
// @Summary Get all conversations for the current user
// @Description Get all conversations for the authenticated user
// @Tags chat
// @Produce json
// @Success 200 {array} models.Conversation
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /chat/conversations [get]
func (h *ChatHandler) GetConversations(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	conversations, err := h.chatService.GetConversations(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get conversations"})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

// GetMessages godoc
// @Summary Get all messages in a conversation
// @Description Get all messages in a specific conversation
// @Tags chat
// @Produce json
// @Param id path int true "Conversation ID"
// @Success 200 {array} models.Message
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /chat/conversations/{id}/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	convIDStr := c.Param("id")
	if convIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Conversation ID is required"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Check if conversation belongs to user
	// (Note: This check should be added in a middleware or in the service layer)

	convID, _ := strconv.ParseUint(convIDStr, 10, 32)
	messages, err := h.chatService.GetMessages(userID.(uint), uint(convID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// StreamChat godoc
// @Summary Stream chat response
// @Description Get a streaming chat response
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Param request body StreamChatRequest true "Chat request"
// @Success 200 {string} string "Stream response"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /chat/stream [post]
func (h *ChatHandler) StreamChat(c *gin.Context) {
	var req StreamChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Create a context that cancels when the connection closes
	ctx, cancelFunc := context.WithCancel(c.Request.Context())
	closeNotify := c.Writer.CloseNotify()
	go func() {
		<-closeNotify
		cancelFunc()
	}()

	// Call chat service to get stream
	chatCh, err := h.chatService.SendMessageStream(
		ctx,
		userID.(uint),
		req.ConversationID,
		req.Content,
		req.Model,
		req.WebSearch,
		req.SearchProvider,
		req.MCPTool,
		req.SystemPrompt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get chat response"})
		cancelFunc()
		return
	}

	// Stream response
	for chunk := range chatCh {
		jsonChunk, err := json.Marshal(chunk)
		if err != nil {
			continue
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", jsonChunk)
		c.Writer.Flush()
	}

	cancelFunc()
}

// StreamChatWithRAG godoc
// @Summary Stream chat response with RAG (knowledge base) support
// @Description Get a streaming chat response with optional RAG context
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Param request body StreamChatWithRAGRequest true "Chat request with RAG"
// @Success 200 {string} string "Stream response"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /chat/stream/rag [post]
func (h *ChatHandler) StreamChatWithRAG(c *gin.Context) {
	var req StreamChatWithRAGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx, cancelFunc := context.WithCancel(c.Request.Context())
	closeNotify := c.Writer.CloseNotify()
	go func() {
		<-closeNotify
		cancelFunc()
	}()

	// Call chat service with RAG
	chatCh, err := h.chatService.SendMessageStreamWithRAG(
		ctx,
		userID.(uint),
		req.ConversationID,
		req.Content,
		req.Model,
		req.WebSearch,
		req.SearchProvider,
		req.MCPTool,
		req.SystemPrompt,
		req.RAGEnabled,
		req.RAGDocumentIDs,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get chat response"})
		cancelFunc()
		return
	}

	// Stream response
	for chunk := range chatCh {
		jsonChunk, err := json.Marshal(chunk)
		if err != nil {
			continue
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", jsonChunk)
		c.Writer.Flush()
	}

	cancelFunc()
}

// GenerateConversationSummary godoc
// @Summary Generate a summary for a conversation
// @Description Generate a summary for a specific conversation
// @Tags chat
// @Accept json
// @Produce json
// @Param id path int true "Conversation ID"
// @Param request body GenerateSummaryRequest true "Summary generation request"
// @Success 200 {object} SummaryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /chat/conversations/{id}/summary [post]
func (h *ChatHandler) GenerateConversationSummary(c *gin.Context) {
	convIDStr := c.Param("id")
	if convIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Conversation ID is required"})
		return
	}

	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Check if conversation belongs to user
	// (Note: This check should be added in a middleware or in the service layer)

	convID, _ := strconv.ParseUint(convIDStr, 10, 32)
	summary, err := h.chatService.GenerateConversationSummary(userID.(uint), uint(convID), req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate summary"})
		return
	}

	c.JSON(http.StatusOK, SummaryResponse{Summary: summary})
}

// GetChatModels godoc
// @Summary Get available chat models
// @Description Get list of available AI models from configuration
// @Tags chat
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /chat/models [get]
func (h *ChatHandler) GetChatModels(c *gin.Context) {
	models := config.GetModelsConfig()
	if models == nil {
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}

	var modelList []map[string]interface{}
	for key, provider := range models.Providers {
		// Skip disabled providers
		if !provider.Enabled {
			continue
		}
		for _, model := range provider.Models {
			modelList = append(modelList, map[string]interface{}{
				"id":       model,
				"name":     model,
				"provider": key,
			})
		}
	}

	// Sort by provider name, then by model name
	sort.Slice(modelList, func(i, j int) bool {
		if modelList[i]["provider"].(string) != modelList[j]["provider"].(string) {
			return modelList[i]["provider"].(string) < modelList[j]["provider"].(string)
		}
		return modelList[i]["name"].(string) < modelList[j]["name"].(string)
	})

	c.JSON(http.StatusOK, modelList)
}

// GetSearchProviders godoc
// @Summary Get available search providers
// @Description Get list of available search providers from configuration
// @Tags chat
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /chat/search-providers [get]
func (h *ChatHandler) GetSearchProviders(c *gin.Context) {
	searchs := config.GetSearchsConfig()
	if searchs == nil {
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}

	var providerList []map[string]interface{}
	for key, provider := range searchs.Providers {
		if !provider.Enabled {
			continue
		}
		providerList = append(providerList, map[string]interface{}{
			"id":   key,
			"name": provider.Name,
		})
	}

	c.JSON(http.StatusOK, providerList)
}

// GetMCPTools godoc
// @Summary Get available MCP tools
// @Description Get list of available MCP tools from connected MCP servers
// @Tags chat
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /chat/mcp-tools [get]
func (h *ChatHandler) GetMCPTools(c *gin.Context) {
	var toolList []map[string]interface{}

	// Get tools from MCPManager (for external MCP servers)
	if h.mcpManager != nil {
		// Get all connected MCP servers from the manager
		servers := h.mcpManager.ListConnected()
		for _, server := range servers {
			// Get tools from each server
			for _, tool := range server.Tools {
				toolList = append(toolList, map[string]interface{}{
					"id":          fmt.Sprintf("%s/%s", server.Name, tool.Name),
					"name":        fmt.Sprintf("[%s] %s", server.Name, tool.Name),
					"server":      server.Name,
					"tool":        tool.Name,
					"description": tool.Description,
				})
			}
		}
	}

	// Get tools from config (for both built-in and external servers)
	mcps := config.GetMCPServersConfig()
	if mcps == nil || mcps.Servers == nil {
		// Return whatever we got from MCPManager
		if len(toolList) == 0 {
			c.JSON(http.StatusOK, []map[string]interface{}{})
			return
		}
		c.JSON(http.StatusOK, toolList)
		return
	}

	// Track which servers we've already added from MCPManager
	addedServers := make(map[string]bool)
	for _, tool := range toolList {
		if server, ok := tool["server"].(string); ok {
			addedServers[server] = true
		}
	}

	// Iterate through all enabled MCP servers from config
	for name, server := range mcps.Servers {
		if !server.Enabled {
			continue
		}

		// Skip if already added from MCPManager
		if addedServers[name] {
			continue
		}

		// For built-in servers, add predefined tools
		if server.Type == "builtin" {
			switch name {
			case "filesystem-local":
				toolList = append(toolList, map[string]interface{}{
					"id":          fmt.Sprintf("%s/read_file", name),
					"name":        fmt.Sprintf("[%s] Read File", server.Name),
					"server":      name,
					"tool":        "read_file",
					"description": "Read file contents",
				})
				toolList = append(toolList, map[string]interface{}{
					"id":          fmt.Sprintf("%s/list_directory", name),
					"name":        fmt.Sprintf("[%s] List Directory", server.Name),
					"server":      name,
					"tool":        "list_directory",
					"description": "List directory contents",
				})
			case "terminal":
				toolList = append(toolList, map[string]interface{}{
					"id":          fmt.Sprintf("%s/execute_command", name),
					"name":        fmt.Sprintf("[%s] Execute Command", server.Name),
					"server":      name,
					"tool":        "execute_command",
					"description": "Execute shell command",
				})
			case "search":
				toolList = append(toolList, map[string]interface{}{
					"id":          fmt.Sprintf("%s/web_search", name),
					"name":        fmt.Sprintf("[%s] Web Search", server.Name),
					"server":      name,
					"tool":        "web_search",
					"description": "Search the web",
				})
			case "code-analysis":
				toolList = append(toolList, map[string]interface{}{
					"id":          fmt.Sprintf("%s/analyze_code", name),
					"name":        fmt.Sprintf("[%s] Analyze Code", server.Name),
					"server":      name,
					"tool":        "analyze_code",
					"description": "Analyze code for issues",
				})
				toolList = append(toolList, map[string]interface{}{
					"id":          fmt.Sprintf("%s/suggest_improvements", name),
					"name":        fmt.Sprintf("[%s] Suggest Improvements", server.Name),
					"server":      name,
					"tool":        "suggest_improvements",
					"description": "Suggest code improvements",
				})
			}
		} else {
			// For external MCP servers not yet connected via MCPManager,
			// add a generic entry indicating the server is available
			toolList = append(toolList, map[string]interface{}{
				"id":          fmt.Sprintf("%s/call_tool", name),
				"name":        fmt.Sprintf("[%s] %s", server.Name, name),
				"server":      name,
				"tool":        "call_tool",
				"description": fmt.Sprintf("Tools from %s MCP server (connect to view tools)", server.Name),
			})
		}
	}

	c.JSON(http.StatusOK, toolList)
}

type CreateConversationRequest struct {
	Title string `json:"title" binding:"required"`
	Model string `json:"model" binding:"required"`
}

type StreamChatRequest struct {
	ConversationID uint   `json:"conversation_id" binding:"required"`
	Content        string `json:"content" binding:"required"`
	Model          string `json:"model"`
	WebSearch      bool   `json:"web_search"`
	SearchProvider string `json:"search_provider"`
	MCPTool        string `json:"mcp_tool"`
	SystemPrompt   string `json:"system_prompt"`
}

// StreamChatWithRAGRequest includes RAG (knowledge base) parameters
type StreamChatWithRAGRequest struct {
	ConversationID uint     `json:"conversation_id" binding:"required"`
	Content        string   `json:"content" binding:"required"`
	Model          string   `json:"model"`
	WebSearch      bool     `json:"web_search"`
	SearchProvider string   `json:"search_provider"`
	MCPTool        string   `json:"mcp_tool"`
	SystemPrompt   string   `json:"system_prompt"`
	RAGEnabled     bool     `json:"rag_enabled"`
	RAGDocumentIDs []string `json:"rag_document_ids"`
}

type GenerateSummaryRequest struct {
	Model string `json:"model"`
}

type SummaryResponse struct {
	Summary string `json:"summary"`
}
