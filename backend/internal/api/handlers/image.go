package handlers

import (
	"backend/internal/models"
	"backend/internal/services/image"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ImageHandler struct {
	service *image.ImageService
}

func NewImageHandler(service *image.ImageService) *ImageHandler {
	return &ImageHandler{service: service}
}

// Generate creates an image from a prompt
func (h *ImageHandler) Generate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.ImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Prompt) > 4000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt too long, max 4000 characters"})
		return
	}

	img, err := h.service.Generate(userID.(uint), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, img)
}

// GetHistory returns user's image generation history
func (h *ImageHandler) GetHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	images, total, err := h.service.GetHistory(userID.(uint), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetImage returns a single image by ID
func (h *ImageHandler) GetImage(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	img, err := h.service.GetImage(id, userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	c.JSON(http.StatusOK, img)
}

// DeleteImage deletes an image
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	if err := h.service.DeleteImage(id, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "image deleted"})
}

// GetModels returns available image generation models
func (h *ImageHandler) GetModels(c *gin.Context) {
	models := h.service.GetAvailableModels()
	c.JSON(http.StatusOK, models)
}

// GetProviders returns available image providers
func (h *ImageHandler) GetProviders(c *gin.Context) {
	providers := h.service.GetProviders()
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// CreateVariation creates a variation of an existing image
func (h *ImageHandler) CreateVariation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		SourceImageID string `json:"source_image_id" binding:"required"`
		Prompt        string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	img, err := h.service.CreateVariation(userID.(uint), req.SourceImageID, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, img)
}

// EditImage edits an image using inpainting
func (h *ImageHandler) EditImage(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		SourceImageID string `json:"source_image_id" binding:"required"`
		MaskURL       string `json:"mask_url" binding:"required"`
		Prompt        string `json:"prompt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	img, err := h.service.EditImage(userID.(uint), req.SourceImageID, req.MaskURL, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, img)
}
