package agent

import (
	"testing"
	"time"

	"backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigValidator(t *testing.T) {
	validator := NewConfigValidator()
	assert.NotNil(t, validator)
}

func TestConfigValidator_ValidateOnLoad(t *testing.T) {
	validator := NewConfigValidator()
	
	server := config.MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
	}
	
	result := validator.ValidateOnLoad(&server)
	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
}

func TestConfigValidator_ValidateBeforeConnect(t *testing.T) {
	validator := NewConfigValidator()
	
	server := config.MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
	}
	
	result := validator.ValidateBeforeConnect(&server)
	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
}

func TestConfigValidator_GetValidationHistory(t *testing.T) {
	validator := NewConfigValidator()
	
	server := config.MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
	}
	
	// 执行验证
	validator.ValidateOnLoad(&server)
	
	// 获取历史
	history := validator.GetValidationHistory("test-server")
	assert.Len(t, history, 1)
	assert.True(t, history[0].IsValid)
}

func TestConfigValidator_GetValidationHistory_Empty(t *testing.T) {
	validator := NewConfigValidator()
	
	// 获取不存在的服务器历史
	history := validator.GetValidationHistory("non-existent")
	assert.Empty(t, history)
}

func TestConfigValidator_ClearCache(t *testing.T) {
	validator := NewConfigValidator()
	
	server := config.MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
	}
	
	// 执行验证
	validator.ValidateOnLoad(&server)
	
	// 验证缓存有数据
	history := validator.GetValidationHistory("test-server")
	assert.Len(t, history, 1)
	
	// 清除缓存
	validator.ClearCache()
	
	// 验证缓存已清空
	history = validator.GetValidationHistory("test-server")
	assert.Empty(t, history)
}

func TestConfigValidator_GetCacheStats(t *testing.T) {
	validator := NewConfigValidator()
	
	server1 := config.MCPServer{
		Name:    "server1",
		Command: "npx",
		Args:    []string{"-y", "@test/server1"},
		Type:    "command",
	}
	
	server2 := config.MCPServer{
		Name:    "server2",
		Command: "npx",
		Args:    []string{"-y", "@test/server2"},
		Type:    "command",
	}
	
	// 执行验证
	validator.ValidateOnLoad(&server1)
	validator.ValidateOnLoad(&server2)
	validator.ValidateOnLoad(&server1) // 第二次验证server1
	
	// 获取缓存统计
	stats := validator.GetCacheStats()
	assert.Len(t, stats, 2)
	assert.Equal(t, 2, stats["server1"]) // server1有2条记录
	assert.Equal(t, 1, stats["server2"]) // server2有1条记录
}

func TestConfigValidator_ValidateInvalidConfig(t *testing.T) {
	validator := NewConfigValidator()
	
	// 无效配置：缺少名称，命令包含危险字符
	server := config.MCPServer{
		Name:    "",
		Command: "npx; rm -rf /",
		Type:    "command",
	}
	
	result := validator.ValidateOnLoad(&server)
	assert.NotNil(t, result)
	assert.False(t, result.IsValid)
	assert.True(t, result.HasErrors())
}

func TestConfigValidator_HistoryLimit(t *testing.T) {
	validator := NewConfigValidator()
	
	server := config.MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
	}
	
	// 执行多次验证（超过限制）
	for i := 0; i < 15; i++ {
		validator.ValidateOnLoad(&server)
		// 添加微小延迟确保时间戳不同
		time.Sleep(1 * time.Millisecond)
	}
	
	// 获取历史
	history := validator.GetValidationHistory("test-server")
	// 应该只保留最近的10条记录
	assert.Len(t, history, 10)
	
	// 验证时间戳顺序（最近的在前）
	for i := 0; i < len(history)-1; i++ {
		assert.True(t, history[i].Timestamp.After(history[i+1].Timestamp) || 
			history[i].Timestamp.Equal(history[i+1].Timestamp))
	}
}