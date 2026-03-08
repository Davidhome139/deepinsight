package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"

	"backend/internal/services/agent"
)

// AgentHandler handles agent-related WebSocket connections and HTTP requests
type AgentHandler struct {
	orchestrator *agent.Orchestrator
	upgrader     websocket.Upgrader
	jwtSecret    string
	// Track active connections per user to prevent duplicates
	activeConnections map[uint]*websocket.Conn
	connMutex         sync.RWMutex
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(orchestrator *agent.Orchestrator, jwtSecret string) *AgentHandler {
	return &AgentHandler{
		orchestrator:      orchestrator,
		jwtSecret:         jwtSecret,
		activeConnections: make(map[uint]*websocket.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Configure properly for production
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections for real-time agent communication
func (h *AgentHandler) HandleWebSocket(c *gin.Context) {
	// Debug: log JWT secret prefix (for troubleshooting)
	secretPrefix := ""
	if len(h.jwtSecret) > 10 {
		secretPrefix = h.jwtSecret[:10]
	}
	fmt.Printf("[Agent WS] Using JWT secret prefix: %s...\n", secretPrefix)

	// Try to get userID from context (set by auth middleware)
	userID, exists := c.Get("userID")

	// If not in context, try to validate token from query string
	if !exists {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		// Parse and validate JWT token
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.jwtSecret), nil
		})
		if err != nil {
			fmt.Printf("[Agent WS] JWT parse error: %v\n", err)
			// Check if the error is due to token expiration
			if errors.Is(err, jwt.ErrTokenExpired) {
				// Return 401 with specific expiration error
				c.JSON(http.StatusUnauthorized, gin.H{"error": "token_expired", "details": "JWT token has expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "details": err.Error()})
			}
			return
		}
		if !parsedToken.Valid {
			fmt.Printf("[Agent WS] JWT token invalid\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Extract user ID from claims
		if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
			if uid, ok := claims["user_id"].(float64); ok {
				userID = uint(uid)
				exists = true
			}
		}
	}

	if !exists {
		fmt.Printf("[Agent WS] UserID not found in token claims\n")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fmt.Printf("[Agent WS] Authenticated user: %v\n", userID)

	// Store userID for this connection
	userIDUint, _ := userID.(uint)

	// Check if there's already an active connection for this user
	h.connMutex.RLock()
	if existingConn, exists := h.activeConnections[userIDUint]; exists {
		// Close existing connection
		h.connMutex.RUnlock()
		existingConn.Close()
		// Remove existing handler (will be done in the message loop of the old connection)
	} else {
		h.connMutex.RUnlock()
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("[Agent WS] WebSocket upgrade error: %v\n", err)
		return
	}
	defer conn.Close()

	// Add new connection to active connections
	h.connMutex.Lock()
	h.activeConnections[userIDUint] = conn
	h.connMutex.Unlock()

	// Generate unique task ID
	taskID := fmt.Sprintf("task-%d-%d", userID, time.Now().UnixNano())

	// Set userID in orchestrator for this connection
	h.orchestrator.SetCurrentUser(userIDUint)

	// Create a mutex for serializing WebSocket writes
	var writeMu sync.Mutex

	// Register event handler for this connection and capture for cleanup
	eventHandler := func(tid string, eventType string, data interface{}) {
		if tid != taskID {
			return
		}

		msg := map[string]interface{}{
			"type": eventType,
			"data": data,
		}

		// Lock to prevent concurrent writes
		writeMu.Lock()
		defer writeMu.Unlock()

		if err := conn.WriteJSON(msg); err != nil {
			// Connection closed or error
			return
		}
	}
	h.orchestrator.AddEventHandler(eventHandler)

	// Main message loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			// Remove handler when connection closes
			h.orchestrator.RemoveEventHandler(eventHandler)
			// Remove connection from active connections
			h.connMutex.Lock()
			delete(h.activeConnections, userIDUint)
			h.connMutex.Unlock()
			break
		}

		var msg struct {
			Type    string                 `json:"type"`
			Task    string                 `json:"task"`
			Model   string                 `json:"model"`
			Command string                 `json:"command"`
			Path    string                 `json:"path"`
			Content string                 `json:"content"`
			Context map[string]interface{} `json:"context"`
			Configs []agent.AgentConfig    `json:"configs"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			conn.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "Invalid message format",
			})
			continue
		}

		switch msg.Type {
		case "start_task":
			h.handleStartTask(conn, taskID, msg.Task, msg.Context)

		case "execute_command":
			h.handleExecuteCommand(conn, taskID, msg.Command)

		case "read_file":
			h.handleReadFile(conn, taskID, msg.Path)

		case "write_file":
			h.handleWriteFile(conn, taskID, msg.Path, msg.Content)

		case "refresh_files":
			h.handleRefreshFiles(conn, taskID)

		case "discover_tools":
			h.handleDiscoverTools(conn)

		case "refresh_tools":
			h.handleRefreshTools(conn)

		case "set_model":
			h.handleSetModel(conn, msg.Model)

		case "get_agent_configs":
			h.handleGetAgentConfigs(conn)

		case "save_agent_configs":
			h.handleSaveAgentConfigs(conn, msg.Configs)

		case "stop_task":
			h.handleStopTask(conn, taskID)
		}
	}
}

func (h *AgentHandler) handleStartTask(conn *websocket.Conn, taskID string, task string, context map[string]interface{}) {
	// Get context values
	osType := "linux"
	if osVal, ok := context["os"].(string); ok {
		osType = osVal
	}

	// Get current files
	files := make(map[string]string)
	if filesVal, ok := context["currentFiles"].([]interface{}); ok {
		for _, f := range filesVal {
			if fileMap, ok := f.(map[string]interface{}); ok {
				path, _ := fileMap["path"].(string)
				content, _ := fileMap["content"].(string)
				if path != "" {
					files[path] = content
				}
			}
		}
	}

	// Create working directory for task
	workingDir := filepath.Join("/tmp", "agent-tasks", taskID)
	os.MkdirAll(workingDir, 0755)

	// Start task
	_, err := h.orchestrator.StartTask(taskID, task, files, osType, workingDir)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
		})
		return
	}

	conn.WriteJSON(map[string]interface{}{
		"type":    "task_started",
		"taskId":  taskID,
		"message": "Task started successfully",
	})
}

func (h *AgentHandler) handleExecuteCommand(conn *websocket.Conn, taskID string, command string) {
	output, err := h.orchestrator.ExecuteCommand(taskID, command)

	response := map[string]interface{}{
		"type":   "terminal_output",
		"output": output,
	}

	if err != nil {
		response["error"] = err.Error()
	}

	conn.WriteJSON(response)
}

func (h *AgentHandler) handleReadFile(conn *websocket.Conn, taskID string, path string) {
	content, err := h.orchestrator.ReadFile(taskID, path)

	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
		})
		return
	}

	conn.WriteJSON(map[string]interface{}{
		"type":    "file_content",
		"path":    path,
		"content": content,
	})
}

func (h *AgentHandler) handleWriteFile(conn *websocket.Conn, taskID string, path string, content string) {
	err := h.orchestrator.WriteFile(taskID, path, content)

	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
		})
		return
	}

	conn.WriteJSON(map[string]interface{}{
		"type":    "file_written",
		"path":    path,
		"message": "File written successfully",
	})
}

func (h *AgentHandler) handleRefreshFiles(conn *websocket.Conn, taskID string) {
	files, err := h.orchestrator.GetFileTree(taskID)

	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
		})
		return
	}

	conn.WriteJSON(map[string]interface{}{
		"type":  "file_tree_update",
		"files": files,
	})
}

func (h *AgentHandler) handleDiscoverTools(conn *websocket.Conn) {
	// Trigger discovery (cached after first run)
	h.orchestrator.DiscoverTools()

	// Wait a bit for discovery to complete (external MCPs may take time)
	time.Sleep(500 * time.Millisecond)

	// Send discovered tools to client
	h.sendDiscoveredTools(conn)
}

func (h *AgentHandler) handleRefreshTools(conn *websocket.Conn) {
	// Force refresh discovery
	h.orchestrator.RefreshTools()

	// Wait a bit for discovery to complete
	time.Sleep(500 * time.Millisecond)

	// Send refreshed tools to client
	h.sendDiscoveredTools(conn)

	conn.WriteJSON(map[string]interface{}{
		"type":    "tools_refreshed",
		"message": "Tools refreshed successfully",
	})
}

func (h *AgentHandler) sendDiscoveredTools(conn *websocket.Conn) {
	// Get discovered MCPs
	mcps := h.orchestrator.GetMCPManager().ListConnected()
	fmt.Printf("[Agent WS] Sending %d MCPs to client\n", len(mcps))
	conn.WriteJSON(map[string]interface{}{
		"type": "mcp_discovered",
		"mcps": mcps,
	})

	// Get available skills
	skills := h.orchestrator.GetSkillRegistry().List()
	skillList := make([]map[string]string, len(skills))
	for i, skill := range skills {
		skillList[i] = map[string]string{
			"name":        skill.Name(),
			"description": skill.Description(),
		}
	}
	fmt.Printf("[Agent WS] Sending %d skills to client\n", len(skills))
	conn.WriteJSON(map[string]interface{}{
		"type":   "skills_discovered",
		"skills": skillList,
	})
}

func (h *AgentHandler) handleSetModel(conn *websocket.Conn, model string) {
	if model == "" {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": "Model name is required",
		})
		return
	}

	// Set the model in the orchestrator
	h.orchestrator.SetModel(model)

	fmt.Printf("[Agent WS] Model changed to: %s\n", model)
	conn.WriteJSON(map[string]interface{}{
		"type":    "model_changed",
		"model":   model,
		"message": "Model changed successfully",
	})
}

func (h *AgentHandler) handleGetAgentConfigs(conn *websocket.Conn) {
	configs := h.orchestrator.GetAgentConfigs()
	conn.WriteJSON(map[string]interface{}{
		"type":    "agent_configs",
		"configs": configs,
	})
}

func (h *AgentHandler) handleSaveAgentConfigs(conn *websocket.Conn, configs []agent.AgentConfig) {
	for _, config := range configs {
		h.orchestrator.UpdateAgentConfig(config.Name, config.Prompt)
	}

	conn.WriteJSON(map[string]interface{}{
		"type":    "agent_configs_saved",
		"message": "Agent configurations saved successfully",
	})
}

func (h *AgentHandler) handleStopTask(conn *websocket.Conn, taskID string) {
	err := h.orchestrator.StopTask(taskID)

	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
		})
		return
	}

	conn.WriteJSON(map[string]interface{}{
		"type":    "task_stopped",
		"message": "Task stopped successfully",
	})
}

// HTTP Handlers

// ListAgents returns all available agents
func (h *AgentHandler) ListAgents(c *gin.Context) {
	agents := h.orchestrator.ListAgents()

	response := make([]map[string]string, len(agents))
	for i, agent := range agents {
		response[i] = map[string]string{
			"name":        agent.Name(),
			"role":        agent.Role(),
			"description": agent.Description(),
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetAgent returns a specific agent's details
func (h *AgentHandler) GetAgent(c *gin.Context) {
	name := c.Param("name")

	agent, ok := h.orchestrator.GetAgent(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"name":        agent.Name(),
		"role":        agent.Role(),
		"description": agent.Description(),
		"prompt":      agent.GetPrompt(),
	})
}

// UpdateAgent updates an agent's configuration
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		Prompt string `json:"prompt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.orchestrator.UpdateAgentConfig(name, req.Prompt); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agent updated successfully"})
}

// ListMCPTools returns available MCP tools
func (h *AgentHandler) ListMCPTools(c *gin.Context) {
	mcps := h.orchestrator.GetMCPManager().ListConnected()

	response := make([]map[string]interface{}, len(mcps))
	for i, mcp := range mcps {
		tools := make([]string, len(mcp.Tools))
		for j, tool := range mcp.Tools {
			tools[j] = tool.Name
		}
		response[i] = map[string]interface{}{
			"name":      mcp.Name,
			"connected": mcp.Connected,
			"tools":     tools,
		}
	}

	c.JSON(http.StatusOK, response)
}

// ListSkills returns available skills
func (h *AgentHandler) ListSkills(c *gin.Context) {
	skills := h.orchestrator.GetSkillRegistry().List()

	response := make([]map[string]string, len(skills))
	for i, skill := range skills {
		response[i] = map[string]string{
			"name":        skill.Name(),
			"description": skill.Description(),
		}
	}

	c.JSON(http.StatusOK, response)
}

// ExecuteSkill executes a skill
func (h *AgentHandler) ExecuteSkill(c *gin.Context) {
	name := c.Param("name")

	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	skill, ok := h.orchestrator.GetSkillRegistry().Get(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
		return
	}

	result, err := skill.Execute(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// =====================================================
// Enhanced Endpoints (Priority 1-3 Features)
// =====================================================

// GetCodebaseContext returns relevant codebase context for a task
func (h *AgentHandler) GetCodebaseContext(c *gin.Context) {
	task := c.Query("task")
	maxFiles := 10 // Default

	if maxFilesStr := c.Query("max_files"); maxFilesStr != "" {
		fmt.Sscanf(maxFilesStr, "%d", &maxFiles)
	}

	ctx := h.orchestrator.GetCodebaseContext(task, maxFiles)
	if ctx == nil {
		c.JSON(http.StatusOK, gin.H{"context": nil, "message": "Codebase not indexed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"context": ctx})
}

// GetPendingApprovals returns all pending approval requests
func (h *AgentHandler) GetPendingApprovals(c *gin.Context) {
	approvals := h.orchestrator.GetPendingApprovals()
	c.JSON(http.StatusOK, gin.H{"approvals": approvals})
}

// SubmitApproval handles approval/rejection of a file write
func (h *AgentHandler) SubmitApproval(c *gin.Context) {
	var response agent.ApprovalResponse
	if err := c.ShouldBindJSON(&response); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.orchestrator.SubmitApproval(response); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SetApprovalRequired enables or disables approval requirements
func (h *AgentHandler) SetApprovalRequired(c *gin.Context) {
	var req struct {
		Required bool `json:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.orchestrator.SetApprovalRequired(req.Required)
	c.JSON(http.StatusOK, gin.H{"approval_required": req.Required})
}

// GetProposedChanges returns all proposed file changes
func (h *AgentHandler) GetProposedChanges(c *gin.Context) {
	changes := h.orchestrator.GetProposedChanges()
	summary := h.orchestrator.GetChangesSummary()

	c.JSON(http.StatusOK, gin.H{
		"changes": changes,
		"summary": summary,
	})
}

// GetUnifiedDiff returns a unified diff of all proposed changes
func (h *AgentHandler) GetUnifiedDiff(c *gin.Context) {
	diff := h.orchestrator.GetUnifiedDiff()
	c.JSON(http.StatusOK, gin.H{"diff": diff})
}

// ApproveAllChanges approves all pending changes
func (h *AgentHandler) ApproveAllChanges(c *gin.Context) {
	h.orchestrator.ApproveAllChanges()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetGitStatus returns the current Git status
func (h *AgentHandler) GetGitStatus(c *gin.Context) {
	status, err := h.orchestrator.GetGitStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}

// CreateTaskBranch creates a Git branch for a task
func (h *AgentHandler) CreateTaskBranch(c *gin.Context) {
	var req struct {
		TaskID      string `json:"task_id"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.orchestrator.CreateTaskBranch(req.TaskID, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CommitChanges commits all staged changes
func (h *AgentHandler) CommitChanges(c *gin.Context) {
	var req struct {
		TaskID  string `json:"task_id"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	commit, err := h.orchestrator.CommitChanges(req.TaskID, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"commit": commit})
}

// PreparePullRequest prepares PR information
func (h *AgentHandler) PreparePullRequest(c *gin.Context) {
	var req struct {
		TaskID      string `json:"task_id"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pr, err := h.orchestrator.PreparePullRequest(req.TaskID, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pull_request": pr})
}

// CheckSyntax checks syntax for a file
func (h *AgentHandler) CheckSyntax(c *gin.Context) {
	filePath := c.Query("file")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file path required"})
		return
	}

	diagnostics, err := h.orchestrator.CheckSyntax(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"diagnostics": diagnostics})
}

// CheckAllSyntax checks syntax for all files
func (h *AgentHandler) CheckAllSyntax(c *gin.Context) {
	results, err := h.orchestrator.CheckAllSyntax()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// ReindexCodebase triggers codebase re-indexing
func (h *AgentHandler) ReindexCodebase(c *gin.Context) {
	if err := h.orchestrator.ReindexCodebase(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Codebase reindexed"})
}

// GetAffectedFiles returns files affected by a symbol change
func (h *AgentHandler) GetAffectedFiles(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol name required"})
		return
	}

	files := h.orchestrator.GetAffectedFiles(symbol)
	c.JSON(http.StatusOK, gin.H{"affected_files": files})
}
