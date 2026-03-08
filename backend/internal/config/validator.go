package config

import (
	"fmt"
	"reflect"
	"strings"
)

// ConfigValidator 配置验证器
type ConfigValidator struct {
	errors []string
}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		errors: make([]string, 0),
	}
}

// ValidateModelProvider 验证模型提供商配置（宽松模式，只验证关键字段）
func (cv *ConfigValidator) ValidateModelProvider(key string, provider ModelProvider) bool {
	cv.errors = cv.errors[:0] // 清空错误

	// 验证必需字段（只验证最基本的）
	if provider.Name == "" {
		cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': name is required", key))
	}

	// 数值范围验证（只在设置了值时验证）
	if provider.Timeout < 0 {
		cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': timeout must be non-negative", key))
	}

	if provider.Temperature < 0 || provider.Temperature > 2 {
		cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': temperature must be between 0 and 2", key))
	}

	if provider.TopP < 0 || provider.TopP > 1 {
		cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': top_p must be between 0 and 1", key))
	}

	return len(cv.errors) == 0
}

// validateAuthFields 验证认证字段
func (cv *ConfigValidator) validateAuthFields(key string, provider ModelProvider) {
	authType := strings.ToLower(provider.AuthType)

	switch authType {
	case "bearer", "":
		// Bearer 认证需要 api_key
		if provider.APIKey == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': api_key is required for bearer auth", key))
		}

	case "tc3-hmac-sha256":
		// TC3 签名认证需要 secret_id 和 secret_key
		if provider.SecretID == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': secret_id is required for TC3 auth", key))
		}
		if provider.SecretKey == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': secret_key is required for TC3 auth", key))
		}

	case "oauth":
		// OAuth 需要 token_url 和 api_key
		if provider.TokenURL == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': token_url is required for OAuth", key))
		}
		if provider.APIKey == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': api_key is required for OAuth", key))
		}
	}

	// 特殊提供商验证
	switch key {
	case "tencentcloud":
		if provider.SecretID == "" || provider.SecretKey == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': secret_id and secret_key are required", key))
		}
	case "minimax":
		if provider.GroupID == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("Provider '%s': group_id is required", key))
		}
	}
}

// ValidateMCPServer 验证 MCP 服务器配置
func (cv *ConfigValidator) ValidateMCPServer(key string, server MCPServer) bool {
	cv.errors = cv.errors[:0] // 清空错误

	// 验证必需字段
	if server.Name == "" {
		cv.errors = append(cv.errors, fmt.Sprintf("MCP server '%s': name is required", key))
	}

	// 非 builtin 类型需要验证命令
	if server.Type != "builtin" {
		if server.Command == "" {
			cv.errors = append(cv.errors, fmt.Sprintf("MCP server '%s': command is required for non-builtin servers", key))
		}

		// 验证命令是否包含危险字符
		dangerousChars := []string{";", "&&", "||", "|", "`", "$", "(", ")"}
		for _, char := range dangerousChars {
			if strings.Contains(server.Command, char) {
				cv.errors = append(cv.errors, fmt.Sprintf("MCP server '%s': command contains dangerous character '%s'", key, char))
				break
			}
		}
	}

	// 验证类型
	validTypes := []string{"", "builtin", "command", "sse"}
	isValidType := false
	for _, t := range validTypes {
		if server.Type == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		cv.errors = append(cv.errors, fmt.Sprintf("MCP server '%s': invalid type '%s', must be one of: builtin, command, sse", key, server.Type))
	}

	return len(cv.errors) == 0
}

// GetErrors 获取验证错误
func (cv *ConfigValidator) GetErrors() []string {
	return cv.errors
}

// SanitizeString 清理字符串输入
func SanitizeString(input string) string {
	// 移除首尾空白
	input = strings.TrimSpace(input)
	// 移除控制字符
	input = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, input)
	return input
}

// SanitizeModelProvider 清理模型提供商配置
func SanitizeModelProvider(provider *ModelProvider) {
	provider.Name = SanitizeString(provider.Name)
	provider.APIKey = SanitizeString(provider.APIKey)
	provider.SecretKey = SanitizeString(provider.SecretKey)
	provider.SecretID = SanitizeString(provider.SecretID)
	provider.GroupID = SanitizeString(provider.GroupID)
	provider.BaseURL = SanitizeString(provider.BaseURL)
	provider.Endpoint = SanitizeString(provider.Endpoint)
	provider.AuthType = SanitizeString(provider.AuthType)
	provider.TokenURL = SanitizeString(provider.TokenURL)

	// 清理 headers
	cleanedHeaders := make(map[string]string)
	for k, v := range provider.Headers {
		cleanKey := SanitizeString(k)
		cleanValue := SanitizeString(v)
		if cleanKey != "" {
			cleanedHeaders[cleanKey] = cleanValue
		}
	}
	provider.Headers = cleanedHeaders

	// 清理 models
	cleanedModels := make([]string, 0, len(provider.Models))
	for _, model := range provider.Models {
		cleanModel := SanitizeString(model)
		if cleanModel != "" {
			cleanedModels = append(cleanedModels, cleanModel)
		}
	}
	provider.Models = cleanedModels
}

// SetDefaults 设置默认值
func SetDefaults(provider *ModelProvider) {
	if provider.Timeout == 0 {
		provider.Timeout = 60
	}
	if provider.AuthType == "" {
		provider.AuthType = "bearer"
	}
	if provider.Endpoint == "" {
		provider.Endpoint = "/v1/chat/completions"
	}
	if provider.Temperature == 0 {
		provider.Temperature = 1.0
	}
	if provider.TopP == 0 {
		provider.TopP = 0.95
	}
}

// MergeProviderConfig 合并提供商配置（用于增量更新）
func MergeProviderConfig(existing, incoming ModelProvider) ModelProvider {
	result := existing

	// 使用反射遍历所有字段
	v := reflect.ValueOf(&result).Elem()
	incomingV := reflect.ValueOf(incoming)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		incomingField := incomingV.Field(i)

		// 只更新非零值
		switch field.Kind() {
		case reflect.String:
			if incomingField.String() != "" {
				field.SetString(incomingField.String())
			}
		case reflect.Bool:
			if incomingField.Bool() != field.Bool() {
				field.SetBool(incomingField.Bool())
			}
		case reflect.Int, reflect.Int64:
			if incomingField.Int() != 0 {
				field.SetInt(incomingField.Int())
			}
		case reflect.Float64:
			if incomingField.Float() != 0 {
				field.SetFloat(incomingField.Float())
			}
		case reflect.Slice:
			if incomingField.Len() > 0 {
				field.Set(incomingField)
			}
		case reflect.Map:
			if incomingField.Len() > 0 {
				field.Set(incomingField)
			}
		}
	}

	return result
}
