# MCP服务器集成经验总结

## 概述

本文档总结了在DouBao项目中集成MCP（Model Context Protocol）服务器的经验教训，特别是针对context7服务器的集成过程。经过5天的调试和修复，我们成功解决了从settings模块测试成功但在chat页面初始化失败的问题。

### 核心问题
- MCP服务器在settings模块测试成功，但在chat页面初始化失败
- 出现多种错误：服务器未找到、连接失败、初始化超时等
- 异步连接时序问题导致状态不一致

### 技术栈
- **后端语言**: Go 1.21+
- **Web框架**: Gin
- **MCP库**: github.com/mark3labs/mcp-go
- **容器化**: Docker + Docker Compose
- **包管理**: npm (用于context7服务器)

## 配置层

### 1. MCP服务器配置文件

**文件位置**: `backend/config/mcpservers.json`

```json
{
  "context7": {
    "args": [
      "-y",
      "@upstash/context7-mcp"
    ],
    "command": "npx",
    "connected": false,
    "enabled": true,
    "env": {
      "NODE_TLS_REJECT_UNAUTHORIZED": "0"
    },
    "fromgalleryid": null,
    "name": "Context7 (上下文管理)",
    "server_type": "command",
    "type": "command"
  }
}
```

### 2. 关键配置项说明

| 配置项 | 说明 | 注意事项 |
|--------|------|----------|
| `command` | 启动命令 | 使用`npx`执行npm包 |
| `args` | 命令参数 | `-y`自动确认，`@upstash/context7-mcp`包名 |
| `env` | 环境变量 | `NODE_TLS_REJECT_UNAUTHORIZED=0`绕过SSL验证 |
| `type` | 服务器类型 | 必须设置为`command` |
| `enabled` | 启用状态 | 控制服务器是否自动发现 |

### 3. 配置加载代码

**文件位置**: `backend/internal/config/mcpservers.go`

```go
// GetMCPServersConfig 获取MCP服务器配置
func GetMCPServersConfig() *MCPServersConfig {
    configMu.RLock()
    defer configMu.RUnlock()
    
    if cachedConfig != nil {
        log.Println("[Config] Returning cached config with", len(cachedConfig.Servers), "servers")
        return cachedConfig
    }
    
    // 从文件加载配置
    configPath := filepath.Join(GetConfigDir(), "mcpservers.json")
    data, err := os.ReadFile(configPath)
    if err != nil {
        log.Printf("[Config] Failed to read MCP servers config: %v", err)
        return &MCPServersConfig{Servers: make(map[string]MCPServer)}
    }
    
    var config MCPServersConfig
    if err := json.Unmarshal(data, &config); err != nil {
        log.Printf("[Config] Failed to parse MCP servers config: %v", err)
        return &MCPServersConfig{Servers: make(map[string]MCPServer)}
    }
    
    cachedConfig = &config
    return cachedConfig
}
```

## 初始化层

### 1. MCP管理器创建

**文件位置**: `backend/internal/services/agent/mcp_manager.go`

```go
// NewMCPManager 创建MCP管理器
func NewMCPManager() *MCPManager {
    log.Println("[MCP] ========== NewMCPManager called ==========")
    m := &MCPManager{
        servers: make(map[string]*config.MCPServer),
    }
    log.Println("[MCP] MCPManager created (discovery will be triggered on demand)")
    return m
}
```

### 2. 服务器发现机制

**关键问题**: 服务器发现是异步的，但`GetServer`调用是同步的

**原始问题代码**:
```go
func (m *MCPManager) GetServer(name string) (*config.MCPServer, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    server, ok := m.servers[name]
    return server, ok
}
```

**修复后的代码**:
```go
func (m *MCPManager) GetServer(name string) (*config.MCPServer, bool) {
    m.mu.RLock()
    server, ok := m.servers[name]
    m.mu.RUnlock()
    
    // 如果服务器未找到，尝试发现它
    if !ok {
        log.Printf("[MCP] Server %s not found, triggering discovery...", name)
        m.Discover()
        
        // 发现后再次尝试
        m.mu.RLock()
        server, ok = m.servers[name]
        m.mu.RUnlock()
        
        if ok {
            log.Printf("[MCP] Server %s found after discovery", name)
        } else {
            log.Printf("[MCP] Server %s still not found after discovery", name)
        }
    }
    
    return server, ok
}
```

### 3. 异步发现与立即注册

**问题**: 服务器在goroutine中异步连接，但映射中没有立即添加

**修复代码**:
```go
func (m *MCPManager) Discover() {
    // ... 配置加载代码
    
    for name, server := range cfg.Servers {
        if !server.Enabled {
            continue
        }
        
        // 创建服务器副本避免竞态条件
        serverCopy := server
        serverCopy.Name = name
        
        // 立即添加到映射中（即使尚未连接）
        m.mu.Lock()
        m.servers[name] = &serverCopy
        m.mu.Unlock()
        
        // 在goroutine中异步连接
        go func(s config.MCPServer) {
            m.connectServer(s)
        }(serverCopy)
    }
}
```

## 连接层

### 1. 服务器连接流程

```go
func (m *MCPManager) connectServer(server config.MCPServer) {
    log.Printf("[MCP] ========== Connecting to server: %s ==========", server.Name)
    
    // 准备环境变量
    env := []string{}
    if len(server.Env) > 0 {
        for k, v := range server.Env {
            env = append(env, fmt.Sprintf("%s=%s", k, v))
        }
    }
    
    // 创建stdio传输
    t := transport.NewStdio(server.Command, env, server.Args...)
    cli := client.NewClient(t)
    
    // 启动客户端
    startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer startCancel()
    
    if err := cli.Start(startCtx); err != nil {
        log.Printf("[MCP] Failed to start client for %s: %v", server.Name, err)
        return
    }
    
    // 初始化（context7需要特殊处理）
    m.initializeServer(server.Name, cli)
}
```

### 2. context7特殊初始化处理

**问题**: context7初始化经常超时（60秒仍不够）

**解决方案**: 使用更短的超时并允许初始化失败

```go
func (m *MCPManager) initializeServer(serverName string, cli *client.Client) {
    if serverName == "context7" {
        // context7使用更短的超时
        log.Printf("[MCP] Initializing context7 with 10s timeout...")
        initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer initCancel()
        
        _, err := cli.Initialize(initCtx, initReq)
        
        if err != nil {
            log.Printf("[MCP] Initialization failed for context7: %v", err)
            // 即使初始化失败，仍然尝试获取工具
            log.Printf("[MCP] Will try to get tools for context7 despite initialization failure")
        } else {
            log.Printf("[MCP] Initialization successful for context7")
        }
    } else {
        // 其他服务器使用标准初始化
        // ... 标准初始化代码
    }
    
    // 获取工具列表
    m.getServerTools(serverName, cli)
}
```

### 3. 连接等待时间调整

**问题**: `ConnectToServer`只等待1秒，但context7需要约2秒

**修复代码**:
```go
func (m *MCPManager) ConnectToServer(serverName string) error {
    // ... 配置检查代码
    
    // 连接到特定服务器
    originalServer.Name = serverName
    m.connectServer(originalServer)
    
    // 等待连接建立 - context7需要约2秒
    var waitTime time.Duration
    if serverName == "context7" {
        waitTime = 5 * time.Second  // 为context7使用更长的超时
        log.Printf("[MCP] Waiting %v for context7 connection...", waitTime)
    } else {
        waitTime = 2 * time.Second
    }
    time.Sleep(waitTime)
    
    // 检查连接是否成功
    updatedServer, ok := m.GetServer(serverName)
    if !ok || !updatedServer.Connected {
        return fmt.Errorf("failed to connect to server %s after %v wait", serverName, waitTime)
    }
    
    return nil
}
```

## 调用层

### 1. 工具调用流程

```go
func (m *MCPManager) CallTool(serverName string, toolName string, args map[string]interface{}) (string, error) {
    // 获取服务器
    server, ok := m.GetServer(serverName)
    if !ok {
        return "", fmt.Errorf("MCP server %s not found", serverName)
    }
    
    // 检查连接状态
    if !server.Connected {
        // 尝试连接服务器
        err := m.ConnectToServer(serverName)
        if err != nil {
            return "", fmt.Errorf("MCP server %s not connected and failed to connect: %v", serverName, err)
        }
        // 获取更新后的服务器信息
        server, ok = m.GetServer(serverName)
        if !ok || !server.Connected {
            return "", fmt.Errorf("MCP server %s still not connected after connection attempt", serverName)
        }
    }
    
    // 调用工具
    callCtx, callCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer callCancel()
    
    callReq := mcp.CallToolRequest{
        Params: mcp.CallToolRequestParams{
            Name: toolName,
            Arguments: args,
        },
    }
    
    result, err := cli.CallTool(callCtx, callReq)
    if err != nil {
        return "", fmt.Errorf("failed to call tool %s on server %s: %v", toolName, serverName, err)
    }
    
    return result.Content[0].Text, nil
}
```

### 2. Chat服务集成

**文件位置**: `backend/internal/services/chat/chat.go`

```go
func (s *chatService) executeMCPTool(serverName string, toolName string, userContent string) string {
    // 检查MCP管理器是否可用
    if s.mcpManager == nil {
        return "[MCP Error: MCP manager not initialized]"
    }
    
    // 获取服务器
    server, ok := s.mcpManager.GetServer(serverName)
    if !ok {
        return fmt.Sprintf("[MCP Error: Server '%s' not found]", serverName)
    }
    
    // 准备参数
    args := map[string]interface{}{
        "input": userContent,
    }
    
    // 调用工具
    result, err := s.mcpManager.CallTool(serverName, toolName, args)
    if err != nil {
        return fmt.Sprintf("[MCP Error: %v]", err)
    }
    
    return result
}
```

## 问题诊断

### 1. 常见错误与解决方案

| 错误信息 | 原因 | 解决方案 |
|----------|------|----------|
| `[Chat] MCP server not found: context7` | 服务器未在映射中 | 在`Discover()`中立即添加服务器到映射 |
| `[Chat] MCP server not connected: context7` | 连接时序问题 | 增加`ConnectToServer`的等待时间 |
| `client not initialized` | 初始化未调用 | 确保调用`cli.Initialize()` |
| `transport error: context deadline exceeded` | 初始化超时 | 减少超时时间，允许初始化失败 |
| `Failed to get tools for context7` | 初始化失败后获取工具 | 即使初始化失败也尝试获取工具 |

### 2. 日志分析要点

**成功连接的关键日志**:
```
[MCP] Initializing context7 with 10s timeout...
[MCP] Initialization successful for context7
[MCP] Getting tools for context7...
[MCP] Successfully connected to server: context7 with 2 tools
```

**失败日志模式**:
```
[MCP] Server context7 not found, triggering discovery...
[MCP] Initialization failed for context7: context deadline exceeded
[Chat] MCP server not connected: context7
```

### 3. Docker环境问题

**问题**: context7命令在Docker容器中不可用

**解决方案**: 在Dockerfile中全局安装

```dockerfile
# 安装context7 MCP服务器
RUN npm install -g @upstash/context7-mcp
```

## 最佳实践

### 1. 设计原则

1. **立即注册原则**: 服务器在开始连接前就应注册到映射中
2. **异步连接原则**: 连接过程应该是异步的，不阻塞主线程
3. **容错处理原则**: 允许初始化失败，继续尝试其他操作
4. **超时优化原则**: 根据服务器特性设置合适的超时时间

### 2. 代码模式

**服务器发现模式**:
```go
// 1. 立即注册
m.servers[name] = &serverCopy

// 2. 异步连接
go m.connectServer(serverCopy)
```

**连接重试模式**:
```go
if !server.Connected {
    err := m.ConnectToServer(serverName)
    if err != nil {
        // 记录错误但继续
        log.Printf("[MCP] Connection failed: %v", err)
    }
}
```

### 3. 配置管理

1. **环境变量**: 使用`NODE_TLS_REJECT_UNAUTHORIZED=0`绕过SSL问题
2. **命令路径**: 确保命令在Docker容器PATH中可用
3. **类型设置**: 确保`type`字段正确设置为`command`

### 4. 调试技巧

1. **日志级别**: 在关键路径添加详细日志
2. **时序分析**: 注意异步操作的时序问题
3. **超时调整**: 根据实际需要调整超时时间
4. **逐步验证**: 从settings模块开始测试，逐步扩展到chat页面

## 总结

经过5天的调试，我们成功解决了MCP服务器集成问题。关键经验包括：

1. **服务器发现机制**需要立即注册，异步连接
2. **连接时序问题**需要通过适当的等待时间解决
3. **初始化过程**需要针对不同服务器进行特殊处理
4. **错误处理**需要容错，允许部分失败

这些经验教训将帮助团队在未来集成其他MCP服务器时避免重复踩坑，提高开发效率。

## 性能影响与监控

### 1. 性能影响分析

#### 资源消耗
- **内存占用**: 每个MCP服务器进程需要独立的内存空间
- **CPU使用**: 初始化过程可能消耗较多CPU资源
- **启动时间**: 异步连接可以减少启动延迟

#### 优化建议
1. **懒加载**: 只在需要时连接服务器
2. **连接池**: 考虑实现连接池复用连接
3. **超时控制**: 合理设置超时避免资源浪费
4. **错误重试**: 实现指数退避重试机制

### 2. 监控方案

#### 日志监控
```go
// 添加监控指标
type MCPMetrics struct {
    ConnectionsActive   int
    ConnectionsFailed   int
    ToolsCalled         int
    AverageResponseTime time.Duration
}

// 在关键路径添加监控点
func (m *MCPManager) CallToolWithMetrics(serverName string, toolName string, args map[string]interface{}) (string, error) {
    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        m.metrics.AverageResponseTime = (m.metrics.AverageResponseTime*time.Duration(m.metrics.ToolsCalled) + duration) / time.Duration(m.metrics.ToolsCalled+1)
        m.metrics.ToolsCalled++
    }()
    
    return m.CallTool(serverName, toolName, args)
}
```

#### 健康检查
```go
// 定期健康检查
func (m *MCPManager) HealthCheck() map[string]bool {
    health := make(map[string]bool)
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    for name, server := range m.servers {
        if server.Connected {
            // 简单的ping检查
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()
            
            _, err := server.Client.ListTools(ctx, mcp.ListToolsRequest{})
            health[name] = err == nil
        } else {
            health[name] = false
        }
    }
    
    return health
}
```

### 3. 运维考虑

#### 部署注意事项
1. **依赖管理**: 确保所有MCP服务器的依赖在Docker镜像中
2. **版本控制**: 记录MCP服务器版本以便问题排查
3. **配置管理**: 使用环境变量管理敏感配置

#### 故障处理
1. **优雅降级**: MCP服务器失败时不影响核心功能
2. **自动恢复**: 实现断线重连机制
3. **告警机制**: 监控关键指标并设置告警阈值

## 扩展与未来工作

### 1. 支持更多MCP服务器类型
- **HTTP服务器**: 支持通过HTTP连接的MCP服务器
- **WebSocket服务器**: 支持实时通信的MCP服务器
- **本地插件**: 支持本地二进制插件

### 2. 功能增强
- **工具发现**: 自动发现和注册可用工具
- **权限控制**: 基于角色的工具访问控制
- **缓存机制**: 工具结果缓存提高性能

### 3. 开发工具
- **测试框架**: MCP服务器集成测试工具
- **调试工具**: 实时监控和调试MCP通信
- **配置生成器**: 可视化MCP服务器配置工具

## 总结

经过5天的调试，我们成功解决了MCP服务器集成问题。关键经验包括：

1. **服务器发现机制**需要立即注册，异步连接
2. **连接时序问题**需要通过适当的等待时间解决
3. **初始化过程**需要针对不同服务器进行特殊处理
4. **错误处理**需要容错，允许部分失败
5. **性能监控**是生产环境部署的关键
6. **运维支持**确保系统稳定运行

这些经验教训将帮助团队在未来集成其他MCP服务器时避免重复踩坑，提高开发效率。建议将本文档作为团队内部参考，并在实际集成过程中不断补充和完善。

---
*文档最后更新: 2026-03-26*
*基于DouBao项目实际开发经验总结*