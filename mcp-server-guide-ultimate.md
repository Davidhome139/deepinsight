# MCP集成：1小时vs一周的教训

## 核心发现
- **MCPtest**：1小时，几次修改，稳定运行
- **newDouBao**：一周多，数十次修改，频繁错误

## 6条黄金法则

### 1. 代码：直接复制成功案例
```go
// 正确（MCPtest）
stdio := transport.NewStdio("playwright-mcp", nil)

// 错误（newDouBao）  
t = transport.NewStdio(cmd, env, args...) // 过度复杂
```

### 2. 环境：Docker容器级设置
- 环境变量在Dockerfile中设置
- Go代码不传递环境变量
- 验证：`docker exec container env | grep PLAYWRIGHT`

### 3. 健康检查：简单至上
- 只检查基本连接状态
- 避免工具调用检查
- Playwright：假设健康，除非明确错误

### 4. 版本：保持一致
- 使用`mcp-go v0.46.0+`
- 测试与生产环境版本一致

### 5. Docker：关键配置
```dockerfile
RUN ln -sf /ms-playwright/chromium-*/chrome-linux/chrome /opt/google/chrome/chrome
ENV PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/opt/google/chrome/chrome
```

### 6. 调试：科学方法
1. 复制MCPtest代码
2. 创建最小测试案例
3. 一次只改一个变量
4. 记录每次结果

## 快速集成三步法

### 第一步：基础（15分钟）
- 复制MCPtest连接代码
- 验证命令：`which playwright-mcp`
- 测试初始化

### 第二步：诊断（30分钟）
- broken pipe → 简化健康检查
- 超时 → Context7设60秒，加重试
- 工具失败 → 检查Docker配置

### 第三步：优化（必要时）
- 简单重试（最多3次）
- 合理超时（Playwright 15s，Context7 60s）
- 基础健康检查

## 问题速查表
| 问题 | 原因 | 解决 |
|------|------|------|
| broken pipe | 进程退出/连接干扰 | 简化连接，移除复杂逻辑 |
| 初始化超时 | 时间不足 | Context7设60秒，加重试 |
| 工具失败 | 环境/路径问题 | 检查Docker配置 |

## 终极建议
1. **从简单开始**：先验证最简单方案
2. **复制成功**：MCPtest怎么做，你就怎么做
3. **逐步添加**：每步验证，避免大改
4. **科学调试**：找根本原因，非表面症状

**记住**：遇到问题先问"**MCPtest是怎么做的？**"，然后直接复制。

---
*经验总结：简单性战胜复杂性*
*适用：所有MCP服务器集成*