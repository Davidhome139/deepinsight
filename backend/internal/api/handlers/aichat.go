package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/services/aichat"

	"github.com/gin-gonic/gin"
)

// AIChatHandler AI-AI 聊天处理器
type AIChatHandler struct {
	service *aichat.AIChatService
}

// NewAIChatHandler 创建处理器
func NewAIChatHandler(service *aichat.AIChatService) *AIChatHandler {
	return &AIChatHandler{
		service: service,
	}
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	Title             string                   `json:"title" binding:"required"`
	Topic             string                   `json:"topic" binding:"required"`
	GlobalConstraint  string                   `json:"global_constraint"`
	MaxRounds         int                      `json:"max_rounds"`
	TerminationConfig models.TerminationConfig `json:"termination_config"`
	AgentA            models.AgentConfig       `json:"agent_a" binding:"required"`
	AgentB            models.AgentConfig       `json:"agent_b" binding:"required"`
}

// CreateSession 创建会话
func (h *AIChatHandler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.MaxRounds == 0 {
		req.MaxRounds = 10
	}

	config := aichat.SessionConfig{
		Title:             req.Title,
		Topic:             req.Topic,
		GlobalConstraint:  req.GlobalConstraint,
		MaxRounds:         req.MaxRounds,
		TerminationConfig: req.TerminationConfig,
		AgentA:            req.AgentA,
		AgentB:            req.AgentB,
	}

	session, err := h.service.CreateSession(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession 获取会话
func (h *AIChatHandler) GetSession(c *gin.Context) {
	id := c.Param("id")
	session, err := h.service.GetSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// ListSessions 列会话
func (h *AIChatHandler) ListSessions(c *gin.Context) {
	filter := aichat.SessionFilter{}

	if status := c.Query("status"); status != "" {
		filter.Status = models.SessionStatus(status)
	}

	if parentID := c.Query("parent_id"); parentID != "" {
		filter.ParentID = &parentID
	}

	sessions, err := h.service.ListSessions(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// DeleteSession 删除会话
func (h *AIChatHandler) DeleteSession(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteSession(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
}

// StartSession 开始会话
func (h *AIChatHandler) StartSession(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.StartSession(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session started"})
}

// PauseSession 暂停会话
func (h *AIChatHandler) PauseSession(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.PauseSession(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session paused"})
}

// StopSession 停止会话
func (h *AIChatHandler) StopSession(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.StopSession(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session stopped"})
}

// DirectorCommandRequest 导演指令请求
type DirectorCommandRequest struct {
	TargetAgent      string `json:"target_agent" binding:"required"` // agent_a, agent_b, both
	Command          string `json:"command" binding:"required"`
	InsertAfterRound *int   `json:"insert_after_round"`
}

// InjectDirectorCommand 注入导演指令
func (h *AIChatHandler) InjectDirectorCommand(c *gin.Context) {
	id := c.Param("id")

	var req DirectorCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := &models.DirectorCommand{
		TargetAgent:      req.TargetAgent,
		Command:          req.Command,
		InsertAfterRound: req.InsertAfterRound,
	}

	if err := h.service.InjectDirectorCommand(id, cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "director command injected"})
}

// CreateBranchRequest 创建分支请求
type CreateBranchRequest struct {
	Title string `json:"title" binding:"required"`
}

// CreateBranch 创建分支
func (h *AIChatHandler) CreateBranch(c *gin.Context) {
	id := c.Param("id")

	roundStr := c.Query("round")
	round, err := strconv.Atoi(roundStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid round parameter"})
		return
	}

	var req CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	branch, err := h.service.CreateBranch(id, round, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, branch)
}

// CreateSnapshotRequest 创建快照请求
type CreateSnapshotRequest struct {
	Title string `json:"title" binding:"required"`
}

// CreateSnapshot 创建快照
func (h *AIChatHandler) CreateSnapshot(c *gin.Context) {
	id := c.Param("id")

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	snapshot, err := h.service.CreateSnapshot(id, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, snapshot)
}

// GetTemplates 获取模板列表
func (h *AIChatHandler) GetTemplates(c *gin.Context) {
	templates, err := h.service.GetTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

// GetTemplate 获取单个模板
func (h *AIChatHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")
	template, err := h.service.GetTemplate(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	c.JSON(http.StatusOK, template)
}

// CreateTemplate 创建自定义模板
func (h *AIChatHandler) CreateTemplate(c *gin.Context) {
	var template models.SessionTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.CreateTemplate(&template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, template)
}

// UpdateTemplate 更新模板
func (h *AIChatHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	var req models.SessionTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.service.UpdateTemplate(id, &req)
	if err != nil {
		if err.Error() == "cannot modify builtin templates" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// DeleteTemplate 删除模板
func (h *AIChatHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteTemplate(id); err != nil {
		if err.Error() == "cannot delete builtin templates" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

// CloneTemplate 克隆模板
func (h *AIChatHandler) CloneTemplate(c *gin.Context) {
	id := c.Param("id")
	clone, err := h.service.CloneTemplate(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, clone)
}

// ExportSession 导出会话
func (h *AIChatHandler) ExportSession(c *gin.Context) {
	id := c.Param("id")
	formatStr := c.DefaultQuery("format", "markdown")

	// 转换格式
	var format aichat.ExportFormat
	switch formatStr {
	case "json":
		format = aichat.ExportFormatJSON
	case "text":
		format = aichat.ExportFormatText
	default:
		format = aichat.ExportFormatMarkdown
	}

	exportService := h.service.GetExportService()
	content, err := exportService.ExportSession(id, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置文件名
	filename := fmt.Sprintf("ai-chat-session-%s.%s", id[:8], formatStr)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.String(http.StatusOK, content)
}

// GetEvaluation 获取会话评估结果
func (h *AIChatHandler) GetEvaluation(c *gin.Context) {
	id := c.Param("id")

	report, err := h.service.GetEvaluation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "evaluation not found"})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *AIChatHandler) GetModels(c *gin.Context) {
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

// GetMCPTools 获取可用的MCP工具列表（从settings模块）
func (h *AIChatHandler) GetMCPTools(c *gin.Context) {
	// Get MCP manager from service
	mcpManager := h.service.GetMCPManager()
	if mcpManager == nil {
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}

	// Ensure MCP servers are discovered
	mcpManager.Discover()

	// Get all servers
	servers := mcpManager.GetAllServers()

	var toolsList []map[string]interface{}
	for serverName, server := range servers {
		if !server.Connected && server.Client == nil {
			// Skip external servers that are not connected
			// But include built-in servers
			if serverName != "terminal" && serverName != "search" && serverName != "code-analysis" && serverName != "filesystem-local" {
				continue
			}
		}

		for _, tool := range server.Tools {
			toolsList = append(toolsList, map[string]interface{}{
				"id":          fmt.Sprintf("%s/%s", serverName, tool.Name),
				"name":        tool.Name,
				"server":      serverName,
				"description": tool.Description,
			})
		}
	}

	// Sort by server name, then by tool name
	sort.Slice(toolsList, func(i, j int) bool {
		if toolsList[i]["server"].(string) != toolsList[j]["server"].(string) {
			return toolsList[i]["server"].(string) < toolsList[j]["server"].(string)
		}
		return toolsList[i]["name"].(string) < toolsList[j]["name"].(string)
	})

	c.JSON(http.StatusOK, toolsList)
}
