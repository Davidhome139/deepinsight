package analytics

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"
)

// FeedbackRating represents user feedback on a response
type FeedbackRating struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	ConversationID string    `json:"conversation_id" gorm:"index"`
	MessageID      string    `json:"message_id" gorm:"index"`
	UserID         uint      `json:"user_id" gorm:"index"`
	Rating         int       `json:"rating"` // 1-5 stars
	Feedback       string    `json:"feedback"`
	Category       string    `json:"category"` // helpful, accurate, fast, unhelpful, inaccurate, slow
	CreatedAt      time.Time `json:"created_at"`
}

// QueryPattern tracks frequently asked queries
type QueryPattern struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	QueryHash    string    `json:"query_hash" gorm:"index"`
	QuerySummary string    `json:"query_summary"` // Shortened/anonymized version
	Category     string    `json:"category"`
	Count        int       `json:"count"`
	AvgRating    float64   `json:"avg_rating"`
	LastSeen     time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
}

// ErrorPattern tracks common error types
type ErrorPattern struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	ErrorType    string    `json:"error_type" gorm:"index"`
	ErrorMessage string    `json:"error_message"`
	Service      string    `json:"service"`
	Model        string    `json:"model"`
	Count        int       `json:"count"`
	LastSeen     time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
}

// DailyAnalytics stores daily aggregated analytics
type DailyAnalytics struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	Date          time.Time `json:"date" gorm:"index"`
	TotalRequests int64     `json:"total_requests"`
	TotalTokens   int64     `json:"total_tokens"`
	UniqueUsers   int64     `json:"unique_users"`
	AvgRating     float64   `json:"avg_rating"`
	ErrorRate     float64   `json:"error_rate"`
	AvgLatency    float64   `json:"avg_latency"`
	CreatedAt     time.Time `json:"created_at"`
}

// Collector handles real-time analytics collection
type Collector struct {
	db            *gorm.DB
	bufferMu      sync.Mutex
	feedbackBuf   []FeedbackRating
	queryBuf      map[string]*QueryPattern
	errorBuf      map[string]*ErrorPattern
	flushInterval time.Duration
	stopCh        chan struct{}
}

// NewCollector creates a new analytics collector
func NewCollector(db *gorm.DB) *Collector {
	c := &Collector{
		db:            db,
		feedbackBuf:   make([]FeedbackRating, 0, 100),
		queryBuf:      make(map[string]*QueryPattern),
		errorBuf:      make(map[string]*ErrorPattern),
		flushInterval: 5 * time.Minute,
		stopCh:        make(chan struct{}),
	}
	return c
}

// Start begins the background flush routine
func (c *Collector) Start(ctx context.Context) {
	go c.flushLoop(ctx)
}

// Stop stops the collector
func (c *Collector) Stop() {
	close(c.stopCh)
	c.flush() // Final flush
}

// flushLoop periodically flushes buffers to database
func (c *Collector) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// flush writes buffered data to database
func (c *Collector) flush() {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	// Flush feedback ratings
	if len(c.feedbackBuf) > 0 {
		c.db.CreateInBatches(c.feedbackBuf, 100)
		c.feedbackBuf = c.feedbackBuf[:0]
	}

	// Flush query patterns
	for _, qp := range c.queryBuf {
		c.db.Save(qp)
	}
	c.queryBuf = make(map[string]*QueryPattern)

	// Flush error patterns
	for _, ep := range c.errorBuf {
		c.db.Save(ep)
	}
	c.errorBuf = make(map[string]*ErrorPattern)
}

// RecordFeedback records user feedback on a response
func (c *Collector) RecordFeedback(conversationID, messageID string, userID uint, rating int, feedback, category string) {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	c.feedbackBuf = append(c.feedbackBuf, FeedbackRating{
		ID:             generateID(),
		ConversationID: conversationID,
		MessageID:      messageID,
		UserID:         userID,
		Rating:         rating,
		Feedback:       feedback,
		Category:       category,
		CreatedAt:      time.Now(),
	})
}

// RecordQuery tracks a query pattern
func (c *Collector) RecordQuery(queryHash, querySummary, category string) {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	if qp, exists := c.queryBuf[queryHash]; exists {
		qp.Count++
		qp.LastSeen = time.Now()
	} else {
		c.queryBuf[queryHash] = &QueryPattern{
			ID:           generateID(),
			QueryHash:    queryHash,
			QuerySummary: querySummary,
			Category:     category,
			Count:        1,
			LastSeen:     time.Now(),
			CreatedAt:    time.Now(),
		}
	}
}

// RecordError tracks an error pattern
func (c *Collector) RecordError(errorType, errorMessage, service, model string) {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	key := errorType + ":" + service + ":" + model
	if ep, exists := c.errorBuf[key]; exists {
		ep.Count++
		ep.LastSeen = time.Now()
	} else {
		c.errorBuf[key] = &ErrorPattern{
			ID:           generateID(),
			ErrorType:    errorType,
			ErrorMessage: truncate(errorMessage, 500),
			Service:      service,
			Model:        model,
			Count:        1,
			LastSeen:     time.Now(),
			CreatedAt:    time.Now(),
		}
	}
}

// GetFeedbackStats returns feedback statistics
func (c *Collector) GetFeedbackStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	var result struct {
		TotalFeedback int64   `json:"total_feedback"`
		AvgRating     float64 `json:"avg_rating"`
		PositiveCount int64   `json:"positive_count"`
		NegativeCount int64   `json:"negative_count"`
	}

	err := c.db.Model(&FeedbackRating{}).
		Select(`
			COUNT(*) as total_feedback,
			COALESCE(AVG(rating), 0) as avg_rating,
			SUM(CASE WHEN rating >= 4 THEN 1 ELSE 0 END) as positive_count,
			SUM(CASE WHEN rating <= 2 THEN 1 ELSE 0 END) as negative_count
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_feedback": result.TotalFeedback,
		"avg_rating":     result.AvgRating,
		"positive_count": result.PositiveCount,
		"negative_count": result.NegativeCount,
		"positive_rate":  float64(result.PositiveCount) / float64(max(result.TotalFeedback, 1)) * 100,
	}, nil
}

// GetTopQueries returns most frequent queries
func (c *Collector) GetTopQueries(limit int) ([]QueryPattern, error) {
	var queries []QueryPattern
	err := c.db.Model(&QueryPattern{}).
		Order("count DESC").
		Limit(limit).
		Find(&queries).Error
	return queries, err
}

// GetTopErrors returns most frequent errors
func (c *Collector) GetTopErrors(limit int) ([]ErrorPattern, error) {
	var errors []ErrorPattern
	err := c.db.Model(&ErrorPattern{}).
		Order("count DESC").
		Limit(limit).
		Find(&errors).Error
	return errors, err
}

// GetLowRatedResponses returns responses with low ratings
func (c *Collector) GetLowRatedResponses(threshold int, limit int) ([]FeedbackRating, error) {
	var ratings []FeedbackRating
	err := c.db.Model(&FeedbackRating{}).
		Where("rating <= ?", threshold).
		Order("created_at DESC").
		Limit(limit).
		Find(&ratings).Error
	return ratings, err
}

// AggregateDailyAnalytics creates daily summary
func (c *Collector) AggregateDailyAnalytics(date time.Time) error {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Get usage stats
	var usageStats struct {
		TotalRequests int64   `json:"total_requests"`
		TotalTokens   int64   `json:"total_tokens"`
		UniqueUsers   int64   `json:"unique_users"`
		AvgLatency    float64 `json:"avg_latency"`
		ErrorRate     float64 `json:"error_rate"`
	}

	c.db.Table("usage_records").
		Select(`
			COUNT(*) as total_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(AVG(latency), 0) as avg_latency,
			COALESCE(AVG(CASE WHEN NOT success THEN 100.0 ELSE 0.0 END), 0) as error_rate
		`).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Scan(&usageStats)

	// Get average rating
	var avgRating float64
	c.db.Model(&FeedbackRating{}).
		Select("COALESCE(AVG(rating), 0)").
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Scan(&avgRating)

	// Upsert daily analytics
	daily := DailyAnalytics{
		ID:            generateID(),
		Date:          startOfDay,
		TotalRequests: usageStats.TotalRequests,
		TotalTokens:   usageStats.TotalTokens,
		UniqueUsers:   usageStats.UniqueUsers,
		AvgRating:     avgRating,
		ErrorRate:     usageStats.ErrorRate,
		AvgLatency:    usageStats.AvgLatency,
		CreatedAt:     time.Now(),
	}

	return c.db.Save(&daily).Error
}

// Helper functions

func generateID() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
