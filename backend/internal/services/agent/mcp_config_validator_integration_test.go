package agent

import (
	"testing"
	"time"

	"backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigValidator_Integration(t *testing.T) {
	// 创建验证器
	validator := NewConfigValidator()
	assert.NotNil(t, validator)
	
	// 创建测试服务器配置
	server := &config.MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
		Env: map[string]string{
			"API_KEY": "test-key",
		},
	}
	
	// 测试验证
	result := validator.ValidateOnLoad(server)
	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
	assert.False(t, result.HasErrors())
	
	// 测试连接前验证
	result = validator.ValidateBeforeConnect(server)
	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
	
	// 测试获取历史
	history := validator.GetValidationHistory("test-server")
	assert.Len(t, history, 2) // 两次验证
	
	// 验证历史记录顺序（最近的在前）
	assert.True(t, history[0].Timestamp.After(history[1].Timestamp) || 
		history[0].Timestamp.Equal(history[1].Timestamp))
	
	// 测试缓存统计
	stats := validator.GetCacheStats()
	assert.Equal(t, 1, len(stats))
	assert.Equal(t, 2, stats["test-server"])
	
	// 测试清除缓存
	validator.ClearCache()
	
	// 验证缓存已清空
	history = validator.GetValidationHistory("test-server")
	assert.Empty(t, history)
	
	stats = validator.GetCacheStats()
	assert.Empty(t, stats)
}

func TestConfigValidator_InvalidConfig(t *testing.T) {
	validator := NewConfigValidator()
	
	// 无效配置：缺少名称，命令包含危险字符
	server := &config.MCPServer{
		Name:    "",
		Command: "npx; rm -rf /",
		Type:    "invalid-type",
	}
	
	result := validator.ValidateOnLoad(server)
	assert.NotNil(t, result)
	assert.False(t, result.IsValid)
	assert.True(t, result.HasErrors())
	
	// 验证错误详情
	assert.NotEmpty(t, result.Errors)
	for _, err := range result.Errors {
		assert.NotEmpty(t, err.Field)
		assert.NotEmpty(t, err.Message)
		assert.Contains(t, []string{"error", "warning"}, err.Severity)
	}
}

func TestConfigValidator_MultipleServers(t *testing.T) {
	validator := NewConfigValidator()
	
	// 创建多个服务器配置
	servers := []*config.MCPServer{
		{
			Name:    "server1",
			Command: "npx",
			Args:    []string{"-y", "@test/server1"},
			Type:    "command",
		},
		{
			Name:    "server2",
			Command: "npx",
			Args:    []string{"-y", "@test/server2"},
			Type:    "command",
		},
		{
			Name:    "server3",
			Command: "npx",
			Args:    []string{"-y", "@test/server3"},
			Type:    "command",
		},
	}
	
	// 为每个服务器执行多次验证
	for i, server := range servers {
		for j := 0; j < i+1; j++ {
			validator.ValidateOnLoad(server)
			time.Sleep(1 * time.Millisecond) // 确保时间戳不同
		}
	}
	
	// 验证缓存统计
	stats := validator.GetCacheStats()
	assert.Equal(t, 3, len(stats))
	assert.Equal(t, 1, stats["server1"])
	assert.Equal(t, 2, stats["server2"])
	assert.Equal(t, 3, stats["server3"])
	
	// 验证每个服务器的历史记录
	for i, server := range servers {
		history := validator.GetValidationHistory(server.Name)
		assert.Len(t, history, i+1)
		
		// 验证历史记录限制（最多10条）
		if i+1 > 10 {
			assert.Len(t, history, 10)
		}
	}
}

func TestConfigValidator_HistoryLimits(t *testing.T) {
	validator := NewConfigValidator()
	
	server := &config.MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
	}
	
	// 执行超过限制次数的验证
	for i := 0; i < 15; i++ {
		validator.ValidateOnLoad(server)
		time.Sleep(1 * time.Millisecond)
	}
	
	// 验证历史记录限制
	history := validator.GetValidationHistory("test-server")
	assert.Len(t, history, 10)
	
	// 验证时间戳顺序（最近的在前）
	for i := 0; i < len(history)-1; i++ {
		assert.True(t, history[i].Timestamp.After(history[i+1].Timestamp) || 
			history[i].Timestamp.Equal(history[i+1].Timestamp))
	}
}