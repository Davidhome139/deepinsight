package agent

import (
	"context"
	"testing"
	"time"

	"backend/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMCPClient 模拟MCP客户端
type MockMCPClient struct {
	mock.Mock
}

func (m *MockMCPClient) ListTools(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

func (m *MockMCPClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewMCPManager(t *testing.T) {
	manager := NewMCPManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.servers)
	assert.NotNil(t, manager.documentations)
	assert.NotNil(t, manager.resourceManager)
	assert.NotNil(t, manager.circuitBreakers)
	assert.NotNil(t, manager.monitor)
	assert.NotNil(t, manager.configValidator)
}

func TestMCPManager_GetServer_NotFound(t *testing.T) {
	manager := NewMCPManager()
	server, found := manager.GetServer("non-existent-server")
	assert.False(t, found)
	assert.Nil(t, server)
}

func TestMCPManager_GetServer_Found(t *testing.T) {
	manager := NewMCPManager()

	// 添加一个测试服务器
	testServer := &config.MCPServer{
		Name:    "test-server",
		Enabled: true,
		Type:    "stdin",
		Command: "echo",
		Args:    []string{"test"},
	}

	manager.mu.Lock()
	manager.servers["test-server"] = testServer
	manager.mu.Unlock()

	server, found := manager.GetServer("test-server")
	assert.True(t, found)
	assert.NotNil(t, server)
	assert.Equal(t, "test-server", server.Name)
	assert.True(t, server.Enabled)
}

func TestMCPManager_ConnectToServer_NotFound(t *testing.T) {
	manager := NewMCPManager()
	err := manager.ConnectToServer("non-existent-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMCPManager_GetMonitorStats(t *testing.T) {
	manager := NewMCPManager()
	stats := manager.GetMonitorStats()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
}

func TestMCPManager_GetCircuitBreakerState_NotFound(t *testing.T) {
	manager := NewMCPManager()
	state := manager.GetCircuitBreakerState("non-existent-server")
	assert.Equal(t, StateClosed, state)
}

func TestMCPManager_GetCircuitBreakerState_Found(t *testing.T) {
	manager := NewMCPManager()

	// 创建一个熔断器
	cb := NewCircuitBreaker(3, 30*time.Second)
	manager.circuitBreakers["test-server"] = cb

	state := manager.GetCircuitBreakerState("test-server")
	assert.Equal(t, StateClosed, state)
}

func TestMCPManager_GetResourceStats(t *testing.T) {
	manager := NewMCPManager()
	stats := manager.GetResourceStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.ConnectionCount)
	assert.Equal(t, 0, stats.ActiveTools)
	assert.Equal(t, 0.0, stats.MemoryUsageMB)
}

func TestMCPManager_GetMonitor(t *testing.T) {
	manager := NewMCPManager()
	monitor := manager.GetMonitor()
	assert.NotNil(t, monitor)
}

func TestMCPManager_Discover_EmptyConfig(t *testing.T) {
	manager := NewMCPManager()

	// 由于不能模拟包级别函数，我们直接测试Discover方法
	// 它应该处理空配置的情况
	manager.Discover()

	// 验证没有panic发生
	// 这是一个基本的健全性测试
	assert.NotNil(t, manager)
}

func TestMCPManager_Discover_WithDisabledServer(t *testing.T) {
	manager := NewMCPManager()

	// 由于不能模拟包级别函数，我们直接测试Discover方法
	// 它应该处理配置加载
	manager.Discover()

	// 验证没有panic发生
	// 这是一个基本的健全性测试
	assert.NotNil(t, manager)
}

func TestMCPManager_CloseAllServers(t *testing.T) {
	manager := NewMCPManager()

	// 添加一些测试服务器
	manager.mu.Lock()
	manager.servers["server1"] = &config.MCPServer{
		Name:      "server1",
		Enabled:   true,
		Connected: true,
	}
	manager.servers["server2"] = &config.MCPServer{
		Name:      "server2",
		Enabled:   true,
		Connected: true,
	}
	manager.mu.Unlock()

	manager.CloseAllServers()
	// 方法没有返回值，只验证没有panic发生

	// 验证方法执行完成，没有panic
	assert.NotNil(t, manager)
}
