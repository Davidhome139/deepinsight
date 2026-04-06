package mcp

import (
	"backend/internal/config"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewDefaultAutomationCoordinator(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	assert.NotNil(t, coordinator)
	assert.Equal(t, mockDepManager, coordinator.depManager)
	assert.Equal(t, mockDocFetcher, coordinator.docFetcher)
	assert.Equal(t, mockConfigGenerator, coordinator.configGenerator)
	assert.Equal(t, mockHotReloadManager, coordinator.hotReloadManager)
	assert.Equal(t, mockMCPManager, coordinator.mcpManager)
	assert.Equal(t, configPath, coordinator.configPath)
	assert.Equal(t, docsPath, coordinator.docsPath)
	assert.False(t, coordinator.status.Running)
	assert.Equal(t, 0, coordinator.status.TotalPackages)
}

func TestDefaultAutomationCoordinator_StartStop(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 设置模拟期望
	mockHotReloadManager.On("Start", mock.Anything).Return(nil)

	// 启动协调器
	ctx := context.Background()
	err := coordinator.Start(ctx)
	assert.NoError(t, err)

	// 检查状态
	status := coordinator.GetStatus()
	assert.True(t, status.Running)

	// 设置停止期望
	mockHotReloadManager.On("Stop").Return(nil)

	// 停止协调器
	err = coordinator.Stop()
	assert.NoError(t, err)

	// 检查状态
	status = coordinator.GetStatus()
	assert.False(t, status.Running)

	// 验证模拟调用
	mockHotReloadManager.AssertExpectations(t)
}

func TestDefaultAutomationCoordinator_AddMCPPackage(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 设置模拟期望
	mockDepManager.On("CheckDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(false, nil)
	mockDepManager.On("InstallDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(nil)

	docInfo := &DocumentationInfo{
		Name:        "test-package",
		Description: "Test package documentation",
		Version:     "1.0.0",
		Homepage:    "https://example.com/test-package",
		Repository:  "https://github.com/test/test-package",
	}
	mockDocFetcher.On("FetchDocumentation", mock.Anything, "test-package", DependencyTypeNPM).Return(docInfo, nil)

	serverConfig := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    "test-package",
			Enabled: true,
			Command: "test-package",
		},
	}
	mockConfigGenerator.On("GenerateServerConfig", docInfo, mock.AnythingOfType("*mcp.DependencyInfo")).Return(serverConfig, nil)
	mockConfigGenerator.On("SaveConfigToFile", serverConfig, configPath).Return(nil)

	mockMCPManager.On("ConnectToServer", "test-package").Return(nil)

	// 启动协调器
	ctx := context.Background()
	mockHotReloadManager.On("Start", mock.Anything).Return(nil)
	err := coordinator.Start(ctx)
	assert.NoError(t, err)

	// 添加包
	packageName := "test-package"
	depType := DependencyTypeNPM

	err = coordinator.AddMCPPackage(packageName, depType)
	assert.NoError(t, err)

	// 等待自动化完成
	time.Sleep(200 * time.Millisecond)

	// 检查状态
	status := coordinator.GetStatus()
	assert.Equal(t, 1, status.TotalPackages)
	assert.Contains(t, status.Packages, packageName)

	pkgStatus := status.Packages[packageName]
	assert.Equal(t, packageName, pkgStatus.PackageName)
	assert.Equal(t, depType, pkgStatus.DependencyType)
	// 自动化完成后状态应该是installed, generated, connected
	assert.Equal(t, "installed", pkgStatus.InstallStatus)
	assert.Equal(t, "generated", pkgStatus.ConfigStatus)
	assert.Equal(t, "connected", pkgStatus.ConnectionStatus)

	// 停止协调器
	mockHotReloadManager.On("Stop").Return(nil)
	err = coordinator.Stop()
	assert.NoError(t, err)

	// 验证模拟调用
	mockDepManager.AssertExpectations(t)
	mockDocFetcher.AssertExpectations(t)
	mockConfigGenerator.AssertExpectations(t)
	mockMCPManager.AssertExpectations(t)
	mockHotReloadManager.AssertExpectations(t)
}

func TestDefaultAutomationCoordinator_AddMCPPackage_Duplicate(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 设置模拟期望
	mockDepManager.On("CheckDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(false, nil)
	mockDepManager.On("InstallDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(nil)

	docInfo := &DocumentationInfo{
		Name:        "test-package",
		Description: "Test package documentation",
		Version:     "1.0.0",
		Homepage:    "https://example.com/test-package",
		Repository:  "https://github.com/test/test-package",
	}
	mockDocFetcher.On("FetchDocumentation", mock.Anything, "test-package", DependencyTypeNPM).Return(docInfo, nil)

	serverConfig := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    "test-package",
			Enabled: true,
			Command: "test-package",
		},
	}
	mockConfigGenerator.On("GenerateServerConfig", docInfo, mock.AnythingOfType("*mcp.DependencyInfo")).Return(serverConfig, nil)
	mockConfigGenerator.On("SaveConfigToFile", serverConfig, configPath).Return(nil)

	mockMCPManager.On("ConnectToServer", "test-package").Return(nil)

	// 启动协调器
	ctx := context.Background()
	mockHotReloadManager.On("Start", mock.Anything).Return(nil)
	err := coordinator.Start(ctx)
	assert.NoError(t, err)

	// 添加包
	packageName := "test-package"
	depType := DependencyTypeNPM

	err = coordinator.AddMCPPackage(packageName, depType)
	assert.NoError(t, err)

	// 等待自动化完成
	time.Sleep(200 * time.Millisecond)

	// 再次添加相同的包应该失败
	err = coordinator.AddMCPPackage(packageName, depType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 检查状态
	status := coordinator.GetStatus()
	assert.Equal(t, 1, status.TotalPackages) // 应该只有一个包

	// 停止协调器
	mockHotReloadManager.On("Stop").Return(nil)
	err = coordinator.Stop()
	assert.NoError(t, err)

	// 验证模拟调用
	mockDepManager.AssertExpectations(t)
	mockDocFetcher.AssertExpectations(t)
	mockConfigGenerator.AssertExpectations(t)
	mockMCPManager.AssertExpectations(t)
	mockHotReloadManager.AssertExpectations(t)
}

func TestDefaultAutomationCoordinator_RemoveMCPPackage(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 设置模拟期望
	mockDepManager.On("CheckDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(false, nil)
	mockDepManager.On("InstallDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(nil)

	docInfo := &DocumentationInfo{
		Name:        "test-package",
		Description: "Test package documentation",
		Version:     "1.0.0",
		Homepage:    "https://example.com/test-package",
		Repository:  "https://github.com/test/test-package",
	}
	mockDocFetcher.On("FetchDocumentation", mock.Anything, "test-package", DependencyTypeNPM).Return(docInfo, nil)

	serverConfig := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    "test-package",
			Enabled: true,
			Command: "test-package",
		},
	}
	mockConfigGenerator.On("GenerateServerConfig", docInfo, mock.AnythingOfType("*mcp.DependencyInfo")).Return(serverConfig, nil)
	mockConfigGenerator.On("SaveConfigToFile", serverConfig, configPath).Return(nil)

	mockMCPManager.On("ConnectToServer", "test-package").Return(nil)

	// 启动协调器
	ctx := context.Background()
	mockHotReloadManager.On("Start", mock.Anything).Return(nil)
	err := coordinator.Start(ctx)
	assert.NoError(t, err)

	// 添加包
	packageName := "test-package"
	depType := DependencyTypeNPM

	err = coordinator.AddMCPPackage(packageName, depType)
	assert.NoError(t, err)

	// 等待自动化完成
	time.Sleep(200 * time.Millisecond)

	// 检查包存在
	status := coordinator.GetStatus()
	assert.Equal(t, 1, status.TotalPackages)
	assert.Contains(t, status.Packages, packageName)

	// 移除包
	err = coordinator.RemoveMCPPackage(packageName)
	assert.NoError(t, err)

	// 检查包已移除
	status = coordinator.GetStatus()
	assert.Equal(t, 0, status.TotalPackages)
	assert.NotContains(t, status.Packages, packageName)

	// 停止协调器
	mockHotReloadManager.On("Stop").Return(nil)
	err = coordinator.Stop()
	assert.NoError(t, err)

	// 验证模拟调用
	mockDepManager.AssertExpectations(t)
	mockDocFetcher.AssertExpectations(t)
	mockConfigGenerator.AssertExpectations(t)
	mockMCPManager.AssertExpectations(t)
	mockHotReloadManager.AssertExpectations(t)
}

func TestDefaultAutomationCoordinator_RemoveMCPPackage_NotFound(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 移除不存在的包应该失败
	err := coordinator.RemoveMCPPackage("non-existent-package")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultAutomationCoordinator_UpdateMCPPackage(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 设置模拟期望
	mockDepManager.On("CheckDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(false, nil)
	mockDepManager.On("InstallDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(nil)
	mockDepManager.On("UpdateDependency", mock.Anything, mock.AnythingOfType("mcp.DependencyInfo")).Return(nil)

	docInfo := &DocumentationInfo{
		Name:        "test-package",
		Description: "Test package documentation",
		Version:     "1.0.0",
		Homepage:    "https://example.com/test-package",
		Repository:  "https://github.com/test/test-package",
	}
	mockDocFetcher.On("FetchDocumentation", mock.Anything, "test-package", DependencyTypeNPM).Return(docInfo, nil)
	mockDocFetcher.On("FetchDocumentation", mock.Anything, "test-package", DependencyTypeGo).Return(docInfo, nil)

	serverConfig := &config.MCPServerWithAutomation{
		MCPServer: config.MCPServer{
			Name:    "test-package",
			Enabled: true,
			Command: "test-package",
		},
	}
	mockConfigGenerator.On("GenerateServerConfig", docInfo, mock.AnythingOfType("*mcp.DependencyInfo")).Return(serverConfig, nil)
	mockConfigGenerator.On("SaveConfigToFile", serverConfig, configPath).Return(nil)

	mockMCPManager.On("ConnectToServer", "test-package").Return(nil)

	// 启动协调器
	ctx := context.Background()
	mockHotReloadManager.On("Start", mock.Anything).Return(nil)
	err := coordinator.Start(ctx)
	assert.NoError(t, err)

	// 添加包
	packageName := "test-package"
	depType := DependencyTypeNPM

	err = coordinator.AddMCPPackage(packageName, depType)
	assert.NoError(t, err)

	// 等待自动化完成
	time.Sleep(200 * time.Millisecond)

	// 更新包
	newDepType := DependencyTypeGo
	err = coordinator.UpdateMCPPackage(packageName, newDepType)
	assert.NoError(t, err)

	// 等待更新完成
	time.Sleep(200 * time.Millisecond)

	// 检查状态
	status := coordinator.GetStatus()
	assert.Contains(t, status.Packages, packageName)

	pkgStatus := status.Packages[packageName]
	assert.Equal(t, newDepType, pkgStatus.DependencyType)
	assert.Equal(t, "updated", pkgStatus.UpdateStatus) // 更新状态应该是updated

	// 停止协调器
	mockHotReloadManager.On("Stop").Return(nil)
	err = coordinator.Stop()
	assert.NoError(t, err)

	// 验证模拟调用
	mockDepManager.AssertExpectations(t)
	mockDocFetcher.AssertExpectations(t)
	mockConfigGenerator.AssertExpectations(t)
	mockMCPManager.AssertExpectations(t)
	mockHotReloadManager.AssertExpectations(t)
}

func TestDefaultAutomationCoordinator_UpdateMCPPackage_NotFound(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 更新不存在的包应该失败
	err := coordinator.UpdateMCPPackage("non-existent-package", DependencyTypeNPM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultAutomationCoordinator_GetPackageStatus(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 添加包
	packageName := "test-package"
	depType := DependencyTypeNPM

	err := coordinator.AddMCPPackage(packageName, depType)
	assert.NoError(t, err)

	// 获取包状态
	pkgStatus, err := coordinator.GetPackageStatus(packageName)
	assert.NoError(t, err)
	assert.NotNil(t, pkgStatus)
	assert.Equal(t, packageName, pkgStatus.PackageName)
	assert.Equal(t, depType, pkgStatus.DependencyType)
	assert.Equal(t, "pending", pkgStatus.InstallStatus)
	assert.Equal(t, "pending", pkgStatus.ConfigStatus)
	assert.Equal(t, "pending", pkgStatus.ConnectionStatus)

	// 获取不存在的包状态应该失败
	_, err = coordinator.GetPackageStatus("non-existent-package")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultAutomationCoordinator_Start_AlreadyRunning(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 启动协调器
	ctx := context.Background()
	mockHotReloadManager.On("Start", mock.Anything).Return(nil)
	err := coordinator.Start(ctx)
	assert.NoError(t, err)

	// 再次启动应该失败
	err = coordinator.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// 停止协调器
	mockHotReloadManager.On("Stop").Return(nil)
	err = coordinator.Stop()
	assert.NoError(t, err)

	mockHotReloadManager.AssertExpectations(t)
}

func TestDefaultAutomationCoordinator_Stop_NotRunning(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 停止未运行的协调器应该成功
	err := coordinator.Stop()
	assert.NoError(t, err)
}

func TestDefaultAutomationCoordinator_GetInstallCommand(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 测试各种包类型的安装命令
	testCases := []struct {
		packageName string
		depType     DependencyType
		expected    string
	}{
		{"test-package", DependencyTypeNPM, "npm install -g test-package"},
		{"github.com/test/package", DependencyTypeGo, "go install github.com/test/package@latest"},
		{"test-package", DependencyTypePip, "pip install test-package"},
		{"test-image", DependencyTypeDocker, "docker pull test-image"},
		{"test-package", DependencyType("unknown"), ""},
	}

	for _, tc := range testCases {
		cmd := coordinator.getInstallCommand(tc.packageName, tc.depType)
		assert.Equal(t, tc.expected, cmd, "Package: %s, Type: %v", tc.packageName, tc.depType)
	}
}

func TestDefaultAutomationCoordinator_GetTestCommand(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 测试各种包类型的测试命令
	testCases := []struct {
		packageName string
		depType     DependencyType
		expected    string
	}{
		{"test-package", DependencyTypeNPM, "npx -y test-package --version"},
		{"test-package", DependencyTypeGo, "test-package --version"},
		{"test-package", DependencyTypePip, "python -m test-package --version"},
		{"test-image", DependencyTypeDocker, "docker run --rm test-image --version"},
		{"test-package", DependencyType("unknown"), ""},
	}

	for _, tc := range testCases {
		cmd := coordinator.getTestCommand(tc.packageName, tc.depType)
		assert.Equal(t, tc.expected, cmd, "Package: %s, Type: %v", tc.packageName, tc.depType)
	}
}

func TestDefaultAutomationCoordinator_UpdatePackageStatus(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockHotReloadManager := new(MockHotReloadManager)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		mockHotReloadManager,
		mockMCPManager,
		configPath,
		docsPath,
	)

	// 添加包
	packageName := "test-package"
	depType := DependencyTypeNPM

	err := coordinator.AddMCPPackage(packageName, depType)
	assert.NoError(t, err)

	// 更新包状态
	coordinator.updatePackageStatus(packageName, "installStatus", "installing", "")

	// 检查状态
	pkgStatus, err := coordinator.GetPackageStatus(packageName)
	assert.NoError(t, err)
	assert.Equal(t, "installing", pkgStatus.InstallStatus)
	assert.Empty(t, pkgStatus.LastError)

	// 更新包状态并设置错误
	coordinator.updatePackageStatus(packageName, "installStatus", "failed", "installation failed")

	// 检查状态
	pkgStatus, err = coordinator.GetPackageStatus(packageName)
	assert.NoError(t, err)
	assert.Equal(t, "failed", pkgStatus.InstallStatus)
	assert.Equal(t, "installation failed", pkgStatus.LastError)

	// 更新不存在的包状态应该无操作
	coordinator.updatePackageStatus("non-existent-package", "installStatus", "installing", "")
	// 不应该panic
}

func TestDefaultAutomationCoordinator_WithoutHotReloadManager(t *testing.T) {
	mockDepManager := new(MockDependencyManager)
	mockDocFetcher := new(MockDocumentationFetcher)
	mockConfigGenerator := new(MockConfigGenerator)
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")
	docsPath := filepath.Join(tempDir, "docs")

	// 创建没有热加载管理器的协调器
	coordinator := NewDefaultAutomationCoordinator(
		mockDepManager,
		mockDocFetcher,
		mockConfigGenerator,
		nil, // 没有热加载管理器
		mockMCPManager,
		configPath,
		docsPath,
	)

	assert.NotNil(t, coordinator)
	assert.Nil(t, coordinator.hotReloadManager)

	// 启动协调器（应该成功，即使没有热加载管理器）
	ctx := context.Background()
	err := coordinator.Start(ctx)
	assert.NoError(t, err)

	// 检查状态
	status := coordinator.GetStatus()
	assert.True(t, status.Running)

	// 停止协调器
	err = coordinator.Stop()
	assert.NoError(t, err)

	// 检查状态
	status = coordinator.GetStatus()
	assert.False(t, status.Running)
}
