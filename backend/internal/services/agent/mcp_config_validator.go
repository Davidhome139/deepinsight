package agent

import (
	"sync"

	"backend/internal/config"
)

// ConfigValidator 配置验证管理器
type ConfigValidator struct {
	mu    sync.RWMutex
	cache map[string][]config.ValidationResult
}

// NewConfigValidator 创建新的配置验证管理器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		cache: make(map[string][]config.ValidationResult),
	}
}

// ValidateOnLoad 在配置加载时验证
func (v *ConfigValidator) ValidateOnLoad(server *config.MCPServer) config.ValidationResult {
	validator := config.NewConfigValidator()
	result := validator.ValidateMCPServerComprehensive(server.Name, *server)

	v.recordValidation(server.Name, result)
	return result
}

// ValidateBeforeConnect 在连接前验证
func (v *ConfigValidator) ValidateBeforeConnect(server *config.MCPServer) config.ValidationResult {
	validator := config.NewConfigValidator()
	result := validator.ValidateMCPServerComprehensive(server.Name, *server)

	// 添加连接前特定验证
	v.validateConnectivity(server, &result)

	v.recordValidation(server.Name, result)
	return result
}

// validateConnectivity 验证连接性（占位符）
func (v *ConfigValidator) validateConnectivity(server *config.MCPServer, result *config.ValidationResult) {
	// 这里可以添加实际的连接测试
	// 暂时只添加警告
	if result.IsValid {
		result.Warnings = append(result.Warnings,
			"Connectivity validation not implemented yet (will be added in phase 2)")
	}
}

// GetValidationHistory 获取验证历史
func (v *ConfigValidator) GetValidationHistory(serverName string) []config.ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if history, exists := v.cache[serverName]; exists {
		return history
	}
	return []config.ValidationResult{}
}

// recordValidation 记录验证结果
func (v *ConfigValidator) recordValidation(serverName string, result config.ValidationResult) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 限制历史记录数量，新的结果放在前面
	history := v.cache[serverName]
	if len(history) >= 10 {
		history = history[:9] // 保留前9个
	}

	// 将新结果插入到前面
	history = append([]config.ValidationResult{result}, history...)
	v.cache[serverName] = history
}

// ClearCache 清除缓存
func (v *ConfigValidator) ClearCache() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.cache = make(map[string][]config.ValidationResult)
}

// GetCacheStats 获取缓存统计
func (v *ConfigValidator) GetCacheStats() map[string]int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	stats := make(map[string]int)
	for serverName, history := range v.cache {
		stats[serverName] = len(history)
	}
	return stats
}
