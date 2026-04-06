package handlers

import (
	"backend/internal/services/analytics"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	service *analytics.AnalyticsService
}

func NewAnalyticsHandler(service *analytics.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

// GetUsageSummary returns aggregated usage statistics
func (h *AnalyticsHandler) GetUsageSummary(c *gin.Context) {
	userID := h.getUserID(c)
	startDate, endDate := h.getDateRange(c)

	summaries, err := h.service.GetUsageSummary(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summaries)
}

// GetDailyUsage returns daily usage statistics
func (h *AnalyticsHandler) GetDailyUsage(c *gin.Context) {
	userID := h.getUserID(c)
	startDate, endDate := h.getDateRange(c)

	daily, err := h.service.GetDailyUsage(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, daily)
}

// GetCostBreakdown returns cost breakdown by service/model
func (h *AnalyticsHandler) GetCostBreakdown(c *gin.Context) {
	userID := h.getUserID(c)
	startDate, endDate := h.getDateRange(c)

	breakdown, err := h.service.GetCostBreakdown(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, breakdown)
}

// GetTotalStats returns overall statistics
func (h *AnalyticsHandler) GetTotalStats(c *gin.Context) {
	userID := h.getUserID(c)
	startDate, endDate := h.getDateRange(c)

	stats, err := h.service.GetTotalStats(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetRecentUsage returns recent usage records
func (h *AnalyticsHandler) GetRecentUsage(c *gin.Context) {
	userID := h.getUserID(c)

	records, err := h.service.GetRecentUsage(userID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, records)
}

// GetDashboard returns all dashboard data in one call
func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	userID := h.getUserID(c)
	startDate, endDate := h.getDateRange(c)

	stats, _ := h.service.GetTotalStats(userID, startDate, endDate)
	daily, _ := h.service.GetDailyUsage(userID, startDate, endDate)
	breakdown, _ := h.service.GetCostBreakdown(userID, startDate, endDate)
	summaries, _ := h.service.GetUsageSummary(userID, startDate, endDate)

	c.JSON(http.StatusOK, gin.H{
		"stats":     stats,
		"daily":     daily,
		"breakdown": breakdown,
		"summaries": summaries,
	})
}

// Helper functions
func (h *AnalyticsHandler) getUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(uint)
	}
	return 0
}

func (h *AnalyticsHandler) getDateRange(c *gin.Context) (time.Time, time.Time) {
	// Default to last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if start := c.Query("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = t
		}
	}

	if end := c.Query("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			endDate = t.Add(24*time.Hour - time.Second) // End of day
		}
	}

	// Handle period shortcuts
	if period := c.Query("period"); period != "" {
		endDate = time.Now()
		switch period {
		case "today":
			startDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())
		case "week":
			startDate = endDate.AddDate(0, 0, -7)
		case "month":
			startDate = endDate.AddDate(0, -1, 0)
		case "year":
			startDate = endDate.AddDate(-1, 0, 0)
		}
	}

	return startDate, endDate
}
