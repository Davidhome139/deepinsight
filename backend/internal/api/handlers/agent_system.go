package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/internal/models"
	"backend/internal/services/agentsystem"

	"github.com/gin-gonic/gin"
)

// AgentSystemHandler handles agent system API requests
type AgentSystemHandler struct {
	agentService       *agentsystem.CustomAgentService
	workflowEngine     *agentsystem.WorkflowEngine
	permissionService  *agentsystem.PermissionService
	marketplaceService *agentsystem.MarketplaceService
	abTestService      *agentsystem.ABTestService
}

// NewAgentSystemHandler creates a new agent system handler
func NewAgentSystemHandler(
	agentService *agentsystem.CustomAgentService,
	workflowEngine *agentsystem.WorkflowEngine,
	permissionService *agentsystem.PermissionService,
	marketplaceService *agentsystem.MarketplaceService,
	abTestService *agentsystem.ABTestService,
) *AgentSystemHandler {
	return &AgentSystemHandler{
		agentService:       agentService,
		workflowEngine:     workflowEngine,
		permissionService:  permissionService,
		marketplaceService: marketplaceService,
		abTestService:      abTestService,
	}
}

// ==================== Agent Endpoints ====================

// GetAgentTemplates returns predefined agent templates
func (h *AgentSystemHandler) GetAgentTemplates(c *gin.Context) {
	templates := h.agentService.GetTemplates()
	c.JSON(http.StatusOK, templates)
}

// CreateAgent creates a new custom agent
func (h *AgentSystemHandler) CreateAgent(c *gin.Context) {
	userID := c.GetUint("userID")

	var agent models.CustomAgent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.agentService.CreateAgent(c.Request.Context(), userID, &agent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetAgent retrieves an agent by ID
func (h *AgentSystemHandler) GetAgent(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")

	agent, err := h.agentService.GetAgent(c.Request.Context(), agentID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// ListAgents returns all agents for the user
func (h *AgentSystemHandler) ListAgents(c *gin.Context) {
	userID := c.GetUint("userID")
	includePublic := c.Query("include_public") == "true"

	agents, err := h.agentService.ListAgents(c.Request.Context(), userID, includePublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// UpdateAgent updates an agent
func (h *AgentSystemHandler) UpdateAgent(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := h.agentService.UpdateAgent(c.Request.Context(), agentID, userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// DeleteAgent deletes an agent
func (h *AgentSystemHandler) DeleteAgent(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")

	if err := h.agentService.DeleteAgent(c.Request.Context(), agentID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agent deleted"})
}

// DuplicateAgent creates a copy of an agent
func (h *AgentSystemHandler) DuplicateAgent(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := h.agentService.DuplicateAgent(c.Request.Context(), agentID, userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, agent)
}

// ExecuteAgent runs an agent
func (h *AgentSystemHandler) ExecuteAgent(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")

	var req struct {
		Input string `json:"input"`
		Model string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	execution, err := h.agentService.ExecuteAgent(c.Request.Context(), agentID, userID, req.Input, req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// GetAgentExecutionHistory returns execution history
func (h *AgentSystemHandler) GetAgentExecutionHistory(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	executions, err := h.agentService.GetExecutionHistory(c.Request.Context(), agentID, userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// ProvideFeedback records feedback for an execution
func (h *AgentSystemHandler) ProvideFeedback(c *gin.Context) {
	userID := c.GetUint("userID")
	executionID := c.Param("executionId")

	var req struct {
		Rating int    `json:"rating"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.agentService.ProvideFeedback(c.Request.Context(), executionID, userID, req.Rating, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feedback recorded"})
}

// ExportAgent exports agent configuration
func (h *AgentSystemHandler) ExportAgent(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")

	export, err := h.agentService.ExportAgent(c.Request.Context(), agentID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, export)
}

// ImportAgent imports an agent from configuration
func (h *AgentSystemHandler) ImportAgent(c *gin.Context) {
	userID := c.GetUint("userID")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := h.agentService.ImportAgent(c.Request.Context(), userID, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, agent)
}

// ==================== Workflow Endpoints ====================

// CreateWorkflow creates a new workflow
func (h *AgentSystemHandler) CreateWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")

	var workflow models.Workflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.workflowEngine.CreateWorkflow(c.Request.Context(), userID, &workflow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetWorkflow retrieves a workflow
func (h *AgentSystemHandler) GetWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	workflow, err := h.workflowEngine.GetWorkflow(c.Request.Context(), workflowID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// ListWorkflows returns all workflows
func (h *AgentSystemHandler) ListWorkflows(c *gin.Context) {
	userID := c.GetUint("userID")
	includePublic := c.Query("include_public") == "true"

	workflows, err := h.workflowEngine.ListWorkflows(c.Request.Context(), userID, includePublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflows)
}

// UpdateWorkflow updates a workflow
func (h *AgentSystemHandler) UpdateWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := h.workflowEngine.UpdateWorkflow(c.Request.Context(), workflowID, userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// DeleteWorkflow deletes a workflow
func (h *AgentSystemHandler) DeleteWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	if err := h.workflowEngine.DeleteWorkflow(c.Request.Context(), workflowID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow deleted"})
}

// AddWorkflowStep adds a step to a workflow
func (h *AgentSystemHandler) AddWorkflowStep(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	var step models.WorkflowStep
	if err := c.ShouldBindJSON(&step); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.workflowEngine.AddStep(c.Request.Context(), workflowID, userID, &step)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UpdateWorkflowStep updates a workflow step
func (h *AgentSystemHandler) UpdateWorkflowStep(c *gin.Context) {
	userID := c.GetUint("userID")
	stepID := c.Param("stepId")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	step, err := h.workflowEngine.UpdateStep(c.Request.Context(), stepID, userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, step)
}

// DeleteWorkflowStep deletes a workflow step
func (h *AgentSystemHandler) DeleteWorkflowStep(c *gin.Context) {
	userID := c.GetUint("userID")
	stepID := c.Param("stepId")

	if err := h.workflowEngine.DeleteStep(c.Request.Context(), stepID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "step deleted"})
}

// AddWorkflowEdge adds an edge between steps
func (h *AgentSystemHandler) AddWorkflowEdge(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	var edge models.WorkflowEdge
	if err := c.ShouldBindJSON(&edge); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.workflowEngine.AddEdge(c.Request.Context(), workflowID, userID, &edge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// DeleteWorkflowEdge deletes an edge
func (h *AgentSystemHandler) DeleteWorkflowEdge(c *gin.Context) {
	userID := c.GetUint("userID")
	edgeID := c.Param("edgeId")

	if err := h.workflowEngine.DeleteEdge(c.Request.Context(), edgeID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "edge deleted"})
}

// StartWorkflow starts executing a workflow
func (h *AgentSystemHandler) StartWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	var req struct {
		Input map[string]interface{} `json:"input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run, eventChan, err := h.workflowEngine.StartWorkflow(c.Request.Context(), workflowID, userID, req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// For non-streaming response, just return the run
	// Events would be consumed via WebSocket in production
	go func() {
		for range eventChan {
			// Consume events
		}
	}()

	c.JSON(http.StatusOK, run)
}

// GetWorkflowRun retrieves a workflow run
func (h *AgentSystemHandler) GetWorkflowRun(c *gin.Context) {
	userID := c.GetUint("userID")
	runID := c.Param("runId")

	run, err := h.workflowEngine.GetWorkflowRun(c.Request.Context(), runID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, run)
}

// GetWorkflowRunHistory returns run history
func (h *AgentSystemHandler) GetWorkflowRunHistory(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, err := h.workflowEngine.GetWorkflowRunHistory(c.Request.Context(), workflowID, userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runs)
}

// ResumeWorkflow resumes a paused workflow
func (h *AgentSystemHandler) ResumeWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")
	runID := c.Param("runId")

	var req struct {
		Approved bool `json:"approved"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.workflowEngine.ResumeWorkflow(c.Request.Context(), runID, userID, req.Approved); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow resumed"})
}

// CancelWorkflow cancels a running workflow
func (h *AgentSystemHandler) CancelWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")
	runID := c.Param("runId")

	if err := h.workflowEngine.CancelWorkflow(c.Request.Context(), runID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow cancelled"})
}

// ExportWorkflow exports workflow configuration
func (h *AgentSystemHandler) ExportWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	export, err := h.workflowEngine.ExportWorkflow(c.Request.Context(), workflowID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, export)
}

// ImportWorkflow imports a workflow
func (h *AgentSystemHandler) ImportWorkflow(c *gin.Context) {
	userID := c.GetUint("userID")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := h.workflowEngine.ImportWorkflow(c.Request.Context(), userID, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, workflow)
}

// ==================== Permission Endpoints ====================

// CreatePermission creates a new permission rule
func (h *AgentSystemHandler) CreatePermission(c *gin.Context) {
	userID := c.GetUint("userID")

	var perm models.ToolPermission
	if err := c.ShouldBindJSON(&perm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.permissionService.CreatePermission(c.Request.Context(), userID, &perm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetPermission retrieves a permission
func (h *AgentSystemHandler) GetPermission(c *gin.Context) {
	userID := c.GetUint("userID")
	permID := c.Param("id")

	perm, err := h.permissionService.GetPermission(c.Request.Context(), permID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perm)
}

// ListPermissions returns all permissions
func (h *AgentSystemHandler) ListPermissions(c *gin.Context) {
	userID := c.GetUint("userID")

	perms, err := h.permissionService.ListPermissions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perms)
}

// UpdatePermission updates a permission
func (h *AgentSystemHandler) UpdatePermission(c *gin.Context) {
	userID := c.GetUint("userID")
	permID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	perm, err := h.permissionService.UpdatePermission(c.Request.Context(), permID, userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, perm)
}

// DeletePermission deletes a permission
func (h *AgentSystemHandler) DeletePermission(c *gin.Context) {
	userID := c.GetUint("userID")
	permID := c.Param("id")

	if err := h.permissionService.DeletePermission(c.Request.Context(), permID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "permission deleted"})
}

// CheckPermission checks if a tool invocation is allowed
func (h *AgentSystemHandler) CheckPermission(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		ToolName   string                 `json:"tool_name"`
		Args       map[string]interface{} `json:"args"`
		AgentID    *string                `json:"agent_id"`
		WorkflowID *string                `json:"workflow_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.permissionService.CheckPermission(c.Request.Context(), userID, req.ToolName, req.Args, req.AgentID, req.WorkflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetInvocationLogs returns tool invocation logs
func (h *AgentSystemHandler) GetInvocationLogs(c *gin.Context) {
	userID := c.GetUint("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	filter := agentsystem.InvocationLogFilter{
		AgentID:    c.Query("agent_id"),
		WorkflowID: c.Query("workflow_id"),
		ToolName:   c.Query("tool_name"),
		Status:     c.Query("status"),
		Limit:      limit,
	}

	logs, err := h.permissionService.GetInvocationLogs(c.Request.Context(), userID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetUsageStats returns usage statistics
func (h *AgentSystemHandler) GetUsageStats(c *gin.Context) {
	userID := c.GetUint("userID")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	stats, err := h.permissionService.GetUsageStats(c.Request.Context(), userID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateDefaultPermissions creates default permissions for user
func (h *AgentSystemHandler) CreateDefaultPermissions(c *gin.Context) {
	userID := c.GetUint("userID")

	if err := h.permissionService.CreateDefaultPermissions(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "default permissions created"})
}

// ==================== Marketplace Endpoints ====================

// SearchMarketplace searches the marketplace
func (h *AgentSystemHandler) SearchMarketplace(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	minRating, _ := strconv.ParseFloat(c.DefaultQuery("min_rating", "0"), 64)

	var tags []string
	if tagsParam := c.Query("tags"); tagsParam != "" {
		json.Unmarshal([]byte(tagsParam), &tags)
	}

	filter := agentsystem.MarketplaceFilter{
		Query:     c.Query("q"),
		Type:      c.Query("type"),
		Category:  c.Query("category"),
		Tags:      tags,
		MinRating: minRating,
		SortBy:    c.DefaultQuery("sort", "downloads"),
		Offset:    offset,
		Limit:     limit,
	}

	result, err := h.marketplaceService.SearchItems(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetMarketplaceItem retrieves a marketplace item
func (h *AgentSystemHandler) GetMarketplaceItem(c *gin.Context) {
	itemID := c.Param("id")

	item, err := h.marketplaceService.GetItem(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DownloadMarketplaceItem downloads and imports an item
func (h *AgentSystemHandler) DownloadMarketplaceItem(c *gin.Context) {
	userID := c.GetUint("userID")
	itemID := c.Param("id")

	result, err := h.marketplaceService.DownloadItem(c.Request.Context(), itemID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// PublishAgentToMarketplace publishes an agent
func (h *AgentSystemHandler) PublishAgentToMarketplace(c *gin.Context) {
	userID := c.GetUint("userID")
	agentID := c.Param("id")

	var listing agentsystem.MarketplaceListing
	if err := c.ShouldBindJSON(&listing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.marketplaceService.PublishAgent(c.Request.Context(), userID, agentID, &listing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// PublishWorkflowToMarketplace publishes a workflow
func (h *AgentSystemHandler) PublishWorkflowToMarketplace(c *gin.Context) {
	userID := c.GetUint("userID")
	workflowID := c.Param("id")

	var listing agentsystem.MarketplaceListing
	if err := c.ShouldBindJSON(&listing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.marketplaceService.PublishWorkflow(c.Request.Context(), userID, workflowID, &listing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// StarMarketplaceItem adds a star
func (h *AgentSystemHandler) StarMarketplaceItem(c *gin.Context) {
	userID := c.GetUint("userID")
	itemID := c.Param("id")

	if err := h.marketplaceService.StarItem(c.Request.Context(), itemID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "starred"})
}

// UnstarMarketplaceItem removes a star
func (h *AgentSystemHandler) UnstarMarketplaceItem(c *gin.Context) {
	userID := c.GetUint("userID")
	itemID := c.Param("id")

	if err := h.marketplaceService.UnstarItem(c.Request.Context(), itemID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unstarred"})
}

// AddMarketplaceReview adds a review
func (h *AgentSystemHandler) AddMarketplaceReview(c *gin.Context) {
	userID := c.GetUint("userID")
	itemID := c.Param("id")

	var review agentsystem.ReviewInput
	if err := c.ShouldBindJSON(&review); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.marketplaceService.AddReview(c.Request.Context(), itemID, userID, &review)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetMarketplaceReviews returns reviews for an item
func (h *AgentSystemHandler) GetMarketplaceReviews(c *gin.Context) {
	itemID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	reviews, err := h.marketplaceService.GetReviews(c.Request.Context(), itemID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reviews)
}

// GetMarketplaceCategories returns categories
func (h *AgentSystemHandler) GetMarketplaceCategories(c *gin.Context) {
	categories, err := h.marketplaceService.GetCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// GetFeaturedItems returns featured items
func (h *AgentSystemHandler) GetFeaturedItems(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	items, err := h.marketplaceService.GetFeatured(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetTrendingItems returns trending items
func (h *AgentSystemHandler) GetTrendingItems(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	items, err := h.marketplaceService.GetTrending(c.Request.Context(), days, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetMyMarketplaceItems returns user's published items
func (h *AgentSystemHandler) GetMyMarketplaceItems(c *gin.Context) {
	userID := c.GetUint("userID")

	items, err := h.marketplaceService.GetUserItems(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// ForkMarketplaceItem forks an item
func (h *AgentSystemHandler) ForkMarketplaceItem(c *gin.Context) {
	userID := c.GetUint("userID")
	itemID := c.Param("id")

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.marketplaceService.ForkItem(c.Request.Context(), itemID, userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ==================== A/B Test Endpoints ====================

// CreateABTest creates a new A/B test
func (h *AgentSystemHandler) CreateABTest(c *gin.Context) {
	userID := c.GetUint("userID")

	var test models.ABTest
	if err := c.ShouldBindJSON(&test); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.abTestService.CreateTest(c.Request.Context(), userID, &test)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetABTest retrieves an A/B test
func (h *AgentSystemHandler) GetABTest(c *gin.Context) {
	userID := c.GetUint("userID")
	testID := c.Param("id")

	test, err := h.abTestService.GetTest(c.Request.Context(), testID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, test)
}

// ListABTests returns user's A/B tests
func (h *AgentSystemHandler) ListABTests(c *gin.Context) {
	userID := c.GetUint("userID")

	tests, err := h.abTestService.ListTests(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tests)
}

// StartABTest starts an A/B test
func (h *AgentSystemHandler) StartABTest(c *gin.Context) {
	userID := c.GetUint("userID")
	testID := c.Param("id")

	if err := h.abTestService.StartTest(c.Request.Context(), testID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test started"})
}
