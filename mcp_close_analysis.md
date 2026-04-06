# MCP服务器无故关闭问题分析

## 问题描述
用户报告"前几次经常出现MCP无故关闭"，需要查找不是项目退出的MCP关闭代码。

## 分析结果

### 1. 找到的MCP关闭代码位置

#### A. 热重载管理器 (HotReloadManager) - **已禁用**
**文件**: `backend/internal/services/mcp/hot_reload_manager.go`
**位置**: `reloadServer` 方法 (第350行)
```go
// 如果服务器已连接，先断开连接
if mcpServer, ok := m.mcpManager.GetServer(serverName); ok && mcpServer.Connected {
    log.Printf("[HotReload] Disconnecting server: %s", serverName)
    if mcpServer.Client != nil {
        mcpServer.Client.Close()
    }
}
```

**触发条件**:
1. 配置文件变化时 (`handleConfigChange` 方法)
2. 定期检查时 (`periodicCheck` 方法，每30秒检查一次)
3. 手动触发重载时 (`TriggerReload` 方法)

**状态**: **已禁用** - 从 `main.go` 可以看到热重载管理器已经被注释掉了。

#### B. MCP管理器中的关闭方法
**文件**: `backend/internal/services/agent/mcp_manager.go`

1. **`Close()` 方法** (第747行)
   - 在项目退出时调用
   - 关闭所有服务器连接
   - **正常行为**

2. **`CloseServer(serverName)` 方法** (第806行)
   - 关闭单个服务器
   - 需要显式调用

3. **`CloseAllServers()` 方法** (第825行)
   - 关闭所有服务器
   - 需要显式调用

4. **错误处理中的关闭** (第1367行)
   - 当检测到 "file already closed" 错误时
   - 关闭当前连接并尝试重新连接
   - **这是错误恢复机制，不是无故关闭**

### 2. 调用分析

#### 热重载管理器调用
- 在 `main.go` 中已被注释掉
- 如果启用，会导致：
  - 配置文件变化时自动重载服务器
  - 每30秒检查服务器连接状态，断开未连接的服务器并尝试重连

#### 其他关闭调用
- 只有测试文件 (`mcp_manager_test.go`) 调用了 `CloseAllServers()`
- 没有发现其他地方显式调用 `CloseServer()` 或 `CloseAllServers()`

### 3. 可能的原因

#### 已排除的原因
1. **热重载管理器** - 已禁用
2. **自动化协调器** - 已禁用
3. **显式关闭调用** - 未找到

#### 可能的原因
1. **错误恢复机制** (第1367行)
   - 当MCP服务器进程异常退出时，会检测到 "file already closed" 错误
   - 系统会自动关闭连接并尝试重新连接
   - 这可能被用户感知为"无故关闭"

2. **资源管理器超时**
   - 连接池可能有空闲超时设置
   - 长时间未使用的连接可能被关闭

3. **MCP服务器进程自身问题**
   - Playwright等浏览器自动化工具可能不稳定
   - 服务器进程可能因内存不足等原因崩溃

### 4. 建议的解决方案

#### 短期方案
1. **禁用错误恢复机制中的关闭** (谨慎考虑)
   - 修改第1367行的错误处理逻辑
   - 不立即关闭连接，而是尝试其他恢复方式

2. **增加连接稳定性**
   - 增加连接重试次数
   - 添加连接健康检查

#### 长期方案
1. **改进错误处理**
   - 区分不同类型的连接错误
   - 对于临时性错误，不立即关闭连接

2. **添加监控和日志**
   - 记录所有连接关闭事件的原因
   - 添加连接状态监控

3. **优化资源管理**
   - 调整连接池参数
   - 添加连接保活机制

### 5. 代码修改建议

如果要完全避免非项目退出的关闭，可以考虑修改错误处理逻辑：

```go
// 当前代码 (第1367行)
if strings.Contains(err.Error(), "file already closed") || strings.Contains(err.Error(), "read |0") {
    log.Printf("[MCP] Detected 'file already closed' error for server %s, attempting to reconnect...", serverName)

    // Close the current client connection
    if server.Client != nil {
        server.Client.Close()  // <-- 这里会关闭连接
        server.Client = nil
    }
    server.Connected = false
    
    // ... 重新连接逻辑
}

// 建议修改
if strings.Contains(err.Error(), "file already closed") || strings.Contains(err.Error(), "read |0") {
    log.Printf("[MCP] Detected 'file already closed' error for server %s, attempting to reconnect...", serverName)

    // 不立即关闭连接，而是标记为需要重新连接
    server.NeedsReconnect = true
    server.LastError = err.Error()
    
    // 异步重新连接
    go func() {
        time.Sleep(1 * time.Second) // 等待1秒后重试
        m.reconnectServer(serverName)
    }()
    
    return "", fmt.Errorf("server %s temporarily unavailable, reconnecting...", serverName)
}
```

### 6. 结论

1. **主要问题可能来自错误恢复机制**：当MCP服务器进程异常时，系统会自动关闭连接并尝试重新连接。

2. **热重载管理器已禁用**：不会导致无故关闭。

3. **建议优先检查MCP服务器进程的稳定性**：特别是Playwright等资源密集型工具。

4. **可以考虑优化错误处理逻辑**：减少不必要的连接关闭。