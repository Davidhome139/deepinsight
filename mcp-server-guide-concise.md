# MCP服务器快速集成指南

## 核心原则：简单优于复杂

**教训**：MCPtest 1小时成功，newDouBao一周折腾。根本差异在于复杂度。

## 6条关键经验

### 1. 连接方式：直接复制成功案例
```go
// 正确：简单直接（MCPtest方式）
stdio := transport.NewStdio("playwright-mcp", nil)

// 错误：过度复杂（newDouBao方式）
t = transport.NewStdio(cmd, env, args...) // 添加各种参数
```

### 2. 环境变量：容器级别设置
- **正确**：Dockerfile中设置环境变量
- **错误**：Go代码中传递环境变量
- **验证**：`docker exec container env | grep PLAYWRIGHT`

### 3. 健康检查：越简单越好
- **问题**：复杂健康检查导致broken pipe
- **解决**：只检查基本连接状态，避免工具调用
- **Playwright特殊处理**：假设连接健康，除非明确错误

### 4. 版本一致性
- MCPtest：`mcp-go v0.46.0`
- newDouBao：`mcp-go v0.43.2`
- **建议**：使用最新稳定版本

### 5. Docker配置关键点
```dockerfile
# Chromium符号链接（必须）
RUN mkdir -p /opt/google/chrome && \
    ln -sf /ms-playwright/chromium-*/chrome-linux/chrome /opt/google/chrome/chrome

# 环境变量（必须）
ENV PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/opt/google/chrome/chrome
```

### 6. 调试方法论
1. **对比成功案例**：直接复制MCPtest代码
2. **简化复现**：创建最小测试案例
3. **一次一变量**：避免同时修改多处
4. **科学记录**：记录每次修改和结果

## 快速检查清单

### 第一阶段：基础集成（15分钟）
- [ ] 复制MCPtest连接代码
- [ ] 验证MCP命令：`which playwright-mcp`
- [ ] 检查Docker环境变量
- [ ] 测试初始化：应发现工具列表

### 第二阶段：问题诊断（30分钟）
- [ ] 检查broken pipe：简化健康检查
- [ ] 检查超时：Context7需要60秒
- [ ] 检查版本：使用v0.46.0+
- [ ] 检查符号链接：`ls -l /opt/google/chrome/chrome`

### 第三阶段：优化（仅在必要时）
- [ ] 添加简单重试（最多3次）
- [ ] 设置合理超时（Playwright 15s，Context7 60s）
- [ ] 添加基础健康检查（不调用工具）

## 常见问题速查

### broken pipe错误
**原因**：进程退出或连接干扰
**解决**：简化连接方式，移除复杂健康检查

### 初始化超时
**原因**：超时时间不足
**解决**：Context7设为60秒，添加重试机制

### 工具发现失败
**原因**：环境变量或符号链接问题
**解决**：验证Docker配置，检查Chromium路径

## 心态建议

1. **从简单开始**：先验证最简单方案
2. **复制成功**：不要重新发明轮子
3. **逐步添加**：每步都验证，避免大改动
4. **科学调试**：理解根本原因，而非表面症状

## 总结

**MCPtest的成功公式**：
```
简单连接 + 正确环境 + 最小管理 = 稳定运行
```

**记住**：当遇到问题时，先问"**MCPtest是怎么做的？**"，然后直接复制。

---
*文档字数：约800字*
*核心经验：简单性战胜复杂性*