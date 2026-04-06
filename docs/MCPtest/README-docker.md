# MCP Go客户端 - Docker部署指南

这是一个使用Go语言编写的MCP（Model Context Protocol）客户端，用于与Playwright MCP服务器进行交互。本文档介绍如何使用Docker部署和运行此应用。

## 项目结构

```
MCPtest/
├── MCPtest01.go      # 主程序文件
├── go.mod            # Go模块定义
├── go.sum            # Go依赖锁定
├── Dockerfile        # Docker构建文件
├── docker-compose.yml # Docker Compose配置
├── .dockerignore     # Docker忽略文件
└── README-docker.md  # 本文档
```

## 快速开始

### 1. 使用Docker Compose（推荐）

```bash
# 构建并启动容器
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止容器
docker-compose down
```

### 2. 使用Docker直接运行

```bash
# 构建Docker镜像
docker build -t mcp-client .

# 运行容器
docker run --rm -it mcp-client

# 后台运行
docker run -d --name mcp-client mcp-client

# 查看日志
docker logs -f mcp-client
```

## Docker镜像说明

### 多阶段构建
1. **构建阶段**：使用`golang:1.25-alpine`构建Go应用
2. **运行阶段**：使用`alpine:latest`作为基础镜像，包含：
   - Node.js和npm
   - Chromium浏览器（Playwright所需）
   - Playwright MCP服务器
   - 必要的字体和库

### 安全特性
- 使用非root用户（`appuser`）运行应用
- 最小化基础镜像大小
- 只安装必要的运行时依赖

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `PLAYWRIGHT_BROWSERS_PATH` | `/ms-playwright` | Playwright浏览器安装路径 |
| `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD` | `1` | 跳过浏览器下载（使用系统安装的Chromium） |

## 网络配置

### 容器网络模式
- **桥接网络**：默认使用Docker桥接网络
- **主机网络**：如果需要访问主机网络，可以修改`docker-compose.yml`中的`network_mode`

### 端口映射
当前应用使用stdio通信，不需要暴露端口。如果需要HTTP服务，可以修改Dockerfile和docker-compose.yml。

## 资源限制

默认配置：
- 内存限制：1GB
- 内存保留：512MB
- CPU：无限制

可以根据需要调整`docker-compose.yml`中的资源限制。

## 持久化存储

如果需要持久化日志或数据，可以取消注释`docker-compose.yml`中的volumes部分：

```yaml
volumes:
  - ./logs:/app/logs
  - ./data:/app/data
```

## 故障排除

### 1. Docker构建失败
```bash
# 清理构建缓存
docker builder prune

# 重新构建
docker build --no-cache -t mcp-client .
```

### 2. 容器启动失败
```bash
# 查看详细日志
docker-compose logs --tail=100

# 进入容器调试
docker-compose exec mcp-client sh
```

### 3. Playwright浏览器问题
确保容器中有正确的浏览器依赖：
```bash
# 检查容器中的浏览器
docker-compose exec mcp-client ls /ms-playwright
```

## 开发说明

### 本地开发
```bash
# 使用npx运行（开发模式）
go run MCPtest01.go

# 构建本地二进制文件
go build -o mcp-client ./MCPtest01.go
```

### 更新依赖
```bash
# 更新Go依赖
go get -u ./...

# 更新Docker镜像
docker-compose build --no-cache
```

## 性能优化

1. **镜像大小优化**：使用多阶段构建，只复制必要的文件
2. **启动时间优化**：使用Alpine基础镜像，减少层数
3. **内存优化**：设置合理的资源限制

## 安全建议

1. **定期更新**：定期更新基础镜像和依赖
2. **最小权限**：使用非root用户运行容器
3. **网络隔离**：使用Docker网络隔离容器
4. **资源限制**：设置内存和CPU限制防止资源耗尽

## 扩展功能

### 添加HTTP服务
如果需要将MCP客户端作为HTTP服务运行，可以：
1. 在Go应用中添加HTTP服务器
2. 在Dockerfile中暴露端口
3. 在docker-compose.yml中配置端口映射

### 添加监控
可以集成Prometheus监控或添加健康检查端点。

## 许可证

本项目使用MIT许可证。详情请参阅LICENSE文件。