package llm

import (
	"context"
	"fmt"
	"strings"

	"backend/internal/models"
	"backend/internal/services/ai"
)

// Client is the interface for LLM operations
type Client interface {
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error)
	StreamComplete(ctx context.Context, prompt string, opts CompletionOptions, callback func(string)) error
}

// CompletionOptions contains options for completion
type CompletionOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
	TopP        float64
	Stop        []string
	UserID      uint // For retrieving user-specific API keys
}

// AIClient wraps the AI service for use by the agent system
type AIClient struct {
	service      ai.AIService
	defaultModel string
	userID       uint
}

// NewClient creates a new LLM client using the configured AI service
func NewClient() Client {
	return &AIClient{
		service:      nil,                 // Will be set later via SetService
		defaultModel: "deepseek-reasoner", // Default to DeepSeek reasoning model
		userID:       0,
	}
}

// NewClientWithService creates a new LLM client with the given AI service
func NewClientWithService(service ai.AIService) Client {
	return &AIClient{
		service:      service,
		defaultModel: "deepseek-reasoner",
		userID:       0,
	}
}

// SetService sets the AI service (called during initialization)
func (c *AIClient) SetService(service ai.AIService) {
	c.service = service
}

// SetUserID sets the current user ID for API key lookup
func (c *AIClient) SetUserID(userID uint) {
	c.userID = userID
}

// Complete generates a completion using the AI service
func (c *AIClient) Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error) {
	if c.service == nil {
		return "", fmt.Errorf("AI service not configured")
	}

	model := opts.Model
	if model == "" {
		model = c.defaultModel
	}

	// Create messages from prompt
	messages := []models.Message{
		{Role: "user", Content: prompt},
	}

	req := &ai.ChatRequest{
		Messages:  messages,
		Model:     model,
		Stream:    false,
		MaxTokens: opts.MaxTokens,
		UserID:    c.userID,
	}

	if opts.Temperature > 0 {
		req.Temperature = opts.Temperature
	}

	ch, err := c.service.ChatStream(ctx, req)
	if err != nil {
		return "", err
	}

	// Collect all chunks
	var result strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		result.WriteString(chunk.Content)
	}

	return result.String(), nil
}

// StreamComplete generates a streaming completion using the AI service
func (c *AIClient) StreamComplete(ctx context.Context, prompt string, opts CompletionOptions, callback func(string)) error {
	if c.service == nil {
		return fmt.Errorf("AI service not configured")
	}

	model := opts.Model
	if model == "" {
		model = c.defaultModel
	}

	// Create messages from prompt
	messages := []models.Message{
		{Role: "user", Content: prompt},
	}

	req := &ai.ChatRequest{
		Messages:  messages,
		Model:     model,
		Stream:    true,
		MaxTokens: opts.MaxTokens,
		UserID:    c.userID,
	}

	if opts.Temperature > 0 {
		req.Temperature = opts.Temperature
	}

	ch, err := c.service.ChatStream(ctx, req)
	if err != nil {
		return err
	}

	// Stream chunks to callback
	for chunk := range ch {
		if chunk.Done {
			break
		}
		callback(chunk.Content)
	}

	return nil
}

// GetDefaultModel returns the default model (DeepSeek reasoning)
func (c *AIClient) GetDefaultModel() string {
	return c.defaultModel
}

// SetModel changes the default model
func (c *AIClient) SetModel(model string) {
	c.defaultModel = model
}
