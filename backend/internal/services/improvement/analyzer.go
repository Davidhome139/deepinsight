package improvement

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AnalysisResult contains the result of feedback analysis
type AnalysisResult struct {
	ID              string            `json:"id" gorm:"primaryKey"`
	AnalysisType    string            `json:"analysis_type"` // weekly, monthly, custom
	StartDate       time.Time         `json:"start_date"`
	EndDate         time.Time         `json:"end_date"`
	Summary         string            `json:"summary"`
	Issues          []IdentifiedIssue `json:"issues" gorm:"serializer:json"`
	Recommendations []Recommendation  `json:"recommendations" gorm:"serializer:json"`
	Metrics         AnalysisMetrics   `json:"metrics" gorm:"serializer:json"`
	CreatedAt       time.Time         `json:"created_at"`
}

// IdentifiedIssue represents a problem found during analysis
type IdentifiedIssue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"` // error, performance, quality, usability
	Severity    string   `json:"severity"` // critical, high, medium, low
	Frequency   int      `json:"frequency"`
	Impact      string   `json:"impact"`
	Examples    []string `json:"examples"`
}

// Recommendation represents a suggested improvement
type Recommendation struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"` // high, medium, low
	Effort      string   `json:"effort"`   // small, medium, large
	IssueRefs   []string `json:"issue_refs"`
	ActionItems []string `json:"action_items"`
}

// AnalysisMetrics contains quantitative metrics from analysis
type AnalysisMetrics struct {
	TotalRequests     int64    `json:"total_requests"`
	TotalErrors       int64    `json:"total_errors"`
	ErrorRate         float64  `json:"error_rate"`
	AvgRating         float64  `json:"avg_rating"`
	AvgLatency        float64  `json:"avg_latency"`
	UniqueUsers       int64    `json:"unique_users"`
	LowRatedResponses int64    `json:"low_rated_responses"`
	TopErrorTypes     []string `json:"top_error_types"`
}

// LLMClient interface for AI-powered analysis
type LLMClient interface {
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// Analyzer performs feedback analysis using AI
type Analyzer struct {
	db        *gorm.DB
	llmClient LLMClient
	logger    *log.Logger
}

// NewAnalyzer creates a new feedback analyzer
func NewAnalyzer(db *gorm.DB, llmClient LLMClient) *Analyzer {
	return &Analyzer{
		db:        db,
		llmClient: llmClient,
		logger:    log.Default(),
	}
}

// AnalyzeWeekly performs weekly feedback analysis
func (a *Analyzer) AnalyzeWeekly(ctx context.Context) (*AnalysisResult, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)
	return a.Analyze(ctx, "weekly", startDate, endDate)
}

// AnalyzeMonthly performs monthly feedback analysis
func (a *Analyzer) AnalyzeMonthly(ctx context.Context) (*AnalysisResult, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -1, 0)
	return a.Analyze(ctx, "monthly", startDate, endDate)
}

// Analyze performs feedback analysis for a given period
func (a *Analyzer) Analyze(ctx context.Context, analysisType string, startDate, endDate time.Time) (*AnalysisResult, error) {
	a.logger.Printf("[Analyzer] Starting %s analysis from %s to %s", analysisType, startDate, endDate)

	// Gather metrics
	metrics, err := a.gatherMetrics(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to gather metrics: %w", err)
	}

	// Get error patterns
	errorPatterns, err := a.getErrorPatterns(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get error patterns: %w", err)
	}

	// Get low-rated responses
	lowRatedSamples, err := a.getLowRatedSamples(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get low rated samples: %w", err)
	}

	// Use LLM to analyze and generate insights
	issues, recommendations, summary, err := a.generateInsights(ctx, metrics, errorPatterns, lowRatedSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to generate insights: %w", err)
	}

	result := &AnalysisResult{
		ID:              fmt.Sprintf("analysis_%s_%d", analysisType, time.Now().Unix()),
		AnalysisType:    analysisType,
		StartDate:       startDate,
		EndDate:         endDate,
		Summary:         summary,
		Issues:          issues,
		Recommendations: recommendations,
		Metrics:         *metrics,
		CreatedAt:       time.Now(),
	}

	// Save result to database
	if err := a.db.Create(result).Error; err != nil {
		a.logger.Printf("[Analyzer] Warning: failed to save analysis result: %v", err)
	}

	a.logger.Printf("[Analyzer] Analysis complete. Found %d issues, %d recommendations", len(issues), len(recommendations))
	return result, nil
}

// gatherMetrics collects quantitative metrics
func (a *Analyzer) gatherMetrics(ctx context.Context, startDate, endDate time.Time) (*AnalysisMetrics, error) {
	var metrics AnalysisMetrics

	// Get usage stats
	a.db.WithContext(ctx).Table("usage_records").
		Select(`
			COUNT(*) as total_requests,
			SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as total_errors,
			COALESCE(AVG(CASE WHEN NOT success THEN 100.0 ELSE 0.0 END), 0) as error_rate,
			COALESCE(AVG(latency), 0) as avg_latency,
			COUNT(DISTINCT user_id) as unique_users
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Scan(&metrics)

	// Get average rating
	a.db.WithContext(ctx).Table("feedback_ratings").
		Select("COALESCE(AVG(rating), 0) as avg_rating").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Scan(&metrics)

	// Count low-rated responses
	a.db.WithContext(ctx).Table("feedback_ratings").
		Select("COUNT(*) as low_rated_responses").
		Where("created_at BETWEEN ? AND ? AND rating <= 2", startDate, endDate).
		Scan(&metrics)

	// Get top error types
	var topErrors []struct {
		ErrorType string `json:"error_type"`
		Count     int    `json:"count"`
	}
	a.db.WithContext(ctx).Table("error_patterns").
		Select("error_type, count").
		Where("last_seen BETWEEN ? AND ?", startDate, endDate).
		Order("count DESC").
		Limit(5).
		Scan(&topErrors)

	for _, e := range topErrors {
		metrics.TopErrorTypes = append(metrics.TopErrorTypes, fmt.Sprintf("%s (%d)", e.ErrorType, e.Count))
	}

	return &metrics, nil
}

// getErrorPatterns retrieves error patterns for analysis
func (a *Analyzer) getErrorPatterns(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	var patterns []struct {
		ErrorType    string `json:"error_type"`
		ErrorMessage string `json:"error_message"`
		Service      string `json:"service"`
		Model        string `json:"model"`
		Count        int    `json:"count"`
	}

	err := a.db.WithContext(ctx).Table("error_patterns").
		Where("last_seen BETWEEN ? AND ?", startDate, endDate).
		Order("count DESC").
		Limit(20).
		Scan(&patterns).Error

	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(patterns))
	for i, p := range patterns {
		result[i] = map[string]interface{}{
			"error_type":    p.ErrorType,
			"error_message": p.ErrorMessage,
			"service":       p.Service,
			"model":         p.Model,
			"count":         p.Count,
		}
	}

	return result, nil
}

// getLowRatedSamples retrieves samples of low-rated responses
func (a *Analyzer) getLowRatedSamples(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	var samples []struct {
		Rating   int    `json:"rating"`
		Feedback string `json:"feedback"`
		Category string `json:"category"`
	}

	err := a.db.WithContext(ctx).Table("feedback_ratings").
		Select("rating, feedback, category").
		Where("created_at BETWEEN ? AND ? AND rating <= 2", startDate, endDate).
		Order("created_at DESC").
		Limit(50).
		Scan(&samples).Error

	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(samples))
	for i, s := range samples {
		result[i] = map[string]interface{}{
			"rating":   s.Rating,
			"feedback": s.Feedback,
			"category": s.Category,
		}
	}

	return result, nil
}

// generateInsights uses LLM to analyze data and generate insights
func (a *Analyzer) generateInsights(ctx context.Context, metrics *AnalysisMetrics, errorPatterns, lowRatedSamples []map[string]interface{}) ([]IdentifiedIssue, []Recommendation, string, error) {
	if a.llmClient == nil {
		// Fallback to rule-based analysis
		return a.generateRuleBasedInsights(metrics, errorPatterns, lowRatedSamples)
	}

	// Prepare data for LLM
	dataJSON, _ := json.Marshal(map[string]interface{}{
		"metrics":           metrics,
		"error_patterns":    errorPatterns,
		"low_rated_samples": lowRatedSamples,
	})

	systemPrompt := `You are an AI system improvement analyst. Analyze the provided metrics, error patterns, and user feedback to identify issues and recommend improvements.

Output your analysis in the following JSON format:
{
  "summary": "Brief summary of findings",
  "issues": [
    {
      "id": "issue_1",
      "title": "Issue title",
      "description": "Detailed description",
      "category": "error|performance|quality|usability",
      "severity": "critical|high|medium|low",
      "frequency": 10,
      "impact": "Impact description",
      "examples": ["example1", "example2"]
    }
  ],
  "recommendations": [
    {
      "id": "rec_1",
      "title": "Recommendation title",
      "description": "Detailed description",
      "priority": "high|medium|low",
      "effort": "small|medium|large",
      "issue_refs": ["issue_1"],
      "action_items": ["Action 1", "Action 2"]
    }
  ]
}`

	userPrompt := fmt.Sprintf("Analyze this data and provide improvement insights:\n\n%s", string(dataJSON))

	response, err := a.llmClient.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		a.logger.Printf("[Analyzer] LLM analysis failed, falling back to rules: %v", err)
		return a.generateRuleBasedInsights(metrics, errorPatterns, lowRatedSamples)
	}

	// Parse LLM response
	var result struct {
		Summary         string            `json:"summary"`
		Issues          []IdentifiedIssue `json:"issues"`
		Recommendations []Recommendation  `json:"recommendations"`
	}

	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		response = response[jsonStart : jsonEnd+1]
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		a.logger.Printf("[Analyzer] Failed to parse LLM response, falling back to rules: %v", err)
		return a.generateRuleBasedInsights(metrics, errorPatterns, lowRatedSamples)
	}

	return result.Issues, result.Recommendations, result.Summary, nil
}

// generateRuleBasedInsights generates insights using predefined rules
func (a *Analyzer) generateRuleBasedInsights(metrics *AnalysisMetrics, errorPatterns, lowRatedSamples []map[string]interface{}) ([]IdentifiedIssue, []Recommendation, string, error) {
	var issues []IdentifiedIssue
	var recommendations []Recommendation
	var summaryParts []string

	issueCounter := 0

	// Check error rate
	if metrics.ErrorRate > 5 {
		issueCounter++
		severity := "medium"
		if metrics.ErrorRate > 10 {
			severity = "high"
		}
		if metrics.ErrorRate > 20 {
			severity = "critical"
		}

		issues = append(issues, IdentifiedIssue{
			ID:          fmt.Sprintf("issue_%d", issueCounter),
			Title:       "High Error Rate",
			Description: fmt.Sprintf("Error rate is %.2f%%, above acceptable threshold of 5%%", metrics.ErrorRate),
			Category:    "error",
			Severity:    severity,
			Frequency:   int(metrics.TotalErrors),
			Impact:      "Users experiencing failed requests",
		})
		summaryParts = append(summaryParts, fmt.Sprintf("High error rate (%.2f%%)", metrics.ErrorRate))
	}

	// Check average rating
	if metrics.AvgRating < 3.5 && metrics.AvgRating > 0 {
		issueCounter++
		issues = append(issues, IdentifiedIssue{
			ID:          fmt.Sprintf("issue_%d", issueCounter),
			Title:       "Low User Satisfaction",
			Description: fmt.Sprintf("Average rating is %.2f out of 5, indicating user dissatisfaction", metrics.AvgRating),
			Category:    "quality",
			Severity:    "high",
			Frequency:   int(metrics.LowRatedResponses),
			Impact:      "Poor user experience affecting retention",
		})
		summaryParts = append(summaryParts, fmt.Sprintf("Low satisfaction rating (%.2f/5)", metrics.AvgRating))
	}

	// Check latency
	if metrics.AvgLatency > 2000 { // > 2 seconds
		issueCounter++
		issues = append(issues, IdentifiedIssue{
			ID:          fmt.Sprintf("issue_%d", issueCounter),
			Title:       "Slow Response Time",
			Description: fmt.Sprintf("Average latency is %.0fms, exceeding 2000ms threshold", metrics.AvgLatency),
			Category:    "performance",
			Severity:    "medium",
			Impact:      "Users waiting too long for responses",
		})
		summaryParts = append(summaryParts, fmt.Sprintf("Slow responses (%.0fms avg)", metrics.AvgLatency))
	}

	// Analyze error patterns
	for i, pattern := range errorPatterns {
		if i >= 3 {
			break // Top 3 errors only
		}
		count := pattern["count"].(int)
		if count > 10 {
			issueCounter++
			issues = append(issues, IdentifiedIssue{
				ID:          fmt.Sprintf("issue_%d", issueCounter),
				Title:       fmt.Sprintf("Recurring Error: %s", pattern["error_type"]),
				Description: fmt.Sprintf("Error '%s' occurred %d times", pattern["error_message"], count),
				Category:    "error",
				Severity:    "medium",
				Frequency:   count,
			})
		}
	}

	// Generate recommendations based on issues
	recCounter := 0
	for _, issue := range issues {
		recCounter++
		var rec Recommendation

		switch issue.Category {
		case "error":
			rec = Recommendation{
				ID:          fmt.Sprintf("rec_%d", recCounter),
				Title:       fmt.Sprintf("Fix: %s", issue.Title),
				Description: "Investigate and resolve the root cause of this error",
				Priority:    issue.Severity,
				Effort:      "medium",
				IssueRefs:   []string{issue.ID},
				ActionItems: []string{
					"Review error logs for this error type",
					"Identify root cause",
					"Implement fix and add tests",
				},
			}
		case "performance":
			rec = Recommendation{
				ID:          fmt.Sprintf("rec_%d", recCounter),
				Title:       "Optimize Response Time",
				Description: "Implement caching or optimize slow operations",
				Priority:    "medium",
				Effort:      "medium",
				IssueRefs:   []string{issue.ID},
				ActionItems: []string{
					"Profile slow endpoints",
					"Add caching for frequent queries",
					"Optimize database queries",
				},
			}
		case "quality":
			rec = Recommendation{
				ID:          fmt.Sprintf("rec_%d", recCounter),
				Title:       "Improve Response Quality",
				Description: "Review and improve prompts and model configurations",
				Priority:    "high",
				Effort:      "large",
				IssueRefs:   []string{issue.ID},
				ActionItems: []string{
					"Analyze low-rated responses",
					"Refine system prompts",
					"Consider model upgrades",
				},
			}
		}

		if rec.ID != "" {
			recommendations = append(recommendations, rec)
		}
	}

	// Generate summary
	summary := "Analysis complete. "
	if len(summaryParts) > 0 {
		summary += "Issues identified: " + strings.Join(summaryParts, "; ") + "."
	} else {
		summary += "No significant issues found. System is performing within acceptable parameters."
	}

	return issues, recommendations, summary, nil
}

// GetLatestAnalysis returns the most recent analysis result
func (a *Analyzer) GetLatestAnalysis(analysisType string) (*AnalysisResult, error) {
	var result AnalysisResult
	err := a.db.Where("analysis_type = ?", analysisType).
		Order("created_at DESC").
		First(&result).Error
	return &result, err
}

// GetAnalysisHistory returns historical analysis results
func (a *Analyzer) GetAnalysisHistory(limit int) ([]AnalysisResult, error) {
	var results []AnalysisResult
	err := a.db.Order("created_at DESC").
		Limit(limit).
		Find(&results).Error
	return results, err
}
