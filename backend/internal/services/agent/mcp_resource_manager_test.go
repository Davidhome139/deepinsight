package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConnectionPool(t *testing.T) {
	pool := NewConnectionPool(10, 5*time.Minute)
	assert.NotNil(t, pool)
}

func TestConnectionPool_Stats(t *testing.T) {
	pool := NewConnectionPool(10, 5*time.Minute)
	stats := pool.Stats()
	assert.Equal(t, 0, stats.ActiveConnections)
	assert.Equal(t, 0, stats.IdleConnections)
}

func TestNewResourceManager(t *testing.T) {
	limits := ResourceLimits{
		MaxConnections:    5,
		MaxMemoryMB:       100,
		ConnectionTimeout: 30 * time.Second,
		IdleTimeout:       5 * time.Minute,
	}

	rm := NewResourceManager(limits)
	assert.NotNil(t, rm)
}

func TestResourceManager_GetStats(t *testing.T) {
	rm := NewResourceManager(DefaultResourceLimits)
	stats := rm.GetStats()
	assert.Equal(t, 0, stats.ConnectionCount)
	assert.Equal(t, 0, stats.ActiveTools)
	assert.Equal(t, 0.0, stats.MemoryUsageMB)
}

func TestResourceManager_GetPoolStats(t *testing.T) {
	rm := NewResourceManager(DefaultResourceLimits)
	stats := rm.GetPoolStats()
	assert.Equal(t, 0, stats.ActiveConnections)
	assert.Equal(t, 0, stats.IdleConnections)
}

func TestResourceManager_CleanupIdleConnections(t *testing.T) {
	rm := NewResourceManager(DefaultResourceLimits)
	cleaned := rm.CleanupIdleConnections()
	assert.Equal(t, 0, cleaned)
}

func TestDefaultResourceLimits(t *testing.T) {
	assert.Equal(t, 10, DefaultResourceLimits.MaxConnections)
	assert.Equal(t, 100, DefaultResourceLimits.MaxMemoryMB)
	assert.Equal(t, 30*time.Second, DefaultResourceLimits.ConnectionTimeout)
	assert.Equal(t, 5*time.Minute, DefaultResourceLimits.IdleTimeout)
}
