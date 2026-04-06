# Playwright Broken Pipe错误修复总结

## 问题描述
从日志中发现Playwright工具调用失败：

```
2026/04/05 11:56:56 [MCP] Calling tool: server=playwright, tool=browser_navigate, args=map[url:https://example.com]
2026/04/05 11:56:56 [MCP] Tool call failed: server=playwright, tool=browser_navigate, error=transport error: failed to write request: write |1: broken pipe
```

同时URL提取失败：
```
[Chat] DEBUG: User content: 使用playwright通过baidu.com查找美伊最新战报。
[Chat] DEBUG: extractURLFromContent returned: 
[Chat] DEBUG: constructURLFromContent returned: 
[Chat] No URL found in content, using default: https://example.com
```

## 问题分析

### 1. **Playwright连接断开（Broken Pipe）**
- 连接成功7分钟后，服务器连接断开
- 错误信息：`write |1: broken pipe`
- 当前错误处理只匹配"file already closed"，不匹配"broken pipe"

### 2. **URL提取失败**
用户请求："使用playwright通过baidu.com查找美伊最新战报。"
但系统：
1. `extractURLFromContent` 使用正则匹配完整URL（如`https://baidu.com`）
2. `constructURLFromContent` 只匹配少数几个域名，不包括`baidu`
3. 最终使用默认URL：`https://example.com`

## 解决方案

### 1. **修复Broken Pipe错误处理**
**修改文件**: `backend/internal/services/agent/mcp_manager.go`

#### 之前（只匹配特定错误）：
```go
if strings.Contains(err.Error(), "file already closed") || strings.Contains(err.Error(), "read |0") {
    log.Printf("[MCP] Detected 'file already closed' error for server %s, marking for reconnection...", serverName)
}
```

#### 之后（匹配更多连接错误）：
```go
if strings.Contains(err.Error(), "file already closed") || strings.Contains(err.Error(), "read |0") || strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "write |1") {
    log.Printf("[MCP] Detected connection error for server %s (%v), marking for reconnection...", serverName, err)
}
```

### 2. **改进URL提取逻辑**
**修改文件**: `backend/internal/services/chat/chat.go`

#### 扩展域名映射：
```go
// Common domain patterns
domainPatterns := map[string]string{
    "github":        "https://github.com/",
    "mark3labs":     "https://github.com/mark3labs/",
    "google":        "https://www.google.com/",
    "baidu":         "https://www.baidu.com/",  // 新增
    "example":       "https://example.com/",
    "stackoverflow": "https://stackoverflow.com/",
    "wikipedia":     "https://en.wikipedia.org/",
    "youtube":       "https://www.youtube.com/",
    "twitter":       "https://twitter.com/",
    // ... 更多域名
}
```

#### 添加中文域名支持：
```go
// Special handling for Chinese domains
chineseDomains := map[string]string{
    "百度":     "https://www.baidu.com/",
    "淘宝":     "https://www.taobao.com/",
    "京东":     "https://www.jd.com/",
    "腾讯":     "https://www.qq.com/",
    "新浪":     "https://www.sina.com.cn/",
    "网易":     "https://www.163.com/",
    "搜狐":     "https://www.sohu.com/",
    "知乎":     "https://www.zhihu.com/",
    "哔哩哔哩": "https://www.bilibili.com/",
    "豆瓣":     "https://www.douban.com/",
}
```

#### 添加"通过XXX"模式匹配：
```go
// Try to extract domain from "通过XXX" pattern (e.g., "通过baidu.com")
if strings.Contains(content, "通过") {
    re := regexp.MustCompile(`通过([^\s.,，。！？]+)`)
    matches := re.FindStringSubmatch(content)
    if len(matches) > 1 {
        domain := matches[1]
        // Clean up domain
        domain = strings.TrimSpace(domain)
        domain = strings.TrimSuffix(domain, "查找")
        domain = strings.TrimSuffix(domain, "搜索")
        domain = strings.TrimSuffix(domain, "查询")
        
        // Check if it looks like a domain
        if strings.Contains(domain, ".") || len(domain) >= 3 {
            // Add protocol if missing
            if !strings.HasPrefix(strings.ToLower(domain), "http") {
                return "https://" + domain
            }
            return domain
        }
    }
}
```

## 修改的文件

### 1. `backend/internal/services/agent/mcp_manager.go`
- **第1546-1547行**：扩展错误匹配，包含"broken pipe"和"write |1"

### 2. `backend/internal/services/chat/chat.go`
- **第1242-1256行**：扩展域名映射，添加baidu等常见域名
- **第1264-1276行**：添加中文域名支持
- **第1302-1327行**：添加"通过XXX"模式匹配

## 预期效果

### 1. **Broken Pipe错误处理**
```
之前：
2026/04/05 11:56:56 [MCP] Tool call failed: server=playwright, tool=browser_navigate, error=transport error: failed to write request: write |1: broken pipe
（没有重新连接尝试）

之后：
2026/04/05 11:56:56 [MCP] Tool call failed: server=playwright, tool=browser_navigate, error=transport error: failed to write request: write |1: broken pipe
2026/04/05 11:56:56 [MCP] Detected connection error for server playwright (transport error: failed to write request: write |1: broken pipe), marking for reconnection...
2026/04/05 11:56:56 [MCP] Starting asynchronous reconnection for server: playwright
```

### 2. **URL提取改进**
```
之前：
用户请求：使用playwright通过baidu.com查找美伊最新战报。
提取结果：https://example.com（默认）

之后：
用户请求：使用playwright通过baidu.com查找美伊最新战报。
提取结果：https://www.baidu.com/
```

### 3. **用户体验提升**
1. **自动重连**：连接断开时自动尝试重新连接
2. **正确URL**：用户提到的网站被正确识别
3. **错误恢复**：系统更健壮，能处理连接问题

## 测试建议

### 1. **测试Broken Pipe处理**
- 模拟Playwright服务器崩溃
- 验证系统是否自动尝试重新连接
- 检查重连成功后工具是否可用

### 2. **测试URL提取**
- 测试各种URL格式：`baidu.com`、`www.baidu.com`、`https://baidu.com`
- 测试中文域名：`百度`、`淘宝`、`京东`
- 测试"通过XXX"模式：`通过github.com`、`通过baidu.com查找`

### 3. **监控连接稳定性**
- 观察长时间运行后连接是否稳定
- 检查重连机制是否正常工作
- 监控错误率是否下降

## 注意事项

### 1. **错误匹配的扩展性**
- 当前匹配了常见连接错误
- 未来可能需要根据实际情况扩展
- 考虑使用更灵活的错误分类

### 2. **URL提取的局限性**
- 当前实现基于关键词匹配
- 对于复杂或模糊的请求可能不准确
- 可以考虑使用AI辅助的URL提取

### 3. **性能考虑**
- 域名映射表可能变得很大
- 正则表达式匹配可能有性能影响
- 需要平衡准确性和性能

## 总结
通过这次修改，我们解决了两个关键问题：

1. **Broken Pipe错误处理**：现在能正确检测和处理连接断开错误，自动尝试重新连接
2. **URL提取改进**：能正确识别用户提到的网站，特别是中文网站和常见模式

这些改进使得Playwright工具更加稳定和易用，提高了系统的整体可靠性和用户体验。