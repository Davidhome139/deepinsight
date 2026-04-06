package handlers

import (
	"net/http"
	"strconv"

	"backend/internal/services/video"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type VideoHandler struct {
	videoService *video.VideoService
}

func NewVideoHandler(db *gorm.DB) *VideoHandler {
	return &VideoHandler{
		videoService: video.NewVideoService(db),
	}
}

func (h *VideoHandler) Generate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req video.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.videoService.Generate(userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *VideoHandler) GetTask(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskID := c.Param("id")
	task, err := h.videoService.GetTask(taskID, userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *VideoHandler) ListTasks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	status := c.Query("status")

	result, err := h.videoService.ListTasks(userID.(uint), page, size, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *VideoHandler) DeleteTask(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskID := c.Param("id")
	if err := h.videoService.DeleteTask(taskID, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

func (h *VideoHandler) GetModels(c *gin.Context) {
	models := h.videoService.GetModels()
	c.JSON(http.StatusOK, gin.H{"models": models})
}

func (h *VideoHandler) GetProviders(c *gin.Context) {
	providers := h.videoService.GetProviders()
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}
