package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"backend/internal/services/branch"

	"github.com/gin-gonic/gin"
)

// BranchHandler handles branch-related API requests
type BranchHandler struct {
	service *branch.BranchService
}

// NewBranchHandler creates a new branch handler
func NewBranchHandler(service *branch.BranchService) *BranchHandler {
	return &BranchHandler{service: service}
}

// =============================================================================
// Branch Endpoints
// =============================================================================

// CreateBranch creates a new branch from a message
// POST /api/v1/chat/conversations/:id/branches
func (h *BranchHandler) CreateBranch(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	var req struct {
		ForkPointMsgID *uint  `json:"fork_point_message_id"`
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	branch, err := h.service.CreateBranch(uint(convID), req.ForkPointMsgID, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, branch)
}

// GetBranches returns all branches for a conversation
// GET /api/v1/chat/conversations/:id/branches
func (h *BranchHandler) GetBranches(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	branches, err := h.service.GetBranches(uint(convID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, branches)
}

// GetBranchTree returns the branch tree structure for visualization
// GET /api/v1/chat/conversations/:id/branches/tree
func (h *BranchHandler) GetBranchTree(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	tree, err := h.service.GetBranchTree(uint(convID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tree)
}

// SwitchBranch switches to a different branch
// POST /api/v1/chat/conversations/:id/branches/:branchId/switch
func (h *BranchHandler) SwitchBranch(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	branchID := c.Param("branchId")
	if branchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch ID required"})
		return
	}

	if err := h.service.SwitchBranch(uint(convID), branchID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "branch switched successfully"})
}

// DeleteBranch deletes a branch
// DELETE /api/v1/chat/conversations/:id/branches/:branchId
func (h *BranchHandler) DeleteBranch(c *gin.Context) {
	branchID := c.Param("branchId")
	if branchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch ID required"})
		return
	}

	if err := h.service.DeleteBranch(branchID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "branch deleted successfully"})
}

// GetBranchMessages returns messages for a specific branch
// GET /api/v1/chat/branches/:branchId/messages
func (h *BranchHandler) GetBranchMessages(c *gin.Context) {
	branchID := c.Param("branchId")
	if branchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch ID required"})
		return
	}

	messages, err := h.service.GetBranchMessages(branchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// =============================================================================
// Message Editing Endpoints
// =============================================================================

// EditMessage edits a message content
// PUT /api/v1/chat/messages/:msgId
func (h *BranchHandler) EditMessage(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	userID, _ := c.Get("user_id")

	var req struct {
		Content    string `json:"content" binding:"required"`
		Regenerate bool   `json:"regenerate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.service.EditMessage(uint(msgID), req.Content, userID.(uint), req.Regenerate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, msg)
}

// EditAndBranch edits a message and creates a new branch
// POST /api/v1/chat/messages/:msgId/branch
func (h *BranchHandler) EditAndBranch(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	userID, _ := c.Get("user_id")

	var req struct {
		Content    string `json:"content" binding:"required"`
		BranchName string `json:"branch_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	branchName := req.BranchName
	if branchName == "" {
		branchName = "Edited Branch"
	}

	branch, msg, err := h.service.CreateBranchFromEdit(uint(msgID), req.Content, userID.(uint), branchName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"branch":  branch,
		"message": msg,
	})
}

// GetMessageVersions returns edit history for a message
// GET /api/v1/chat/messages/:msgId/versions
func (h *BranchHandler) GetMessageVersions(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	versions, err := h.service.GetMessageVersions(uint(msgID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// RevertMessage reverts a message to a previous version
// POST /api/v1/chat/messages/:msgId/revert
func (h *BranchHandler) RevertMessage(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	userID, _ := c.Get("user_id")

	var req struct {
		Version int `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.service.RevertMessage(uint(msgID), req.Version, userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, msg)
}

// RegenerateResponse regenerates the AI response for a message
// POST /api/v1/chat/messages/:msgId/regenerate
func (h *BranchHandler) RegenerateResponse(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	var req struct {
		Model string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.service.RegenerateResponse(c.Request.Context(), uint(msgID), req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, msg)
}

// RegenerateResponseStream regenerates the AI response with streaming output
// POST /api/v1/chat/messages/:msgId/regenerate/stream
func (h *BranchHandler) RegenerateResponseStream(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	var req struct {
		Model string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Create cancellable context
	ctx, cancelFunc := context.WithCancel(c.Request.Context())
	closeNotify := c.Writer.CloseNotify()
	go func() {
		<-closeNotify
		cancelFunc()
	}()

	// Get streaming channel
	ch, _, err := h.service.RegenerateResponseStream(ctx, uint(msgID), req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		cancelFunc()
		return
	}

	// Stream response
	for chunk := range ch {
		jsonChunk, err := json.Marshal(chunk)
		if err != nil {
			continue
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", jsonChunk)
		c.Writer.Flush()
	}

	cancelFunc()
}

// MultiRegenerate generates multiple alternative responses
// POST /api/v1/chat/messages/:msgId/multi-regenerate
func (h *BranchHandler) MultiRegenerate(c *gin.Context) {
	msgID, err := strconv.ParseUint(c.Param("msgId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	var req struct {
		Model string `json:"model"`
		Count int    `json:"count"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Count < 1 || req.Count > 5 {
		req.Count = 3
	}

	messages, err := h.service.MultiRegenerate(c.Request.Context(), uint(msgID), req.Model, req.Count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// =============================================================================
// Parallel Exploration Endpoints
// =============================================================================

// StartParallelExploration starts exploring multiple models in parallel
// POST /api/v1/chat/conversations/:id/parallel-explore
func (h *BranchHandler) StartParallelExploration(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	var req struct {
		SourceMsgID uint     `json:"source_message_id" binding:"required"`
		Prompt      string   `json:"prompt" binding:"required"`
		Models      []string `json:"models" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Models) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least 2 models required for parallel exploration"})
		return
	}

	if len(req.Models) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 5 models allowed"})
		return
	}

	exploration, err := h.service.StartParallelExploration(c.Request.Context(), uint(convID), req.SourceMsgID, req.Prompt, req.Models)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, exploration)
}

// GetParallelExploration returns a parallel exploration session
// GET /api/v1/chat/parallel-explorations/:explorationId
func (h *BranchHandler) GetParallelExploration(c *gin.Context) {
	explorationID := c.Param("explorationId")
	if explorationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exploration ID required"})
		return
	}

	exploration, err := h.service.GetParallelExploration(explorationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exploration)
}

// GetParallelExplorations returns all explorations for a conversation
// GET /api/v1/chat/conversations/:id/parallel-explorations
func (h *BranchHandler) GetParallelExplorations(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	explorations, err := h.service.GetParallelExplorations(uint(convID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, explorations)
}

// SelectExplorationBranch selects a branch from parallel exploration
// POST /api/v1/chat/conversations/:id/parallel-explorations/:explorationId/select
func (h *BranchHandler) SelectExplorationBranch(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	var req struct {
		BranchID string `json:"branch_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.SelectExplorationBranch(uint(convID), req.BranchID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "branch selected successfully"})
}

// GetMessageTree returns the message tree structure
// GET /api/v1/chat/conversations/:id/message-tree
func (h *BranchHandler) GetMessageTree(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	branchID := c.Query("branch_id")
	var branchIDPtr *string
	if branchID != "" {
		branchIDPtr = &branchID
	}

	tree, err := h.service.GetMessageTree(uint(convID), branchIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tree)
}
