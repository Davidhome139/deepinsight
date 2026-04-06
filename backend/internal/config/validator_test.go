package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidationResult_New(t *testing.T) {
	result := ValidationResult{
		IsValid:   true,
		Errors:    []ValidationError{},
		Warnings:  []string{},
		Timestamp: time.Now(),
	}

	assert.True(t, result.IsValid)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings)
}

func TestValidationError_String(t *testing.T) {
	err := ValidationError{
		Field:    "command",
		Message:  "command is required",
		Severity: "error",
	}

	assert.Equal(t, "command: command is required (error)", err.String())
}

func TestValidationResult_HasErrors(t *testing.T) {
	result := ValidationResult{
		IsValid: false,
		Errors: []ValidationError{
			{Field: "command", Message: "command is required", Severity: "error"},
		},
		Warnings: []string{},
	}

	assert.True(t, result.HasErrors())
	assert.False(t, result.HasWarnings())
}

func TestValidationResult_HasWarnings(t *testing.T) {
	result := ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []string{"warning message"},
	}

	assert.False(t, result.HasErrors())
	assert.True(t, result.HasWarnings())
}

func TestValidationResult_ErrorMessages(t *testing.T) {
	result := ValidationResult{
		IsValid: false,
		Errors: []ValidationError{
			{Field: "command", Message: "command is required", Severity: "error"},
			{Field: "name", Message: "name is required", Severity: "error"},
		},
		Warnings: []string{},
	}

	messages := result.ErrorMessages()
	assert.Len(t, messages, 2)
	assert.Contains(t, messages[0], "command: command is required")
	assert.Contains(t, messages[1], "name: name is required")
}

func TestValidateMCPServerComprehensive(t *testing.T) {
	validator := NewConfigValidator()

	// 测试有效配置
	validServer := MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
	}

	result := validator.ValidateMCPServerComprehensive("test", validServer)
	assert.True(t, result.IsValid)
	assert.False(t, result.HasErrors())

	// 测试无效配置
	invalidServer := MCPServer{
		Name:    "",
		Command: "npx; rm -rf /",
		Type:    "invalid-type",
	}

	result = validator.ValidateMCPServerComprehensive("invalid", invalidServer)
	assert.False(t, result.IsValid)
	assert.True(t, result.HasErrors())
}

func TestValidateMCPServerComprehensive_URLValidation(t *testing.T) {
	validator := NewConfigValidator()

	// 测试无效URL
	server := MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
		URL:     "invalid-url", // 无效URL
	}

	result := validator.ValidateMCPServerComprehensive("test", server)
	assert.False(t, result.IsValid)
	assert.True(t, result.HasErrors())

	// 测试有效URL
	server.URL = "https://example.com"
	result = validator.ValidateMCPServerComprehensive("test", server)
	assert.True(t, result.IsValid)
	assert.False(t, result.HasErrors())
}

func TestValidateMCPServerComprehensive_EnvValidation(t *testing.T) {
	validator := NewConfigValidator()

	// 测试空环境变量
	server := MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
		Type:    "command",
		Env: map[string]string{
			"API_KEY": "", // 空值
		},
	}

	result := validator.ValidateMCPServerComprehensive("test", server)
	assert.True(t, result.IsValid) // 空环境变量只是警告，不是错误
	assert.True(t, result.HasWarnings())
}
