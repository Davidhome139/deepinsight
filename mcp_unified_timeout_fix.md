# MCP服务器统一超时设置修复总结

## 问题描述
之前MCP服务器的初始化超时设置不一致：
- Context7：30秒（之前修复的）
- Playwright：10秒
- 其他服务器：30秒

根据要求，需要统一修改为15秒，避免30秒太慢的问题。

## 解决方案

### 1. **统一超时设置**
将所有MCP服务器的初始化超时统一设置为**15秒**。

### 2. **修改文件**
**修改文件**: `backend/internal/services/agent/mcp_manager.go`

#### 修改点1：Context7和Playwright的超时设置（第670-682行）
```go
// 之前（差异化设置）：
var initTimeout time.Duration
if server.Name == "context7" {
    initTimeout = 30 * time.Second
    log.Printf("[MCP] Initializing %s with longer timeout (%v) due to network dependencies...", server.Name, initTimeout)
} else {
    initTimeout = 10 * time.Second
    log.Printf("[MCP] Initializing %s with %v timeout...", server.Name, initTimeout)
}

// 之后（统一设置）：
initTimeout := 15 * time.Second
log.Printf("[MCP] Initializing %s with unified timeout (%v)...", server.Name, initTimeout)
```

#### 修改点2：其他服务器的超时设置（第718-719行）
```go
// 之前：
initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)

// 之后：
initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
```

## 修改效果

### 1. **超时设置对比**
| 服务器类型 | 之前超时 | 之后超时 | 变化 |
|-----------|----------|----------|------|
| Context7 | 30秒 | 15秒 | -15秒 |
| Playwright | 10秒 | 15秒 | +5秒 |
| 其他服务器 | 30秒 | 15秒 | -15秒 |

### 2. **性能影响**
- **Context7**：减少15秒等待时间，提高响应速度
- **Playwright**：增加5秒等待时间，确保稳定初始化
- **整体**：更平衡的超时设置，避免过长等待

### 3. **日志输出变化**
```
之前（Context7）：
[MCP] Initializing context7 with longer timeout (30s) due to network dependencies...

之前（Playwright）：
[MCP] Initializing playwright with 10s timeout...

之后（统一）：
[MCP] Initializing context7 with unified timeout (15s)...
[MCP] Initializing playwright with unified timeout (15s)...
```

## 设计考虑

### 1. **为什么选择15秒？**
- **足够时间**：大多数MCP服务器能在15秒内完成初始化
- **用户体验**：避免用户等待过久
- **系统资源**：减少长时间占用资源

### 2. **平衡考虑**
- Context7从30秒减到15秒：网络依赖可能仍需要时间
- Playwright从10秒增到15秒：确保复杂场景下的稳定初始化
- 统一设置：简化维护，避免差异化配置的复杂性

### 3. **错误处理**
即使15秒超时，系统仍然：
1. 标记服务器为"已连接但有错误"
2. 跳过工具发现
3. 允许后续重连尝试
4. 提供清晰的错误信息

## 测试建议

### 1. **功能测试**
- 重启MCP服务，观察初始化日志
- 验证所有服务器是否在15秒内完成初始化
- 测试工具调用功能

### 2. **性能测试**
- 记录各服务器的实际初始化时间
- 监控超时失败率
- 比较修改前后的性能差异

### 3. **边界测试**
- 测试网络缓慢情况下的表现
- 验证超时后的错误处理
- 检查重连机制是否正常工作

## 监控指标

### 1. **关键指标**
- 初始化成功率
- 平均初始化时间
- 超时失败率
- 重连次数

### 2. **告警设置**
- 初始化失败率 > 5%
- 平均初始化时间 > 10秒
- 连续重连失败 > 3次

## 扩展性考虑

### 1. **配置化超时**
未来可以考虑将超时设置移到配置文件中：
```yaml
mcp:
  timeouts:
    initialization: 15s
    tool_discovery: 30s
    tool_call: 60s
```

### 2. **动态调整**
- 根据历史成功率动态调整超时
- 实现自适应超时算法
- 添加熔断器模式

### 3. **服务器特定配置**
如果需要，仍可支持服务器特定的超时：
```go
// 未来可能的扩展
timeouts := map[string]time.Duration{
    "context7":    20 * time.Second,
    "playwright":  10 * time.Second,
    "default":     15 * time.Second,
}
```

## 总结

通过这次修改，我们实现了：

1. **统一超时**：所有MCP服务器使用15秒初始化超时
2. **性能优化**：减少不必要的长时间等待
3. **简化维护**：消除差异化配置的复杂性
4. **平衡设计**：在速度和稳定性之间取得平衡

这个修改应该能提高系统的响应速度，同时保持足够的稳定性。如果某些服务器在15秒内仍然无法完成初始化，可能需要进一步优化服务器本身或考虑服务器特定的超时设置。