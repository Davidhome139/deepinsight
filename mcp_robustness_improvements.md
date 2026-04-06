# MCP服务器鲁棒性改进总结

## 问题背景
用户报告"前几次经常出现MCP无故关闭"，需要增加系统的鲁棒性，避免非项目退出时的MCP服务器关闭。

## 解决方案
按照"不立即关闭连接，而是标记为需要重新连接，然后异步尝试重新连接"的思路，对系统进行了以下改进：

### 1. 扩展MCPServer结构体
在 `MCPServer` 结构体中添加了新的运行时字段：
- `NeedsReconnect bool` - 标记服务器是否需要重新连接
- `LastReconnectAttempt time.Time` - 记录最后一次重新连接尝试时间

### 2. 修改错误处理逻辑
在 `CallTool` 方法的错误处理中，当检测到 "file already closed" 错误时：
- **不再立即关闭连接**
- **标记服务器为需要重新连接** (`server.NeedsReconnect = true`)
- **记录错误信息和时间戳**
- **异步启动重新连接过程**
- **返回临时错误信息**，告知用户服务器正在重新连接

### 3. 添加异步重新连接机制
新增 `asyncReconnectServer` 方法：
- 使用指数退避策略（1, 2, 4, 8, 16秒，最大30秒）
- 最多重试5次
- 在后台异步执行，不阻塞主线程
- 成功重新连接后清除标记
- 所有重试失败后标记为永久断开

### 4. 增强CallTool方法的鲁棒性
在 `CallTool` 方法开始时检查：
- 如果服务器标记为需要重新连接
- 检查最后一次重新连接尝试时间
- 如果超过30秒，尝试立即重新连接
- 否则返回错误，告知用户服务器正在重新连接

### 5. 添加健康检查机制
新增健康检查系统：
- **自动启动**：在 `NewMCPManager` 中启动健康检查goroutine
- **定期检查**：每60秒检查一次所有服务器
- **智能重连**：对于标记为需要重新连接的服务器，如果超过5分钟没有尝试，自动尝试重新连接
- **优雅关闭**：在 `Close` 方法中停止健康检查goroutine

## 修改的文件

### 1. `backend/internal/config/mcpservers.go`
- 扩展 `MCPServer` 结构体，添加 `NeedsReconnect` 和 `LastReconnectAttempt` 字段

### 2. `backend/internal/services/agent/mcp_manager.go`
- 在 `MCPManager` 结构体中添加健康检查管理字段
- 修改 `NewMCPManager` 函数，初始化健康检查上下文并启动goroutine
- 添加 `startHealthCheck` 和 `performHealthCheck` 方法
- 修改 `CallTool` 方法的错误处理逻辑
- 添加 `asyncReconnectServer` 方法
- 更新 `Close` 方法以停止健康检查goroutine

## 核心改进点

### 1. **避免不必要的连接关闭**
```go
// 之前：立即关闭连接
if server.Client != nil {
    server.Client.Close()
    server.Client = nil
}
server.Connected = false

// 现在：标记为需要重新连接
server.NeedsReconnect = true
server.LastError = err.Error()
server.LastReconnectAttempt = time.Now()
```

### 2. **异步重新连接**
```go
// 在后台异步重新连接
go m.asyncReconnectServer(serverName, toolName, args)

// 返回临时错误
return "", fmt.Errorf("server %s temporarily unavailable, reconnecting...", serverName)
```

### 3. **指数退避重试**
```go
// Exponential backoff: 1, 2, 4, 8, 16 seconds
delay := baseDelay * time.Duration(1<<(attempt-1))
if delay > maxDelay {
    delay = maxDelay
}
```

### 4. **健康检查**
```go
// 每60秒检查一次
ticker := time.NewTicker(60 * time.Second)

// 对于需要重新连接的服务器，如果超过5分钟没有尝试，自动重连
if time.Since(server.LastReconnectAttempt) > 5*time.Minute {
    go func(name string) {
        err := m.ConnectToServer(name)
        // ... 处理重连结果
    }(serverName)
}
```

## 预期效果

### 1. **减少无故关闭**
- 服务器进程异常时，不再立即关闭连接
- 连接状态得到保留，减少状态不一致问题

### 2. **提高用户体验**
- 用户收到明确的"服务器正在重新连接"提示
- 系统在后台自动尝试恢复，用户无需手动干预

### 3. **增加系统稳定性**
- 指数退避策略避免频繁重试导致的资源浪费
- 健康检查机制确保长时间断开的服务器能被自动恢复

### 4. **更好的错误处理**
- 区分临时性错误和永久性错误
- 对于临时性错误，提供优雅的降级方案

## 测试建议

### 1. **模拟服务器崩溃**
- 手动停止Playwright等MCP服务器进程
- 观察系统是否标记为需要重新连接
- 验证异步重新连接是否正常工作

### 2. **测试错误恢复**
- 在工具调用过程中模拟"file already closed"错误
- 验证是否返回正确的错误信息
- 检查重新连接日志

### 3. **验证健康检查**
- 让服务器长时间处于断开状态
- 观察健康检查是否自动尝试重新连接
- 验证重连成功后标记是否被清除

## 注意事项

### 1. **资源管理**
- 异步重新连接可能创建多个goroutine，但指数退退避和最大重试次数限制了资源使用
- 健康检查每60秒运行一次，开销较小

### 2. **状态一致性**
- 服务器标记为需要重新连接时，`Connected` 字段可能为 `true`，但实际连接已断开
- 这反映了"应该连接但当前断开"的状态，比直接设置为 `false` 更准确

### 3. **向后兼容性**
- 新增字段都有 `omitempty` 标签，JSON序列化时不会影响现有代码
- 所有修改都是增量式的，不影响现有功能

## 总结
通过这次修改，MCP服务器管理系统现在具有更强的鲁棒性：
1. **避免无故关闭**：不再立即关闭连接，而是标记为需要重新连接
2. **自动恢复**：在后台异步尝试重新连接，使用指数退避策略
3. **健康监控**：定期检查服务器状态，自动恢复长时间断开的连接
4. **用户友好**：提供明确的错误信息，告知用户系统正在自动恢复

这些改进应该能显著减少用户感知到的"MCP无故关闭"问题，提高系统的整体稳定性和用户体验。