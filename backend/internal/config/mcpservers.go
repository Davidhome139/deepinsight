package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"
)

// MCPServersConfig MCP 服务器配置结构体
type MCPServersConfig struct {
	Servers  map[string]MCPServer `mapstructure:"mcpServers"`
	Settings MCPServerSettings    `mapstructure:"settings"`
}

// MCPManagerInterface defines the interface for MCP manager
type MCPManagerInterface interface {
	ConnectToServer(serverName string) error
	GetServer(serverName string) (*MCPServer, bool)
	Discover()
}

// Global MCP manager reference
var (
	mcpManagerInstance MCPManagerInterface
	mcpManagerMu       sync.RWMutex
)

// SetMCPManager sets the global MCP manager instance
func SetMCPManager(manager MCPManagerInterface) {
	mcpManagerMu.Lock()
	defer mcpManagerMu.Unlock()
	mcpManagerInstance = manager
	log.Println("[Config] MCP manager instance set")
}

// GetMCPManager gets the global MCP manager instance
func GetMCPManager() MCPManagerInterface {
	mcpManagerMu.RLock()
	defer mcpManagerMu.RUnlock()
	return mcpManagerInstance
}

type MCPServer struct {
	Name          string            `mapstructure:"name"`
	Enabled       bool              `mapstructure:"enabled"`
	Type          string            `mapstructure:"type" json:"server_type,omitempty"`
	Command       string            `mapstructure:"command"`
	Args          []string          `mapstructure:"args"`
	Env           map[string]string `mapstructure:"env"`
	FromGalleryId []string          `mapstructure:"fromGalleryId"`
	URL           string            `mapstructure:"url"`
	// Runtime fields (not persisted to config)
	Client               *client.Client `json:"-" mapstructure:"-"`
	Connected            bool           `json:"connected" mapstructure:"-"`
	Tools                []mcp.Tool     `json:"tools" mapstructure:"-"`
	LastError            string         `json:"last_error,omitempty" mapstructure:"-"`
	NeedsReconnect       bool           `json:"needs_reconnect,omitempty" mapstructure:"-"`
	LastReconnectAttempt time.Time      `json:"last_reconnect_attempt,omitempty" mapstructure:"-"`
}

type MCPServerSettings struct {
	AutoDiscover bool `mapstructure:"auto_discover"`
	Timeout      int  `mapstructure:"timeout"`
	MaxTools     int  `mapstructure:"max_tools"`
}

// MCPServerAutomationInfo MCP服务器自动化信息
type MCPServerAutomationInfo struct {
	AutoInstall    bool   `json:"autoInstall" mapstructure:"autoInstall"`
	AutoUpdate     bool   `json:"autoUpdate" mapstructure:"autoUpdate"`
	PackageManager string `json:"packageManager" mapstructure:"packageManager"`
	PackageName    string `json:"packageName" mapstructure:"packageName"`
	InstallScript  string `json:"installScript" mapstructure:"installScript"`
	InstallStatus  string `json:"installStatus" mapstructure:"installStatus"` // pending, installing, installed, failed
	UpdateStatus   string `json:"updateStatus" mapstructure:"updateStatus"`   // pending, updating, updated, failed
}

// MCPServerWithAutomation 带自动化信息的MCP服务器
type MCPServerWithAutomation struct {
	MCPServer
	AutomationInfo *MCPServerAutomationInfo `json:"automationInfo,omitempty" mapstructure:"automationInfo"`
}

// MCPServersConfigWithAutomation 带自动化信息的MCP服务器配置
type MCPServersConfigWithAutomation struct {
	Servers  map[string]MCPServerWithAutomation `json:"mcpServers" mapstructure:"mcpServers"`
	Settings MCPServerSettings                  `json:"settings" mapstructure:"settings"`
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
	v.SetConfigType("json") // 显式设置配置文件类型为JSON
	SetupViperEnv(v, "MCPSERVERS")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg MCPServersConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 从环境变量中读取Env
	for key, server := range cfg.Servers {
		// 从环境变量中读取Env
		if server.Env == nil {
			server.Env = make(map[string]string)
		}

		// 这里需要根据实际的Env键名来读取环境变量
		// 由于Env的键名是动态的，我们需要一种方式来知道哪些Env键需要从环境变量中读取
		// 这里暂时不实现，因为Env的键名是用户自定义的

		// 更新server
		cfg.Servers[key] = server
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
	config := mcpServersConfig
	mcpServersMu.RUnlock()

	// 如果配置未加载，尝试自动加载
	if config == nil {
		configDir := "config"
		configPath := filepath.Join(configDir, "mcpservers.json")

		log.Printf("[MCP] Config not loaded, attempting to load from %s", configPath)

		// 检查配置文件是否存在
		if _, err := os.Stat(configPath); err == nil {
			log.Printf("[MCP] Config file exists at %s", configPath)
			// 配置文件存在，尝试加载
			if loadedConfig, err := LoadMCPServersConfig(configPath); err == nil {
				log.Printf("[MCP] Successfully loaded config with %d servers", len(loadedConfig.Servers))
				return loadedConfig
			} else {
				log.Printf("[MCP] Failed to load config: %v", err)
			}
		} else {
			log.Printf("[MCP] Config file does not exist at %s: %v", configPath, err)
		}

		// 如果配置文件不存在或加载失败，返回空配置
		log.Println("[MCP] Returning empty config")
		return &MCPServersConfig{
			Servers: make(map[string]MCPServer),
		}
	}

	log.Printf("[MCP] Returning cached config with %d servers", len(config.Servers))
	return config
}

// UpdateMCPServer 更新 MCP 服务器
func UpdateMCPServer(name string, server MCPServer) {
	mcpServersMu.Lock()
	defer mcpServersMu.Unlock()

	// 确保配置路径是JSON文件
	if mcpServersConfigPath == "" || filepath.Ext(mcpServersConfigPath) != ".json" {
		// 如果路径为空或不是JSON文件，使用默认的JSON配置路径
		configDir := "config"
		mcpServersConfigPath = filepath.Join(configDir, "mcpservers.json")
	}

	if mcpServersConfig == nil {
		mcpServersConfig = &MCPServersConfig{
			Servers: make(map[string]MCPServer),
		}
	}

	// 提取Env中的敏感信息并保存到.env文件
	configDir := filepath.Dir(mcpServersConfigPath)
	keyValuePairs := make(map[string]string)

	// 构建环境变量键名前缀
	prefix := "MCPSERVERS_SERVERS_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_ENV"

	// 保存Env中的敏感信息到.env
	if server.Env != nil {
		for key, value := range server.Env {
			// 构建完整的环境变量键名
			envKey := prefix + "_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
			keyValuePairs[envKey] = value
		}
	}

	// 更新.env文件
	if len(keyValuePairs) > 0 {
		if err := UpdateEnvFile(configDir, keyValuePairs); err != nil {
			log.Printf("Failed to update .env file: %v", err)
		}
	}

	// 清空Env后再保存到mcpservers.json
	sanitizedServer := server
	sanitizedServer.Env = nil

	// 保存到mcpservers.json
	v := viper.New()
	v.SetConfigFile(mcpServersConfigPath)
	v.SetConfigType("json") // 显式设置配置文件类型为JSON

	// 读取现有配置
	if err := v.ReadInConfig(); err != nil {
		log.Printf("Failed to read MCP servers config: %v", err)
		// 如果读取失败，初始化一个新的配置结构
		v.Set("mcpServers", make(map[string]interface{}))
		v.Set("settings", MCPServerSettings{})
	}

	// 将sanitizedServer转换为map[string]interface{}以便正确序列化
	// 使用与现有配置相同的字段名（保持大小写一致）
	serverMap := make(map[string]interface{})
	// 检查现有配置中的字段名，保持一致性
	existingServers := v.Get("mcpServers").(map[string]interface{})
	var existingServerMap map[string]interface{}
	if existingServer, ok := existingServers[name]; ok {
		if server, ok := existingServer.(map[string]interface{}); ok {
			existingServerMap = server
		}
	}

	// 如果存在现有配置，使用相同的字段名
	if existingServerMap != nil {
		// 复制现有字段名
		for key, value := range existingServerMap {
			serverMap[key] = value
		}
		// 更新字段值，统一使用小写字段名
		serverMap["name"] = sanitizedServer.Name
		serverMap["enabled"] = sanitizedServer.Enabled
		serverMap["type"] = sanitizedServer.Type
		serverMap["command"] = sanitizedServer.Command
		serverMap["args"] = sanitizedServer.Args
		serverMap["env"] = sanitizedServer.Env
		serverMap["fromGalleryId"] = sanitizedServer.FromGalleryId
	} else {
		// 新服务器，使用标准字段名（小写）
		serverMap["name"] = sanitizedServer.Name
		serverMap["enabled"] = sanitizedServer.Enabled
		serverMap["type"] = sanitizedServer.Type
		serverMap["command"] = sanitizedServer.Command
		serverMap["args"] = sanitizedServer.Args
		serverMap["env"] = sanitizedServer.Env
		serverMap["fromGalleryId"] = sanitizedServer.FromGalleryId
	}

	// 更新指定服务器的配置
	servers := v.Get("mcpServers").(map[string]interface{})
	servers[name] = serverMap
	v.Set("mcpServers", servers)

	// 保存到文件
	log.Printf("UpdateMCPServer: Saving config to %s", mcpServersConfigPath)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save MCP servers config: %v", err)
	} else {
		log.Printf("UpdateMCPServer: Config saved successfully for server %s", name)
		// 记录保存的内容
		log.Printf("UpdateMCPServer: Saved serverMap: %+v", serverMap)
	}

	// 更新内存中的配置
	// 直接设置内存中的配置，而不是通过重新加载文件，避免死锁
	// 使用原始server对象（包含Env）而不是sanitizedServer
	mcpServersConfig.Servers[name] = server

	// 如果服务器被启用，立即尝试连接
	if server.Enabled {
		log.Printf("UpdateMCPServer: Server %s is enabled, attempting to connect immediately", name)

		// 获取MCP管理器并尝试连接
		mcpManager := GetMCPManager()
		if mcpManager != nil {
			go func() {
				log.Printf("UpdateMCPServer: Starting immediate connection attempt for %s", name)
				// 先触发发现，确保服务器被注册
				mcpManager.Discover()

				// 等待一小段时间让发现过程完成
				time.Sleep(500 * time.Millisecond)

				// 尝试连接服务器
				if err := mcpManager.ConnectToServer(name); err != nil {
					log.Printf("UpdateMCPServer: Failed to connect to server %s: %v", name, err)
				} else {
					log.Printf("UpdateMCPServer: Successfully connected to server %s", name)
				}
			}()
		} else {
			log.Printf("UpdateMCPServer: MCP manager not available, connection will be attempted on next discovery")
		}
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

// AddMCPServer 添加 MCP 服务器
func AddMCPServer(name string, server MCPServer) {
	mcpServersMu.Lock()
	defer mcpServersMu.Unlock()

	if mcpServersConfig == nil {
		mcpServersConfig = &MCPServersConfig{
			Servers: make(map[string]MCPServer),
		}
	}

	// 如果Servers为nil，初始化它
	if mcpServersConfig.Servers == nil {
		mcpServersConfig.Servers = make(map[string]MCPServer)
	}

	// 提取Env中的敏感信息并保存到.env文件
	configDir := filepath.Dir(mcpServersConfigPath)
	keyValuePairs := make(map[string]string)

	// 构建环境变量键名前缀
	prefix := "MCPSERVERS_SERVERS_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_ENV"

	// 保存Env中的敏感信息到.env
	if server.Env != nil {
		for key, value := range server.Env {
			// 构建完整的环境变量键名
			envKey := prefix + "_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
			keyValuePairs[envKey] = value
		}
	}

	// 更新.env文件
	if len(keyValuePairs) > 0 {
		if err := UpdateEnvFile(configDir, keyValuePairs); err != nil {
			log.Printf("Failed to update .env file: %v", err)
		}
	}

	// 清空Env后再保存到mcpservers.json
	sanitizedServer := server
	sanitizedServer.Env = nil

	// 保存到mcpservers.json
	v := viper.New()
	v.SetConfigFile(mcpServersConfigPath)
	v.SetConfigType("json") // 显式设置配置文件类型为JSON

	// 读取现有配置
	if err := v.ReadInConfig(); err != nil {
		log.Printf("Failed to read MCP servers config: %v", err)
		// 如果读取失败，初始化一个新的配置结构
		v.Set("mcpServers", make(map[string]interface{}))
		v.Set("settings", MCPServerSettings{})
	}

	// 将sanitizedServer转换为map[string]interface{}以便正确序列化
	// 新服务器，使用标准字段名（小写）
	serverMap := make(map[string]interface{})
	serverMap["name"] = sanitizedServer.Name
	serverMap["enabled"] = sanitizedServer.Enabled
	serverMap["type"] = sanitizedServer.Type
	serverMap["command"] = sanitizedServer.Command
	serverMap["args"] = sanitizedServer.Args
	serverMap["env"] = sanitizedServer.Env
	serverMap["fromGalleryId"] = sanitizedServer.FromGalleryId

	// 添加新服务器的配置
	servers := v.Get("mcpServers").(map[string]interface{})
	servers[name] = serverMap
	v.Set("mcpServers", servers)

	// 保存到文件
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save MCP servers config: %v", err)
	}

	// 更新内存中的配置
	// 直接设置内存中的配置，而不是通过重新加载文件，避免死锁
	// 使用原始server对象（包含Env）而不是sanitizedServer
	mcpServersConfig.Servers[name] = server
}
