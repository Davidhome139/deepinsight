# Playwright默认URL配置化改造总结

## 问题描述
之前Playwright的默认URL是硬编码在代码中的：
```go
url = "https://www.baidu.com"
```

根据要求，需要将配置移到配置文件中，实现配置化。

## 解决方案

### 1. **扩展配置结构**
**修改文件**: `backend/internal/config/config.go`

#### 添加新的配置结构：
```go
type ToolsConfig struct {
    Playwright PlaywrightConfig `mapstructure:"playwright"`
}

type PlaywrightConfig struct {
    DefaultURL string `mapstructure:"default_url"`
}
```

#### 更新Config结构体：
```go
type Config struct {
    // ... 其他配置
    Tools    ToolsConfig    `mapstructure:"tools"`  // 新增
    OS       string         `mapstructure:"os"`
}
```

#### 更新配置同步：
```go
func syncConfigToViper() {
    // ... 其他配置同步
    viper.Set("tools", GlobalConfig.Tools)  // 新增
}
```

### 2. **更新配置文件**
**修改文件**: `backend/config/config.yaml`

#### 添加tools配置节：
```yaml
tools:
    playwright:
        default_url: "https://www.baidu.com"
```

### 3. **修改代码使用配置**
**修改文件**: `backend/internal/services/chat/chat.go`

#### 修改默认URL获取逻辑：
```go
// 之前（硬编码）：
if url == "" {
    url = "https://www.baidu.com"
    fmt.Printf("[Chat] No URL found in content, using default: %s\n", url)
}

// 之后（从配置获取）：
if url == "" {
    defaultURL := s.getPlaywrightDefaultURL()
    url = defaultURL
    fmt.Printf("[Chat] No URL found in content, using default from config: %s\n", url)
}
```

#### 添加配置获取方法：
```go
func (s *chatService) getPlaywrightDefaultURL() string {
    // Try to get from config
    if config.GlobalConfig != nil && config.GlobalConfig.Tools.Playwright.DefaultURL != "" {
        return config.GlobalConfig.Tools.Playwright.DefaultURL
    }
    
    // Fallback to hardcoded default
    return "https://www.baidu.com"
}
```

## 修改的文件

### 1. `backend/internal/config/config.go`
- **第16行**：在Config结构体中添加Tools字段
- **第112-119行**：添加ToolsConfig和PlaywrightConfig结构体定义
- **第209行**：在syncConfigToViper函数中添加tools配置同步

### 2. `backend/config/config.yaml`
- **第34-37行**：添加tools配置节，包含playwright的default_url

### 3. `backend/internal/services/chat/chat.go`
- **第931-934行**：修改默认URL获取逻辑，使用配置
- **第2164-2175行**：添加getPlaywrightDefaultURL方法

## 配置优势

### 1. **灵活性**
- 可以随时修改默认URL，无需重新编译代码
- 支持不同环境使用不同的默认URL

### 2. **可维护性**
- 配置集中管理，便于维护
- 配置变更有明确的记录

### 3. **向后兼容**
- 如果配置不存在或为空，使用硬编码的默认值
- 确保系统始终有可用的默认URL

### 4. **扩展性**
- 可以轻松添加其他工具的配置
- 支持复杂的配置结构

## 使用示例

### 1. **修改默认URL**
在 `config.yaml` 中修改：
```yaml
tools:
    playwright:
        default_url: "https://www.google.com"  # 改为Google
```

### 2. **添加其他工具配置**
未来可以扩展：
```yaml
tools:
    playwright:
        default_url: "https://www.baidu.com"
        timeout: 30
        headless: true
    context7:
        default_library: "next.js"
        cache_ttl: 3600
```

### 3. **环境特定配置**
可以通过环境变量覆盖：
```bash
export APP_TOOLS_PLAYWRIGHT_DEFAULT_URL="https://www.example.com"
```

## 测试建议

### 1. **配置加载测试**
- 验证配置是否正确加载
- 检查默认URL是否正确获取

### 2. **回退机制测试**
- 测试当配置不存在时的回退行为
- 验证硬编码默认值是否有效

### 3. **配置更新测试**
- 修改配置文件后，验证是否生效
- 测试热重载（如果支持）

### 4. **错误处理测试**
- 测试无效URL的处理
- 验证错误日志是否清晰

## 注意事项

### 1. **配置验证**
- 确保URL格式有效
- 考虑添加URL验证逻辑

### 2. **性能考虑**
- 配置获取是轻量级的
- 避免频繁的配置读取

### 3. **安全性**
- 配置文件中不包含敏感信息
- URL应该是公开可访问的

### 4. **文档同步**
- 更新相关文档说明配置选项
- 提供配置示例

## 总结
通过这次改造，我们实现了：

1. **配置化**：将硬编码的默认URL移到配置文件中
2. **灵活性**：可以轻松修改默认行为
3. **可维护性**：配置集中管理，便于维护
4. **向后兼容**：确保现有功能不受影响

这个改进使得系统更加灵活和可配置，为未来的功能扩展奠定了基础。