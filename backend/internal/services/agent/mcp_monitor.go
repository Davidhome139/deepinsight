package agent

import (
	"backend/internal/config"
	"fmt"
	"sync"
	"time"
)

// Metrics 监控指标接口
type Metrics interface {
	RecordRequest(server, tool string, duration time.Duration, err error)
	RecordConnectionChange(delta int)
	RecordCacheHit()
	RecordCacheMiss()
	GetStats() MonitorStats
}

// MonitorStats 监控统计信息
type MonitorStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalConnections   int
	CacheHits          int64
	CacheMisses        int64
	AverageLatency     time.Duration
	ValidationFailures int64
}

// Monitor 监控包装器
type Monitor struct {
	mu    sync.RWMutex
	stats MonitorStats
	// 请求历史用于计算平均延迟
	requestHistory []time.Duration
	maxHistorySize int

	// 验证历史
	validationHistory map[string][]config.ValidationResult
}

// NewMonitor 创建新的监控器
func NewMonitor() *Monitor {
	return &Monitor{
		stats:             MonitorStats{},
		requestHistory:    make([]time.Duration, 0),
		maxHistorySize:    1000, // 保留最近1000个请求的延迟数据
		validationHistory: make(map[string][]config.ValidationResult),
	}
}

// RecordRequest 记录请求
func (m *Monitor) RecordRequest(server, tool string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalRequests++

	if err == nil {
		m.stats.SuccessfulRequests++
	} else {
		m.stats.FailedRequests++
	}

	// 更新请求历史
	m.requestHistory = append(m.requestHistory, duration)
	if len(m.requestHistory) > m.maxHistorySize {
		m.requestHistory = m.requestHistory[1:]
	}

	// 计算平均延迟
	var total time.Duration
	for _, d := range m.requestHistory {
		total += d
	}
	if len(m.requestHistory) > 0 {
		m.stats.AverageLatency = total / time.Duration(len(m.requestHistory))
	}
}

// RecordConnectionChange 记录连接变化
func (m *Monitor) RecordConnectionChange(delta int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TotalConnections += delta
	if m.stats.TotalConnections < 0 {
		m.stats.TotalConnections = 0
	}
}

// RecordCacheHit 记录缓存命中
func (m *Monitor) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.CacheHits++
}

// RecordCacheMiss 记录缓存未命中
func (m *Monitor) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.CacheMisses++
}

// GetStats 获取统计信息
func (m *Monitor) GetStats() MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// GetCacheHitRate 获取缓存命中率
func (m *Monitor) GetCacheHitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := m.stats.CacheHits + m.stats.CacheMisses
	if total == 0 {
		return 0.0
	}
	return float64(m.stats.CacheHits) / float64(total)
}

// GetSuccessRate 获取成功率
func (m *Monitor) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.stats.TotalRequests == 0 {
		return 0.0
	}
	return float64(m.stats.SuccessfulRequests) / float64(m.stats.TotalRequests)
}

// ResetStats 重置统计信息
func (m *Monitor) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = MonitorStats{}
	m.requestHistory = make([]time.Duration, 0)
}

// MonitorMiddleware 监控中间件
type MonitorMiddleware struct {
	monitor *Monitor
}

// NewMonitorMiddleware 创建新的监控中间件
func NewMonitorMiddleware(monitor *Monitor) *MonitorMiddleware {
	return &MonitorMiddleware{
		monitor: monitor,
	}
}

// WrapRequest 包装请求函数
func (mw *MonitorMiddleware) WrapRequest(server, tool string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	mw.monitor.RecordRequest(server, tool, duration, err)
	return err
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Healthy     bool
	Message     string
	LastCheck   time.Time
	ErrorCount  int
	SuccessRate float64
}

// HealthChecker 健康检查器
type HealthChecker struct {
	monitor       *Monitor
	lastCheck     time.Time
	checkInterval time.Duration
	mu            sync.RWMutex
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(monitor *Monitor, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		monitor:       monitor,
		checkInterval: interval,
		lastCheck:     time.Now(),
	}
}

// Check 执行健康检查
func (hc *HealthChecker) Check() HealthCheckResult {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	stats := hc.monitor.GetStats()
	hc.lastCheck = time.Now()

	// 简单的健康检查逻辑
	result := HealthCheckResult{
		LastCheck:   hc.lastCheck,
		SuccessRate: hc.monitor.GetSuccessRate(),
		ErrorCount:  int(stats.FailedRequests),
	}

	// 如果总请求数大于0且成功率低于90%，认为不健康
	if stats.TotalRequests > 10 && result.SuccessRate < 0.9 {
		result.Healthy = false
		result.Message = fmt.Sprintf("Low success rate: %.2f%%", result.SuccessRate*100)
	} else if stats.TotalConnections == 0 {
		result.Healthy = false
		result.Message = "No active connections"
	} else {
		result.Healthy = true
		result.Message = "System is healthy"
	}

	return result
}

// ShouldCheck 检查是否应该执行健康检查
func (hc *HealthChecker) ShouldCheck() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return time.Since(hc.lastCheck) > hc.checkInterval
}

// GetLastCheck 获取上次检查时间
func (hc *HealthChecker) GetLastCheck() time.Time {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastCheck
}

// PerformanceAlert 性能告警
type PerformanceAlert struct {
	Type      string
	Message   string
	Threshold float64
	Actual    float64
	Timestamp time.Time
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	monitor        *Monitor
	alerts         []PerformanceAlert
	maxAlerts      int
	latencyWarning time.Duration
	mu             sync.RWMutex
}

// NewPerformanceMonitor 创建新的性能监控器
func NewPerformanceMonitor(monitor *Monitor, latencyWarning time.Duration) *PerformanceMonitor {
	return &PerformanceMonitor{
		monitor:        monitor,
		alerts:         make([]PerformanceAlert, 0),
		maxAlerts:      100,
		latencyWarning: latencyWarning,
	}
}

// CheckPerformance 检查性能
func (pm *PerformanceMonitor) CheckPerformance() []PerformanceAlert {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	stats := pm.monitor.GetStats()
	var newAlerts []PerformanceAlert

	// 检查延迟
	if stats.AverageLatency > pm.latencyWarning {
		newAlerts = append(newAlerts, PerformanceAlert{
			Type:      "HighLatency",
			Message:   fmt.Sprintf("Average latency is high: %v", stats.AverageLatency),
			Threshold: float64(pm.latencyWarning),
			Actual:    float64(stats.AverageLatency),
			Timestamp: time.Now(),
		})
	}

	// 检查缓存命中率
	cacheHitRate := pm.monitor.GetCacheHitRate()
	if cacheHitRate < 0.5 && stats.CacheHits+stats.CacheMisses > 100 {
		newAlerts = append(newAlerts, PerformanceAlert{
			Type:      "LowCacheHitRate",
			Message:   fmt.Sprintf("Cache hit rate is low: %.2f%%", cacheHitRate*100),
			Threshold: 50.0,
			Actual:    cacheHitRate * 100,
			Timestamp: time.Now(),
		})
	}

	// 检查成功率
	successRate := pm.monitor.GetSuccessRate()
	if successRate < 0.8 && stats.TotalRequests > 50 {
		newAlerts = append(newAlerts, PerformanceAlert{
			Type:      "LowSuccessRate",
			Message:   fmt.Sprintf("Success rate is low: %.2f%%", successRate*100),
			Threshold: 80.0,
			Actual:    successRate * 100,
			Timestamp: time.Now(),
		})
	}

	// 添加新告警
	pm.alerts = append(pm.alerts, newAlerts...)

	// 限制告警数量
	if len(pm.alerts) > pm.maxAlerts {
		pm.alerts = pm.alerts[len(pm.alerts)-pm.maxAlerts:]
	}

	return newAlerts
}

// GetAlerts 获取所有告警
func (pm *PerformanceMonitor) GetAlerts() []PerformanceAlert {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.alerts
}

// ClearAlerts 清除告警
func (pm *PerformanceMonitor) ClearAlerts() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.alerts = make([]PerformanceAlert, 0)
}

// RecordValidationFailure 记录验证失败
func (m *Monitor) RecordValidationFailure(server string, result config.ValidationResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.ValidationFailures++

	// 记录验证失败详情
	history := m.validationHistory[server]
	if len(history) >= 5 { // 限制历史记录数量
		history = history[1:]
	}

	history = append(history, result)
	m.validationHistory[server] = history
}

// GetValidationStats 获取验证统计
func (m *Monitor) GetValidationStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"validationFailures": m.stats.ValidationFailures,
		"serversWithHistory": len(m.validationHistory),
	}
}

// GetValidationHistory 获取验证历史
func (m *Monitor) GetValidationHistory(server string) []config.ValidationResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if history, exists := m.validationHistory[server]; exists {
		return history
	}
	return []config.ValidationResult{}
}
