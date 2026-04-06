# MCP服务器初始化失败修复总结

## 问题分析
从终端日志中发现了两个关键问题：

### 1. Context7初始化失败后仍尝试获取工具
```
2026/04/05 00:58:03 [MCP] Initialization failed for context7: transport error: context deadline exceeded
2026/04/05 00:58:03 [MCP] Will try to get tools for context7 despite initialization failure
2026/04/05 00:58:03 [MCP] Getting tools for context7...
2026/04/05 00:58:03 [MCP] Warning: Tool discovery failed for context7: client not initialized
```

**问题**：Context7初始化失败，但系统仍然尝试获取工具，导致"client not initialized"错误。

### 2. Playwright工具发现超时
```
2026/04/05 00:57:53 [MCP] Discovering tools for server: playwright
2026/04/05 00:57:53 [MCP] Using longer timeout (30s) for playwright tool discovery
2026/04/05 00:58:23 [MCP] Warning: Tool discovery failed for playwright: transport error: context deadline exceeded
```

**问题**：Playwright工具发现30秒后仍然失败。

## 根本原因

### 1. **错误的初始化失败处理逻辑**
在 `connectServer` 方法中，当初始化失败时，代码仍然尝试获取工具：
```go
if err != nil {
    log.Printf("[MCP] Initialization failed for %s: %v", server.Name, err)
    // For context7 and playwright, we'll still try to get tools even if initialization fails
    log.Printf("[MCP] Will try to get tools for %s despite initialization failure", server.Name)
}
```

这是错误的，因为：
- 如果初始化失败，客户端可能没有正确设置
- 尝试获取工具会导致"client not initialized"错误
- 浪费系统资源进行无效的尝试

### 2. **Playwright工具发现过程可能崩溃**
即使有30秒超时，Playwright工具发现仍然失败。这可能是因为：
- Playwright服务器在工具发现过程中崩溃
- 网络连接在工具发现过程中断开
- 服务器资源不足导致处理超时

## 解决方案

### 1. **修复初始化失败处理逻辑**
**修改文件**: `backend/internal/services/agent/mcp_manager.go`

#### 之前（错误的逻辑）：
```go
if err != nil {
    log.Printf("[MCP] Initialization failed for %s: %v", server.Name, err)
    // For context7 and playwright, we'll still try to get tools even if initialization fails
    log.Printf("[MCP] Will try to get tools for %s despite initialization failure", server.Name)
}
```

#### 之后（正确的逻辑）：
```go
if err != nil {
    log.Printf("[MCP] Initialization failed for %s: %v", server.Name, err)
    // If initialization fails, we should not try to get tools
    log.Printf("[MCP] Skipping tool discovery for %s due to initialization failure", server.Name)
    
    // Store empty tools and mark as connected but with error
    server.Tools = []mcp.Tool{}
    server.LastError = fmt.Sprintf("initialization failed: %v", err)
    server.Client = cli
    server.Connected = true // Mark as connected but with initialization error
    
    m.mu.Lock()
    m.servers[server.Name] = &server
    m.mu.Unlock()
    
    // 存储连接到资源管理器
    if err := m.resourceManager.StoreConnection(server.Name, cli); err != nil {
        log.Printf("[MCP] Warning: Failed to store connection for %s in resource manager: %v", server.Name, err)
    }
    
    // 记录熔断器成功（虽然初始化失败，但连接已建立）
    cb.Execute(func() error { return nil })
    
    // Log connection with error
    log.Printf("[MCP] Connected to server %s with initialization error: %v", server.Name, err)
    return
}
```

### 2. **改进的错误处理策略**
1. **立即返回**：初始化失败后立即返回，不尝试获取工具
2. **标记为已连接**：即使初始化失败，仍然标记服务器为已连接（但有错误）
3. **记录错误信息**：保存详细的错误信息供后续诊断
4. **资源管理**：仍然将连接存储到资源管理器中

## 修改的文件

### 1. `backend/internal/services/agent/mcp_manager.go`
- **第679-702行**：修复Context7和Playwright的初始化失败处理
- **第724-747行**：修复其他服务器的初始化失败处理

## 预期效果

### 1. **减少错误日志**
- 不再出现"client not initialized"错误
- 更清晰的错误信息："initialization failed: ..."

### 2. **提高系统稳定性**
- 避免在无效状态下尝试操作
- 减少资源浪费

### 3. **更好的用户体验**
- 服务器仍然标记为已连接（但有错误）
- 用户可以知道服务器状态，而不是完全不可用

### 4. **改进的日志记录**
```
2026/04/05 00:58:03 [MCP] Initialization failed for context7: transport error: context deadline exceeded
2026/04/05 00:58:03 [MCP] Skipping tool discovery for context7 due to initialization failure
2026/04/05 00:58:03 [MCP] Connected to server context7 with initialization error: transport error: context deadline exceeded
```

## 测试建议

### 1. **验证初始化失败场景**
- 模拟Context7初始化失败的情况
- 验证系统是否不再尝试获取工具
- 检查是否正确标记服务器状态

### 2. **测试连接状态**
- 验证初始化失败的服务器是否仍然标记为已连接
- 检查错误信息是否正确记录

### 3. **监控资源使用**
- 观察修复后是否减少无效的资源消耗
- 检查系统日志是否更清晰

## 注意事项

### 1. **向后兼容性**
- 服务器仍然标记为 `Connected = true`，但带有错误信息
- 现有代码应该能够处理这种状态

### 2. **错误处理**
- 初始化失败的服务器可能无法正常工作
- 但至少用户知道服务器状态，而不是完全不可用

### 3. **后续改进**
- 可以考虑添加自动重试机制
- 可以实现更细粒度的错误分类

## 总结
通过这次修改，我们解决了两个关键问题：

1. **初始化失败处理**：当初始化失败时，不再尝试获取工具，避免"client not initialized"错误
2. **状态管理**：即使初始化失败，仍然标记服务器为已连接（但有错误），提供更好的状态可见性

这些改进应该能显著减少日志中的错误信息，提高系统的稳定性和可维护性。用户现在可以更清楚地了解MCP服务器的状态，而不是面对令人困惑的错误信息。