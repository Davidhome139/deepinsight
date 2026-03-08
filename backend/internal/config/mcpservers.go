package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// MCPServersConfig MCP 服务器配置
type MCPServersConfig struct {
	Servers  map[string]MCPServer `mapstructure:"servers"`
	Settings MCPServerSettings    `mapstructure:"settings"`
}

type MCPServer struct {
	Name         string            `mapstructure:"name"`
	Enabled      bool              `mapstructure:"enabled"`
	Type         string            `mapstructure:"type" json:"server_type"`
	Command      string            `mapstructure:"command"`
	Args         []string          `mapstructure:"args"`
	Env          map[string]string `mapstructure:"env"`
	AllowedPaths []string          `mapstructure:"allowed_paths"`
}

type MCPServerSettings struct {
	AutoDiscover bool `mapstructure:"auto_discover"`
	Timeout      int  `mapstructure:"timeout"`
	MaxTools     int  `mapstructure:"max_tools"`
}

var (
	mcpServersConfig     *MCPServersConfig
	mcpServersConfigPath string
	mcpServersMu         sync.RWMutex
)

// LoadMCPServersConfig 加载 MCP 服务器配置
func LoadMCPServersConfig(path string) (*MCPServersConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg MCPServersConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	mcpServersMu.Lock()
	mcpServersConfig = &cfg
	mcpServersConfigPath = path
	mcpServersMu.Unlock()

	log.Printf("Loaded MCP servers config from %s, servers: %d", path, len(cfg.Servers))
	return &cfg, nil
}

// GetMCPServersConfig 获取 MCP 服务器配置
func GetMCPServersConfig() *MCPServersConfig {
	mcpServersMu.RLock()
	defer mcpServersMu.RUnlock()
	return mcpServersConfig
}

// UpdateMCPServer 更新 MCP 服务器
func UpdateMCPServer(name string, server MCPServer) {
	mcpServersMu.Lock()
	defer mcpServersMu.Unlock()

	if mcpServersConfig == nil {
		mcpServersConfig = &MCPServersConfig{
			Servers: make(map[string]MCPServer),
		}
	}
	mcpServersConfig.Servers[name] = server

	v := viper.New()
	v.SetConfigFile(mcpServersConfigPath)
	v.Set("servers", mcpServersConfig.Servers)
	v.Set("settings", mcpServersConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save MCP servers config: %v", err)
	}
}

// DeleteMCPServer 删除 MCP 服务器
func DeleteMCPServer(name string) {
	mcpServersMu.Lock()
	defer mcpServersMu.Unlock()

	if mcpServersConfig == nil || mcpServersConfig.Servers == nil {
		return
	}
	delete(mcpServersConfig.Servers, name)

	v := viper.New()
	v.SetConfigFile(mcpServersConfigPath)
	v.Set("servers", mcpServersConfig.Servers)
	v.Set("settings", mcpServersConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save MCP servers config: %v", err)
	}
}
