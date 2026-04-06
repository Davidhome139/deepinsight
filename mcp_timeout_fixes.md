# MCP服务器超时问题修复总结

## 问题分析
从终端日志中发现了两个关键问题：

### 1. Playwright工具发现超时
```
2026/04/05 00:52:59 [MCP] Successfully connected to server: playwright with 21 tools
2026/04/05 00:52:59 [MCP] Discovering tools for server: playwright
2026/04/05 00:53:09 [MCP] Warning: Tool discovery failed for playwright: transport error: context deadline exceeded
```

**问题**：Playwright初始化成功并发现了21个工具，但10秒后的工具发现失败了（超时）。

### 2. Context7初始化失败后仍尝试获取工具
```
2026/04/05 00:53:08 [MCP] Initialization failed for context7: transport error: context deadline exceeded
2026/04/05 00:53:08 [MCP] Will try to get tools for context7 despite initialization failure
2026/04/05 00:53:08 [MCP] Getting tools for context7...
2026/04/05 00:53:08 [MCP] Warning: Tool discovery failed for context7: client not initialized
```

**问题**：Context7初始化失败，但系统仍然尝试获取工具，导致"client not initialized"错误。

## 解决方案

### 1. 增加工具发现超时时间
**修改文件**: `backend/internal/services/agent/mcp_manager.go`

#### 在 `connectServer` 方法中：
```go
// 之前：所有服务器使用15秒超时
toolsCtx, toolsCancel := context.WithTimeout(context.Background(), 15*time.Second)

// 现在：Playwright和Context7使用30秒超时，其他服务器使用15秒
var toolsTimeout time.Duration
if server.Name == "playwright" || server.Name == "context7" {
    toolsTimeout = 30 * time.Second
    log.Printf("[MCP] Using longer timeout (%v) for %s tool listing", toolsTimeout, server.Name)
} else {
    toolsTimeout = 15 * time.Second
}
toolsCtx, toolsCancel := context.WithTimeout(context.Background(), toolsTimeout)
```

#### 在 `DiscoverTools` 方法中：
```go
// 之前：所有服务器使用10秒超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

// 现在：Playwright和Context7使用30秒超时，其他服务器使用15秒
var timeout time.Duration
if serverName == "playwright" || serverName == "context7" {
    timeout = 30 * time.Second
    log.Printf("[MCP] Using longer timeout (%v) for %s tool discovery", timeout, serverName)
} else {
    timeout = 15 * time.Second
}
ctx, cancel := context.WithTimeout(context.Background(), timeout)
```

### 2. 修复初始化失败后的逻辑
**问题**：当客户端初始化失败时，不应该尝试获取工具。

**修复**：
```go
// 之前：无论初始化是否成功都尝试获取工具
toolsResult, err := cli.ListTools(toolsCtx, mcp.ListToolsRequest{})

// 现在：检查客户端是否已初始化
var toolsResult *mcp.ListToolsResult
var toolsErr error

// Check if client is properly initialized before trying to get tools
if cli != nil {
    // ... 获取工具
    toolsResult, toolsErr = cli.ListTools(toolsCtx, mcp.ListToolsRequest{})
} else {
    toolsErr = fmt.Errorf("client not initialized")
}
```

## 修改的文件

### 1. `backend/internal/services/agent/mcp_manager.go`
- **第708-745行**：修改 `connectServer` 方法中的工具发现逻辑
- **第970-985行**：修改 `DiscoverTools` 方法中的超时设置

## 预期效果

### 1. 减少工具发现超时错误
- Playwright和Context7等复杂服务器现在有更长的超时时间（30秒）
- 标准服务器仍然使用合理的超时时间（15秒）

### 2. 避免无效的工具发现尝试
- 当客户端初始化失败时，不再尝试获取工具
- 减少"client not initialized"错误日志

### 3. 提高系统稳定性
- 更合理的超时设置减少因网络延迟或服务器响应慢导致的连接失败
- 更好的错误处理逻辑提高系统的鲁棒性

## 测试建议

### 1. 验证超时设置
- 启动系统并观察Playwright和Context7的连接日志
- 确认工具发现不再出现"context deadline exceeded"错误

### 2. 测试初始化失败场景
- 模拟Context7初始化失败的情况
- 验证系统是否不再尝试获取工具，而是记录适当的错误信息

### 3. 监控连接性能
- 观察工具发现的实际耗时
- 根据实际情况调整超时时间

## 注意事项

### 1. 超时时间的平衡
- 30秒对于工具发现来说可能仍然不够，但这是一个合理的起点
- 如果仍然出现超时，可以考虑进一步增加或实现动态超时调整

### 2. 资源占用
- 更长的超时时间意味着连接可能保持更长时间
- 但这也避免了因超时过早而导致的重新连接尝试

### 3. 用户体验
- 用户可能会注意到工具加载时间变长
- 但比工具发现失败要好，因为失败会导致功能不可用

## 总结
通过这次修改，我们解决了两个关键问题：

1. **工具发现超时**：为Playwright和Context7等复杂服务器增加了超时时间
2. **初始化失败处理**：避免在客户端未初始化时尝试获取工具

这些改进应该能显著减少日志中的错误信息，提高MCP服务器的连接成功率，从而提供更稳定的用户体验。