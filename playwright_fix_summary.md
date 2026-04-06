# Playwright MCP 服务器修复总结

## 问题描述

用户报告了两个问题：

1. **工具推荐问题**：当用户查询包含 "playwright" 时，系统错误地推荐了 `search/web_search` 而不是 `playwright/browser_navigate`
2. **连接问题**：Playwright MCP 服务器没有连接，导致工具执行失败

## 解决方案

### 1. 修复工具推荐问题

**问题**：在 `detectMCPIntentSemantic` 函数中，当用户查询包含 "playwright" 时，系统没有正确推荐 Playwright 工具。

**修复**：在 [chat.go](file:///d:/apps/newDouBao/backend/internal/services/chat/chat.go#L1276-1345) 的 `detectMCPIntentSemantic` 函数中添加了特殊处理：

```go
// Special handling for playwright queries - if user explicitly mentions playwright,
// we should recommend playwright/browser_navigate regardless of other factors
contentLower := strings.ToLower(content)
if strings.Contains(contentLower, "playwright") {
    fmt.Printf("[Chat] Special handling: User explicitly mentioned 'playwright', recommending playwright/browser_navigate\n")
    return "playwright/browser_navigate"
}
```

**效果**：现在当用户查询包含 "playwright" 时，系统会正确推荐 `playwright/browser_navigate`。

### 2. 修复连接问题（按用户要求修改）

**用户要求**："MCPserver都改成项目启动和MCP新增的时候启动，只有项目退出才关闭。不搞按需启动。"

**修改内容**：移除了 Playwright 的按需连接策略，改为在项目启动时启动所有 MCP 服务器。

#### 修改的文件和位置：

1. **`mcp_manager.go` - `Discover` 方法**：
   - 移除了对 Playwright 的特殊处理：`if name == "playwright" { continue }`
   - 现在 Playwright 会像其他服务器一样在项目启动时连接

2. **`mcp_manager.go` - `connectServer` 方法**：
   - 移除了 Playwright 的特殊处理代码块
   - 现在 Playwright 会正常连接，而不是标记为已连接但客户端为 nil

3. **`mcp_manager.go` - `ConnectToServer` 方法**：
   - 移除了对 Playwright 的特殊处理：`if serverName == "playwright" { return nil }`
   - 现在 Playwright 可以像其他服务器一样被连接

4. **`mcp_manager.go` - `CallTool` 方法**：
   - 移除了对 Playwright 的特殊处理：`if serverName == "playwright" { return m.callPlaywrightToolOnDemand(...) }`
   - 现在 Playwright 工具调用会走正常流程

5. **`mcp_manager.go` - 删除了 `callPlaywrightToolOnDemand` 方法**：
   - 由于不再需要按需连接，删除了这个方法

## 修改后的行为

1. **项目启动时**：所有启用的 MCP 服务器（包括 Playwright）都会启动
2. **工具推荐**：当用户查询包含 "playwright" 时，系统会正确推荐 `playwright/browser_navigate`
3. **工具执行**：Playwright 工具会正常执行，因为服务器已经在运行
4. **项目退出时**：所有 MCP 服务器（包括 Playwright）都会关闭

## 测试验证

1. **工具推荐测试**：创建了 `test_playwright_fix.py` 测试脚本，验证修复后的工具推荐逻辑
2. **连接逻辑测试**：创建了 `test_playwright_connection.py` 测试脚本，分析连接问题
3. **代码编译**：所有修改都成功编译，没有语法错误

## 预期效果

1. 用户查询 "使用playwright查找百度最新的美伊战报。" 时：
   - 系统会推荐 `playwright/browser_navigate`
   - Playwright MCP 服务器已经启动，可以正常执行工具
   - 用户会得到基于浏览器自动化的结果

2. 所有 MCP 服务器都在项目启动时启动，项目退出时关闭，符合用户要求。

## 注意事项

1. **资源消耗**：由于 Playwright 浏览器环境较重，在项目启动时就启动可能会增加资源消耗
2. **启动时间**：Playwright 服务器启动可能需要一些时间，可能会影响项目启动速度
3. **稳定性**：如果 Playwright 服务器启动失败，可能会影响其他 MCP 服务器的启动（但代码中使用了异步连接，可以缓解这个问题）

## 后续优化建议

1. **连接状态监控**：添加对 Playwright 服务器连接状态的监控
2. **健康检查**：定期检查 Playwright 服务器是否正常运行
3. **优雅降级**：如果 Playwright 服务器启动失败，提供优雅的降级方案
4. **配置选项**：允许用户配置是否在启动时启动 Playwright 服务器