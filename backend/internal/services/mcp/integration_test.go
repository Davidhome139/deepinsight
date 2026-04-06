package mcp

import (
	"backend/internal/config"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestMCPAutomationIntegration 测试MCP自动化集成
func TestMCPAutomationIntegration(t *testing.T) {
	// 跳过集成测试，除非明确要求
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "mcp_docs")

	// 创建模拟对象
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	// 创建自动化协调器
	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 测试1: 启动协调器
	t.Run("StartCoordinator", func(t *testing.T) {
		// 设置模拟期望
		mockHotReloadManager.On("Start", mock.Anything).Return(nil)

		ctx := context.Background()
		err := coordinator.Start(ctx)
		assert.NoError(t, err)

		// 检查状态
		status := coordinator.GetStatus()
		assert.True(t, status.Running)
	})

	// 测试2: 添加MCP包
	t.Run("AddMCPPackage", func(t *testing.T) {
		packageName := "test-mcp-server"
		depType := DependencyTypeNPM

		// 设置模拟期望
		mockDepManager.On("CheckDependency", mock.Anything, mock.AnythingOfType("DependencyInfo")).Return(false, nil)
		mockDepManager.On("InstallDependency", mock.Anything, mock.AnythingOfType("DependencyInfo")).Return(nil)

		docInfo := &DocumentationInfo{
			Name:        packageName,
			Description: "Test MCP server",
			Version:     "1.0.0",
			Homepage:    "https://example.com/test-mcp",
		}
		mockDocFetcher.On("FetchDocumentation", mock.Anything, packageName, depType).Return(docInfo, nil)

		serverConfig := &config.MCPServerWithAutomation{
			MCPServer: config.MCPServer{
				Name:    packageName,
				Enabled: true,
				Command: packageName,
			},
		}
		mockConfigGenerator.On("GenerateServerConfig", docInfo, mock.AnythingOfType("*DependencyInfo")).Return(serverConfig, nil)
		mockConfigGenerator.On("SaveConfigToFile", serverConfig, configPath).Return(nil)

		mockMCPManager.On("ConnectToServer", packageName).Return(nil)

		// 添加包
		err := coordinator.AddMCPPackage(packageName, depType)
		assert.NoError(t, err)

		// 等待自动化完成
		time.Sleep(500 * time.Millisecond)

		// 检查包状态
		pkgStatus, err := coordinator.GetPackageStatus(packageName)
		assert.NoError(t, err)
		assert.NotNil(t, pkgStatus)
		assert.Equal(t, packageName, pkgStatus.PackageName)
		assert.Equal(t, depType, pkgStatus.DependencyType)
	})

	// 测试3: 更新MCP包
	t.Run("UpdateMCPPackage", func(t *testing.T) {
		packageName := "test-mcp-server"
		newDepType := DependencyTypeGo

		// 设置模拟期望
		mockDepManager.On("UpdateDependency", mock.Anything, mock.AnythingOfType("DependencyInfo")).Return(nil)
		mockMCPManager.On("ConnectToServer", packageName).Return(nil)

		// 更新包
		err := coordinator.UpdateMCPPackage(packageName, newDepType)
		assert.NoError(t, err)

		// 等待更新完成
		time.Sleep(500 * time.Millisecond)

		// 检查包状态
		pkgStatus, err := coordinator.GetPackageStatus(packageName)
		assert.NoError(t, err)
		assert.Equal(t, newDepType, pkgStatus.DependencyType)
	})

	// 测试4: 移除MCP包
	t.Run("RemoveMCPPackage", func(t *testing.T) {
		packageName := "test-mcp-server"

		// 设置模拟期望
		mockDepManager.On("RemoveDependency", mock.Anything, mock.AnythingOfType("DependencyInfo")).Return(nil)

		// 移除包
		err := coordinator.RemoveMCPPackage(packageName)
		assert.NoError(t, err)

		// 检查包已移除
		status := coordinator.GetStatus()
		assert.NotContains(t, status.Packages, packageName)
	})

	// 测试5: 停止协调器
	t.Run("StopCoordinator", func(t *testing.T) {
		// 设置模拟期望
		mockHotReloadManager.On("Stop").Return(nil)

		err := coordinator.Stop()
		assert.NoError(t, err)

		// 检查状态
		status := coordinator.GetStatus()
		assert.False(t, status.Running)
	})

	// 验证所有模拟调用
	mockDepManager.AssertExpectations(t)
	mockDocFetcher.AssertExpectations(t)
	mockConfigGenerator.AssertExpectations(t)
	mockMCPManager.AssertExpectations(t)
	mockHotReloadManager.AssertExpectations(t)
}

// TestFactoryAdapterIntegration 测试工厂适配器集成
func TestFactoryAdapterIntegration(t *testing.T) {
	// 创建工厂
	depFactory := NewDependencyManagerFactory()
	docFactory := NewDocumentationFetcherFactory()

	// 创建适配器
	depManager := NewFactoryDependencyManager(depFactory)
	docFetcher := NewFactoryDocumentationFetcher(docFactory)

	// 测试依赖管理器适配器
	t.Run("DependencyManagerAdapter", func(t *testing.T) {
		ctx := context.Background()
		depInfo := DependencyInfo{
			Name:        "test-package",
			PackageName: "test-package",
			Type:        DependencyTypeNPM,
		}

		// 这些调用应该成功（即使实际安装会失败，因为我们在测试环境中）
		// 我们主要测试适配器是否正确创建了管理器
		// 在测试环境中，我们期望错误，因为npm命令可能不存在
		// 但我们主要测试适配器是否工作
		assert.NotPanics(t, func() {
			depManager.CheckDependency(ctx, depInfo)
		}, "Adapter should handle manager creation without panic")
	})

	// 测试文档获取器适配器
	t.Run("DocumentationFetcherAdapter", func(t *testing.T) {
		ctx := context.Background()

		// 测试应该处理错误而不崩溃
		assert.NotPanics(t, func() {
			docFetcher.FetchDocumentation(ctx, "test-package", DependencyTypeNPM)
		}, "Adapter should handle fetcher creation without panic")
	})
}

// TestConfigGeneratorIntegration 测试配置生成器集成
func TestConfigGeneratorIntegration(t *testing.T) {
	generator := NewDefaultConfigGenerator()

	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 测试配置生成
	t.Run("GenerateNPMConfig", func(t *testing.T) {
		docInfo := &DocumentationInfo{
			Name:        "@organization/test-mcp",
			Description: "Test MCP server",
			Version:     "1.0.0",
			Homepage:    "https://example.com/test-mcp",
			Repository:  "https://github.com/organization/test-mcp",
		}

		depInfo := &DependencyInfo{
			Name:        "test-mcp",
			PackageName: "@organization/test-mcp",
			Type:        DependencyTypeNPM,
			InstallCmd:  "npm install -g @organization/test-mcp",
			TestCmd:     "test-mcp --version",
		}

		server, err := generator.GenerateServerConfig(docInfo, depInfo)
		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "test-mcp", server.Name)
		assert.True(t, server.Enabled)
		assert.Equal(t, "command", server.Type)
		assert.Equal(t, "test-mcp", server.Command)
	})

	// 测试配置保存
	t.Run("SaveConfigToFile", func(t *testing.T) {
		server := &config.MCPServerWithAutomation{
			MCPServer: config.MCPServer{
				Name:    "test-server",
				Enabled: true,
				Command: "test-server",
			},
		}

		err := generator.SaveConfigToFile(server, configPath)
		assert.NoError(t, err)

		// 验证文件存在
		_, err = os.Stat(configPath)
		assert.NoError(t, err)
	})
}

// TestHotReloadManagerIntegration 测试热加载管理器集成
func TestHotReloadManagerIntegration(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建模拟MCP管理器
	mockMCPManager := new(MockMCPManager)

	// 创建热加载管理器
	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// 测试启动和停止
	t.Run("StartStop", func(t *testing.T) {
		ctx := context.Background()

		// 启动管理器
		err := manager.Start(ctx)
		assert.NoError(t, err)

		// 检查状态
		status := manager.GetStatus()
		assert.True(t, status.Running)

		// 停止管理器
		err = manager.Stop()
		assert.NoError(t, err)

		// 检查状态
		status = manager.GetStatus()
		assert.False(t, status.Running)
	})
}
