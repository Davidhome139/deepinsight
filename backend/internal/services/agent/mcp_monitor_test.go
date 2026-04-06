package agent

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMonitor(t *testing.T) {
	monitor := NewMonitor()
	assert.NotNil(t, monitor)

	stats := monitor.GetStats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.Equal(t, 0, stats.TotalConnections)
	assert.Equal(t, int64(0), stats.CacheHits)
	assert.Equal(t, int64(0), stats.CacheMisses)
	assert.Equal(t, time.Duration(0), stats.AverageLatency)
}

func TestMonitor_RecordRequest_Success(t *testing.T) {
	monitor := NewMonitor()

	monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, nil)

	stats := monitor.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.Greater(t, stats.AverageLatency, time.Duration(0))
}

func TestMonitor_RecordRequest_Failure(t *testing.T) {
	monitor := NewMonitor()

	monitor.RecordRequest("server1", "tool1", 50*time.Millisecond, errors.New("error"))

	stats := monitor.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.SuccessfulRequests)
	assert.Equal(t, int64(1), stats.FailedRequests)
}

func TestMonitor_RecordConnectionChange(t *testing.T) {
	monitor := NewMonitor()

	monitor.RecordConnectionChange(2)
	stats := monitor.GetStats()
	assert.Equal(t, 2, stats.TotalConnections)

	monitor.RecordConnectionChange(-1)
	stats = monitor.GetStats()
	assert.Equal(t, 1, stats.TotalConnections)

	// 测试不会变成负数
	monitor.RecordConnectionChange(-5)
	stats = monitor.GetStats()
	assert.Equal(t, 0, stats.TotalConnections)
}

func TestMonitor_RecordCacheHitMiss(t *testing.T) {
	monitor := NewMonitor()

	monitor.RecordCacheHit()
	monitor.RecordCacheHit()
	monitor.RecordCacheMiss()

	stats := monitor.GetStats()
	assert.Equal(t, int64(2), stats.CacheHits)
	assert.Equal(t, int64(1), stats.CacheMisses)
}

func TestMonitor_GetCacheHitRate(t *testing.T) {
	monitor := NewMonitor()

	// 没有缓存访问
	rate := monitor.GetCacheHitRate()
	assert.Equal(t, 0.0, rate)

	// 只有命中
	monitor.RecordCacheHit()
	monitor.RecordCacheHit()
	rate = monitor.GetCacheHitRate()
	assert.Equal(t, 1.0, rate)

	// 命中和未命中混合
	monitor.RecordCacheMiss()
	rate = monitor.GetCacheHitRate()
	assert.InDelta(t, 0.666, rate, 0.001) // 2/3 ≈ 0.666
}

func TestMonitor_GetSuccessRate(t *testing.T) {
	monitor := NewMonitor()

	// 没有请求
	rate := monitor.GetSuccessRate()
	assert.Equal(t, 0.0, rate)

	// 只有成功请求
	monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, nil)
	monitor.RecordRequest("server1", "tool2", 200*time.Millisecond, nil)
	rate = monitor.GetSuccessRate()
	assert.Equal(t, 1.0, rate)

	// 成功和失败混合
	monitor.RecordRequest("server1", "tool1", 150*time.Millisecond, errors.New("error"))
	rate = monitor.GetSuccessRate()
	assert.InDelta(t, 0.666, rate, 0.001) // 2/3 ≈ 0.666
}

func TestMonitor_ResetStats(t *testing.T) {
	monitor := NewMonitor()

	// 记录一些数据
	monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, nil)
	monitor.RecordConnectionChange(2)
	monitor.RecordCacheHit()

	// 重置
	monitor.ResetStats()

	stats := monitor.GetStats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, 0, stats.TotalConnections)
	assert.Equal(t, int64(0), stats.CacheHits)
}

func TestMonitor_AverageLatency(t *testing.T) {
	monitor := NewMonitor()

	// 记录多个请求
	monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, nil)
	monitor.RecordRequest("server1", "tool2", 200*time.Millisecond, nil)
	monitor.RecordRequest("server1", "tool3", 300*time.Millisecond, nil)

	stats := monitor.GetStats()
	expectedAvg := (100 + 200 + 300) / 3 * time.Millisecond
	assert.InDelta(t, float64(expectedAvg), float64(stats.AverageLatency), float64(1*time.Millisecond))
}

func TestNewMonitorMiddleware(t *testing.T) {
	monitor := NewMonitor()
	middleware := NewMonitorMiddleware(monitor)

	assert.NotNil(t, middleware)
	assert.Equal(t, monitor, middleware.monitor)
}

func TestMonitorMiddleware_WrapRequest(t *testing.T) {
	monitor := NewMonitor()
	middleware := NewMonitorMiddleware(monitor)

	called := false
	err := middleware.WrapRequest("server1", "tool1", func() error {
		called = true
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)

	stats := monitor.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.SuccessfulRequests)
}

func TestNewHealthChecker(t *testing.T) {
	monitor := NewMonitor()
	checker := NewHealthChecker(monitor, 1*time.Minute)

	assert.NotNil(t, checker)
	assert.True(t, time.Since(checker.GetLastCheck()) < 1*time.Second)
}

func TestHealthChecker_Check_Healthy(t *testing.T) {
	monitor := NewMonitor()
	checker := NewHealthChecker(monitor, 1*time.Minute)

	// 记录一些成功请求
	for i := 0; i < 20; i++ {
		monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, nil)
	}
	monitor.RecordConnectionChange(2)

	result := checker.Check()

	assert.True(t, result.Healthy)
	assert.Contains(t, result.Message, "healthy")
	assert.Equal(t, 0, result.ErrorCount)
	assert.Equal(t, 1.0, result.SuccessRate)
	assert.True(t, time.Since(result.LastCheck) < 1*time.Second)
}

func TestHealthChecker_Check_Unhealthy_LowSuccessRate(t *testing.T) {
	monitor := NewMonitor()
	checker := NewHealthChecker(monitor, 1*time.Minute)

	// 记录大量失败请求
	for i := 0; i < 15; i++ {
		if i < 5 {
			monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, nil)
		} else {
			monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, errors.New("error"))
		}
	}

	result := checker.Check()

	assert.False(t, result.Healthy)
	assert.Contains(t, result.Message, "Low success rate")
	assert.Greater(t, result.ErrorCount, 0)
	assert.Less(t, result.SuccessRate, 0.9)
}

func TestHealthChecker_Check_Unhealthy_NoConnections(t *testing.T) {
	monitor := NewMonitor()
	checker := NewHealthChecker(monitor, 1*time.Minute)

	// 没有连接
	result := checker.Check()

	assert.False(t, result.Healthy)
	assert.Contains(t, result.Message, "No active connections")
}

func TestHealthChecker_ShouldCheck(t *testing.T) {
	monitor := NewMonitor()
	checker := NewHealthChecker(monitor, 100*time.Millisecond)

	// 刚创建时不应该立即检查（lastCheck是创建时间）
	assert.False(t, checker.ShouldCheck())

	// 等待超过检查间隔
	time.Sleep(150 * time.Millisecond)
	assert.True(t, checker.ShouldCheck())

	// 执行检查
	checker.Check()

	// 立即再次检查应该返回false
	assert.False(t, checker.ShouldCheck())

	// 再次等待超过检查间隔
	time.Sleep(150 * time.Millisecond)
	assert.True(t, checker.ShouldCheck())
}

func TestNewPerformanceMonitor(t *testing.T) {
	monitor := NewMonitor()
	perfMonitor := NewPerformanceMonitor(monitor, 1*time.Second)

	assert.NotNil(t, perfMonitor)
	assert.Equal(t, monitor, perfMonitor.monitor)
	assert.Equal(t, 1*time.Second, perfMonitor.latencyWarning)
}

func TestPerformanceMonitor_CheckPerformance(t *testing.T) {
	monitor := NewMonitor()
	perfMonitor := NewPerformanceMonitor(monitor, 100*time.Millisecond)

	// 记录高延迟请求
	for i := 0; i < 20; i++ {
		monitor.RecordRequest("server1", "tool1", 150*time.Millisecond, nil)
	}

	alerts := perfMonitor.CheckPerformance()

	assert.NotEmpty(t, alerts)
	assert.Equal(t, "HighLatency", alerts[0].Type)
	assert.Contains(t, alerts[0].Message, "Average latency is high")
}

func TestPerformanceMonitor_CheckPerformance_LowCacheHitRate(t *testing.T) {
	monitor := NewMonitor()
	perfMonitor := NewPerformanceMonitor(monitor, 1*time.Second)

	// 记录低缓存命中率
	for i := 0; i < 150; i++ {
		if i < 50 {
			monitor.RecordCacheHit()
		} else {
			monitor.RecordCacheMiss()
		}
	}

	alerts := perfMonitor.CheckPerformance()

	assert.NotEmpty(t, alerts)
	assert.Equal(t, "LowCacheHitRate", alerts[0].Type)
	assert.Contains(t, alerts[0].Message, "Cache hit rate is low")
}

func TestPerformanceMonitor_CheckPerformance_LowSuccessRate(t *testing.T) {
	monitor := NewMonitor()
	perfMonitor := NewPerformanceMonitor(monitor, 1*time.Second)

	// 记录低成功率
	for i := 0; i < 100; i++ {
		if i < 60 {
			monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, nil)
		} else {
			monitor.RecordRequest("server1", "tool1", 100*time.Millisecond, errors.New("error"))
		}
	}

	alerts := perfMonitor.CheckPerformance()

	assert.NotEmpty(t, alerts)
	assert.Equal(t, "LowSuccessRate", alerts[0].Type)
	assert.Contains(t, alerts[0].Message, "Success rate is low")
}

func TestPerformanceMonitor_GetAlerts(t *testing.T) {
	monitor := NewMonitor()
	perfMonitor := NewPerformanceMonitor(monitor, 100*time.Millisecond)

	// 生成一些告警
	for i := 0; i < 20; i++ {
		monitor.RecordRequest("server1", "tool1", 150*time.Millisecond, nil)
	}
	perfMonitor.CheckPerformance()

	alerts := perfMonitor.GetAlerts()
	assert.NotEmpty(t, alerts)

	// 清除告警
	perfMonitor.ClearAlerts()
	alerts = perfMonitor.GetAlerts()
	assert.Empty(t, alerts)
}
