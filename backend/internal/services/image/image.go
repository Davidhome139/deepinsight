package image

import (
	"backend/internal/models"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ImageProvider defines the interface for image generation providers
type ImageProvider interface {
	Generate(ctx context.Context, req *ImageRequest) (*ImageResult, error)
	CreateVariation(ctx context.Context, imageURL string, req *ImageRequest) (*ImageResult, error)
	EditImage(ctx context.Context, imageURL, maskURL string, req *ImageRequest) (*ImageResult, error)
	GetName() string
	GetModels() []ImageModel
}

// ImageModel represents an available image model
type ImageModel struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Description string   `json:"description"`
	Sizes       []string `json:"sizes"`
	Qualities   []string `json:"qualities,omitempty"`
	Styles      []string `json:"styles,omitempty"`
	Features    []string `json:"features"` // text2image, image2image, inpaint, variations
}

// ImageRequest represents an image generation/edit request
type ImageRequest struct {
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Size           string `json:"size,omitempty"`
	N              int    `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	// For variations and editing
	SourceImageURL string  `json:"source_image_url,omitempty"`
	MaskURL        string  `json:"mask_url,omitempty"`
	Strength       float64 `json:"strength,omitempty"` // 0.0-1.0 for img2img
}

// ImageResult represents the result of image generation
type ImageResult struct {
	ImageURL       string `json:"image_url"`
	RevisedPrompt  string `json:"revised_prompt,omitempty"`
	GenerationTime int64  `json:"generation_time"`
}

// ImageService handles image generation across multiple providers
type ImageService struct {
	db        *gorm.DB
	providers map[string]ImageProvider
}

// NewImageService creates a new image service with all providers
func NewImageService(db *gorm.DB, apiKey string) *ImageService {
	s := &ImageService{
		db:        db,
		providers: make(map[string]ImageProvider),
	}

	// Register providers
	s.providers["aliyun"] = NewAliyunWanxProvider(db, apiKey)
	s.providers["stability"] = NewStabilityImageProvider(db)

	return s
}

// Generate creates an image from a prompt
func (s *ImageService) Generate(userID uint, req models.ImageGenerationRequest) (*models.GeneratedImage, error) {
	provider := s.getProvider(req.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()

	// Set defaults
	if req.Size == "" {
		req.Size = "1024*1024"
	}
	if req.N == 0 {
		req.N = 1
	}

	imageReq := &ImageRequest{
		Model:   req.Model,
		Prompt:  req.Prompt,
		Size:    req.Size,
		N:       req.N,
		Quality: req.Quality,
		Style:   req.Style,
	}

	result, err := provider.Generate(ctx, imageReq)
	if err != nil {
		return nil, err
	}

	generationTime := time.Since(startTime).Milliseconds()

	// Save to database
	image := &models.GeneratedImage{
		ID:             uuid.New().String(),
		UserID:         userID,
		Prompt:         req.Prompt,
		RevisedPrompt:  result.RevisedPrompt,
		Model:          req.Model,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		ImageURL:       result.ImageURL,
		Cost:           models.CalculateImageCost(req.Model, req.Size, req.Quality),
		GenerationTime: generationTime,
		CreatedAt:      time.Now(),
	}

	if err := s.db.Create(image).Error; err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	return image, nil
}

// CreateVariation creates a variation of an existing image
func (s *ImageService) CreateVariation(userID uint, sourceImageID string, prompt string) (*models.GeneratedImage, error) {
	// Get source image
	var sourceImage models.GeneratedImage
	if err := s.db.Where("id = ? AND user_id = ?", sourceImageID, userID).First(&sourceImage).Error; err != nil {
		return nil, fmt.Errorf("source image not found")
	}

	provider := s.getProvider(sourceImage.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()

	imageReq := &ImageRequest{
		Model:  sourceImage.Model,
		Prompt: prompt,
		Size:   sourceImage.Size,
	}

	result, err := provider.CreateVariation(ctx, sourceImage.ImageURL, imageReq)
	if err != nil {
		return nil, err
	}

	generationTime := time.Since(startTime).Milliseconds()

	// Save to database
	image := &models.GeneratedImage{
		ID:             uuid.New().String(),
		UserID:         userID,
		Prompt:         prompt,
		RevisedPrompt:  result.RevisedPrompt,
		Model:          sourceImage.Model,
		Size:           sourceImage.Size,
		ImageURL:       result.ImageURL,
		Cost:           models.CalculateImageCost(sourceImage.Model, sourceImage.Size, ""),
		GenerationTime: generationTime,
		CreatedAt:      time.Now(),
	}

	if err := s.db.Create(image).Error; err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	return image, nil
}

// EditImage edits an image using inpainting
func (s *ImageService) EditImage(userID uint, sourceImageID string, maskURL string, prompt string) (*models.GeneratedImage, error) {
	// Get source image
	var sourceImage models.GeneratedImage
	if err := s.db.Where("id = ? AND user_id = ?", sourceImageID, userID).First(&sourceImage).Error; err != nil {
		return nil, fmt.Errorf("source image not found")
	}

	provider := s.getProvider(sourceImage.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()

	imageReq := &ImageRequest{
		Model:  sourceImage.Model,
		Prompt: prompt,
		Size:   sourceImage.Size,
	}

	result, err := provider.EditImage(ctx, sourceImage.ImageURL, maskURL, imageReq)
	if err != nil {
		return nil, err
	}

	generationTime := time.Since(startTime).Milliseconds()

	// Save to database
	image := &models.GeneratedImage{
		ID:             uuid.New().String(),
		UserID:         userID,
		Prompt:         prompt,
		RevisedPrompt:  result.RevisedPrompt,
		Model:          sourceImage.Model,
		Size:           sourceImage.Size,
		ImageURL:       result.ImageURL,
		Cost:           models.CalculateImageCost(sourceImage.Model, sourceImage.Size, ""),
		GenerationTime: generationTime,
		CreatedAt:      time.Now(),
	}

	if err := s.db.Create(image).Error; err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	return image, nil
}

// getProvider returns the appropriate provider for a model
func (s *ImageService) getProvider(model string) ImageProvider {
	// Map models to providers
	switch model {
	case "stable-diffusion-xl", "stable-diffusion-v2", "sd-inpaint":
		if p, ok := s.providers["stability"]; ok {
			return p
		}
	}
	// Default to Aliyun
	return s.providers["aliyun"]
}

// GetHistory returns user's image generation history
func (s *ImageService) GetHistory(userID uint, limit, offset int) ([]models.GeneratedImage, int64, error) {
	var images []models.GeneratedImage
	var total int64

	query := s.db.Model(&models.GeneratedImage{}).Where("user_id = ?", userID)
	query.Count(&total)

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&images).Error

	return images, total, err
}

// GetImage returns a single image by ID
func (s *ImageService) GetImage(id string, userID uint) (*models.GeneratedImage, error) {
	var image models.GeneratedImage
	err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// DeleteImage deletes an image
func (s *ImageService) DeleteImage(id string, userID uint) error {
	return s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.GeneratedImage{}).Error
}

// GetAvailableModels returns available models from all providers
func (s *ImageService) GetAvailableModels() []map[string]interface{} {
	var models []map[string]interface{}
	for _, provider := range s.providers {
		for _, m := range provider.GetModels() {
			models = append(models, map[string]interface{}{
				"id":          m.ID,
				"name":        m.Name,
				"description": m.Description,
				"provider":    m.Provider,
				"sizes":       m.Sizes,
				"qualities":   m.Qualities,
				"styles":      m.Styles,
				"features":    m.Features,
			})
		}
	}
	return models
}

// GetProviders returns list of available providers
func (s *ImageService) GetProviders() []map[string]interface{} {
	var providers []map[string]interface{}
	for name, provider := range s.providers {
		providers = append(providers, map[string]interface{}{
			"id":     name,
			"name":   provider.GetName(),
			"models": provider.GetModels(),
		})
	}
	return providers
}

// ============ Aliyun Wanx Provider ============

type AliyunWanxProvider struct {
	db      *gorm.DB
	apiKey  string
	baseURL string
}

func NewAliyunWanxProvider(db *gorm.DB, apiKey string) *AliyunWanxProvider {
	return &AliyunWanxProvider{
		db:      db,
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis",
	}
}

func (p *AliyunWanxProvider) GetName() string {
	return "Aliyun Wanx (通义万象)"
}

func (p *AliyunWanxProvider) GetModels() []ImageModel {
	return []ImageModel{
		{
			ID:          "wanx-v1",
			Name:        "通义万象 v1",
			Provider:    "aliyun",
			Description: "高质量文生图模型",
			Sizes:       []string{"1024*1024", "720*1280", "1280*720"},
			Qualities:   []string{"standard"},
			Features:    []string{"text2image"},
		},
		{
			ID:          "wanx-sketch-to-image-v1",
			Name:        "通义万象草图生图",
			Provider:    "aliyun",
			Description: "根据草图生成图像",
			Sizes:       []string{"1024*1024", "720*1280", "1280*720"},
			Qualities:   []string{"standard"},
			Features:    []string{"image2image"},
		},
	}
}

func (p *AliyunWanxProvider) Generate(ctx context.Context, req *ImageRequest) (*ImageResult, error) {
	// Build Wanx API request
	apiReq := map[string]interface{}{
		"model": req.Model,
		"input": map[string]interface{}{
			"prompt": req.Prompt,
		},
		"parameters": map[string]interface{}{
			"size": req.Size,
			"n":    req.N,
		},
	}

	if req.NegativePrompt != "" {
		apiReq["input"].(map[string]interface{})["negative_prompt"] = req.NegativePrompt
	}

	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Submit task
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var submitResp struct {
		Output struct {
			TaskID     string `json:"task_id"`
			TaskStatus string `json:"task_status"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &submitResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if submitResp.Code != "" {
		return nil, fmt.Errorf("API error: %s - %s", submitResp.Code, submitResp.Message)
	}

	taskID := submitResp.Output.TaskID
	if taskID == "" {
		return nil, fmt.Errorf("no task ID returned")
	}

	// Poll for task completion
	return p.pollTaskCompletion(ctx, taskID)
}

func (p *AliyunWanxProvider) pollTaskCompletion(ctx context.Context, taskID string) (*ImageResult, error) {
	taskURL := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/tasks/%s", taskID)
	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < 60; i++ { // Max 2 minutes
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}

		taskReq, _ := http.NewRequestWithContext(ctx, "GET", taskURL, nil)
		taskReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		taskResp, err := client.Do(taskReq)
		if err != nil {
			continue
		}

		taskBody, _ := io.ReadAll(taskResp.Body)
		taskResp.Body.Close()

		var taskResult struct {
			Output struct {
				TaskStatus string `json:"task_status"`
				Results    []struct {
					URL string `json:"url"`
				} `json:"results"`
			} `json:"output"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(taskBody, &taskResult); err != nil {
			continue
		}

		if taskResult.Output.TaskStatus == "SUCCEEDED" {
			if len(taskResult.Output.Results) > 0 {
				return &ImageResult{
					ImageURL: taskResult.Output.Results[0].URL,
				}, nil
			}
		} else if taskResult.Output.TaskStatus == "FAILED" {
			return nil, fmt.Errorf("image generation failed: %s", taskResult.Message)
		}
	}

	return nil, fmt.Errorf("image generation timed out")
}

func (p *AliyunWanxProvider) CreateVariation(ctx context.Context, imageURL string, req *ImageRequest) (*ImageResult, error) {
	// Aliyun doesn't have native variation support, use img2img with the original
	return p.Generate(ctx, req)
}

func (p *AliyunWanxProvider) EditImage(ctx context.Context, imageURL, maskURL string, req *ImageRequest) (*ImageResult, error) {
	// Aliyun doesn't have native inpainting, return error
	return nil, fmt.Errorf("inpainting not supported by Aliyun Wanx")
}

// ============ Stability AI Provider ============

type StabilityImageProvider struct {
	db *gorm.DB
}

func NewStabilityImageProvider(db *gorm.DB) *StabilityImageProvider {
	return &StabilityImageProvider{db: db}
}

func (p *StabilityImageProvider) GetName() string {
	return "Stability AI"
}

func (p *StabilityImageProvider) GetModels() []ImageModel {
	return []ImageModel{
		{
			ID:          "stable-diffusion-xl",
			Name:        "Stable Diffusion XL",
			Provider:    "stability",
			Description: "High quality image generation",
			Sizes:       []string{"1024x1024", "1152x896", "896x1152", "1344x768", "768x1344"},
			Qualities:   []string{"standard", "hd"},
			Styles:      []string{"photographic", "anime", "digital-art", "cinematic"},
			Features:    []string{"text2image", "image2image", "inpaint", "variations"},
		},
		{
			ID:          "sd-inpaint",
			Name:        "Stable Diffusion Inpaint",
			Provider:    "stability",
			Description: "Image inpainting and editing",
			Sizes:       []string{"512x512", "1024x1024"},
			Features:    []string{"inpaint"},
		},
	}
}

func (p *StabilityImageProvider) getAPIKey() (string, error) {
	var setting models.ProviderSetting
	err := p.db.Where("provider = ? AND enabled = ?", "stability", true).First(&setting).Error
	if err != nil {
		return "", fmt.Errorf("Stability AI not configured")
	}
	return setting.APIKey, nil
}

func (p *StabilityImageProvider) Generate(ctx context.Context, req *ImageRequest) (*ImageResult, error) {
	apiKey, err := p.getAPIKey()
	if err != nil {
		return nil, err
	}

	url := "https://api.stability.ai/v1/generation/stable-diffusion-xl-1024-v1-0/text-to-image"

	// Build request body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("text_prompts[0][text]", req.Prompt)
	writer.WriteField("text_prompts[0][weight]", "1")
	if req.NegativePrompt != "" {
		writer.WriteField("text_prompts[1][text]", req.NegativePrompt)
		writer.WriteField("text_prompts[1][weight]", "-1")
	}

	// Parse size
	width, height := 1024, 1024
	fmt.Sscanf(req.Size, "%dx%d", &width, &height)
	writer.WriteField("width", fmt.Sprintf("%d", width))
	writer.WriteField("height", fmt.Sprintf("%d", height))

	if req.Style != "" {
		writer.WriteField("style_preset", req.Style)
	}

	writer.Close()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, body)
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result struct {
		Artifacts []struct {
			Base64       string `json:"base64"`
			FinishReason string `json:"finishReason"`
		} `json:"artifacts"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Artifacts) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	// For now, return base64 data URI (in production, upload to storage)
	imageData := result.Artifacts[0].Base64
	dataURI := "data:image/png;base64," + imageData

	return &ImageResult{ImageURL: dataURI}, nil
}

func (p *StabilityImageProvider) CreateVariation(ctx context.Context, imageURL string, req *ImageRequest) (*ImageResult, error) {
	apiKey, err := p.getAPIKey()
	if err != nil {
		return nil, err
	}

	// Download source image
	imageData, err := downloadImageAsBase64(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	url := "https://api.stability.ai/v1/generation/stable-diffusion-xl-1024-v1-0/image-to-image"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add image
	imageBytes, _ := base64.StdEncoding.DecodeString(imageData)
	part, _ := writer.CreateFormFile("init_image", "image.png")
	part.Write(imageBytes)

	writer.WriteField("text_prompts[0][text]", req.Prompt)
	writer.WriteField("image_strength", "0.35") // Low strength for variations

	writer.Close()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, body)
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result struct {
		Artifacts []struct {
			Base64 string `json:"base64"`
		} `json:"artifacts"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Artifacts) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	dataURI := "data:image/png;base64," + result.Artifacts[0].Base64
	return &ImageResult{ImageURL: dataURI}, nil
}

func (p *StabilityImageProvider) EditImage(ctx context.Context, imageURL, maskURL string, req *ImageRequest) (*ImageResult, error) {
	apiKey, err := p.getAPIKey()
	if err != nil {
		return nil, err
	}

	// Download images
	imageData, err := downloadImageAsBase64(ctx, imageURL)
	if err != nil {
		return nil, err
	}
	maskData, err := downloadImageAsBase64(ctx, maskURL)
	if err != nil {
		return nil, err
	}

	url := "https://api.stability.ai/v1/generation/stable-diffusion-xl-1024-v1-0/image-to-image/masking"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add images
	imageBytes, _ := base64.StdEncoding.DecodeString(imageData)
	imgPart, _ := writer.CreateFormFile("init_image", "image.png")
	imgPart.Write(imageBytes)

	maskBytes, _ := base64.StdEncoding.DecodeString(maskData)
	maskPart, _ := writer.CreateFormFile("mask_image", "mask.png")
	maskPart.Write(maskBytes)

	writer.WriteField("text_prompts[0][text]", req.Prompt)
	writer.WriteField("mask_source", "MASK_IMAGE_WHITE")

	writer.Close()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, body)
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result struct {
		Artifacts []struct {
			Base64 string `json:"base64"`
		} `json:"artifacts"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Artifacts) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	dataURI := "data:image/png;base64," + result.Artifacts[0].Base64
	return &ImageResult{ImageURL: dataURI}, nil
}

// Helper function to download image and convert to base64
func downloadImageAsBase64(ctx context.Context, url string) (string, error) {
	if len(url) > 100 && url[:5] == "data:" {
		// Already a data URI
		parts := bytes.SplitN([]byte(url), []byte(","), 2)
		if len(parts) == 2 {
			return string(parts[1]), nil
		}
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
