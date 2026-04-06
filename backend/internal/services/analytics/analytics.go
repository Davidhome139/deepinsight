package analytics

import (
	"backend/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnalyticsService struct {
	db *gorm.DB
}

func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

// RecordUsage records a single API usage event
func (s *AnalyticsService) RecordUsage(userID uint, service, model, provider string, inputTokens, outputTokens int, latencyMs int64, success bool, errorMsg string) error {
	record := models.UsageRecord{
		ID:           uuid.New().String(),
		UserID:       userID,
		Service:      service,
		Model:        model,
		Provider:     provider,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Cost:         models.CalculateCost(model, inputTokens, outputTokens),
		Latency:      latencyMs,
		Success:      success,
		ErrorMsg:     errorMsg,
		CreatedAt:    time.Now(),
	}
	return s.db.Create(&record).Error
}

// GetUsageSummary returns aggregated usage statistics
func (s *AnalyticsService) GetUsageSummary(userID uint, startDate, endDate time.Time) ([]models.UsageSummary, error) {
	var summaries []models.UsageSummary

	query := s.db.Model(&models.UsageRecord{}).
		Select(`
			service,
			model,
			provider,
			COUNT(*) as total_calls,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_calls,
			SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failed_calls,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			AVG(latency) as avg_latency
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("service, model, provider")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Scan(&summaries).Error
	return summaries, err
}

// GetDailyUsage returns daily usage statistics
func (s *AnalyticsService) GetDailyUsage(userID uint, startDate, endDate time.Time) ([]models.DailyUsage, error) {
	var dailyUsage []models.DailyUsage

	query := s.db.Model(&models.UsageRecord{}).
		Select(`
			TO_CHAR(created_at, 'YYYY-MM-DD') as date,
			service,
			COUNT(*) as total_calls,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("TO_CHAR(created_at, 'YYYY-MM-DD'), service").
		Order("date")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Scan(&dailyUsage).Error
	return dailyUsage, err
}

// GetCostBreakdown returns cost breakdown by service/model
func (s *AnalyticsService) GetCostBreakdown(userID uint, startDate, endDate time.Time) ([]models.CostBreakdown, error) {
	var breakdown []models.CostBreakdown
	var totalCost float64

	// Get total cost first
	totalQuery := s.db.Model(&models.UsageRecord{}).
		Select("COALESCE(SUM(cost), 0)").
		Where("created_at BETWEEN ? AND ?", startDate, endDate)
	if userID > 0 {
		totalQuery = totalQuery.Where("user_id = ?", userID)
	}
	totalQuery.Scan(&totalCost)

	// Get breakdown
	query := s.db.Model(&models.UsageRecord{}).
		Select(`
			service,
			model,
			provider,
			SUM(cost) as cost
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("service, model, provider").
		Order("cost DESC")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Scan(&breakdown).Error
	if err != nil {
		return nil, err
	}

	// Calculate percentages
	for i := range breakdown {
		if totalCost > 0 {
			breakdown[i].Percent = (breakdown[i].Cost / totalCost) * 100
		}
	}

	return breakdown, nil
}

// GetTotalStats returns overall statistics
func (s *AnalyticsService) GetTotalStats(userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	var result struct {
		TotalCalls  int64   `json:"total_calls"`
		TotalTokens int64   `json:"total_tokens"`
		TotalCost   float64 `json:"total_cost"`
		SuccessRate float64 `json:"success_rate"`
		AvgLatency  float64 `json:"avg_latency"`
	}

	query := s.db.Model(&models.UsageRecord{}).
		Select(`
			COUNT(*) as total_calls,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(AVG(CASE WHEN success THEN 100.0 ELSE 0.0 END), 0) as success_rate,
			COALESCE(AVG(latency), 0) as avg_latency
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate)

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_calls":  result.TotalCalls,
		"total_tokens": result.TotalTokens,
		"total_cost":   result.TotalCost,
		"success_rate": result.SuccessRate,
		"avg_latency":  result.AvgLatency,
	}, nil
}

// GetRecentUsage returns recent usage records
func (s *AnalyticsService) GetRecentUsage(userID uint, limit int) ([]models.UsageRecord, error) {
	var records []models.UsageRecord

	query := s.db.Model(&models.UsageRecord{}).
		Order("created_at DESC").
		Limit(limit)

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Find(&records).Error
	return records, err
}
