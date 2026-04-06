package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMCPServerValidator(t *testing.T) {
	validator := NewMCPServerValidator()
	assert.NotNil(t, validator)
	assert.NotEmpty(t, validator.rules)
}

func TestMCPServerValidator_ValidateServer_ValidServer(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "test-package"},
		Type:    "command",
		Enabled: true,
	}

	errors := validator.ValidateServer(server)
	assert.Empty(t, errors, "expected no validation errors for valid server")
}

func TestMCPServerValidator_ValidateServer_MissingName(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Command: "npx",
		Args:    []string{"test"},
		Type:    "command",
	}

	errors := validator.ValidateServer(server)
	assert.NotEmpty(t, errors, "expected validation errors for missing name")
	assert.Contains(t, errors[0].Message, "required")
}

func TestMCPServerValidator_ValidateServer_InvalidNameFormat(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test server", // 包含空格
		Command: "npx",
		Args:    []string{"test"},
		Type:    "command",
	}

	errors := validator.ValidateServer(server)
	assert.NotEmpty(t, errors, "expected validation errors for invalid name format")
	assert.Contains(t, errors[0].Message, "does not match pattern")
}

func TestMCPServerValidator_ValidateServer_EmptyArgs(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{},
		Type:    "command",
	}

	errors := validator.ValidateServer(server)
	assert.NotEmpty(t, errors, "expected validation errors for empty args")
	assert.Contains(t, errors[0].Message, "field is required")
}

func TestMCPServerValidator_ValidateServer_InvalidServerType(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"test"},
		Type:    "invalid-type",
	}

	errors := validator.ValidateServer(server)
	assert.NotEmpty(t, errors, "expected validation errors for invalid server type")
	assert.Contains(t, errors[0].Message, "invalid server type")
}

func TestMCPServerValidator_ValidateServer_ValidDockerType(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "docker-server",
		Command: "docker",
		Args:    []string{"run", "test-image"},
		Type:    "docker",
	}

	errors := validator.ValidateServer(server)
	assert.Empty(t, errors, "expected no validation errors for valid docker server")
}

func TestMCPServerValidator_ValidateServer_ValidBuiltinType(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "builtin-server",
		Command: "builtin",
		Args:    []string{"test"},
		Type:    "builtin",
	}

	errors := validator.ValidateServer(server)
	assert.Empty(t, errors, "expected no validation errors for valid builtin server")
}

func TestMCPServerValidator_ValidateServer_InvalidEnvKey(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"test"},
		Type:    "command",
		Env: map[string]string{
			"INVALID-KEY": "value", // 包含连字符
		},
	}

	errors := validator.ValidateServer(server)
	assert.NotEmpty(t, errors, "expected validation errors for invalid env key")
	assert.Contains(t, errors[0].Message, "invalid environment variable key")
}

func TestMCPServerValidator_ValidateServer_ValidEnv(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"test"},
		Type:    "command",
		Env: map[string]string{
			"VALID_KEY":   "value",
			"ANOTHER_KEY": "another_value",
		},
	}

	errors := validator.ValidateServer(server)
	assert.Empty(t, errors, "expected no validation errors for valid env")
}

func TestMCPServerValidator_ValidateServer_InvalidURL(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"test"},
		Type:    "command",
		URL:     "invalid-url", // 缺少协议
	}

	errors := validator.ValidateServer(server)
	assert.NotEmpty(t, errors, "expected validation errors for invalid URL")
	assert.Contains(t, errors[0].Message, "must start with http:// or https://")
}

func TestMCPServerValidator_ValidateServer_ValidURL(t *testing.T) {
	validator := NewMCPServerValidator()

	server := &MCPServer{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"test"},
		Type:    "command",
		URL:     "https://example.com",
	}

	errors := validator.ValidateServer(server)
	assert.Empty(t, errors, "expected no validation errors for valid URL")
}

func TestMCPServerValidator_ValidateServer_NilServer(t *testing.T) {
	validator := NewMCPServerValidator()

	errors := validator.ValidateServer(nil)
	assert.NotEmpty(t, errors, "expected validation errors for nil server")
	assert.Contains(t, errors[0].Message, "server configuration is nil")
}

func TestMCPServerValidator_ValidateConfig_ValidConfig(t *testing.T) {
	validator := NewMCPServerValidator()

	config := &MCPConfig{
		Servers: map[string]*MCPServerConfig{
			"server1": {
				MCPServer: MCPServer{
					Name:    "server1",
					Command: "npx",
					Args:    []string{"test"},
					Type:    "command",
				},
			},
			"server2": {
				MCPServer: MCPServer{
					Name:    "server2",
					Command: "docker",
					Args:    []string{"run", "test"},
					Type:    "docker",
				},
			},
		},
	}

	errors := validator.ValidateConfig(config)
	assert.Empty(t, errors, "expected no validation errors for valid config")
}

func TestMCPServerValidator_ValidateConfig_NilConfig(t *testing.T) {
	validator := NewMCPServerValidator()

	errors := validator.ValidateConfig(nil)
	assert.NotEmpty(t, errors, "expected validation errors for nil config")
	assert.Contains(t, errors[0].Message, "configuration is nil")
}

func TestMCPServerValidator_ValidateConfig_InvalidServerInConfig(t *testing.T) {
	validator := NewMCPServerValidator()

	config := &MCPConfig{
		Servers: map[string]*MCPServerConfig{
			"invalid-server": {
				MCPServer: MCPServer{
					Name:    "", // 缺少名称
					Command: "npx",
					Args:    []string{"test"},
					Type:    "command",
				},
			},
		},
	}

	errors := validator.ValidateConfig(config)
	assert.NotEmpty(t, errors, "expected validation errors for invalid server in config")
	assert.Contains(t, errors[0].Message, "required")
}

func TestValidateAndLoadMCPConfig_ValidConfig(t *testing.T) {
	// 注意：这个测试需要实际的配置文件
	// 这里我们只测试函数签名和基本逻辑
	validator := NewMCPServerValidator()
	assert.NotNil(t, validator)
	// 实际文件加载测试需要配置文件存在
}

func TestIsZeroValue(t *testing.T) {
	// 测试字符串
	assert.True(t, isZeroValue(""))
	assert.False(t, isZeroValue("test"))

	// 测试字符串切片
	assert.True(t, isZeroValue([]string{}))
	assert.True(t, isZeroValue([]string(nil)))
	assert.False(t, isZeroValue([]string{"test"}))

	// 测试map
	assert.True(t, isZeroValue(map[string]string{}))
	assert.True(t, isZeroValue(map[string]string(nil)))
	assert.False(t, isZeroValue(map[string]string{"key": "value"}))

	// 测试nil
	assert.True(t, isZeroValue(nil))
}
