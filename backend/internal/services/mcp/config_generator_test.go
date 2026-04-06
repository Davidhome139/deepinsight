package mcp

import (
	"backend/internal/config"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfigGenerator(t *testing.T) {
	generator := NewDefaultConfigGenerator()
	assert.NotNil(t, generator)
}

func TestDefaultConfigGenerator_GenerateServerConfig_NPM(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	doc := &DocumentationInfo{
		Name:        "@upstash/context7-mcp",
		Description: "Context7 MCP server",
		Homepage:    "https://npmjs.com/package/@upstash/context7-mcp",
	}

	depInfo := &DependencyInfo{
		Name:        "context7",
		PackageName: "@upstash/context7-mcp",
		Type:        DependencyTypeNPM,
		InstallCmd:  "npm install -g @upstash/context7-mcp",
		TestCmd:     "npx -y @upstash/context7-mcp --version",
	}

	server, err := generator.GenerateServerConfig(doc, depInfo)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "context7", server.Name)
	assert.Equal(t, true, server.Enabled)
	assert.Equal(t, "command", server.Type)
	assert.Equal(t, "npx", server.Command)
	assert.Equal(t, []string{"-y", "@upstash/context7-mcp"}, server.Args)
	assert.NotNil(t, server.AutomationInfo)
	assert.Equal(t, true, server.AutomationInfo.AutoInstall)
	assert.Equal(t, "npm", server.AutomationInfo.PackageManager)
	assert.Equal(t, "@upstash/context7-mcp", server.AutomationInfo.PackageName)
}

func TestDefaultConfigGenerator_GenerateServerConfig_ScopedPackage(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	doc := &DocumentationInfo{
		Name:        "@organization/package-mcp",
		Description: "Test MCP server",
		Homepage:    "https://example.com",
	}

	depInfo := &DependencyInfo{
		Name:        "package",
		PackageName: "@organization/package-mcp",
		Type:        DependencyTypeNPM,
		InstallCmd:  "npm install -g @organization/package-mcp",
		TestCmd:     "npx -y @organization/package-mcp --version",
	}

	server, err := generator.GenerateServerConfig(doc, depInfo)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "package", server.Name) // 应该去除@organization/前缀和-mcp后缀
	assert.Equal(t, []string{"-y", "@organization/package-mcp"}, server.Args)
}

func TestDefaultConfigGenerator_GenerateServerConfig_Go(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	doc := &DocumentationInfo{
		Name:        "github.com/mark3labs/mcp-go",
		Description: "MCP Go SDK",
		Homepage:    "https://github.com/mark3labs/mcp-go",
	}

	depInfo := &DependencyInfo{
		Name:        "mcp-go",
		PackageName: "github.com/mark3labs/mcp-go",
		Type:        DependencyTypeGo,
		InstallCmd:  "go install github.com/mark3labs/mcp-go@latest",
		TestCmd:     "mcp-go --version",
	}

	server, err := generator.GenerateServerConfig(doc, depInfo)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "mcp-go", server.Name)
	assert.Equal(t, "mcp-go", server.Command)
	assert.Equal(t, []string{}, server.Args)
	assert.NotNil(t, server.AutomationInfo)
	assert.Equal(t, "go", server.AutomationInfo.PackageManager)
}

func TestDefaultConfigGenerator_GenerateServerConfig_Pip(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	doc := &DocumentationInfo{
		Name:        "mcp-server",
		Description: "Python MCP server",
		Homepage:    "https://pypi.org/project/mcp-server",
	}

	depInfo := &DependencyInfo{
		Name:        "mcp-server",
		PackageName: "mcp-server",
		Type:        DependencyTypePip,
		InstallCmd:  "pip install mcp-server",
		TestCmd:     "python -m mcp_server --version",
	}

	server, err := generator.GenerateServerConfig(doc, depInfo)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "mcp", server.Name) // extractServerName会移除-server后缀
	assert.Equal(t, "python", server.Command)
	assert.Equal(t, []string{"-m", "mcp-server"}, server.Args)
	assert.NotNil(t, server.AutomationInfo)
	assert.Equal(t, "pip", server.AutomationInfo.PackageManager)
}

func TestDefaultConfigGenerator_GenerateServerConfig_Docker(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	doc := &DocumentationInfo{
		Name:        "nginx",
		Description: "Nginx web server",
		Homepage:    "https://hub.docker.com/_/nginx",
	}

	depInfo := &DependencyInfo{
		Name:        "nginx",
		PackageName: "nginx",
		Type:        DependencyTypeDocker,
		InstallCmd:  "docker pull nginx",
		TestCmd:     "docker run --rm nginx --version",
	}

	server, err := generator.GenerateServerConfig(doc, depInfo)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "nginx", server.Name)
	assert.Equal(t, "docker", server.Command)
	assert.Equal(t, []string{"run", "--rm", "-i", "nginx"}, server.Args)
	assert.NotNil(t, server.AutomationInfo)
	assert.Equal(t, "docker", server.AutomationInfo.PackageManager)
}

func TestDefaultConfigGenerator_GenerateServerConfig_InvalidInput(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	// 测试空文档
	_, err := generator.GenerateServerConfig(nil, &DependencyInfo{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "documentation info is required")

	// 测试空依赖信息
	_, err = generator.GenerateServerConfig(&DocumentationInfo{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency info is required")
}

func TestDefaultConfigGenerator_ExtractServerName(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	testCases := []struct {
		input    string
		expected string
	}{
		{"@upstash/context7-mcp", "context7"},
		{"@organization/package-mcp-server", "package-mcp"}, // 先移除-server，再移除-mcp
		{"simple-package", "simple-package"},
		{"mcp-server", "mcp"},
		{"@scope/name-mcp", "name"},
		{"", "mcp-server"},    // 空输入返回默认值
		{"@single", "single"}, // 只有@前缀，移除@
	}

	for _, tc := range testCases {
		result := generator.extractServerName(tc.input)
		assert.Equal(t, tc.expected, result, "Input: %s", tc.input)
	}
}

func TestDefaultConfigGenerator_SaveConfigToFile(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	server := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    "test-server",
			Enabled: true,
			Type:    "command",
			Command: "npx",
			Args:    []string{"-y", "test-package"},
		},
		AutomationInfo: &config.MCPServerAutomationInfo{
			AutoInstall:    true,
			AutoUpdate:     true,
			PackageManager: "npm",
			PackageName:    "test-package",
			InstallScript:  "npm install -g test-package",
			InstallStatus:  "pending",
			UpdateStatus:   "pending",
		},
	}

	// 保存配置
	err := generator.SaveConfigToFile(server, configPath)
	assert.NoError(t, err)

	// 验证文件存在
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// 读取并验证文件内容
	data, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	// 验证JSON格式
	var configData map[string]interface{}
	err = json.Unmarshal(data, &configData)
	assert.NoError(t, err)

	// 验证服务器配置存在
	servers, ok := configData["mcpServers"].(map[string]interface{})
	assert.True(t, ok)

	serverConfig, ok := servers["test-server"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test-server", serverConfig["name"])
	assert.Equal(t, true, serverConfig["enabled"])
	assert.Equal(t, "command", serverConfig["type"])
	assert.Equal(t, "npx", serverConfig["command"])

	// 验证自动化信息
	automationInfo, ok := serverConfig["automationInfo"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, automationInfo["autoInstall"])
	assert.Equal(t, "npm", automationInfo["packageManager"])
	assert.Equal(t, "test-package", automationInfo["packageName"])
}

func TestDefaultConfigGenerator_SaveConfigToFile_UpdateExisting(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建初始配置
	initialConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"existing-server": map[string]interface{}{
				"name":    "existing-server",
				"enabled": true,
				"type":    "command",
				"command": "echo",
				"args":    []string{"hello"},
			},
		},
	}

	data, err := json.MarshalIndent(initialConfig, "", "  ")
	assert.NoError(t, err)

	err = os.WriteFile(configPath, data, 0644)
	assert.NoError(t, err)

	// 添加新服务器
	newServer := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    "new-server",
			Enabled: true,
			Type:    "command",
			Command: "npx",
			Args:    []string{"-y", "new-package"},
		},
	}

	err = generator.SaveConfigToFile(newServer, configPath)
	assert.NoError(t, err)

	// 读取并验证文件内容
	data, err = os.ReadFile(configPath)
	assert.NoError(t, err)

	var configData map[string]interface{}
	err = json.Unmarshal(data, &configData)
	assert.NoError(t, err)

	servers, ok := configData["mcpServers"].(map[string]interface{})
	assert.True(t, ok)

	// 验证两个服务器都存在
	assert.Contains(t, servers, "existing-server")
	assert.Contains(t, servers, "new-server")
}

func TestDefaultConfigGenerator_SaveConfigToFile_InvalidServer(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 测试空服务器
	err := generator.SaveConfigToFile(nil, configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server config is required")
}

func TestDefaultConfigGenerator_UpdateExistingConfig(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	existingConfig := &config.MCPServersConfigWithAutomation{
		Servers: make(map[string]config.MCPServerWithAutomation),
	}

	newServer := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    "new-server",
			Enabled: true,
		},
	}

	err := generator.UpdateExistingConfig(existingConfig, newServer)
	assert.NoError(t, err)
	assert.Contains(t, existingConfig.Servers, "new-server")
	assert.True(t, existingConfig.Servers["new-server"].Enabled)
}

func TestDefaultConfigGenerator_UpdateExistingConfig_InvalidInput(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	// 测试空现有配置
	err := generator.UpdateExistingConfig(nil, &config.MCPServerWithAutomation{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "existing config is required")

	// 测试空新服务器
	err = generator.UpdateExistingConfig(&config.MCPServersConfigWithAutomation{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "new server config is required")

	// 测试未初始化的Servers map
	existingConfig := &config.MCPServersConfigWithAutomation{}
	newServer := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name: "test-server",
		},
	}

	err = generator.UpdateExistingConfig(existingConfig, newServer)
	assert.NoError(t, err)
	assert.NotNil(t, existingConfig.Servers)
	assert.Contains(t, existingConfig.Servers, "test-server")
}
