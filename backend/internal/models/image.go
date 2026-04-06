package models

import (
	"time"
)

// GeneratedImage represents an AI-generated image
type GeneratedImage struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id" gorm:"index"`
	Prompt         string    `json:"prompt"`
	RevisedPrompt  string    `json:"revised_prompt,omitempty"`
	Model          string    `json:"model"`   // dall-e-3, dall-e-2
	Size           string    `json:"size"`    // 1024x1024, 1024x1792, 1792x1024
	Quality        string    `json:"quality"` // standard, hd
	Style          string    `json:"style"`   // vivid, natural
	ImageURL       string    `json:"image_url"`
	LocalPath      string    `json:"local_path,omitempty"`
	Cost           float64   `json:"cost"`
	GenerationTime int64     `json:"generation_time"` // ms
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
}

// ImageGenerationRequest represents a request to generate an image
type ImageGenerationRequest struct {
	Prompt  string `json:"prompt" binding:"required"`
	Model   string `json:"model"`   // default: dall-e-3
	Size    string `json:"size"`    // default: 1024x1024
	Quality string `json:"quality"` // default: standard
	Style   string `json:"style"`   // default: vivid
	N       int    `json:"n"`       // number of images (1 for dall-e-3)
}

// ImagePricing defines pricing per image
var ImagePricing = map[string]map[string]float64{
	"dall-e-3": {
		"1024x1024-standard": 0.040,
		"1024x1024-hd":       0.080,
		"1024x1792-standard": 0.080,
		"1024x1792-hd":       0.120,
		"1792x1024-standard": 0.080,
		"1792x1024-hd":       0.120,
	},
	"dall-e-2": {
		"1024x1024": 0.020,
		"512x512":   0.018,
		"256x256":   0.016,
	},
}

// CalculateImageCost calculates the cost for image generation
func CalculateImageCost(model, size, quality string) float64 {
	if pricing, ok := ImagePricing[model]; ok {
		key := size
		if model == "dall-e-3" {
			key = size + "-" + quality
		}
		if cost, ok := pricing[key]; ok {
			return cost
		}
	}
	return 0.04 // Default cost
}
