package mcp

import (
	"backend/internal/config"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// HotReloadManager 热加载管理器接口
type HotReloadManager interface {
	Start(ctx context.Context) error
	Stop() error
	AddWatchPath(path string) error
	RemoveWatchPath(path string) error
	GetStatus() HotReloadStatus
	TriggerReload(serverName string) error
}

// HotReloadStatus 热加载状态
type HotReloadStatus struct {
	Running      bool                          `json:"running"`
	WatchedPaths []string                      `json:"watchedPaths"`
	LastReload   time.Time                     `json:"lastReload"`
	ReloadCount  int                           `json:"reloadCount"`
	ServerStatus map[string]ServerReloadStatus `json:"serverStatus"`
}

// ServerReloadStatus 服务器重载状态
type ServerReloadStatus struct {
	LastReloadAttempt time.Time `json:"lastReloadAttempt"`
	ReloadCount       int       `json:"reloadCount"`
	LastError         string    `json:"lastError,omitempty"`
	Status            string    `json:"status"` // idle, reloading, success, failed
}

// DefaultHotReloadManager 默认热加载管理器
type DefaultHotReloadManager struct {
	mcpManager config.MCPManagerInterface
	watcher    *fsnotify.Watcher
	mu         sync.RWMutex
	status     HotReloadStatus
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	configPath string
}

func NewDefaultHotReloadManager(mcpManager config.MCPManagerInterface, configPath string) (*DefaultHotReloadManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DefaultHotReloadManager{
		mcpManager: mcpManager,
		watcher:    watcher,
		status: HotReloadStatus{
			Running:      false,
			WatchedPaths: []string{},
			LastReload:   time.Time{},
			ReloadCount:  0,
			ServerStatus: make(map[string]ServerReloadStatus),
		},
		ctx:        ctx,
		cancel:     cancel,
		configPath: configPath,
	}, nil
}

func (m *DefaultHotReloadManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.Running {
		return fmt.Errorf("hot reload manager is already running")
	}

	// 添加配置文件路径到监控
	if err := m.AddWatchPath(m.configPath); err != nil {
		return fmt.Errorf("failed to watch config path: %v", err)
	}

	// 启动监控goroutine
	m.wg.Add(1)
	go m.watchFiles()

	// 启动定期检查goroutine
	m.wg.Add(1)
	go m.periodicCheck()

	m.status.Running = true
	log.Printf("[HotReload] Hot reload manager started, watching: %s", m.configPath)

	return nil
}

func (m *DefaultHotReloadManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.status.Running {
		return nil
	}

	// 取消上下文
	m.cancel()

	// 关闭文件监控器
	if err := m.watcher.Close(); err != nil {
		log.Printf("[HotReload] Error closing watcher: %v", err)
	}

	// 等待所有goroutine完成
	m.wg.Wait()

	m.status.Running = false
	log.Println("[HotReload] Hot reload manager stopped")

	return nil
}

func (m *DefaultHotReloadManager) AddWatchPath(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查路径是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 如果路径不存在，尝试创建目录
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
		// 创建空文件
		if filepath.Ext(path) != "" {
			if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
				return fmt.Errorf("failed to create file: %v", err)
			}
		}
	}

	// 添加监控
	if err := m.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to add watch path: %v", err)
	}

	// 更新状态
	m.status.WatchedPaths = append(m.status.WatchedPaths, path)
	log.Printf("[HotReload] Added watch path: %s", path)

	return nil
}

func (m *DefaultHotReloadManager) RemoveWatchPath(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 移除监控
	if err := m.watcher.Remove(path); err != nil {
		return fmt.Errorf("failed to remove watch path: %v", err)
	}

	// 更新状态
	for i, p := range m.status.WatchedPaths {
		if p == path {
			m.status.WatchedPaths = append(m.status.WatchedPaths[:i], m.status.WatchedPaths[i+1:]...)
			break
		}
	}

	log.Printf("[HotReload] Removed watch path: %s", path)
	return nil
}

func (m *DefaultHotReloadManager) GetStatus() HotReloadStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *DefaultHotReloadManager) TriggerReload(serverName string) error {
	m.mu.Lock()

	// 更新服务器状态
	serverStatus := m.status.ServerStatus[serverName]
	serverStatus.LastReloadAttempt = time.Now()
	serverStatus.Status = "reloading"
	m.status.ServerStatus[serverName] = serverStatus

	m.mu.Unlock()

	log.Printf("[HotReload] Triggering reload for server: %s", serverName)

	// 执行重载
	err := m.reloadServer(serverName)

	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新状态
	serverStatus = m.status.ServerStatus[serverName]
	serverStatus.ReloadCount++

	if err != nil {
		serverStatus.Status = "failed"
		serverStatus.LastError = err.Error()
		log.Printf("[HotReload] Failed to reload server %s: %v", serverName, err)
	} else {
		serverStatus.Status = "success"
		serverStatus.LastError = ""
		m.status.LastReload = time.Now()
		m.status.ReloadCount++
		log.Printf("[HotReload] Successfully reloaded server: %s", serverName)
	}

	m.status.ServerStatus[serverName] = serverStatus

	return err
}

func (m *DefaultHotReloadManager) watchFiles() {
	defer m.wg.Done()

	log.Println("[HotReload] File watcher started")

	for {
		select {
		case <-m.ctx.Done():
			log.Println("[HotReload] File watcher stopped")
			return

		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			// 只处理写入和重命名事件
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Rename == fsnotify.Rename {
				log.Printf("[HotReload] File changed: %s (op: %v)", event.Name, event.Op)

				// 如果是配置文件，触发重新加载
				if event.Name == m.configPath {
					m.handleConfigChange(event.Name)
				}
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[HotReload] Watcher error: %v", err)
		}
	}
}

func (m *DefaultHotReloadManager) handleConfigChange(configPath string) {
	log.Printf("[HotReload] Config file changed: %s", configPath)

	// 等待一小段时间，避免频繁重载
	time.Sleep(500 * time.Millisecond)

	// 重新加载配置
	cfg, err := config.LoadMCPServersConfig(configPath)
	if err != nil {
		log.Printf("[HotReload] Failed to reload config: %v", err)
		return
	}

	log.Printf("[HotReload] Config reloaded, %d servers found", len(cfg.Servers))

	// 检查哪些服务器需要重载
	for name, server := range cfg.Servers {
		if server.Enabled {
			// 触发服务器重载
			go func(serverName string) {
				if err := m.TriggerReload(serverName); err != nil {
					log.Printf("[HotReload] Failed to reload server %s: %v", serverName, err)
				}
			}(name)
		}
	}
}

func (m *DefaultHotReloadManager) periodicCheck() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("[HotReload] Periodic checker started")

	for {
		select {
		case <-m.ctx.Done():
			log.Println("[HotReload] Periodic checker stopped")
			return

		case <-ticker.C:
			// 检查所有启用的服务器是否需要重载
			cfg := config.GetMCPServersConfig()
			if cfg == nil {
				continue
			}

			for name, server := range cfg.Servers {
				if server.Enabled {
					// 检查服务器连接状态
					if mcpServer, ok := m.mcpManager.GetServer(name); ok {
						if !mcpServer.Connected {
							log.Printf("[HotReload] Server %s is not connected, attempting to reload", name)
							go m.TriggerReload(name)
						}
					}
				}
			}
		}
	}
}

func (m *DefaultHotReloadManager) reloadServer(serverName string) error {
	// 获取服务器配置
	cfg := config.GetMCPServersConfig()
	if cfg == nil {
		return fmt.Errorf("failed to get MCP servers config")
	}

	server, exists := cfg.Servers[serverName]
	if !exists {
		return fmt.Errorf("server %s not found in config", serverName)
	}

	if !server.Enabled {
		return fmt.Errorf("server %s is disabled", serverName)
	}

	log.Printf("[HotReload] Reloading server: %s", serverName)

	// 如果服务器已连接，先断开连接
	if mcpServer, ok := m.mcpManager.GetServer(serverName); ok && mcpServer.Connected {
		log.Printf("[HotReload] Disconnecting server: %s", serverName)
		if mcpServer.Client != nil {
			mcpServer.Client.Close()
		}
	}

	// 重新连接服务器
	if err := m.mcpManager.ConnectToServer(serverName); err != nil {
		return fmt.Errorf("failed to reconnect server %s: %v", serverName, err)
	}

	return nil
}
