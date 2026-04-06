# 开发环境指南

## 快速开始

### Windows
```batch
# 启动开发环境
scripts\dev.bat up

# 查看日志
scripts\dev.bat logs backend

# 停止服务
scripts\dev.bat down
```

### Linux/Mac
```bash
# 添加执行权限（首次使用）
chmod +x scripts/dev.sh

# 启动开发环境
./scripts/dev.sh up

# 查看日志
./scripts/dev.sh logs backend

# 停止服务
./scripts/dev.sh down
```

## 服务访问

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端 | http://localhost:5173 | Vite 开发服务器 |
| 后端 API | http://localhost:8080 | Go + Gin |
| 数据库 | localhost:5432 | PostgreSQL + pgvector |
| Redis | localhost:6379 | 缓存服务 |

## 开发命令

### Windows (scripts\dev.bat)

| 命令 | 说明 |
|------|------|
| `up` | 启动所有开发服务 |
| `down` | 停止所有服务 |
| `restart` | 重启所有服务 |
| `logs [service]` | 查看日志，service可选：backend/frontend/db/redis |
| `build` | 重新构建并启动 |
| `clean` | 清理所有数据和容器 |
| `shell-backend` | 进入后端容器 shell |
| `shell-frontend` | 进入前端容器 shell |
| `db` | 连接 PostgreSQL 数据库 |

### Linux/Mac (./scripts/dev.sh)

| 命令 | 说明 |
|------|------|
| `up` | 启动所有开发服务 |
| `down` | 停止所有服务 |
| `restart` | 重启所有服务 |
| `logs [service]` | 查看日志 |
| `build` | 重新构建并启动 |
| `clean` | 清理所有数据和容器 |
| `shell-backend` | 进入后端容器 shell |
| `shell-frontend` | 进入前端容器 shell |
| `db` | 连接 PostgreSQL 数据库 |
| `status` | 查看服务状态 |

## 热重载 (Hot Reload)

### 后端 (Go)
- 使用 [Air](https://github.com/cosmtrek/air) 实现热重载
- 修改 `.go` 文件后自动重新编译运行
- 配置文件: `backend/.air.toml`

### 前端 (Vue)
- 使用 Vite 内置热模块替换 (HMR)
- 修改 `.vue` 或 `.ts` 文件后页面自动更新
- 无需手动刷新

## 代码调试

### 后端调试
```bash
# 进入后端容器
scripts\dev.bat shell-backend
# 或
./scripts/dev.sh shell-backend

# 在容器内使用 dlv 调试
dlv debug cmd/main.go
```

### 前端调试
- 使用浏览器开发者工具 (F12)
- Vue DevTools 扩展
- 源码映射已启用

## 数据持久化

开发环境使用 Docker Volumes 持久化数据：

| Volume | 说明 |
|--------|------|
| `postgres_data_dev` | 数据库数据 |
| `redis_data_dev` | Redis 数据 |
| `go_mod_cache` | Go 模块缓存 |
| `node_modules` | Node 模块缓存 |

**注意**: 运行 `clean` 命令会清除所有数据！

## 配置文件

### 后端配置
- 路径: `backend/config/config.yaml`
- 修改后会自动生效（部分配置需重启）

### 前端配置
- Vite 配置: `frontend/vite.config.ts`
- 环境变量: `frontend/.env` (如需要)

## 常见问题

### 1. 端口冲突
如果 5173、8080、5432、6379 端口被占用，修改 `docker-compose.dev.yaml`：

```yaml
ports:
  - "8081:8080"  # 使用 8081 代替 8080
```

### 2. 内存不足
Docker Desktop 默认内存可能不足，建议分配至少 4GB 内存。

### 3. 构建失败
```bash
# 清理并重新构建
scripts\dev.bat clean
scripts\dev.bat build
```

### 4. 数据库连接失败
```bash
# 检查数据库状态
scripts\dev.bat logs db

# 手动连接数据库
scripts\dev.bat db
```

### 5. 热重载不生效
```bash
# 重启服务
scripts\dev.bat restart
```

## 与生产环境的区别

| 特性 | 开发环境 | 生产环境 |
|------|----------|----------|
| 热重载 | ✅ 支持 | ❌ 不支持 |
| 调试工具 | ✅ dlv | ❌ 无 |
| SSL/HTTPS | ❌ HTTP | ✅ HTTPS |
| Nginx | ❌ Vite DevServer | ✅ Nginx |
| 代码优化 | ❌ 未优化 | ✅ 已优化 |
| 日志级别 | debug | info |

## 切换到生产模式

```bash
# 停止开发环境
docker-compose -f docker-compose.dev.yaml down

# 启动生产环境
docker-compose up --build
```

## 目录挂载

开发环境挂载以下目录实现实时同步：

```
./backend:/app          # 后端代码
./frontend:/app         # 前端代码
./backend/config:/app/config    # 配置文件
./backend/plans:/app/plans      # Agent 执行计划
```

## 网络架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Host Machine                          │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐ │
│  │ localhost    │  │ localhost    │  │ localhost          │ │
│  │ :5173        │  │ :8080        │  │ :5432     :6379    │ │
│  └──────┬───────┘  └──────┬───────┘  └─────────┬──────────┘ │
│         │                 │                    │            │
│  ┌──────┴─────────────────┴────────────────────┴──────────┐ │
│  │              Docker Compose Network                      │ │
│  │  ┌──────────┐  ┌──────────┐  ┌────────┐  ┌────────┐    │ │
│  │  │ frontend │  │ backend  │  │   db   │  │ redis  │    │ │
│  │  │ :5173    │  │ :8080    │  │ :5432  │  │ :6379  │    │ │
│  │  └──────────┘  └──────────┘  └────────┘  └────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```
