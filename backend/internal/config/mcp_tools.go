package config

import (
	"encoding/json"
	"fmt"
	"time"
)

// MCPTool represents a single tool from an MCP server
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
	
	// Usage information
	UsageScenarios []string           `json:"usageScenarios,omitempty"`
	InputParams    []ToolParameter    `json:"inputParams,omitempty"`
	OutputFormat   string             `json:"outputFormat,omitempty"`
	ExampleInput   map[string]interface{} `json:"exampleInput,omitempty"`
	ExampleOutput  interface{}        `json:"exampleOutput,omitempty"`
	
	// Metadata
	LastDiscovered time.Time          `json:"lastDiscovered,omitempty"`
	CallCount      int                `json:"callCount,omitempty"`
	SuccessRate    float64            `json:"successRate,omitempty"`
}

// ToolParameter represents a parameter for a tool
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// MCPServerDocumentation represents the complete documentation for an MCP server
type MCPServerDocumentation struct {
	// Server information
	ServerName    string    `json:"serverName"`
	ServerVersion string    `json:"serverVersion,omitempty"`
	Description   string    `json:"description"`
	
	// Overall server documentation
	Overview      string    `json:"overview"`
	UseCases      []string  `json:"useCases,omitempty"`
	Prerequisites []string  `json:"prerequisites,omitempty"`
	Limitations   []string  `json:"limitations,omitempty"`
	
	// Tools documentation
	Tools         []MCPTool `json:"tools"`
	
	// Summary for quick reference
	Summary       string    `json:"summary"`
	
	// Metadata
	LastUpdated   time.Time `json:"lastUpdated"`
	DiscoveryCount int      `json:"discoveryCount"`
}

// MCPServerWithDocs extends the basic MCPServer with documentation
type MCPServerWithDocs struct {
	MCPServer
	Documentation *MCPServerDocumentation `json:"documentation,omitempty"`
	ToolsDiscovered bool                  `json:"toolsDiscovered"`
	LastToolDiscovery time.Time           `json:"lastToolDiscovery,omitempty"`
}

// ConvertToJSON converts the documentation to JSON string
func (doc *MCPServerDocumentation) ConvertToJSON() (string, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal documentation: %w", err)
	}
	return string(data), nil
}

// GenerateSummary generates a summary of the server documentation
func (doc *MCPServerDocumentation) GenerateSummary() string {
	summary := fmt.Sprintf("MCP Server: %s\n", doc.ServerName)
	if doc.Description != "" {
		summary += fmt.Sprintf("Description: %s\n", doc.Description)
	}
	
	summary += fmt.Sprintf("Tools available: %d\n", len(doc.Tools))
	for i, tool := range doc.Tools {
		summary += fmt.Sprintf("  %d. %s: %s\n", i+1, tool.Name, tool.Description)
	}
	
	if len(doc.UseCases) > 0 {
		summary += "Use cases:\n"
		for _, useCase := range doc.UseCases {
			summary += fmt.Sprintf("  - %s\n", useCase)
		}
	}
	
	return summary
}

// GetToolByName finds a tool by name
func (doc *MCPServerDocumentation) GetToolByName(name string) (*MCPTool, bool) {
	for _, tool := range doc.Tools {
		if tool.Name == name {
			return &tool, true
		}
	}
	return nil, false
}

// ExtractInputSchema extracts input parameters from tool schema
func ExtractInputSchema(schema map[string]interface{}) []ToolParameter {
	var params []ToolParameter
	
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for paramName, paramData := range properties {
			paramMap, ok := paramData.(map[string]interface{})
			if !ok {
				continue
			}
			
			param := ToolParameter{
				Name: paramName,
			}
			
			if typeVal, ok := paramMap["type"].(string); ok {
				param.Type = typeVal
			}
			
			if desc, ok := paramMap["description"].(string); ok {
				param.Description = desc
			}
			
			// Check if required
			if requiredList, ok := schema["required"].([]interface{}); ok {
				for _, req := range requiredList {
					if reqName, ok := req.(string); ok && reqName == paramName {
						param.Required = true
						break
					}
				}
			}
			
			if defVal, ok := paramMap["default"]; ok {
				param.Default = defVal
			}
			
			if enumVal, ok := paramMap["enum"].([]interface{}); ok {
				for _, e := range enumVal {
					if eStr, ok := e.(string); ok {
						param.Enum = append(param.Enum, eStr)
					}
				}
			}
			
			params = append(params, param)
		}
	}
	
	return params
}