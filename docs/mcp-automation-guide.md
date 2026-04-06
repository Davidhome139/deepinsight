# MCP自动化集成系统指南

## 概述

MCP（Model Context Protocol）自动化集成系统是一个智能化的MCP服务器管理框架，能够自动处理MCP包的依赖安装、文档获取、配置生成和热加载。系统支持多种包管理器（NPM、Go、Pip、Docker）并提供完整的API接口。

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    MCP自动化集成系统                         │
├─────────────────────────────────────────────────────────────┤
│  API层                                                      │
│  ├── /api/v1/mcp-automation/add                            │
│  ├── /api/v1/mcp-automation/remove                         │
│  ├── /api/v1/mcp-automation/update                         │
│  ├── /api/v1/mcp-automation/status                         │
│  └── /api/v1/mcp-automation/status/:packageName            │
├─────────────────────────────────────────────────────────────┤
│  自动化协调层                                                │
│  ├── 依赖管理器 (NPM/Go/Pip/Docker)                         │
│  ├── 文档获取器                                             │
│  ├── 配置生成器                                             │
│  └── 热加载管理器                                           │
├─────────────────────────────────────────────────────────────┤
│  配置层                                                     │
│  ├── MCP注册表 (mcp_registry.json)                         │
│  ├── 服务器配置 (mcpservers.json)                          │
│  └── 文档存储 (mcp_docs/)                                  │
└─────────────────────────────────────────────────────────────┘
```

## 快速开始

### 1. 启动系统

系统在应用启动时自动初始化。检查日志确认MCP自动化协调器已启动：

```bash
[Main] Initializing MCP automation coordinator...
[Main] MCP automation coordinator started successfully
```

### 2. 添加MCP包

使用API添加新的MCP包：

```bash
# 添加NPM包
curl -X POST http://localhost:8080/api/v1/mcp-automation/add \
  -H "Content-Type: application/json" \
  -d '{
    "packageName": "@upstash/context7-mcp",
    "packageType": "npm"
  }'

# 添加Go包
curl -X POST http://localhost:8080/api/v1/mcp-automation/add \
  -H "Content-Type: application/json" \
  -d '{
    "packageName": "github.com/example/mcp-server",
    "packageType": "go"
  }'
```

### 3. 检查状态

查看自动化状态：

```bash
# 查看所有包状态
curl http://localhost:8080/api/v1/mcp-automation/status

# 查看特定包状态
curl http://localhost:8080/api/v1/mcp-automation/status/context7
```

## API参考

### 添加MCP包

**端点**: `POST /api/v1/mcp-automation/add`

**请求体**:
```json
{
  "packageName": "string",
  "packageType": "npm|go|pip|docker"
}
```

**响应**:
```json
{
  "success": true,
  "message": "MCP package added successfully",
  "package": {
    "name": "string",
    "type": "string"
  }
}
```

### 移除MCP包

**端点**: `DELETE /api/v1/mcp-automation/remove`

**请求体**:
```json
{
  "packageName": "string"
}
```

### 更新MCP包

**端点**: `PUT /api/v1/mcp-automation/update`

**请求体**:
```json
{
  "packageName": "string",
  "packageType": "npm|go|pip|docker"
}
```

### 获取状态

**端点**: `GET /api/v1/mcp-automation/status`

**响应**:
```json
{
  "running": true,
  "totalPackages": 2,
  "activeJobs": 0,
  "lastSync": "2026-03-28T23:10:00Z",
  "packages": {
    "context7": {
      "packageName": "context7",
      "dependencyType": "npm",
      "installStatus": "installed",
      "configStatus": "generated",
      "connectionStatus": "connected",
      "updateStatus": "pending",
      "lastUpdated": "2026-03-28T23:10:00Z",
      "error": null
    }
  }
}
```

## 自动化流程

当添加新的MCP包时，系统执行以下自动化流程：

1. **依赖检查与安装**
   - 检查包是否已安装
   - 根据包类型执行安装命令
   - 验证安装结果

2. **文档获取**
   - 从包注册表获取元数据
   - 提取描述、主页、仓库等信息
   - 保存文档到本地存储

3. **配置生成**
   - 基于包信息生成MCP服务器配置
   - 设置合理的默认值
   - 添加自动化标记

4. **服务器连接**
   - 更新配置文件
   - 触发MCP管理器重新加载
   - 建立服务器连接

5. **热加载监控**
   - 监控配置文件变化
   - 自动重新加载配置
   - 保持连接状态

## 包类型支持

### NPM包
- **包名格式**: `@scope/package-name` 或 `package-name`
- **安装命令**: `npm install -g <package-name>`
- **测试命令**: `<binary-name> --version`

### Go包
- **包名格式**: `github.com/user/repo` 或 `module/path`
- **安装命令**: `go install <package-name>@latest`
- **测试命令**: `<binary-name> --version`

### Pip包
- **包名格式**: `package-name`
- **安装命令**: `pip install <package-name>`
- **测试命令**: `python -m <module> --version`

### Docker镜像
- **镜像名格式**: `image:tag` 或 `registry/image:tag`
- **安装命令**: `docker pull <image>`
- **测试命令**: `docker run --rm <image> --version`

## 配置管理

### MCP注册表
位置: `config/mcp_registry.json`

```json
{
  "mcp_registry": {
    "servers": {
      "context7": {
        "name": "context7",
        "package_name": "@upstash/context7-mcp",
        "package_type": "npm",
        "install_command": "npm install -g @upstash/context7-mcp",
        "test_command": "context7-mcp --version"
      }
    }
  }
}
```

### 服务器配置
位置: `config/mcpservers.json`

```json
{
  "mcpServers": {
    "context7": {
      "name": "context7",
      "enabled": true,
      "type": "stdio",
      "command": "context7-mcp",
      "args": [],
      "automationInfo": {
        "autoInstall": true,
        "autoUpdate": true,
        "packageManager": "npm",
        "packageName": "@upstash/context7-mcp",
        "installScript": "npm install -g @upstash/context7-mcp",
        "installStatus": "installed",
        "updateStatus": "pending"
      }
    }
  }
}
```

## 脚本工具

### 依赖安装脚本
- **Linux/Mac**: `scripts/install_mcp_deps.sh`
- **Windows**: `scripts/install_mcp_deps.bat`

```bash
# 列出可用服务器
./scripts/install_mcp_deps.sh --list

# 安装所有依赖
./scripts/install_mcp_deps.sh

# 安装特定服务器
./scripts/install_mcp_deps.sh --server context7,playwright

# 按类别安装
./scripts/install_mcp_deps.sh --category documentation
```

### Docker入口脚本
位置: `scripts/docker_entrypoint.sh`

Docker容器启动时自动执行：
1. 安装系统依赖
2. 初始化MCP依赖
3. 启动应用程序

## 故障排除

### 常见问题

1. **依赖安装失败**
   - 检查网络连接
   - 验证包名是否正确
   - 查看安装日志: `/var/log/mcp/install.log`

2. **服务器连接失败**
   - 检查MCP服务器是否正在运行
   - 验证配置文件格式
   - 查看应用日志

3. **热加载不工作**
   - 确认文件监控权限
   - 检查配置文件路径
   - 查看热加载管理器状态

### 日志位置
- **应用日志**: 控制台输出
- **安装日志**: `/var/log/mcp/install.log`
- **入口点日志**: `/var/log/mcp/entrypoint.log`
- **自动化日志**: 应用日志中的`[Automation]`标记

### 调试模式
设置环境变量启用详细日志：

```bash
export MCP_AUTOMATION_DEBUG=true
export MCP_HOTRELOAD_DEBUG=true
```

## 扩展开发

### 添加新的包类型

1. 在`dependency_manager.go`中添加新的管理器
2. 在`documentation_fetcher.go`中添加新的获取器
3. 在`config_generator.go`中添加配置生成逻辑
4. 更新工厂类支持新类型

### 自定义自动化规则

覆盖默认行为：

```go
type CustomAutomationCoordinator struct {
    *DefaultAutomationCoordinator
}

func (c *CustomAutomationCoordinator) automatePackage(packageName string, depType DependencyType) {
    // 自定义自动化逻辑
    c.DefaultAutomationCoordinator.automatePackage(packageName, depType)
}
```

## 性能优化

### 缓存策略
- 文档缓存: 24小时
- 依赖检查缓存: 1小时
- 配置缓存: 内存中

### 并发控制
- 最大并发作业: 3
- 超时设置: 30秒
- 重试机制: 3次

### 资源监控
- 内存使用监控
- CPU使用监控
- 磁盘空间检查

## 安全考虑

### 权限管理
- 依赖安装需要适当权限
- 配置文件访问控制
- API认证和授权

### 输入验证
- 包名格式验证
- 命令注入防护
- 路径遍历防护

### 安全审计
- 安装操作日志
- 配置变更记录
- 错误跟踪和报告

## 版本兼容性

### 系统要求
- Go 1.21+
- Node.js 18+
- Docker 20.10+
- 支持的操作系统: Linux, macOS, Windows

### API版本
- 当前版本: v1
- 向后兼容性: 保证
- 弃用策略: 提前通知

## 贡献指南

### 代码规范
- 遵循Go标准规范
- 添加单元测试
- 编写文档注释

### 测试要求
- 单元测试覆盖率 >80%
- 集成测试关键路径
- 性能基准测试

### 提交流程
1. 创建功能分支
2. 编写测试用例
3. 实现功能
4. 运行测试套件
5. 提交Pull Request

## 许可证

本项目采用MIT许可证。详见LICENSE文件。

## 支持与反馈

- 问题报告: GitHub Issues
- 功能请求: GitHub Discussions
- 文档改进: Pull Requests
- 安全漏洞: 安全报告通道