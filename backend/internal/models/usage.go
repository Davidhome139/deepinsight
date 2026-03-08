package models

import (
	"time"
)

// UsageRecord tracks API usage for analytics
type UsageRecord struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"index"`
	Service      string    `json:"service" gorm:"index"` // chat, ai-chat, image, tts, rag
	Model        string    `json:"model" gorm:"index"`
	Provider     string    `json:"provider"` // deepseek, qwen, hunyuan, openai
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	Cost         float64   `json:"cost"`    // Estimated cost in USD
	Latency      int64     `json:"latency"` // Response time in ms
	Success      bool      `json:"success"`
	ErrorMsg     string    `json:"error_msg,omitempty"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

// UsageSummary aggregated usage statistics
type UsageSummary struct {
	Service      string  `json:"service"`
	Model        string  `json:"model"`
	Provider     string  `json:"provider"`
	TotalCalls   int64   `json:"total_calls"`
	SuccessCalls int64   `json:"success_calls"`
	FailedCalls  int64   `json:"failed_calls"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
	AvgLatency   float64 `json:"avg_latency"`
}

// DailyUsage usage statistics per day
type DailyUsage struct {
	Date        string  `json:"date"`
	Service     string  `json:"service"`
	TotalCalls  int64   `json:"total_calls"`
	TotalTokens int64   `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`
}

// CostBreakdown cost breakdown by service/model
type CostBreakdown struct {
	Service  string  `json:"service"`
	Model    string  `json:"model"`
	Provider string  `json:"provider"`
	Cost     float64 `json:"cost"`
	Percent  float64 `json:"percent"`
}

// ModelPricing defines pricing per 1M tokens
var ModelPricing = map[string]struct {
	InputPer1M  float64
	OutputPer1M float64
}{
	"deepseek-chat":     {InputPer1M: 0.14, OutputPer1M: 0.28},
	"deepseek-reasoner": {InputPer1M: 0.55, OutputPer1M: 2.19},
	"qwen-turbo":        {InputPer1M: 0.30, OutputPer1M: 0.60},
	"qwen-plus":         {InputPer1M: 0.80, OutputPer1M: 2.00},
	"qwen-max":          {InputPer1M: 2.00, OutputPer1M: 6.00},
	"hunyuan-lite":      {InputPer1M: 0.00, OutputPer1M: 0.00}, // Free tier
	"hunyuan-standard":  {InputPer1M: 0.50, OutputPer1M: 1.00},
	"hunyuan-pro":       {InputPer1M: 3.00, OutputPer1M: 9.00},
	"gpt-4o":            {InputPer1M: 2.50, OutputPer1M: 10.00},
	"gpt-4o-mini":       {InputPer1M: 0.15, OutputPer1M: 0.60},
	"dall-e-3":          {InputPer1M: 0.00, OutputPer1M: 40.00}, // Per image ($0.04)
	"tts-1":             {InputPer1M: 15.00, OutputPer1M: 0.00}, // Per 1M chars
	"tts-1-hd":          {InputPer1M: 30.00, OutputPer1M: 0.00},
}

// CalculateCost calculates the cost for a usage record
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := ModelPricing[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1000000 * pricing.InputPer1M
	outputCost := float64(outputTokens) / 1000000 * pricing.OutputPer1M
	return inputCost + outputCost
}
