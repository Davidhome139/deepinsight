package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"backend/internal/models"
	"backend/internal/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VideoProvider defines the interface for video generation providers
type VideoProvider interface {
	Generate(ctx context.Context, req *GenerateRequest) (*VideoTask, error)
	GetTaskStatus(ctx context.Context, taskID string) (*VideoTask, error)
	GetName() string
	GetModels() []VideoModel
}

// VideoModel represents an available video model
type VideoModel struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Description string   `json:"description"`
	MaxDuration int      `json:"max_duration"` // seconds
	Resolutions []string `json:"resolutions"`
	InputTypes  []string `json:"input_types"` // text2video, image2video
}

// VideoService manages video generation across multiple providers
type VideoService struct {
	db        *gorm.DB
	providers map[string]VideoProvider
}

// GenerateRequest represents a video generation request
type GenerateRequest struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Prompt     string `json:"prompt"`
	ImageURL   string `json:"image_url,omitempty"`
	Duration   int    `json:"duration,omitempty"`   // seconds
	Resolution string `json:"resolution,omitempty"` // e.g., "1280x720"
	FPS        int    `json:"fps,omitempty"`
	Style      string `json:"style,omitempty"`
}

// VideoTask represents a video generation task
type VideoTask struct {
	ID           string     `json:"id" gorm:"primaryKey"`
	UserID       uint       `json:"user_id" gorm:"index"`
	Provider     string     `json:"provider"`
	Model        string     `json:"model"`
	Prompt       string     `json:"prompt"`
	ImageURL     string     `json:"image_url,omitempty"`
	Status       string     `json:"status"`   // pending, processing, completed, failed
	Progress     int        `json:"progress"` // 0-100
	VideoURL     string     `json:"video_url,omitempty"`
	ThumbnailURL string     `json:"thumbnail_url,omitempty"`
	Duration     int        `json:"duration,omitempty"`
	Resolution   string     `json:"resolution,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
	ErrorMsg     string     `json:"error_msg,omitempty"`
	ExternalID   string     `json:"external_id,omitempty"` // Provider's task ID
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// NewVideoService creates a new video service with all providers
func NewVideoService(db *gorm.DB) *VideoService {
	s := &VideoService{
		db:        db,
		providers: make(map[string]VideoProvider),
	}

	// Register providers
	s.providers["baidu-air"] = NewBaiduAirProvider(db)
	s.providers["stability"] = NewStabilityProvider(db)
	s.providers["local"] = NewLocalFFmpegProvider()

	// Auto-migrate video tasks table
	db.AutoMigrate(&VideoTask{})

	return s
}

// Generate starts a video generation task
func (s *VideoService) Generate(userID uint, req *GenerateRequest) (*VideoTask, error) {
	// Select provider
	provider, ok := s.providers[req.Provider]
	if !ok {
		// Default to baidu-air
		provider = s.providers["baidu-air"]
		req.Provider = "baidu-air"
	}

	// Create task record
	task := &VideoTask{
		ID:         uuid.New().String(),
		UserID:     userID,
		Provider:   req.Provider,
		Model:      req.Model,
		Prompt:     req.Prompt,
		ImageURL:   req.ImageURL,
		Status:     "pending",
		Progress:   0,
		Duration:   req.Duration,
		Resolution: req.Resolution,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Start generation in background
	go s.processGeneration(task, provider, req)

	return task, nil
}

// processGeneration handles the actual video generation
func (s *VideoService) processGeneration(task *VideoTask, provider VideoProvider, req *GenerateRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Update status to processing
	s.updateTaskStatus(task.ID, "processing", 10, "")

	// Call provider
	result, err := provider.Generate(ctx, req)
	if err != nil {
		s.updateTaskStatus(task.ID, "failed", 0, err.Error())
		return
	}

	// Store external ID for polling
	if result.ExternalID != "" {
		s.db.Model(&VideoTask{}).Where("id = ?", task.ID).Update("external_id", result.ExternalID)
	}

	// Poll for completion
	s.pollTaskCompletion(ctx, task.ID, provider, result.ExternalID)
}

// pollTaskCompletion polls the provider for task completion
func (s *VideoService) pollTaskCompletion(ctx context.Context, taskID string, provider VideoProvider, externalID string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	progress := 20
	for {
		select {
		case <-ctx.Done():
			s.updateTaskStatus(taskID, "failed", 0, "timeout")
			return
		case <-ticker.C:
			status, err := provider.GetTaskStatus(ctx, externalID)
			if err != nil {
				continue
			}

			switch status.Status {
			case "completed":
				now := time.Now()
				s.db.Model(&VideoTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
					"status":        "completed",
					"progress":      100,
					"video_url":     status.VideoURL,
					"thumbnail_url": status.ThumbnailURL,
					"file_size":     status.FileSize,
					"updated_at":    now,
					"completed_at":  &now,
				})
				return
			case "failed":
				s.updateTaskStatus(taskID, "failed", 0, status.ErrorMsg)
				return
			case "processing":
				progress = min(progress+10, 90)
				s.updateTaskStatus(taskID, "processing", progress, "")
			}
		}
	}
}

// updateTaskStatus updates task status in database
func (s *VideoService) updateTaskStatus(taskID, status string, progress int, errorMsg string) {
	updates := map[string]interface{}{
		"status":     status,
		"progress":   progress,
		"updated_at": time.Now(),
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	s.db.Model(&VideoTask{}).Where("id = ?", taskID).Updates(updates)
}

// GetTask returns a single task by ID
func (s *VideoService) GetTask(taskID string, userID uint) (*VideoTask, error) {
	var task VideoTask
	err := s.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error
	return &task, err
}

// ListTasks returns user's video tasks
func (s *VideoService) ListTasks(userID uint, page, size int, status string) (*TaskListResponse, error) {
	var tasks []VideoTask
	var total int64

	query := s.db.Model(&VideoTask{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	err := query.Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&tasks).Error

	return &TaskListResponse{
		Total: int(total),
		Items: tasks,
		Page:  page,
		Size:  size,
	}, err
}

// DeleteTask deletes a video task
func (s *VideoService) DeleteTask(taskID string, userID uint) error {
	return s.db.Where("id = ? AND user_id = ?", taskID, userID).Delete(&VideoTask{}).Error
}

// GetModels returns all available models from all providers
func (s *VideoService) GetModels() []VideoModel {
	var models []VideoModel
	for _, provider := range s.providers {
		models = append(models, provider.GetModels()...)
	}
	return models
}

// GetProviders returns list of available providers
func (s *VideoService) GetProviders() []map[string]interface{} {
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

// TaskListResponse for paginated task listing
type TaskListResponse struct {
	Total int         `json:"total"`
	Items []VideoTask `json:"items"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// ============ Baidu Air Provider ============

type BaiduAirProvider struct {
	db *gorm.DB
}

func NewBaiduAirProvider(db *gorm.DB) *BaiduAirProvider {
	return &BaiduAirProvider{db: db}
}

func (p *BaiduAirProvider) GetName() string {
	return "Baidu Air"
}

func (p *BaiduAirProvider) GetModels() []VideoModel {
	return []VideoModel{
		{
			ID:          "musesteamer-air-i2v",
			Name:        "Muse Steamer Air I2V",
			Provider:    "baidu-air",
			Description: "Image to video generation",
			MaxDuration: 5,
			Resolutions: []string{"1280x720", "720x480"},
			InputTypes:  []string{"image2video"},
		},
		{
			ID:          "musesteamer-2.0-i2v",
			Name:        "Muse Steamer 2.0 I2V",
			Provider:    "baidu-air",
			Description: "Enhanced image to video",
			MaxDuration: 10,
			Resolutions: []string{"1920x1080", "1280x720"},
			InputTypes:  []string{"image2video"},
		},
	}
}

func (p *BaiduAirProvider) Generate(ctx context.Context, req *GenerateRequest) (*VideoTask, error) {
	// Get settings from database
	var setting models.ProviderSetting
	err := database.DB.Where("provider = ? AND enabled = ?", "baidu-air", true).First(&setting).Error
	if err != nil {
		return nil, fmt.Errorf("baidu Air not configured")
	}

	apiURL := setting.BaseURL
	if apiURL == "" {
		apiURL = "https://qianfan.baidubce.com/video/generations"
	}

	payload := map[string]interface{}{
		"model": req.Model,
		"content": []map[string]interface{}{
			{"type": "text", "text": req.Prompt},
			{"type": "image_url", "image_url": map[string]string{"url": req.ImageURL}},
		},
	}

	jsonData, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+setting.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var result struct {
		TaskID string `json:"task_id"`
	}
	json.Unmarshal(body, &result)

	return &VideoTask{ExternalID: result.TaskID, Status: "processing"}, nil
}

func (p *BaiduAirProvider) GetTaskStatus(ctx context.Context, taskID string) (*VideoTask, error) {
	var setting models.ProviderSetting
	database.DB.Where("provider = ? AND enabled = ?", "baidu-air", true).First(&setting)

	url := fmt.Sprintf("https://qianfan.baidubce.com/video/generations/%s", taskID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+setting.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Status  string `json:"status"`
		Content struct {
			VideoURL string `json:"video_url"`
		} `json:"content"`
	}
	json.Unmarshal(body, &result)

	status := "processing"
	if result.Status == "SUCCEEDED" {
		status = "completed"
	} else if result.Status == "FAILED" {
		status = "failed"
	}

	return &VideoTask{
		Status:   status,
		VideoURL: result.Content.VideoURL,
	}, nil
}

// ============ Stability AI Provider ============

type StabilityProvider struct {
	db *gorm.DB
}

func NewStabilityProvider(db *gorm.DB) *StabilityProvider {
	return &StabilityProvider{db: db}
}

func (p *StabilityProvider) GetName() string {
	return "Stability AI"
}

func (p *StabilityProvider) GetModels() []VideoModel {
	return []VideoModel{
		{
			ID:          "stable-video-diffusion",
			Name:        "Stable Video Diffusion",
			Provider:    "stability",
			Description: "Image to video using SVD",
			MaxDuration: 4,
			Resolutions: []string{"1024x576", "576x1024"},
			InputTypes:  []string{"image2video"},
		},
	}
}

func (p *StabilityProvider) Generate(ctx context.Context, req *GenerateRequest) (*VideoTask, error) {
	var setting models.ProviderSetting
	err := database.DB.Where("provider = ? AND enabled = ?", "stability", true).First(&setting).Error
	if err != nil {
		return nil, fmt.Errorf("stability AI not configured")
	}

	// Stability AI video generation endpoint
	url := "https://api.stability.ai/v2beta/image-to-video"

	// Download image first
	imageResp, err := http.Get(req.ImageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer imageResp.Body.Close()
	imageData, _ := io.ReadAll(imageResp.Body)

	// Create multipart request
	var buf bytes.Buffer
	buf.Write(imageData)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, &buf)
	httpReq.Header.Set("Authorization", "Bearer "+setting.APIKey)
	httpReq.Header.Set("Content-Type", "image/png")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	json.Unmarshal(body, &result)

	return &VideoTask{ExternalID: result.ID, Status: "processing"}, nil
}

func (p *StabilityProvider) GetTaskStatus(ctx context.Context, taskID string) (*VideoTask, error) {
	var setting models.ProviderSetting
	database.DB.Where("provider = ? AND enabled = ?", "stability", true).First(&setting)

	url := fmt.Sprintf("https://api.stability.ai/v2beta/image-to-video/result/%s", taskID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+setting.APIKey)
	req.Header.Set("Accept", "video/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return &VideoTask{Status: "processing"}, nil
	}

	if resp.StatusCode == http.StatusOK {
		// Video is ready - in real implementation, save to storage
		return &VideoTask{
			Status:   "completed",
			VideoURL: fmt.Sprintf("/api/v1/video/download/%s", taskID),
		}, nil
	}

	return &VideoTask{Status: "failed", ErrorMsg: "Unknown error"}, nil
}

// ============ Local FFmpeg Provider ============

type LocalFFmpegProvider struct{}

func NewLocalFFmpegProvider() *LocalFFmpegProvider {
	return &LocalFFmpegProvider{}
}

func (p *LocalFFmpegProvider) GetName() string {
	return "Local (FFmpeg)"
}

func (p *LocalFFmpegProvider) GetModels() []VideoModel {
	return []VideoModel{
		{
			ID:          "image-slideshow",
			Name:        "Image Slideshow",
			Provider:    "local",
			Description: "Create video from images using FFmpeg",
			MaxDuration: 60,
			Resolutions: []string{"1920x1080", "1280x720", "720x480"},
			InputTypes:  []string{"image2video"},
		},
		{
			ID:          "text-animation",
			Name:        "Text Animation",
			Provider:    "local",
			Description: "Animated text video",
			MaxDuration: 30,
			Resolutions: []string{"1920x1080", "1280x720"},
			InputTypes:  []string{"text2video"},
		},
	}
}

func (p *LocalFFmpegProvider) Generate(ctx context.Context, req *GenerateRequest) (*VideoTask, error) {
	// Check if ffmpeg is available
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not installed")
	}

	taskID := uuid.New().String()

	// For image slideshow, we'd download the image and create a video
	// This is a simplified implementation
	go func() {
		// Simulate processing time
		time.Sleep(10 * time.Second)
		// In real implementation: download image, run ffmpeg, upload result
	}()

	return &VideoTask{
		ExternalID: taskID,
		Status:     "processing",
	}, nil
}

func (p *LocalFFmpegProvider) GetTaskStatus(ctx context.Context, taskID string) (*VideoTask, error) {
	// Check if output file exists (simplified)
	// In real implementation, track job status
	return &VideoTask{
		Status: "processing",
	}, nil
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Legacy support - GenerateResponse for backward compatibility
type GenerateResponse struct {
	ID     string `json:"id"`
	TaskID string `json:"task_id"`
}

func (r *GenerateResponse) GetTaskID() string {
	return r.TaskID
}
