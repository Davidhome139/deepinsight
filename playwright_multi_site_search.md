# Playwright多站点搜索功能实现总结

## 需求描述
用户希望Playwright在搜索时能同时访问多个搜索引擎（百度和维基百科），而不是只访问一个默认URL。

## 解决方案

### 1. **配置结构改造**
**修改文件**: `backend/internal/config/config.go`

#### 配置结构变化：
```go
// 之前（单URL）：
type PlaywrightConfig struct {
    DefaultURL string `mapstructure:"default_url"`
}

// 之后（多URL）：
type PlaywrightConfig struct {
    DefaultURLs []string `mapstructure:"default_urls"`
}
```

### 2. **配置文件更新**
**修改文件**: `backend/config/config.yaml`

#### 配置内容：
```yaml
tools:
    playwright:
        default_urls:
            - "https://www.baidu.com"
            - "https://en.wikipedia.org"
```

### 3. **逻辑实现**
**修改文件**: `backend/internal/services/chat/chat.go`

#### 核心逻辑：
1. **URL提取策略**：
   - 如果用户指定了具体URL，只使用该URL
   - 如果用户没有指定URL，使用配置中的多个默认URL

2. **多URL处理**：
   - 当有多个URL时，调用 `handleMultipleURLs` 方法
   - 顺序访问每个URL
   - 合并所有结果

3. **向后兼容**：
   - 保留 `getPlaywrightDefaultURL` 方法用于单URL场景
   - 新增 `getPlaywrightDefaultURLs` 方法用于多URL场景

### 4. **新增方法**

#### `getPlaywrightDefaultURLs()`:
```go
func (s *chatService) getPlaywrightDefaultURLs() []string {
    if config.GlobalConfig != nil && len(config.GlobalConfig.Tools.Playwright.DefaultURLs) > 0 {
        return config.GlobalConfig.Tools.Playwright.DefaultURLs
    }
    return []string{"https://www.baidu.com"}
}
```

#### `handleMultipleURLs()`:
```go
func (s *chatService) handleMultipleURLs(serverName, toolName string, urls []string, userContent string) string {
    // 顺序访问每个URL
    for i, url := range urls {
        args := map[string]interface{}{"url": url}
        result, err := s.mcpManager.CallTool(serverName, toolName, args)
        // 处理结果...
    }
    // 合并所有结果...
}
```

#### `extractTextFromHTML()`:
```go
func extractTextFromHTML(html string) string {
    // 从HTML中提取可读文本
    // 移除脚本和样式标签
    // 移除HTML标签
    // 整理空白字符
}
```

## 工作流程

### 1. **用户请求示例**：
```
"使用playwright查找美伊最新战报"
```

### 2. **处理流程**：
```
1. 提取URL：用户没有指定具体URL → url = ""
2. 获取默认URLs：从配置读取 → ["https://www.baidu.com", "https://en.wikipedia.org"]
3. 检测多URL：len(urls) > 1 → 调用handleMultipleURLs
4. 顺序访问：
   - 导航到百度，搜索"美伊最新战报"
   - 导航到维基百科，搜索"美伊最新战报"
5. 合并结果：将两个站点的结果合并返回
```

### 3. **日志输出**：
```
[Chat] No specific URL found, using default URLs from config: [https://www.baidu.com https://en.wikipedia.org]
[Chat] Handling multiple URLs: [https://www.baidu.com https://en.wikipedia.org]
[Chat] Navigating to URL 1/2: https://www.baidu.com
[Chat] Successfully navigated to https://www.baidu.com, content length: 1500
[Chat] Navigating to URL 2/2: https://en.wikipedia.org
[Chat] Successfully navigated to https://en.wikipedia.org, content length: 1800
```

## 结果格式

### 多站点搜索结果：
```
# Multi-Site Search Results

**Search Query:** 美伊最新战报
**Searched Sites:** 2

## Site 1/2
🌐 **https://www.baidu.com**
百度搜索结果：美伊战争最新进展...
[页面内容摘要]

---

## Site 2/2
🌐 **https://en.wikipedia.org**
Wikipedia: Iran–United States relations...
[页面内容摘要]

**Summary:** Searched 2 sites for: 美伊最新战报
```

## 配置灵活性

### 1. **添加更多搜索引擎**：
```yaml
tools:
    playwright:
        default_urls:
            - "https://www.baidu.com"
            - "https://en.wikipedia.org"
            - "https://www.google.com"
            - "https://www.bing.com"
```

### 2. **特定场景配置**：
```yaml
# 中文搜索
tools:
    playwright:
        default_urls:
            - "https://www.baidu.com"
            - "https://www.sogou.com"

# 学术搜索
tools:
    playwright:
        default_urls:
            - "https://scholar.google.com"
            - "https://www.ncbi.nlm.nih.gov"
```

### 3. **环境特定配置**：
```bash
# 开发环境 - 测试站点
export APP_TOOLS_PLAYWRIGHT_DEFAULT_URLS='["https://example.com","https://test.com"]'

# 生产环境 - 真实搜索引擎
export APP_TOOLS_PLAYWRIGHT_DEFAULT_URLS='["https://www.baidu.com","https://en.wikipedia.org"]'
```

## 性能考虑

### 1. **顺序访问**：
- 优点：简单可靠，避免并发问题
- 缺点：总时间 = 各站点时间之和

### 2. **内容截断**：
- 每个站点结果限制在1000字符内
- HTML内容提取后限制在2000字符内
- 避免返回过多数据

### 3. **错误处理**：
- 单个站点失败不影响其他站点
- 记录失败原因，继续处理其他URL
- 最终结果包含成功和失败信息

## 测试建议

### 1. **配置测试**：
- 测试空配置时的回退行为
- 测试单URL和多URL配置
- 测试无效URL的处理

### 2. **功能测试**：
- 测试用户指定URL时的行为
- 测试多站点搜索的完整流程
- 测试单个站点失败时的处理

### 3. **性能测试**：
- 测试多个站点的总耗时
- 测试大页面内容的处理
- 测试并发访问（如果需要）

### 4. **结果验证**：
- 验证结果合并的正确性
- 验证HTML提取的准确性
- 验证错误信息的清晰度

## 扩展可能性

### 1. **并行访问**：
```go
// 未来可以改为并行访问
func (s *chatService) handleMultipleURLsParallel(urls []string) {
    var wg sync.WaitGroup
    results := make(chan string, len(urls))
    
    for _, url := range urls {
        wg.Add(1)
        go func(u string) {
            defer wg.Done()
            result := navigateToURL(u)
            results <- result
        }(url)
    }
    
    wg.Wait()
    close(results)
}
```

### 2. **智能站点选择**：
- 根据查询语言自动选择搜索引擎
- 根据查询类型选择专业站点
- 根据历史成功率动态调整

### 3. **结果分析**：
- 自动对比不同站点的结果
- 提取关键信息生成摘要
- 识别矛盾或补充信息

## 总结

通过这次改造，我们实现了：

1. **多站点搜索**：Playwright可以同时访问多个搜索引擎
2. **配置驱动**：搜索站点完全由配置文件控制
3. **灵活扩展**：可以轻松添加更多搜索引擎
4. **智能处理**：根据用户输入决定搜索策略
5. **结果整合**：将多个站点的结果合并返回

这个功能使得搜索更加全面和有用，用户可以通过一次请求获取多个来源的信息，提高了信息获取的效率和覆盖面。