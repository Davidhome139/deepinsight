# MiniMax MCP (Model Context Protocol) 集成文档

## 概述

本文档描述如何在 MCP 架构中集成 MiniMax 模型服务。基于 MiniMax 最新的 ChatCompletion Pro API，提供完整的配置、使用和最佳实践指南。

## MCP 架构简介

Model Context Protocol (MCP) 是一个标准化的协议，用于在应用程序和 AI 模型之间进行通信。它提供了：

- **统一接口**: 标准化的 API 调用方式，支持多个 AI 提供商
- **上下文管理**: 自动管理对话历史和上下文窗口
- **工具集成**: 支持函数调用和工具使用（Function Calling）
- **流式处理**: 实时流式响应，提升用户体验
- **适配器模式**: 通过适配器隔离不同提供商的 API 差异

## MiniMax MCP 配置

### 1. 基础配置 (models.yaml)

在 `backend/config/models.yaml` 中配置 MiniMax：

```yaml
providers:
  minimax:
    name: "MiniMax"
    enabled: true
    api_key: "your_api_key_here"
    group_id: "your_group_id_here"
    base_url: "https://api.minimax.chat/v1"
    endpoint: "/text/chatcompletion_pro"
    auth_type: "bearer"  # Bearer Token 认证
    headers:
      Content-Type: "application/json"
    timeout: 60
    models:
      - "abab6.5-chat"
      - "abab6.5s-chat"
      - "abab6.5t-chat"
      - "abab6.5g-chat"
      - "abab5.5-chat"
      - "abab5.5s-chat"
```

**配置说明**：
- `api_key`: 从 MiniMax 开放平台获取的 API Key
- `group_id`: 账号的 Group ID，用于 URL 参数
- `endpoint`: 推荐使用 `/text/chatcompletion_pro`（最新接口）
- `auth_type`: 设为 `bearer`，表示使用 Bearer Token 认证

### 2. MCP 服务器配置 (mcpservers.yaml)

在 `backend/config/mcpservers.yaml` 中添加 MiniMax MCP 服务：

```yaml
servers:
  minimax:
    name: "MiniMax MCP Server"
    enabled: true
    type: "llm"
    provider: "minimax"
    config:
      default_model: "abab6.5-chat"
      temperature: 0.7
      max_tokens: 4096
      stream: true
      mask_sensitive_info: false
```

## MCP 适配器实现

### MinimaxAdapter 完整实现

MiniMax 适配器负责将标准的 MCP 请求转换为 MiniMax API 格式：

```go
package ai

import (
    "encoding/json"
    "fmt"
    "strings"
    "backend/internal/config"
)

// MinimaxAdapter MiniMax 适配器（OpenAI 兼容格式）
type MinimaxAdapter struct{}

func (a *MinimaxAdapter) BuildRequest(req *ChatRequest) (interface{}, error) {
    messages := make([]map[string]interface{}, 0, len(req.Messages))
    for _, m := range req.Messages {
        message := map[string]interface{}{
            "role":    m.Role,
            "content": m.Content,
        }
        messages = append(messages, message)
    }

    payload := map[string]interface{}{
        "model":    req.Model,
        "messages": messages,
        "stream":   true,
    }

    // 可选参数
    if req.Temperature > 0 {
        payload["temperature"] = req.Temperature
    }
    if req.MaxTokens > 0 {
        payload["max_tokens"] = req.MaxTokens
    }

    // MiniMax 特有参数
    payload["tokens_to_generate"] = 512
    payload["mask_sensitive_info"] = false

    return payload, nil
}

func (a *MinimaxAdapter) BuildHeaders(apiKey string, provider config.ModelProvider) map[string]string {
    headers := map[string]string{
        "Content-Type":  "application/json",
        "Authorization": fmt.Sprintf("Bearer %s", apiKey),
    }

    // 添加配置中的自定义 headers
    for k, v := range provider.Headers {
        headers[k] = v
    }

    return headers
}

func (a *MinimaxAdapter) GetEndpoint(provider config.ModelProvider) string {
    baseURL := strings.TrimRight(provider.BaseURL, "/")
    endpoint := provider.Endpoint
    if endpoint == "" {
        endpoint = "/text/chatcompletion_pro"
    }
    
    // MiniMax 需要在 URL 中添加 GroupId 参数
    if provider.GroupID != "" {
        return fmt.Sprintf("%s%s?GroupId=%s", baseURL, endpoint, provider.GroupID)
    }
    
    return baseURL + endpoint
}

func (a *MinimaxAdapter) ParseStreamResponse(line string) (string, bool, error) {
    if !strings.HasPrefix(line, "data: ") {
        return "", false, nil
    }

    data := strings.TrimPrefix(line, "data: ")
    if data == "[DONE]" {
        return "", true, nil
    }

    // 解析 MiniMax 流式响应（OpenAI 兼容格式）
    var resp struct {
        Choices []struct {
            Delta struct {
                Content string `json:"content"`
            } `json:"delta"`
            FinishReason *string `json:"finish_reason"`
        } `json:"choices"`
    }

    if err := json.Unmarshal([]byte(data), &resp); err != nil {
        return "", false, nil // 忽略解析错误，继续处理下一行
    }

    if len(resp.Choices) > 0 {
        choice := resp.Choices[0]
        // 检查是否结束
        if choice.FinishReason != nil && *choice.FinishReason != "" {
            return "", true, nil
        }
        return choice.Delta.Content, false, nil
    }

    return "", false, nil
}
```

### 适配器注册

在 `adapter.go` 中注册 MiniMax 适配器：

```go
// GetAdapter 根据提供商名称获取对应的适配器
func GetAdapter(providerName string) ProviderAdapter {
    switch providerName {
    case "claude":
        return &ClaudeAdapter{}
    case "aliyun":
        return &AliyunAdapter{}
    case "minimax":
        return &MinimaxAdapter{}  // 使用 OpenAI 兼容适配器
    case "openai", "deepseek", "zhipu", "moonshot", "doubao":
        return &OpenAIAdapter{}
    default:
        return &OpenAIAdapter{}
    }
}

// DetectProviderFromModel 根据模型名称检测提供商
func DetectProviderFromModel(model string) string {
    model = strings.ToLower(model)
    
    if strings.HasPrefix(model, "abab") {
        return "minimax"
    }
    // ... 其他模型检测
    
    return "unknown"
}
```

## MCP 工具集成

### 1. 注册 MiniMax 工具

```go
// 在 MCP 管理器中注册 MiniMax 工具
func RegisterMinimaxTools(mcpManager *MCPManager) {
    mcpManager.RegisterTool("minimax_chat", Tool{
        Name:        "minimax_chat",
        Description: "与 MiniMax 模型进行对话",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "model": map[string]interface{}{
                    "type":        "string",
                    "description": "使用的模型名称",
                    "enum":        []string{"abab6.5-chat", "abab6.5s-chat", "abab5.5-chat", "abab5.5s-chat"},
                },
                "message": map[string]interface{}{
                    "type":        "string",
                    "description": "用户消息",
                },
            },
            "required": []string{"message"},
        },
        Handler: handleMinimaxChat,
    })
}
```

### 2. 工具处理器

```go
func handleMinimaxChat(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    message, ok := params["message"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid message parameter")
    }

    model := "abab6.5-chat"
    if m, ok := params["model"].(string); ok {
        model = m
    }

    // 调用 AI 服务
    req := &ai.ChatRequest{
        Model: model,
        Messages: []models.Message{
            {Role: "user", Content: message},
        },
        Stream: true,
    }

    ch, err := aiService.ChatStream(ctx, req)
    if err != nil {
        return nil, err
    }

    var response string
    for chunk := range ch {
        if !chunk.Done {
            response += chunk.Content
        }
    }

    return map[string]interface{}{
        "content": response,
        "model":   model,
    }, nil
}
```

## 使用示例

### 1. 基本对话

通过统一的 AI Service 调用 MiniMax：

```go
// 创建聊天请求
req := &ai.ChatRequest{
    UserID: userID,
    Model:  "abab6.5-chat",  // 自动检测为 minimax 提供商
    Messages: []models.Message{
        {Role: "user", Content: "你好，请介绍一下 MiniMax"},
    },
    Stream: true,
}

// 调用 AI 服务（自动使用 MiniMax 适配器）
ch, err := aiService.ChatStream(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 处理流式响应
var response string
for chunk := range ch {
    if !chunk.Done {
        response += chunk.Content
        fmt.Print(chunk.Content)  // 实时输出
    }
}
```

### 2. 前端调用示例 (JavaScript)

```javascript
// 基本对话
async function chat(message) {
  const response = await fetch('/api/v1/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      model: 'abab6.5-chat',
      message: message,
      stream: true
    })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    
    const chunk = decoder.decode(value);
    const lines = chunk.split('\n');
    
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.slice(6);
        if (data === '[DONE]') return;
        
        try {
          const parsed = JSON.parse(data);
          if (parsed.content) {
            console.log(parsed.content);
            // 更新 UI
          }
        } catch (e) {
          // 忽略解析错误
        }
      }
    }
  }
}

// 使用
chat('介绍一下人工智能的发展历程');
```

### 3. WebSocket 流式对话

```javascript
const ws = new WebSocket('ws://localhost/api/v1/ws/agent');

ws.onopen = () => {
  ws.send(JSON.stringify({
    action: 'chat',
    model: 'abab6.5-chat',
    message: '请解释量子计算的基本原理',
    stream: true
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch (data.type) {
    case 'chunk':
      // 接收到内容片段
      document.getElementById('output').textContent += data.content;
      break;
    case 'done':
      // 生成完成
      console.log('对话结束');
      break;
    case 'error':
      // 错误处理
      console.error('错误:', data.message);
      break;
  }
};
```

### 4. 上下文管理

```go
// 会话上下文管理器
type ConversationContext struct {
    ConversationID uint
    Messages       []models.Message
    Model          string
    MaxTokens      int
    mu             sync.RWMutex
}

func NewConversationContext(convID uint, model string) *ConversationContext {
    return &ConversationContext{
        ConversationID: convID,
        Model:          model,
        MaxTokens:      4000,
        Messages:       []models.Message{},
    }
}

func (ctx *ConversationContext) AddMessage(role, content string) {
    ctx.mu.Lock()
    defer ctx.mu.Unlock()
    
    ctx.Messages = append(ctx.Messages, models.Message{
        ConversationID: ctx.ConversationID,
        Role:          role,
        Content:       content,
    })
    
    // 自动修剪上下文
    ctx.trimIfNeeded()
}

func (ctx *ConversationContext) trimIfNeeded() {
    // 简单策略：保留最近 20 条消息（10 轮对话）
    if len(ctx.Messages) > 20 {
        // 保留 system 消息
        systemMsgs := []models.Message{}
        for _, msg := range ctx.Messages {
            if msg.Role == "system" {
                systemMsgs = append(systemMsgs, msg)
            }
        }
        
        // 保留最近的对话
        recentMsgs := ctx.Messages[len(ctx.Messages)-20:]
        ctx.Messages = append(systemMsgs, recentMsgs...)
    }
}

func (ctx *ConversationContext) GetMessages() []models.Message {
    ctx.mu.RLock()
    defer ctx.mu.RUnlock()
    return ctx.Messages
}
```

## 性能优化

### 1. HTTP 连接池

复用 HTTP 连接以提高性能：

```go
type MinimaxClient struct {
    httpClient *http.Client
    apiKey     string
    groupID    string
    baseURL    string
}

func NewMinimaxClient(apiKey, groupID, baseURL string) *MinimaxClient {
    return &MinimaxClient{
        httpClient: &http.Client{
            Timeout: 60 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 20,
                MaxConnsPerHost:     20,
                IdleConnTimeout:     90 * time.Second,
                TLSHandshakeTimeout: 10 * time.Second,
            },
        },
        apiKey:  apiKey,
        groupID: groupID,
        baseURL: baseURL,
    }
}
```

### 2. 请求缓存

对于相同的请求，可以缓存响应：

```go
import (
    "crypto/md5"
    "encoding/hex"
    "encoding/json"
    "sync"
    "time"
)

type CachedResponse struct {
    Content   string
    Timestamp time.Time
    TTL       time.Duration
}

type ResponseCache struct {
    cache map[string]CachedResponse
    mu    sync.RWMutex
}

func NewResponseCache() *ResponseCache {
    return &ResponseCache{
        cache: make(map[string]CachedResponse),
    }
}

func (c *ResponseCache) Get(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if cached, ok := c.cache[key]; ok {
        if time.Since(cached.Timestamp) < cached.TTL {
            return cached.Content, true
        }
        // 过期，删除缓存
        delete(c.cache, key)
    }
    return "", false
}

func (c *ResponseCache) Set(key, content string, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.cache[key] = CachedResponse{
        Content:   content,
        Timestamp: time.Now(),
        TTL:       ttl,
    }
}

func generateCacheKey(req *ChatRequest) string {
    data, _ := json.Marshal(req)
    hash := md5.Sum(data)
    return hex.EncodeToString(hash[:])
}
```

### 3. 并发请求控制

使用信号量控制并发数：

```go
import "golang.org/x/sync/semaphore"

type RateLimiter struct {
    sem *semaphore.Weighted
}

func NewRateLimiter(maxConcurrent int64) *RateLimiter {
    return &RateLimiter{
        sem: semaphore.NewWeighted(maxConcurrent),
    }
}

func (r *RateLimiter) Acquire(ctx context.Context) error {
    return r.sem.Acquire(ctx, 1)
}

func (r *RateLimiter) Release() {
    r.sem.Release(1)
}

// 使用示例
limiter := NewRateLimiter(10)  // 最多 10 个并发请求

func (s *UniversalAIService) ChatStreamWithLimit(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error) {
    // 获取信号量
    if err := limiter.Acquire(ctx); err != nil {
        return nil, err
    }
    
    // 请求完成后释放
    defer limiter.Release()
    
    return s.ChatStream(ctx, req)
}
```

## 监控和日志

### 1. 结构化日志

```go
import "go.uber.org/zap"

type MinimaxLogger struct {
    logger *zap.Logger
}

func NewMinimaxLogger() *MinimaxLogger {
    logger, _ := zap.NewProduction()
    return &MinimaxLogger{logger: logger}
}

func (l *MinimaxLogger) LogRequest(req *ChatRequest) {
    l.logger.Info("MiniMax request",
        zap.String("model", req.Model),
        zap.Int("messages", len(req.Messages)),
        zap.Bool("stream", req.Stream),
    )
}

func (l *MinimaxLogger) LogResponse(duration time.Duration, tokens int) {
    l.logger.Info("MiniMax response",
        zap.Duration("latency", duration),
        zap.Int("tokens", tokens),
    )
}

func (l *MinimaxLogger) LogError(err error) {
    l.logger.Error("MiniMax error",
        zap.Error(err),
    )
}
```

### 2. 指标收集

```go
import (
    "sync/atomic"
    "time"
)

type MinimaxMetrics struct {
    TotalRequests    int64
    SuccessRequests  int64
    FailedRequests   int64
    TotalTokens      int64
    TotalLatency     int64
}

func (m *MinimaxMetrics) RecordRequest(success bool, latency time.Duration, tokens int) {
    atomic.AddInt64(&m.TotalRequests, 1)
    
    if success {
        atomic.AddInt64(&m.SuccessRequests, 1)
        atomic.AddInt64(&m.TotalTokens, int64(tokens))
        atomic.AddInt64(&m.TotalLatency, int64(latency))
    } else {
        atomic.AddInt64(&m.FailedRequests, 1)
    }
}

func (m *MinimaxMetrics) GetStats() map[string]interface{} {
    total := atomic.LoadInt64(&m.TotalRequests)
    success := atomic.LoadInt64(&m.SuccessRequests)
    failed := atomic.LoadInt64(&m.FailedRequests)
    tokens := atomic.LoadInt64(&m.TotalTokens)
    latency := atomic.LoadInt64(&m.TotalLatency)
    
    avgLatency := int64(0)
    if success > 0 {
        avgLatency = latency / success
    }
    
    return map[string]interface{}{
        "total_requests":    total,
        "success_requests":  success,
        "failed_requests":   failed,
        "success_rate":      float64(success) / float64(total) * 100,
        "total_tokens":      tokens,
        "avg_latency_ms":    avgLatency / 1000000, // 转换为毫秒
    }
}
```

## 故障处理

### 1. 自动重试策略

```go
func (c *MinimaxClient) ChatWithRetry(ctx context.Context, req *ChatRequest, maxRetries int) (<-chan ChatChunk, error) {
    var lastErr error
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        ch, err := c.ChatStream(ctx, req)
        if err == nil {
            return ch, nil
        }
        
        lastErr = err
        
        // 判断是否应该重试
        if !shouldRetry(err) {
            return nil, err
        }
        
        // 指数退避
        backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
        select {
        case <-time.After(backoff):
            continue
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func shouldRetry(err error) bool {
    // 5xx 错误和 429 错误应该重试
    if httpErr, ok := err.(*HTTPError); ok {
        return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
    }
    return false
}
```

### 2. 熔断器模式

```go
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    failures     int
    lastFailTime time.Time
    state        string // "closed", "open", "half-open"
    mu           sync.Mutex
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        maxFailures:  maxFailures,
        resetTimeout: resetTimeout,
        state:        "closed",
    }
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    // 检查是否可以重置
    if cb.state == "open" && time.Since(cb.lastFailTime) > cb.resetTimeout {
        cb.state = "half-open"
        cb.failures = 0
    }
    
    // 熔断器开启，拒绝请求
    if cb.state == "open" {
        return fmt.Errorf("circuit breaker is open")
    }
    
    // 执行请求
    err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        return err
    }
    
    // 成功，重置熔断器
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

### 3. 降级策略

```go
func (c *MinimaxClient) ChatWithFallback(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error) {
    // 尝试使用首选模型
    ch, err := c.ChatStream(ctx, req)
    if err == nil {
        return ch, nil
    }
    
    log.Printf("Primary model failed, trying fallback: %v", err)
    
    // 降级策略 1: 使用速度优化版本
    if strings.Contains(req.Model, "abab6.5-chat") {
        req.Model = "abab6.5s-chat"
        ch, err = c.ChatStream(ctx, req)
        if err == nil {
            return ch, nil
        }
    }
    
    // 降级策略 2: 使用轻量级模型
    if strings.HasPrefix(req.Model, "abab6.5") {
        req.Model = "abab5.5-chat"
        return c.ChatStream(ctx, req)
    }
    
    return nil, err
}
```

## 最佳实践

### 1. 模型选择

根据不同场景选择合适的模型：

| 场景 | 推荐模型 | 原因 |
|-----|---------|------|
| 复杂推理、代码生成 | abab6.5-chat | 能力最强 |
| 快速响应场景 | abab6.5s-chat | 速度快 |
| 简单对话 | abab6.5g-chat | 成本低 |
| 极简场景 | abab5.5s-chat | 性价比高 |

### 2. 上下文窗口管理

```go
// 智能上下文修剪策略
func (ctx *ConversationContext) SmartTrim() {
    if len(ctx.Messages) <= 10 {
        return
    }
    
    // 1. 保留所有 system 消息
    systemMsgs := []models.Message{}
    otherMsgs := []models.Message{}
    
    for _, msg := range ctx.Messages {
        if msg.Role == "system" {
            systemMsgs = append(systemMsgs, msg)
        } else {
            otherMsgs = append(otherMsgs, msg)
        }
    }
    
    // 2. 保留最近 N 轮对话
    recentCount := 10  // 可根据模型上下文长度调整
    if len(otherMsgs) > recentCount*2 {
        otherMsgs = otherMsgs[len(otherMsgs)-recentCount*2:]
    }
    
    ctx.Messages = append(systemMsgs, otherMsgs...)
}
```

### 3. 流式输出缓冲

对于流式输出，可以实现缓冲机制提升性能：

```go
type StreamBuffer struct {
    buffer   []string
    size     int
    callback func(string)
}

func NewStreamBuffer(size int, callback func(string)) *StreamBuffer {
    return &StreamBuffer{
        buffer:   make([]string, 0, size),
        size:     size,
        callback: callback,
    }
}

func (sb *StreamBuffer) Add(content string) {
    sb.buffer = append(sb.buffer, content)
    if len(sb.buffer) >= sb.size {
        sb.Flush()
    }
}

func (sb *StreamBuffer) Flush() {
    if len(sb.buffer) > 0 {
        sb.callback(strings.Join(sb.buffer, ""))
        sb.buffer = sb.buffer[:0]
    }
}
```

### 4. 错误处理

完善的错误处理流程：

```go
func HandleMinimaxError(err error) error {
    if httpErr, ok := err.(*HTTPError); ok {
        switch httpErr.StatusCode {
        case 401:
            return fmt.Errorf("API Key 无效，请检查配置")
        case 403:
            return fmt.Errorf("配额不足，请充值")
        case 429:
            return fmt.Errorf("请求频率过高，请稍后重试")
        case 500, 502, 503:
            return fmt.Errorf("服务暂时不可用，请稍后重试")
        default:
            return fmt.Errorf("请求失败: %s", httpErr.Message)
        }
    }
    return err
}
```

### 5. 安全性

- **API Key 管理**: 使用环境变量或密钥管理服务
- **敏感信息脱敏**: 启用 `mask_sensitive_info`
- **请求限流**: 实现客户端限流避免超限
- **HTTPS 传输**: 确保使用 HTTPS
- **输入验证**: 验证用户输入，防止注入攻击

### 6. 成本控制

```go
type CostTracker struct {
    inputTokens  int64
    outputTokens int64
    mu           sync.Mutex
}

func (ct *CostTracker) Record(usage Usage) {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    ct.inputTokens += int64(usage.PromptTokens)
    ct.outputTokens += int64(usage.CompletionTokens)
}

func (ct *CostTracker) EstimateCost() float64 {
    // abab6.5-chat 示例价格（请根据实际价格调整）
    inputPrice := 0.03 / 1000   // ¥0.03 per 1K tokens
    outputPrice := 0.06 / 1000  // ¥0.06 per 1K tokens
    
    inputCost := float64(ct.inputTokens) * inputPrice
    outputCost := float64(ct.outputTokens) * outputPrice
    
    return inputCost + outputCost
}
```

## 测试

### 单元测试示例

```go
func TestMinimaxAdapter(t *testing.T) {
    adapter := &MinimaxAdapter{}
    
    // 测试构建请求
    req := &ChatRequest{
        Model: "abab6.5-chat",
        Messages: []models.Message{
            {Role: "user", Content: "Hello"},
        },
        Temperature: 0.7,
        MaxTokens:   1000,
    }
    
    payload, err := adapter.BuildRequest(req)
    assert.NoError(t, err)
    assert.NotNil(t, payload)
    
    // 验证请求格式
    payloadMap := payload.(map[string]interface{})
    assert.Equal(t, "abab6.5-chat", payloadMap["model"])
    assert.Equal(t, true, payloadMap["stream"])
    assert.Equal(t, 0.7, payloadMap["temperature"])
}

func TestMinimaxEndpoint(t *testing.T) {
    adapter := &MinimaxAdapter{}
    provider := config.ModelProvider{
        BaseURL: "https://api.minimax.chat/v1",
        Endpoint: "/text/chatcompletion_pro",
        GroupID: "test-group-id",
    }
    
    endpoint := adapter.GetEndpoint(provider)
    expected := "https://api.minimax.chat/v1/text/chatcompletion_pro?GroupId=test-group-id"
    assert.Equal(t, expected, endpoint)
}
```

### 集成测试

```go
func TestMinimaxIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    apiKey := os.Getenv("MINIMAX_API_KEY")
    groupID := os.Getenv("MINIMAX_GROUP_ID")
    
    if apiKey == "" || groupID == "" {
        t.Skip("MINIMAX_API_KEY or MINIMAX_GROUP_ID not set")
    }
    
    provider := config.ModelProvider{
        APIKey:  apiKey,
        GroupID: groupID,
        BaseURL: "https://api.minimax.chat/v1",
        Endpoint: "/text/chatcompletion_pro",
    }
    
    service := NewUniversalAIService("minimax", provider)
    
    req := &ai.ChatRequest{
        Model: "abab6.5-chat",
        Messages: []models.Message{
            {Role: "user", Content: "你好"},
        },
        Stream: true,
    }
    
    ctx := context.Background()
    ch, err := service.ChatStream(ctx, req)
    assert.NoError(t, err)
    
    var response string
    for chunk := range ch {
        if !chunk.Done {
            response += chunk.Content
        }
    }
    
    assert.NotEmpty(t, response)
    t.Logf("Response: %s", response)
}
```

## 故障排查

### 常见问题

| 问题 | 原因 | 解决方案 |
|-----|------|---------|
| 401 Unauthorized | API Key 错误 | 检查 API Key 是否正确配置 |
| Group ID 错误 | Group ID 未配置或错误 | 检查 URL 中的 GroupId 参数 |
| 超时 | 网络问题或模型响应慢 | 增加 timeout 设置，检查网络 |
| 429 Too Many Requests | 请求频率过高 | 实现限流，降低请求频率 |
| 流式输出中断 | 网络不稳定 | 实现重试机制 |

### 调试技巧

1. **启用详细日志**：
```go
log.SetLevel(log.DebugLevel)
```

2. **打印请求和响应**：
```go
fmt.Printf("Request: %+v\n", req)
fmt.Printf("Response: %+v\n", resp)
```

3. **使用 HTTP 调试工具**：
```bash
# 使用 curl 测试
curl -X POST "https://api.minimax.chat/v1/text/chatcompletion_pro?GroupId=YOUR_GROUP_ID" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"abab6.5-chat","messages":[{"role":"user","content":"你好"}],"stream":false}'
```

## 参考资源

- **MiniMax API 文档**: [./minimaxapi.md](./minimaxapi.md)
- **官方文档**: https://platform.minimaxi.com/docs
- **MCP 协议规范**: https://modelcontextprotocol.io/
- **项目配置**: [../../config/models.yaml](../../config/models.yaml)
- **适配器代码**: [../../internal/services/ai/adapter.go](../../internal/services/ai/adapter.go)

## 更新日志

- **2024-02**: 更新文档以匹配 ChatCompletion Pro API
- **2024-01**: 添加 abab6.5 系列模型支持
- **2024-01**: 完善错误处理和重试机制
- **2023-12**: 初始版本，支持基本对话功能
