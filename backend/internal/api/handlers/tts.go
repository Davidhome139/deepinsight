package handlers

import (
	"backend/internal/services/tts"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TTSHandler struct {
	service *tts.TTSService
}

func NewTTSHandler(service *tts.TTSService) *TTSHandler {
	return &TTSHandler{service: service}
}

// SpeakRequest represents a TTS request
type SpeakRequest struct {
	Text  string  `json:"text" binding:"required"`
	Voice string  `json:"voice"`
	Model string  `json:"model"`
	Speed float64 `json:"speed"`
}

// Speak converts text to speech
func (h *TTSHandler) Speak(c *gin.Context) {
	var req SpeakRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Text) > 4096 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text too long, max 4096 characters"})
		return
	}

	audioData, err := h.service.Speak(req.Text, req.Voice, req.Model, req.Speed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Disposition", "inline; filename=speech.mp3")
	c.Data(http.StatusOK, "audio/mpeg", audioData)
}

// GetVoices returns available TTS voices
func (h *TTSHandler) GetVoices(c *gin.Context) {
	voices := h.service.GetAvailableVoices()
	c.JSON(http.StatusOK, voices)
}

// GetModels returns available TTS models
func (h *TTSHandler) GetModels(c *gin.Context) {
	models := h.service.GetAvailableModels()
	c.JSON(http.StatusOK, models)
}
