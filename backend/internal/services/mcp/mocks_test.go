package mcp

import (
	"backend/internal/config"
	"context"
	
	"github.com/stretchr/testify/mock"
)

// MockDependencyManager 模拟依赖管理器
type MockDependencyManager struct {
	mock.Mock
}

func (m *MockDependencyManager) CheckDependency(ctx context.Context, info DependencyInfo) (bool, error) {
	args := m.Called(ctx, info)
	return args.Bool(0), args.Error(1)
}

func (m *MockDependencyManager) InstallDependency(ctx context.Context, info DependencyInfo) error {
	args := m.Called(ctx, info)
	return args.Error(0)
}

func (m *MockDependencyManager) UpdateDependency(ctx context.Context, info DependencyInfo) error {
	args := m.Called(ctx, info)
	return args.Error(0)
}

func (m *MockDependencyManager) RemoveDependency(ctx context.Context, info DependencyInfo) error {
	args := m.Called(ctx, info)
	return args.Error(0)
}

func (m *MockDependencyManager) GetDependencyVersion(ctx context.Context, info DependencyInfo) (string, error) {
	args := m.Called(ctx, info)
	return args.String(0), args.Error(1)
}

// MockDocumentationFetcher 模拟文档获取器
type MockDocumentationFetcher struct {
	mock.Mock
}

func (m *MockDocumentationFetcher) FetchDocumentation(ctx context.Context, packageName string, depType DependencyType) (*DocumentationInfo, error) {
	args := m.Called(ctx, packageName, depType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DocumentationInfo), args.Error(1)
}

// MockConfigGenerator 模拟配置生成器
type MockConfigGenerator struct {
	mock.Mock
}

func (m *MockConfigGenerator) GenerateServerConfig(doc *DocumentationInfo, depInfo *DependencyInfo) (*config.MCPServerWithAutomation, error) {
	args := m.Called(doc, depInfo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.MCPServerWithAutomation), args.Error(1)
}

func (m *MockConfigGenerator) SaveConfigToFile(server *config.MCPServerWithAutomation, configPath string) error {
	args := m.Called(server, configPath)
	return args.Error(0)
}

func (m *MockConfigGenerator) UpdateExistingConfig(existingConfig *config.MCPServersConfigWithAutomation, newServer *config.MCPServerWithAutomation) error {
	args := m.Called(existingConfig, newServer)
	return args.Error(0)
}

// MockHotReloadManager 模拟热加载管理器
type MockHotReloadManager struct {
	mock.Mock
}

func (m *MockHotReloadManager) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHotReloadManager) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockHotReloadManager) AddWatchPath(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockHotReloadManager) RemoveWatchPath(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockHotReloadManager) GetStatus() HotReloadStatus {
	args := m.Called()
	return args.Get(0).(HotReloadStatus)
}

func (m *MockHotReloadManager) TriggerReload(serverName string) error {
	args := m.Called(serverName)
	return args.Error(0)
}

// MockMCPManager 模拟MCP管理器
type MockMCPManager struct {
	mock.Mock
}

func (m *MockMCPManager) ConnectToServer(serverName string) error {
	args := m.Called(serverName)
	return args.Error(0)
}

func (m *MockMCPManager) GetServer(serverName string) (*config.MCPServer, bool) {
	args := m.Called(serverName)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*config.MCPServer), args.Bool(1)
}

func (m *MockMCPManager) Discover() {
	m.Called()
}

// MockMCPClient 模拟MCP客户端
type MockMCPClient struct {
	mock.Mock
}

func (m *MockMCPClient) Close() error {
	args := m.Called()
	return args.Error(0)
}