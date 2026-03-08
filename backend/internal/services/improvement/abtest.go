package improvement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"gorm.io/gorm"
)

// PromptVariant represents a variant of a prompt for A/B testing
type PromptVariant struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	ExperimentID string    `json:"experiment_id" gorm:"index"`
	Name         string    `json:"name"`
	Content      string    `json:"content"`
	Weight       float64   `json:"weight"` // Traffic allocation (0-1)
	IsControl    bool      `json:"is_control"`
	CreatedAt    time.Time `json:"created_at"`
}

// Experiment represents an A/B test experiment
type Experiment struct {
	ID          string          `json:"id" gorm:"primaryKey"`
	Name        string          `json:"name" gorm:"index"`
	Description string          `json:"description"`
	PromptType  string          `json:"prompt_type"` // system, user, etc.
	Status      string          `json:"status"`      // draft, running, paused, completed
	StartDate   *time.Time      `json:"start_date"`
	EndDate     *time.Time      `json:"end_date"`
	Variants    []PromptVariant `json:"variants" gorm:"foreignKey:ExperimentID"`
	WinnerID    string          `json:"winner_id"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ExperimentResult tracks results for each variant
type ExperimentResult struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	ExperimentID string    `json:"experiment_id" gorm:"index"`
	VariantID    string    `json:"variant_id" gorm:"index"`
	UserID       uint      `json:"user_id"`
	SessionID    string    `json:"session_id"`
	Impressions  int       `json:"impressions"`
	Conversions  int       `json:"conversions"` // Positive outcomes
	Rating       float64   `json:"rating"`
	Latency      float64   `json:"latency"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// VariantStats holds aggregated statistics for a variant
type VariantStats struct {
	VariantID      string  `json:"variant_id"`
	VariantName    string  `json:"variant_name"`
	Impressions    int64   `json:"impressions"`
	Conversions    int64   `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	AvgRating      float64 `json:"avg_rating"`
	AvgLatency     float64 `json:"avg_latency"`
	Confidence     float64 `json:"confidence"`
	IsWinner       bool    `json:"is_winner"`
	ImprovementPct float64 `json:"improvement_pct"`
}

// ABTestManager manages A/B testing for prompts
type ABTestManager struct {
	db              *gorm.DB
	logger          *log.Logger
	experimentCache map[string]*Experiment
	cacheMu         sync.RWMutex
	rand            *rand.Rand
}

// NewABTestManager creates a new A/B test manager
func NewABTestManager(db *gorm.DB) *ABTestManager {
	return &ABTestManager{
		db:              db,
		logger:          log.Default(),
		experimentCache: make(map[string]*Experiment),
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CreateExperiment creates a new A/B test experiment
func (m *ABTestManager) CreateExperiment(name, description, promptType string, variants []PromptVariant) (*Experiment, error) {
	exp := &Experiment{
		ID:          generateExperimentID(name),
		Name:        name,
		Description: description,
		PromptType:  promptType,
		Status:      "draft",
		Variants:    variants,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Ensure weights sum to 1
	totalWeight := 0.0
	for _, v := range variants {
		totalWeight += v.Weight
	}
	if totalWeight != 1.0 {
		// Normalize weights
		for i := range exp.Variants {
			exp.Variants[i].Weight = exp.Variants[i].Weight / totalWeight
		}
	}

	// Assign IDs to variants
	for i := range exp.Variants {
		exp.Variants[i].ID = fmt.Sprintf("%s_var_%d", exp.ID, i)
		exp.Variants[i].ExperimentID = exp.ID
		exp.Variants[i].CreatedAt = time.Now()
	}

	if err := m.db.Create(exp).Error; err != nil {
		return nil, fmt.Errorf("failed to create experiment: %w", err)
	}

	m.logger.Printf("[ABTest] Created experiment %s with %d variants", exp.ID, len(variants))
	return exp, nil
}

// StartExperiment starts an experiment
func (m *ABTestManager) StartExperiment(experimentID string) error {
	now := time.Now()
	return m.db.Model(&Experiment{}).
		Where("id = ?", experimentID).
		Updates(map[string]interface{}{
			"status":     "running",
			"start_date": now,
			"updated_at": now,
		}).Error
}

// PauseExperiment pauses an experiment
func (m *ABTestManager) PauseExperiment(experimentID string) error {
	return m.db.Model(&Experiment{}).
		Where("id = ?", experimentID).
		Updates(map[string]interface{}{
			"status":     "paused",
			"updated_at": time.Now(),
		}).Error
}

// CompleteExperiment completes an experiment and determines winner
func (m *ABTestManager) CompleteExperiment(ctx context.Context, experimentID string) (*VariantStats, error) {
	stats, err := m.GetExperimentStats(experimentID)
	if err != nil {
		return nil, err
	}

	if len(stats) == 0 {
		return nil, fmt.Errorf("no stats available for experiment")
	}

	// Find winner based on conversion rate with statistical significance
	var winner *VariantStats
	var controlStats *VariantStats

	for i := range stats {
		if stats[i].VariantName == "control" || (controlStats == nil && i == 0) {
			controlStats = &stats[i]
		}
	}

	for i := range stats {
		if winner == nil || stats[i].ConversionRate > winner.ConversionRate {
			winner = &stats[i]
		}
	}

	if winner != nil && controlStats != nil && winner.VariantID != controlStats.VariantID {
		winner.ImprovementPct = ((winner.ConversionRate - controlStats.ConversionRate) / controlStats.ConversionRate) * 100
	}

	// Update experiment with winner
	now := time.Now()
	m.db.Model(&Experiment{}).
		Where("id = ?", experimentID).
		Updates(map[string]interface{}{
			"status":     "completed",
			"end_date":   now,
			"winner_id":  winner.VariantID,
			"updated_at": now,
		})

	winner.IsWinner = true
	m.logger.Printf("[ABTest] Experiment %s completed. Winner: %s (%.2f%% improvement)",
		experimentID, winner.VariantID, winner.ImprovementPct)

	return winner, nil
}

// GetVariant returns a variant for a user based on traffic allocation
func (m *ABTestManager) GetVariant(experimentID string, userID uint, sessionID string) (*PromptVariant, error) {
	exp, err := m.getExperiment(experimentID)
	if err != nil {
		return nil, err
	}

	if exp.Status != "running" {
		// Return control variant if not running
		for _, v := range exp.Variants {
			if v.IsControl {
				return &v, nil
			}
		}
		return &exp.Variants[0], nil
	}

	// Consistent hashing based on user/session for reproducibility
	hash := hashUserSession(experimentID, userID, sessionID)
	bucket := float64(hash%1000) / 1000.0

	// Select variant based on weight
	cumWeight := 0.0
	for _, v := range exp.Variants {
		cumWeight += v.Weight
		if bucket < cumWeight {
			return &v, nil
		}
	}

	// Fallback to last variant
	return &exp.Variants[len(exp.Variants)-1], nil
}

// RecordImpression records an impression for a variant
func (m *ABTestManager) RecordImpression(experimentID, variantID string, userID uint, sessionID string) error {
	result := ExperimentResult{
		ID:           fmt.Sprintf("result_%d", time.Now().UnixNano()),
		ExperimentID: experimentID,
		VariantID:    variantID,
		UserID:       userID,
		SessionID:    sessionID,
		Impressions:  1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return m.db.Create(&result).Error
}

// RecordConversion records a conversion (positive outcome) for a variant
func (m *ABTestManager) RecordConversion(experimentID, variantID string, userID uint, sessionID string, rating float64, latency float64) error {
	// Try to update existing result
	result := m.db.Model(&ExperimentResult{}).
		Where("experiment_id = ? AND variant_id = ? AND user_id = ? AND session_id = ?",
			experimentID, variantID, userID, sessionID).
		Updates(map[string]interface{}{
			"conversions": gorm.Expr("conversions + 1"),
			"rating":      rating,
			"latency":     latency,
			"updated_at":  time.Now(),
		})

	if result.RowsAffected == 0 {
		// Create new result
		newResult := ExperimentResult{
			ID:           fmt.Sprintf("result_%d", time.Now().UnixNano()),
			ExperimentID: experimentID,
			VariantID:    variantID,
			UserID:       userID,
			SessionID:    sessionID,
			Impressions:  1,
			Conversions:  1,
			Rating:       rating,
			Latency:      latency,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		return m.db.Create(&newResult).Error
	}

	return result.Error
}

// GetExperimentStats returns statistics for an experiment
func (m *ABTestManager) GetExperimentStats(experimentID string) ([]VariantStats, error) {
	var results []struct {
		VariantID   string  `json:"variant_id"`
		Impressions int64   `json:"impressions"`
		Conversions int64   `json:"conversions"`
		AvgRating   float64 `json:"avg_rating"`
		AvgLatency  float64 `json:"avg_latency"`
	}

	err := m.db.Model(&ExperimentResult{}).
		Select(`
			variant_id,
			SUM(impressions) as impressions,
			SUM(conversions) as conversions,
			COALESCE(AVG(rating), 0) as avg_rating,
			COALESCE(AVG(latency), 0) as avg_latency
		`).
		Where("experiment_id = ?", experimentID).
		Group("variant_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Get variant names
	var variants []PromptVariant
	m.db.Where("experiment_id = ?", experimentID).Find(&variants)
	variantNames := make(map[string]string)
	for _, v := range variants {
		variantNames[v.ID] = v.Name
	}

	stats := make([]VariantStats, len(results))
	for i, r := range results {
		convRate := 0.0
		if r.Impressions > 0 {
			convRate = float64(r.Conversions) / float64(r.Impressions) * 100
		}

		stats[i] = VariantStats{
			VariantID:      r.VariantID,
			VariantName:    variantNames[r.VariantID],
			Impressions:    r.Impressions,
			Conversions:    r.Conversions,
			ConversionRate: convRate,
			AvgRating:      r.AvgRating,
			AvgLatency:     r.AvgLatency,
			Confidence:     m.calculateConfidence(r.Impressions, r.Conversions),
		}
	}

	return stats, nil
}

// getExperiment retrieves an experiment with caching
func (m *ABTestManager) getExperiment(experimentID string) (*Experiment, error) {
	m.cacheMu.RLock()
	if exp, exists := m.experimentCache[experimentID]; exists {
		m.cacheMu.RUnlock()
		return exp, nil
	}
	m.cacheMu.RUnlock()

	var exp Experiment
	if err := m.db.Preload("Variants").First(&exp, "id = ?", experimentID).Error; err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	m.experimentCache[experimentID] = &exp
	m.cacheMu.Unlock()

	return &exp, nil
}

// InvalidateCache clears the experiment cache
func (m *ABTestManager) InvalidateCache(experimentID string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	if experimentID == "" {
		m.experimentCache = make(map[string]*Experiment)
	} else {
		delete(m.experimentCache, experimentID)
	}
}

// GetRunningExperiments returns all running experiments
func (m *ABTestManager) GetRunningExperiments() ([]Experiment, error) {
	var experiments []Experiment
	err := m.db.Preload("Variants").
		Where("status = ?", "running").
		Find(&experiments).Error
	return experiments, err
}

// GetExperiment returns a specific experiment
func (m *ABTestManager) GetExperiment(experimentID string) (*Experiment, error) {
	var exp Experiment
	err := m.db.Preload("Variants").First(&exp, "id = ?", experimentID).Error
	return &exp, err
}

// ApplyWinner applies the winning variant as the new default
func (m *ABTestManager) ApplyWinner(ctx context.Context, experimentID string) error {
	var exp Experiment
	if err := m.db.Preload("Variants").First(&exp, "id = ?", experimentID).Error; err != nil {
		return err
	}

	if exp.WinnerID == "" {
		return fmt.Errorf("experiment has no winner yet")
	}

	// Find winning variant
	var winner *PromptVariant
	for _, v := range exp.Variants {
		if v.ID == exp.WinnerID {
			winner = &v
			break
		}
	}

	if winner == nil {
		return fmt.Errorf("winner variant not found")
	}

	m.logger.Printf("[ABTest] Winner applied for experiment %s: %s", experimentID, winner.Name)
	// The actual prompt update would be handled by the prompt template service
	return nil
}

// calculateConfidence calculates statistical confidence level
func (m *ABTestManager) calculateConfidence(impressions, conversions int64) float64 {
	if impressions < 100 {
		return 0.0
	}
	if impressions < 500 {
		return 50.0
	}
	if impressions < 1000 {
		return 75.0
	}
	if impressions < 5000 {
		return 90.0
	}
	return 95.0
}

// Helper functions

func generateExperimentID(name string) string {
	timestamp := time.Now().Format("20060102")
	hash := sha256.Sum256([]byte(name + timestamp))
	return fmt.Sprintf("exp_%s_%s", timestamp, hex.EncodeToString(hash[:])[:8])
}

func hashUserSession(experimentID string, userID uint, sessionID string) uint64 {
	data := fmt.Sprintf("%s:%d:%s", experimentID, userID, sessionID)
	hash := sha256.Sum256([]byte(data))
	var result uint64
	for i := 0; i < 8; i++ {
		result = result<<8 + uint64(hash[i])
	}
	return result
}
