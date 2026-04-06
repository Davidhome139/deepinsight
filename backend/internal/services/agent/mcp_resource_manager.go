package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
)

// ResourceLimits 资源限制配置
type ResourceLimits struct {
	MaxConnections    int
	MaxMemoryMB       int
	ConnectionTimeout time.Duration
	IdleTimeout       time.Duration
}

// ConnectionPool 连接池接口
type ConnectionPool interface {
	Get(ctx context.Context, serverName string) (*client.Client, error)
	Put(serverName string, conn *client.Client) error
	CloseAll() error
	Stats() PoolStats
}

// PoolStats 连接池统计
type PoolStats struct {
	ActiveConnections int
	IdleConnections   int
	TotalRequests     int64
	FailedRequests    int64
}

// ResourceManager 资源管理器
type ResourceManager struct {
	limits      ResourceLimits
	pool        ConnectionPool
	mu          sync.RWMutex
	connections map[string]*client.Client
	stats       ResourceStats
}

// ResourceStats 资源使用统计
type ResourceStats struct {
	MemoryUsageMB   float64
	ConnectionCount int
	ActiveTools     int
	LastCleanupTime time.Time
}

// mcpConnectionPool MCP连接池实现
type mcpConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*connectionEntry
	maxSize     int
	timeout     time.Duration
}

type connectionEntry struct {
	client   *client.Client
	lastUsed time.Time
	useCount int64
	isActive bool
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(maxSize int, timeout time.Duration) ConnectionPool {
	return &mcpConnectionPool{
		connections: make(map[string]*connectionEntry),
		maxSize:     maxSize,
		timeout:     timeout,
	}
}

// Get 从连接池获取连接
func (p *mcpConnectionPool) Get(ctx context.Context, serverName string) (*client.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, exists := p.connections[serverName]
	if !exists || !entry.isActive {
		return nil, ErrConnectionNotFound
	}

	// 检查连接是否超时
	if time.Since(entry.lastUsed) > p.timeout {
		entry.client.Close()
		delete(p.connections, serverName)
		return nil, ErrConnectionTimeout
	}

	entry.lastUsed = time.Now()
	entry.useCount++
	return entry.client, nil
}

// Put 将连接放回连接池
func (p *mcpConnectionPool) Put(serverName string, conn *client.Client) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查连接池大小
	if len(p.connections) >= p.maxSize {
		// 清理最久未使用的连接
		p.cleanupOldest()
	}

	p.connections[serverName] = &connectionEntry{
		client:   conn,
		lastUsed: time.Now(),
		isActive: true,
	}
	return nil
}

// CloseAll 关闭所有连接
func (p *mcpConnectionPool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, entry := range p.connections {
		if err := entry.client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection %s: %w", name, err))
		}
		delete(p.connections, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

// Stats 获取连接池统计信息
func (p *mcpConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	active := 0
	for _, entry := range p.connections {
		if entry.isActive {
			active++
		}
	}

	return PoolStats{
		ActiveConnections: active,
		IdleConnections:   len(p.connections) - active,
	}
}

// cleanupOldest 清理最久未使用的连接
func (p *mcpConnectionPool) cleanupOldest() {
	var oldestName string
	var oldestTime time.Time

	for name, entry := range p.connections {
		if oldestName == "" || entry.lastUsed.Before(oldestTime) {
			oldestName = name
			oldestTime = entry.lastUsed
		}
	}

	if oldestName != "" {
		if entry := p.connections[oldestName]; entry != nil {
			entry.client.Close()
		}
		delete(p.connections, oldestName)
	}
}

// NewResourceManager 创建新的资源管理器
func NewResourceManager(limits ResourceLimits) *ResourceManager {
	return &ResourceManager{
		limits:      limits,
		pool:        NewConnectionPool(limits.MaxConnections, limits.IdleTimeout),
		connections: make(map[string]*client.Client),
		stats: ResourceStats{
			LastCleanupTime: time.Now(),
		},
	}
}

// GetConnection 获取连接
func (rm *ResourceManager) GetConnection(ctx context.Context, serverName string) (*client.Client, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// 首先尝试从资源管理器获取
	if conn, exists := rm.connections[serverName]; exists && conn != nil {
		return conn, nil
	}

	// 从连接池获取
	return rm.pool.Get(ctx, serverName)
}

// StoreConnection 存储连接
func (rm *ResourceManager) StoreConnection(serverName string, conn *client.Client) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 更新统计信息
	rm.stats.ConnectionCount++
	rm.connections[serverName] = conn

	// 同时存储到连接池
	return rm.pool.Put(serverName, conn)
}

// CloseConnection 关闭连接
func (rm *ResourceManager) CloseConnection(serverName string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if conn, exists := rm.connections[serverName]; exists && conn != nil {
		if err := conn.Close(); err != nil {
			return err
		}
		delete(rm.connections, serverName)
		rm.stats.ConnectionCount--
	}

	return nil
}

// CloseAll 关闭所有连接
func (rm *ResourceManager) CloseAll() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var errs []error
	for name, conn := range rm.connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close connection %s: %w", name, err))
			}
		}
		delete(rm.connections, name)
	}

	rm.stats.ConnectionCount = 0

	// 同时关闭连接池中的所有连接
	if err := rm.pool.CloseAll(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close pool connections: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing resources: %v", errs)
	}
	return nil
}

// GetStats 获取资源统计信息
func (rm *ResourceManager) GetStats() ResourceStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.stats
}

// GetPoolStats 获取连接池统计信息
func (rm *ResourceManager) GetPoolStats() PoolStats {
	return rm.pool.Stats()
}

// CleanupIdleConnections 清理空闲连接
func (rm *ResourceManager) CleanupIdleConnections() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	cleaned := 0
	now := time.Now()

	for name, conn := range rm.connections {
		// 这里需要获取连接的最后使用时间
		// 由于client.Client没有提供最后使用时间，我们暂时跳过这个检查
		// 在实际实现中，需要跟踪连接的最后使用时间
		_ = conn
		_ = now

		// 简化实现：清理超过空闲超时的连接
		// 注意：这需要跟踪每个连接的最后使用时间
		// 暂时不实现具体清理逻辑
		_ = name
	}

	rm.stats.LastCleanupTime = time.Now()
	return cleaned
}

// DefaultResourceLimits 默认资源限制
var DefaultResourceLimits = ResourceLimits{
	MaxConnections:    10,
	MaxMemoryMB:       100,
	ConnectionTimeout: 30 * time.Second,
	IdleTimeout:       5 * time.Minute,
}
