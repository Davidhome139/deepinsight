package mcp

import (
	"backend/internal/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigGenerator 配置生成器接口
type ConfigGenerator interface {
	GenerateServerConfig(doc *DocumentationInfo, depInfo *DependencyInfo) (*config.MCPServerWithAutomation, error)
	SaveConfigToFile(server *config.MCPServerWithAutomation, configPath string) error
	UpdateExistingConfig(existingConfig *config.MCPServersConfigWithAutomation, newServer *config.MCPServerWithAutomation) error
}

// DefaultConfigGenerator 默认配置生成器
type DefaultConfigGenerator struct{}

func NewDefaultConfigGenerator() *DefaultConfigGenerator {
	return &DefaultConfigGenerator{}
}

func (g *DefaultConfigGenerator) GenerateServerConfig(doc *DocumentationInfo, depInfo *DependencyInfo) (*config.MCPServerWithAutomation, error) {
	if doc == nil {
		return nil, fmt.Errorf("documentation info is required")
	}
	if depInfo == nil {
		return nil, fmt.Errorf("dependency info is required")
	}

	// 生成服务器名称（从包名中提取）
	serverName := g.extractServerName(depInfo.PackageName)

	// 根据依赖类型生成配置
	var serverConfig *config.MCPServerWithAutomation

	switch depInfo.Type {
	case DependencyTypeNPM:
		serverConfig = g.generateNPMConfig(serverName, doc, depInfo)
	case DependencyTypeGo:
		serverConfig = g.generateGoConfig(serverName, doc, depInfo)
	case DependencyTypePip:
		serverConfig = g.generatePipConfig(serverName, doc, depInfo)
	case DependencyTypeDocker:
		serverConfig = g.generateDockerConfig(serverName, doc, depInfo)
	default:
		// 默认使用NPM配置
		serverConfig = g.generateNPMConfig(serverName, doc, depInfo)
	}

	return serverConfig, nil
}

func (g *DefaultConfigGenerator) extractServerName(packageName string) string {
	if packageName == "" {
		return "mcp-server"
	}

	// 移除作用域前缀（如@organization/）
	name := packageName
	if strings.HasPrefix(name, "@") {
		parts := strings.Split(name, "/")
		if len(parts) > 1 {
			name = parts[1]
		} else {
			// 如果只有@没有/，移除@
			name = strings.TrimPrefix(name, "@")
		}
	}

	// 移除常见的后缀
	suffixes := []string{"-mcp", "-server", "-mcp-server"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
		}
	}

	// 如果名称仍然包含特殊字符，使用最后一部分
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		name = parts[len(parts)-1]
	}

	// 确保名称不为空
	if name == "" {
		name = "mcp-server"
	}

	return name
}

func (g *DefaultConfigGenerator) generateNPMConfig(serverName string, doc *DocumentationInfo, depInfo *DependencyInfo) *config.MCPServerWithAutomation {
	return &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    serverName,
			Enabled: true,
			Type:    "command",
			Command: "npx",
			Args:    []string{"-y", depInfo.PackageName},
		},
		AutomationInfo: &config.MCPServerAutomationInfo{
			AutoInstall:    true,
			AutoUpdate:     true,
			PackageManager: "npm",
			PackageName:    depInfo.PackageName,
			InstallScript:  depInfo.InstallCmd,
			InstallStatus:  "pending",
			UpdateStatus:   "pending",
		},
	}
}

func (g *DefaultConfigGenerator) generateGoConfig(serverName string, doc *DocumentationInfo, depInfo *DependencyInfo) *config.MCPServerWithAutomation {
	// 对于Go包，命令通常是二进制名称
	command := serverName
	if strings.Contains(depInfo.PackageName, "/") {
		parts := strings.Split(depInfo.PackageName, "/")
		command = parts[len(parts)-1]
	}

	return &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    serverName,
			Enabled: true,
			Type:    "command",
			Command: command,
			Args:    []string{},
		},
		AutomationInfo: &config.MCPServerAutomationInfo{
			AutoInstall:    true,
			AutoUpdate:     true,
			PackageManager: "go",
			PackageName:    depInfo.PackageName,
			InstallScript:  depInfo.InstallCmd,
			InstallStatus:  "pending",
			UpdateStatus:   "pending",
		},
	}
}

func (g *DefaultConfigGenerator) generatePipConfig(serverName string, doc *DocumentationInfo, depInfo *DependencyInfo) *config.MCPServerWithAutomation {
	// 对于Python包，命令通常是模块名
	command := "python"
	args := []string{"-m", depInfo.PackageName}

	// 如果包名包含点，可能是模块路径
	if strings.Contains(depInfo.PackageName, ".") {
		args = []string{"-m", depInfo.PackageName}
	}

	return &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    serverName,
			Enabled: true,
			Type:    "command",
			Command: command,
			Args:    args,
		},
		AutomationInfo: &config.MCPServerAutomationInfo{
			AutoInstall:    true,
			AutoUpdate:     true,
			PackageManager: "pip",
			PackageName:    depInfo.PackageName,
			InstallScript:  depInfo.InstallCmd,
			InstallStatus:  "pending",
			UpdateStatus:   "pending",
		},
	}
}

func (g *DefaultConfigGenerator) generateDockerConfig(serverName string, doc *DocumentationInfo, depInfo *DependencyInfo) *config.MCPServerWithAutomation {
	return &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    serverName,
			Enabled: true,
			Type:    "command",
			Command: "docker",
			Args:    []string{"run", "--rm", "-i", depInfo.PackageName},
		},
		AutomationInfo: &config.MCPServerAutomationInfo{
			AutoInstall:    true,
			AutoUpdate:     true,
			PackageManager: "docker",
			PackageName:    depInfo.PackageName,
			InstallScript:  depInfo.InstallCmd,
			InstallStatus:  "pending",
			UpdateStatus:   "pending",
		},
	}
}

func (g *DefaultConfigGenerator) SaveConfigToFile(server *config.MCPServerWithAutomation, configPath string) error {
	if server == nil {
		return fmt.Errorf("server config is required")
	}

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// 读取现有配置或创建新配置
	var configData map[string]interface{}

	if _, err := os.Stat(configPath); err == nil {
		// 文件存在，读取现有配置
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read existing config: %v", err)
		}

		if err := json.Unmarshal(data, &configData); err != nil {
			// 如果JSON解析失败，创建新的配置结构
			configData = make(map[string]interface{})
		}
	} else {
		// 文件不存在，创建新的配置结构
		configData = make(map[string]interface{})
	}

	// 确保mcpServers字段存在
	if _, ok := configData["mcpServers"]; !ok {
		configData["mcpServers"] = make(map[string]interface{})
	}

	servers, ok := configData["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
		configData["mcpServers"] = servers
	}

	// 转换服务器配置为map
	serverMap := make(map[string]interface{})
	serverMap["name"] = server.Name
	serverMap["enabled"] = server.Enabled
	serverMap["type"] = server.Type
	serverMap["command"] = server.Command
	serverMap["args"] = server.Args

	// 添加自动化信息
	if server.AutomationInfo != nil {
		automationMap := make(map[string]interface{})
		automationMap["autoInstall"] = server.AutomationInfo.AutoInstall
		automationMap["autoUpdate"] = server.AutomationInfo.AutoUpdate
		automationMap["packageManager"] = server.AutomationInfo.PackageManager
		automationMap["packageName"] = server.AutomationInfo.PackageName
		automationMap["installScript"] = server.AutomationInfo.InstallScript
		automationMap["installStatus"] = server.AutomationInfo.InstallStatus
		automationMap["updateStatus"] = server.AutomationInfo.UpdateStatus
		serverMap["automationInfo"] = automationMap
	}

	// 添加到服务器列表
	servers[server.Name] = serverMap

	// 写入文件
	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (g *DefaultConfigGenerator) UpdateExistingConfig(existingConfig *config.MCPServersConfigWithAutomation, newServer *config.MCPServerWithAutomation) error {
	if existingConfig == nil {
		return fmt.Errorf("existing config is required")
	}
	if newServer == nil {
		return fmt.Errorf("new server config is required")
	}

	// 确保Servers map已初始化
	if existingConfig.Servers == nil {
		existingConfig.Servers = make(map[string]config.MCPServerWithAutomation)
	}

	// 添加或更新服务器配置
	existingConfig.Servers[newServer.Name] = *newServer

	return nil
}
