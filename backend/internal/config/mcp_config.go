package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MCPServerConfig represents the configuration for an MCP server
type MCPServerConfig struct {
	MCPServer
	Documentation     *MCPServerDocumentation `json:"documentation,omitempty"`
	ToolsDiscovered   bool                    `json:"toolsDiscovered"`
	LastToolDiscovery time.Time               `json:"lastToolDiscovery,omitempty"`
}

// MCPConfig represents the complete MCP configuration
type MCPConfig struct {
	Servers  map[string]*MCPServerConfig `json:"mcpservers"`
	Settings struct {
		AutoDiscover bool `json:"autodiscover"`
	} `json:"settings"`
}

// LoadMCPConfig loads MCP configuration from file
func LoadMCPConfig(configPath string) (*MCPConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Initialize empty maps if nil
	if config.Servers == nil {
		config.Servers = make(map[string]*MCPServerConfig)
	}

	return &config, nil
}

// LoadAndValidateMCPConfig loads and validates MCP configuration
func LoadAndValidateMCPConfig(configPath string) (*MCPConfig, []ValidationError) {
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

// SaveMCPConfig saves MCP configuration to file
func SaveMCPConfig(config *MCPConfig, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateServerDocumentation updates documentation for a server in config
func (c *MCPConfig) UpdateServerDocumentation(serverName string, doc *MCPServerDocumentation) error {
	server, exists := c.Servers[serverName]
	if !exists {
		return fmt.Errorf("server %s not found in config", serverName)
	}

	server.Documentation = doc
	server.ToolsDiscovered = true
	server.LastToolDiscovery = time.Now()
	c.Servers[serverName] = server

	return nil
}

// GetServerConfig gets configuration for a specific server
func (c *MCPConfig) GetServerConfig(serverName string) (*MCPServerConfig, bool) {
	server, exists := c.Servers[serverName]
	return server, exists
}

// AddServer adds a new server to the configuration
func (c *MCPConfig) AddServer(serverName string, config *MCPServerConfig) {
	if c.Servers == nil {
		c.Servers = make(map[string]*MCPServerConfig)
	}
	c.Servers[serverName] = config
}

// RemoveServer removes a server from the configuration
func (c *MCPConfig) RemoveServer(serverName string) {
	delete(c.Servers, serverName)
}

// GetEnabledServers returns all enabled servers
func (c *MCPConfig) GetEnabledServers() []*MCPServerConfig {
	var enabled []*MCPServerConfig
	for _, server := range c.Servers {
		if server.Enabled {
			enabled = append(enabled, server)
		}
	}
	return enabled
}

// GetServerNames returns all server names
func (c *MCPConfig) GetServerNames() []string {
	var names []string
	for name := range c.Servers {
		names = append(names, name)
	}
	return names
}

// GetServerSummaries returns summaries for all servers with documentation
func (c *MCPConfig) GetServerSummaries() map[string]string {
	summaries := make(map[string]string)
	for name, server := range c.Servers {
		if server.Documentation != nil {
			summaries[name] = server.Documentation.Summary
		}
	}
	return summaries
}

// LoadDocumentation loads documentation from file
func LoadDocumentation(docsPath, serverName string) (*MCPServerDocumentation, error) {
	filename := filepath.Join(docsPath, fmt.Sprintf("%s_docs.json", serverName))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation file: %w", err)
	}

	var doc MCPServerDocumentation
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse documentation: %w", err)
	}

	return &doc, nil
}

// SaveDocumentation saves documentation to file
func SaveDocumentation(doc *MCPServerDocumentation, docsPath, serverName string) error {
	// Ensure directory exists
	if err := os.MkdirAll(docsPath, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	docJSON, err := doc.ConvertToJSON()
	if err != nil {
		return err
	}

	filename := filepath.Join(docsPath, fmt.Sprintf("%s_docs.json", serverName))
	if err := os.WriteFile(filename, []byte(docJSON), 0644); err != nil {
		return fmt.Errorf("failed to write documentation file: %w", err)
	}

	return nil
}

// MCPRegistryServer represents a server in the MCP registry
type MCPRegistryServer struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	PackageName    string                 `json:"package_name"`
	PackageType    string                 `json:"package_type"`
	Homepage       string                 `json:"homepage"`
	Repository     string                 `json:"repository"`
	InstallCommand string                 `json:"install_command"`
	TestCommand    string                 `json:"test_command"`
	DefaultConfig  map[string]interface{} `json:"default_config"`
}

// MCPRegistry represents the MCP registry structure
type MCPRegistry struct {
	Version     string                       `json:"version"`
	LastUpdated string                       `json:"last_updated"`
	Servers     map[string]MCPRegistryServer `json:"servers"`
	Categories  map[string][]string          `json:"categories"`
	Settings    map[string]interface{}       `json:"settings"`
}

// MCPRegistryConfig represents the complete MCP registry configuration
type MCPRegistryConfig struct {
	MCPRegistry MCPRegistry `json:"mcp_registry"`
}

// LoadMCPRegistry loads MCP registry from file
func LoadMCPRegistry() (*MCPRegistryConfig, error) {
	configDir := GetConfigDir()
	registryPath := filepath.Join(configDir, "mcp_registry.json")

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	var registry MCPRegistryConfig
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &registry, nil
}
