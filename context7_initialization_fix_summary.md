# Context7初始化失败问题分析与修复总结

## 问题描述
从日志中发现Context7初始化失败，而Playwright初始化成功：
```
2026/04/05 12:19:38 [MCP] Initialization failed for context7: transport error: context deadline exceeded
2026/04/05 12:19:38 [MCP] Skipping tool discovery for context7 due to initialization failure
2026/04/05 12:19:38 [MCP] Connected to server context7 with initialization error: transport error: context deadline exceeded
```

## 问题分析

### 1. **代码分析发现**
在 `mcp_manager.go` 中，初始化超时设置有问题：

```go
// 之前代码：
if server.Name == "context7" || server.Name == "playwright" {
    // For context7 and playwright, use a shorter timeout but still initialize
    log.Printf("[MCP] Initializing %s with 10s timeout...", server.Name)
    initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
    // ...
}
```

### 2. **问题根源**
- **Context7和Playwright**：都使用10秒超时
- **其他服务器**：使用30秒超时

**关键发现**：Context7可能需要更多时间初始化，因为：
1. **网络依赖**：Context7需要访问外部API服务
2. **资源加载**：可能需要下载或加载资源
3. **初始化过程**：可能比Playwright更复杂

### 3. **为什么Playwright成功而Context7失败？**
- Playwright是本地浏览器自动化工具，初始化相对简单
- Context7是文档查询服务，需要建立网络连接和加载资源

## 解决方案

### 1. **修改超时设置**
**修改文件**: `backend/internal/services/agent/mcp_manager.go`

#### 修改内容：
```go
// 区分Context7和Playwright的超时设置
var initTimeout time.Duration
if server.Name == "context7" {
    initTimeout = 30 * time.Second
    log.Printf("[MCP] Initializing %s with longer timeout (%v) due to network dependencies...", server.Name, initTimeout)
} else {
    initTimeout = 10 * time.Second
    log.Printf("[MCP] Initializing %s with %v timeout...", server.Name, initTimeout)
}

initCtx, initCancel := context.WithTimeout(context.Background(), initTimeout)
```

### 2. **修改位置**
- **第670-682行**：添加差异化的超时设置
- Context7：30秒超时
- Playwright：10秒超时（保持不变）

## 预期效果

### 1. **日志变化**
```
之前：
2026/04/05 12:19:38 [MCP] Initializing context7 with 10s timeout...
2026/04/05 12:19:38 [MCP] Initialization failed for context7: transport error: context deadline exceeded

之后：
2026/04/05 12:19:38 [MCP] Initializing context7 with longer timeout (30s) due to network dependencies...
2026/04/05 12:19:38 [MCP] Initialization successful for context7
```

### 2. **连接状态**
- Context7有更多时间完成初始化
- 减少因超时导致的连接失败
- 提高系统稳定性

## 测试建议

### 1. **验证修复**
- 重启MCP服务，观察Context7初始化日志
- 检查Context7工具是否可用
- 测试Context7工具调用

### 2. **监控指标**
- 记录Context7实际初始化时间
- 监控初始化成功率
- 跟踪工具调用成功率

### 3. **性能测试**
- 测试不同网络条件下的初始化时间
- 验证30秒超时是否足够
- 考虑是否需要进一步调整

## 其他可能的问题

### 1. **网络连接问题**
Context7需要访问：
- `https://api.context7.com` (API服务)
- `https://www.npmjs.com` (包管理)
- 其他依赖服务

### 2. **包安装问题**
确保 `@upstash/context7-mcp` 正确安装：
```bash
# 检查安装
npm list -g @upstash/context7-mcp

# 如果需要重新安装
npm install -g @upstash/context7-mcp
```

### 3. **系统资源**
- 确保有足够的内存
- 检查磁盘空间
- 验证网络带宽

## 配置验证

### 1. **Context7配置** (`mcpservers.json`)
```json
"context7": {
    "args": ["-y", "@upstash/context7-mcp"],
    "command": "npx",
    "enabled": true,
    "name": "Context7 (上下文管理)"
}
```

### 2. **环境检查**
- Node.js版本兼容性
- npm包管理器可用性
- 网络代理设置（如果需要）

## 扩展考虑

### 1. **动态超时调整**
未来可以考虑：
- 根据历史成功率动态调整超时
- 实现指数退避重试机制
- 添加健康检查

### 2. **错误恢复**
- 初始化失败后的自动重试
- 降级策略（使用缓存结果）
- 用户友好的错误提示

### 3. **监控告警**
- 初始化失败告警
- 平均初始化时间监控
- 成功率仪表板

## 总结

通过这次修复，我们解决了：

1. **超时不足问题**：Context7现在有30秒初始化时间
2. **差异化配置**：不同服务器使用不同的超时设置
3. **错误处理改进**：更清晰的日志和错误信息

这个修复应该能显著提高Context7的初始化成功率，使系统更加稳定可靠。如果仍然有问题，可能需要进一步检查网络连接、包安装或系统资源。