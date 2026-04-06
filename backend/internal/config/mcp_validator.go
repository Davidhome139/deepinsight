package config

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// MCPServerValidator MCP服务器配置验证器
type MCPServerValidator struct {
	rules []MCPServerValidationRule
}

// MCPServerValidationRule MCP服务器验证规则
type MCPServerValidationRule struct {
	Field     string
	Required  bool
	Pattern   *regexp.Regexp
	Min       int
	Max       int
	Validator func(interface{}) error
}

// NewMCPServerValidator 创建MCP服务器验证器
func NewMCPServerValidator() *MCPServerValidator {
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	envKeyPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	
	return &MCPServerValidator{
		rules: []MCPServerValidationRule{
			{
				Field:    "Name",
				Required: true,
				Pattern:  namePattern,
				Validator: func(v interface{}) error {
					name, ok := v.(string)
					if !ok {
						return fmt.Errorf("name must be a string")
					}
					if len(name) == 0 {
						return fmt.Errorf("name cannot be empty")
					}
					if len(name) > 50 {
						return fmt.Errorf("name cannot exceed 50 characters")
					}
					return nil
				},
			},
			{
				Field:    "Command",
				Required: true,
				Validator: func(v interface{}) error {
					cmd, ok := v.(string)
					if !ok {
						return fmt.Errorf("command must be a string")
					}
					if len(cmd) == 0 {
						return fmt.Errorf("command cannot be empty")
					}
					return nil
				},
			},
			{
				Field:    "Args",
				Required: true,
				Validator: func(v interface{}) error {
					args, ok := v.([]string)
					if !ok {
						return fmt.Errorf("args must be a string array")
					}
					if len(args) == 0 {
						return fmt.Errorf("args cannot be empty")
					}
					return nil
				},
			},
			{
				Field: "Env",
				Validator: func(v interface{}) error {
					if v == nil {
						return nil
					}
					
					env, ok := v.(map[string]string)
					if !ok {
						return fmt.Errorf("env must be a map[string]string or nil")
					}
					
					for key := range env {
						if !envKeyPattern.MatchString(key) {
							return fmt.Errorf("invalid environment variable key: %s", key)
						}
					}
					return nil
				},
			},
			{
				Field:    "Type",
				Required: true,
				Validator: func(v interface{}) error {
					typ, ok := v.(string)
					if !ok {
						return fmt.Errorf("type must be a string")
					}
					
					validTypes := map[string]bool{
						"command": true,
						"docker":  true,
						"builtin": true,
					}
					
					if !validTypes[typ] {
						return fmt.Errorf("invalid server type: %s", typ)
					}
					return nil
				},
			},
			{
				Field: "URL",
				Validator: func(v interface{}) error {
					if v == nil {
						return nil
					}
					
					url, ok := v.(string)
					if !ok {
						return fmt.Errorf("url must be a string or nil")
					}
					
					if url != "" {
						if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
							return fmt.Errorf("url must start with http:// or https://")
						}
					}
					return nil
				},
			},
		},
	}
}

// ValidateServer 验证单个MCP服务器配置
func (v *MCPServerValidator) ValidateServer(server *MCPServer) []ValidationError {
	var errors []ValidationError
	
	if server == nil {
		return []ValidationError{{
			Field:    "server",
			Message:  "server configuration is nil",
			Severity: "error",
		}}
	}
	
	// 应用规则验证
	for _, rule := range v.rules {
		var fieldValue interface{}
		var fieldName string
		
		switch rule.Field {
		case "Name":
			fieldValue = server.Name
			fieldName = "name"
		case "Command":
			fieldValue = server.Command
			fieldName = "command"
		case "Args":
			fieldValue = server.Args
			fieldName = "args"
		case "Env":
			fieldValue = server.Env
			fieldName = "env"
		case "Type":
			fieldValue = server.Type
			fieldName = "type"
		case "URL":
			fieldValue = server.URL
			fieldName = "url"
		default:
			continue
		}
		
		// 检查必填字段
		if rule.Required {
			if isZeroValue(fieldValue) {
				errors = append(errors, ValidationError{
					Field:    fieldName,
					Message:  "field is required",
					Severity: "error",
				})
				continue
			}
		}
		
		// 跳过空值非必填字段
		if isZeroValue(fieldValue) {
			continue
		}
		
		// 应用模式验证
		if rule.Pattern != nil {
			strValue := fmt.Sprintf("%v", fieldValue)
			if !rule.Pattern.MatchString(strValue) {
				errors = append(errors, ValidationError{
					Field:    fieldName,
					Message:  fmt.Sprintf("does not match pattern: %s", rule.Pattern.String()),
					Severity: "error",
				})
			}
		}
		
		// 应用自定义验证器
		if rule.Validator != nil {
			if err := rule.Validator(fieldValue); err != nil {
				errors = append(errors, ValidationError{
					Field:    fieldName,
					Message:  err.Error(),
					Severity: "error",
				})
			}
		}
	}
	
	// 额外验证：检查命令是否存在（对于command类型）
	if server.Type == "command" && server.Command != "" {
		if !isCommandAvailable(server.Command) {
			errors = append(errors, ValidationError{
				Field:    "command",
				Message:  fmt.Sprintf("command '%s' is not available in PATH", server.Command),
				Severity: "warning",
			})
		}
	}
	
	// 验证FromGalleryId字段
	if server.FromGalleryId != nil && len(server.FromGalleryId) > 0 {
		for i, id := range server.FromGalleryId {
			if strings.TrimSpace(id) == "" {
				errors = append(errors, ValidationError{
					Field:    fmt.Sprintf("fromGalleryId[%d]", i),
					Message:  "gallery ID cannot be empty",
					Severity: "error",
				})
			}
		}
	}
	
	return errors
}

// ValidateConfig 验证完整MCP配置
func (v *MCPServerValidator) ValidateConfig(config *MCPConfig) []ValidationError {
	var errors []ValidationError
	
	if config == nil {
		return []ValidationError{{
			Field:    "config",
			Message:  "configuration is nil",
			Severity: "error",
		}}
	}
	
	// 验证每个服务器
	for name, server := range config.Servers {
		serverErrors := v.ValidateServer(&server.MCPServer)
		for _, err := range serverErrors {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("servers.%s.%s", name, err.Field),
				Message:  err.Message,
				Severity: err.Severity,
			})
		}
	}
	
	return errors
}

// ValidateAndLoadMCPConfig 加载并验证MCP配置
func ValidateAndLoadMCPConfig(configPath string) (*MCPConfig, []ValidationError) {
	config, err := LoadMCPConfig(configPath)
	if err != nil {
		return nil, []ValidationError{{
			Field:    "config",
			Message:  fmt.Sprintf("failed to load config: %v", err),
			Severity: "error",
		}}
	}
	
	validator := NewMCPServerValidator()
	errors := validator.ValidateConfig(config)
	
	return config, errors
}

// 辅助函数
func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}
	
	switch val := v.(type) {
	case string:
		return val == ""
	case []string:
		return val == nil || len(val) == 0
	case map[string]string:
		return val == nil || len(val) == 0
	default:
		return false
	}
}

func isCommandAvailable(cmd string) bool {
	// 检查命令是否在PATH中
	if filepath.IsAbs(cmd) {
		// 如果是绝对路径，检查文件是否存在
		return true // 简化版本，实际实现需要检查文件是否存在
	}
	
	// 检查命令是否在PATH中
	_, err := exec.LookPath(cmd)
	return err == nil
}