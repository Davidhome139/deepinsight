package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db         *gorm.DB
	redis      *redis.Client
	startTime  time.Time
	checksMu   sync.RWMutex
	lastChecks map[string]*HealthCheckResult

	// Prometheus metrics
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	healthStatus    *prometheus.GaugeVec
}

// HealthCheckResult contains the result of a health check
type HealthCheckResult struct {
	Status    string    `json:"status"` // healthy, degraded, unhealthy
	Message   string    `json:"message,omitempty"`
	Duration  string    `json:"duration"`
	CheckedAt time.Time `json:"checked_at"`
}

// HealthResponse is the response for health endpoints
type HealthResponse struct {
	Status    string                        `json:"status"`
	Timestamp time.Time                     `json:"timestamp"`
	Uptime    string                        `json:"uptime"`
	Version   string                        `json:"version,omitempty"`
	Checks    map[string]*HealthCheckResult `json:"checks,omitempty"`
}

// SystemInfo contains system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemAlloc     uint64 `json:"mem_alloc_bytes"`
	MemSys       uint64 `json:"mem_sys_bytes"`
	Uptime       string `json:"uptime"`
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *gorm.DB, redisClient *redis.Client) *HealthHandler {
	h := &HealthHandler{
		db:         db,
		redis:      redisClient,
		startTime:  time.Now(),
		lastChecks: make(map[string]*HealthCheckResult),
	}

	// Initialize Prometheus metrics
	h.initMetrics()

	return h
}

// initMetrics initializes Prometheus metrics
func (h *HealthHandler) initMetrics() {
	h.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "newdoubao_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	h.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "newdoubao_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	h.healthStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "newdoubao_health_status",
			Help: "Health status of components (1=healthy, 0=unhealthy)",
		},
		[]string{"component"},
	)

	// Register metrics
	prometheus.MustRegister(h.requestsTotal)
	prometheus.MustRegister(h.requestDuration)
	prometheus.MustRegister(h.healthStatus)
}

// LivenessProbe returns simple liveness status
// GET /health/live
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

// ReadinessProbe checks if the service is ready to accept traffic
// GET /health/ready
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]*HealthCheckResult)
	overallStatus := "healthy"

	// Check database
	dbCheck := h.checkDatabase(ctx)
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check Redis
	redisCheck := h.checkRedis(ctx)
	checks["redis"] = redisCheck
	if redisCheck.Status != "healthy" {
		overallStatus = "degraded"
	}

	// Store results
	h.checksMu.Lock()
	h.lastChecks = checks
	h.checksMu.Unlock()

	// Update Prometheus metrics
	for name, check := range checks {
		if check.Status == "healthy" {
			h.healthStatus.WithLabelValues(name).Set(1)
		} else {
			h.healthStatus.WithLabelValues(name).Set(0)
		}
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
		Checks:    checks,
	}

	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// HealthCheck returns detailed health status
// GET /health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	checks := make(map[string]*HealthCheckResult)
	overallStatus := "healthy"

	// Run all checks in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	checkFuncs := map[string]func(context.Context) *HealthCheckResult{
		"database": h.checkDatabase,
		"redis":    h.checkRedis,
		"memory":   h.checkMemory,
		"disk":     h.checkDisk,
	}

	for name, checkFunc := range checkFuncs {
		wg.Add(1)
		go func(n string, cf func(context.Context) *HealthCheckResult) {
			defer wg.Done()
			result := cf(ctx)
			mu.Lock()
			checks[n] = result
			if result.Status == "unhealthy" {
				overallStatus = "unhealthy"
			} else if result.Status == "degraded" && overallStatus != "unhealthy" {
				overallStatus = "degraded"
			}
			mu.Unlock()
		}(name, checkFunc)
	}

	wg.Wait()

	// Store results
	h.checksMu.Lock()
	h.lastChecks = checks
	h.checksMu.Unlock()

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
		Version:   "1.0.0",
		Checks:    checks,
	}

	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// GetMetrics returns Prometheus metrics
// GET /health/metrics
func (h *HealthHandler) GetMetrics() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

// GetSystemInfo returns system information
// GET /health/system
func (h *HealthHandler) GetSystemInfo(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	info := SystemInfo{
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		MemAlloc:     memStats.Alloc,
		MemSys:       memStats.Sys,
		Uptime:       time.Since(h.startTime).String(),
	}

	c.JSON(http.StatusOK, info)
}

// checkDatabase checks database connectivity
func (h *HealthHandler) checkDatabase(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		CheckedAt: time.Now(),
	}

	if h.db == nil {
		result.Status = "unhealthy"
		result.Message = "database not configured"
		result.Duration = time.Since(start).String()
		return result
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		result.Status = "unhealthy"
		result.Message = fmt.Sprintf("failed to get DB: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		result.Status = "unhealthy"
		result.Message = fmt.Sprintf("ping failed: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	// Check connection pool stats
	stats := sqlDB.Stats()
	if stats.OpenConnections >= stats.MaxOpenConnections-2 {
		result.Status = "degraded"
		result.Message = fmt.Sprintf("connection pool near limit: %d/%d", stats.OpenConnections, stats.MaxOpenConnections)
	} else {
		result.Status = "healthy"
		result.Message = fmt.Sprintf("connections: %d/%d", stats.OpenConnections, stats.MaxOpenConnections)
	}

	result.Duration = time.Since(start).String()
	return result
}

// checkRedis checks Redis connectivity
func (h *HealthHandler) checkRedis(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		CheckedAt: time.Now(),
	}

	if h.redis == nil {
		result.Status = "degraded"
		result.Message = "redis not configured"
		result.Duration = time.Since(start).String()
		return result
	}

	pong, err := h.redis.Ping(ctx).Result()
	if err != nil {
		result.Status = "unhealthy"
		result.Message = fmt.Sprintf("ping failed: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	result.Status = "healthy"
	result.Message = pong
	result.Duration = time.Since(start).String()
	return result
}

// checkMemory checks memory usage
func (h *HealthHandler) checkMemory(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		CheckedAt: time.Now(),
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Alert if using more than 1GB
	allocMB := memStats.Alloc / 1024 / 1024
	if allocMB > 1024 {
		result.Status = "degraded"
		result.Message = fmt.Sprintf("high memory usage: %d MB", allocMB)
	} else {
		result.Status = "healthy"
		result.Message = fmt.Sprintf("memory: %d MB", allocMB)
	}

	result.Duration = time.Since(start).String()
	return result
}

// checkDisk checks disk usage
func (h *HealthHandler) checkDisk(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		CheckedAt: time.Now(),
	}

	// Check current working directory disk space
	wd, err := os.Getwd()
	if err != nil {
		result.Status = "degraded"
		result.Message = fmt.Sprintf("cannot get working directory: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	// Try to stat the directory to verify access
	_, err = os.Stat(wd)
	if err != nil {
		result.Status = "degraded"
		result.Message = fmt.Sprintf("cannot access directory: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	// On Windows, we can't easily get disk usage without syscalls
	// Just report that directory is accessible
	result.Status = "healthy"
	result.Message = fmt.Sprintf("directory accessible: %s", wd)
	result.Duration = time.Since(start).String()
	return result
}

// MetricsMiddleware is a middleware to collect request metrics
func (h *HealthHandler) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())

		h.requestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		h.requestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}
