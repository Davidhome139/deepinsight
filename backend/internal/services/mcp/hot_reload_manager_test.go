package mcp

import (
	"backend/internal/config"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultHotReloadManager(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建初始配置
	err := os.WriteFile(configPath, []byte(`{}`), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.watcher)
	assert.Equal(t, configPath, manager.configPath)
}

func TestDefaultHotReloadManager_StartStop(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建配置文件
	err := os.WriteFile(configPath, []byte(`{}`), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 启动管理器
	ctx := context.Background()
	err = manager.Start(ctx)
	assert.NoError(t, err)

	// 检查状态
	status := manager.GetStatus()
	assert.True(t, status.Running)
	assert.Contains(t, status.WatchedPaths, configPath)

	// 停止管理器
	err = manager.Stop()
	assert.NoError(t, err)

	// 检查状态
	status = manager.GetStatus()
	assert.False(t, status.Running)
}

func TestDefaultHotReloadManager_AddRemoveWatchPath(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建配置文件
	err := os.WriteFile(configPath, []byte(`{}`), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 添加监控路径
	testPath := filepath.Join(tempDir, "test.json")
	err = manager.AddWatchPath(testPath)
	assert.NoError(t, err)

	// 检查状态
	status := manager.GetStatus()
	assert.Contains(t, status.WatchedPaths, testPath)

	// 移除监控路径
	err = manager.RemoveWatchPath(testPath)
	assert.NoError(t, err)

	// 检查状态
	status = manager.GetStatus()
	assert.NotContains(t, status.WatchedPaths, testPath)
}

func TestDefaultHotReloadManager_AddWatchPath_CreateMissing(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 不创建配置文件，让管理器自动创建
	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 添加监控路径（文件不存在）
	err = manager.AddWatchPath(configPath)
	assert.NoError(t, err)

	// 检查文件是否被创建
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// 检查状态
	status := manager.GetStatus()
	assert.Contains(t, status.WatchedPaths, configPath)
}

func TestDefaultHotReloadManager_TriggerReload(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建配置文件
	err := os.WriteFile(configPath, []byte(`{
		"mcpServers": {
			"test-server": {
				"name": "test-server",
				"enabled": true,
				"type": "command",
				"command": "echo",
				"args": ["test"]
			}
		}
	}`), 0644)
	assert.NoError(t, err)

	// 设置模拟期望
	mockServer := &config.MCPServer{
		Name:      "test-server",
		Enabled:   true,
		Type:      "command",
		Command:   "echo",
		Args:      []string{"test"},
		Connected: false,
	}

	mockMCPManager.On("GetServer", "test-server").Return(mockServer, true)
	mockMCPManager.On("ConnectToServer", "test-server").Return(nil)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 触发重载
	err = manager.TriggerReload("test-server")
	assert.NoError(t, err)

	// 检查状态
	status := manager.GetStatus()
	serverStatus, exists := status.ServerStatus["test-server"]
	assert.True(t, exists)
	assert.Equal(t, "success", serverStatus.Status)
	assert.Equal(t, 1, serverStatus.ReloadCount)
	assert.Empty(t, serverStatus.LastError)

	// 验证模拟调用
	mockMCPManager.AssertExpectations(t)
}

func TestDefaultHotReloadManager_TriggerReload_ServerNotFound(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建空配置文件
	err := os.WriteFile(configPath, []byte(`{}`), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 触发重载（服务器不存在）
	err = manager.TriggerReload("non-existent-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 检查状态
	status := manager.GetStatus()
	serverStatus, exists := status.ServerStatus["non-existent-server"]
	assert.True(t, exists)
	assert.Equal(t, "failed", serverStatus.Status)
	assert.NotEmpty(t, serverStatus.LastError)
}

func TestDefaultHotReloadManager_TriggerReload_ServerDisabled(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建配置文件（服务器禁用）
	configData := `{
		"mcpServers": {
			"test-server": {
				"name": "test-server",
				"enabled": false,
				"type": "command",
				"command": "echo",
				"args": ["test"]
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configData), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 触发重载（服务器禁用）
	err = manager.TriggerReload("test-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")

	// 检查状态
	status := manager.GetStatus()
	serverStatus, exists := status.ServerStatus["test-server"]
	assert.True(t, exists)
	assert.Equal(t, "failed", serverStatus.Status)
	assert.NotEmpty(t, serverStatus.LastError)
}

func TestDefaultHotReloadManager_GetStatus(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	err := os.WriteFile(configPath, []byte(`{}`), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 获取初始状态
	status := manager.GetStatus()
	assert.False(t, status.Running)
	assert.Empty(t, status.WatchedPaths)
	assert.Equal(t, time.Time{}, status.LastReload)
	assert.Equal(t, 0, status.ReloadCount)
	assert.Empty(t, status.ServerStatus)

	// 启动管理器
	ctx := context.Background()
	err = manager.Start(ctx)
	assert.NoError(t, err)

	// 获取运行状态
	status = manager.GetStatus()
	assert.True(t, status.Running)
	assert.Contains(t, status.WatchedPaths, configPath)

	// 停止管理器
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestDefaultHotReloadManager_Start_AlreadyRunning(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	err := os.WriteFile(configPath, []byte(`{}`), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 启动管理器
	ctx := context.Background()
	err = manager.Start(ctx)
	assert.NoError(t, err)

	// 再次启动应该返回错误
	err = manager.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// 停止管理器
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestDefaultHotReloadManager_Stop_NotRunning(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	err := os.WriteFile(configPath, []byte(`{}`), 0644)
	assert.NoError(t, err)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 停止未运行的管理器应该成功
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestDefaultHotReloadManager_ReloadServer_WithConnectedClient(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 创建配置文件
	configData := `{
		"mcpServers": {
			"test-server": {
				"name": "test-server",
				"enabled": true,
				"type": "command",
				"command": "echo",
				"args": ["test"]
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configData), 0644)
	assert.NoError(t, err)

	// 创建模拟的已连接服务器（Client为nil，简化测试）
	mockServer := &config.MCPServer{
		Name:      "test-server",
		Enabled:   true,
		Type:      "command",
		Command:   "echo",
		Args:      []string{"test"},
		Connected: true,
		Client:    nil, // 简化测试，不测试Client关闭
	}

	// 设置模拟期望
	mockMCPManager.On("GetServer", "test-server").Return(mockServer, true)
	mockMCPManager.On("ConnectToServer", "test-server").Return(nil)

	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 触发重载
	err = manager.TriggerReload("test-server")
	assert.NoError(t, err)

	// 验证模拟调用
	mockMCPManager.AssertExpectations(t)
}

func TestDefaultHotReloadManager_ReloadServer_ConfigLoadError(t *testing.T) {
	mockMCPManager := new(MockMCPManager)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcpservers.json")

	// 不创建配置文件，让配置加载失败
	manager, err := NewDefaultHotReloadManager(mockMCPManager, configPath)
	assert.NoError(t, err)

	// 触发重载
	err = manager.TriggerReload("test-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get MCP servers config")
}
