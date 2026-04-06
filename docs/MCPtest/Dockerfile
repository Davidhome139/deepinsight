# 使用多阶段构建
# 第一阶段：构建Go应用
FROM golang:1.25-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建Go应用
RUN CGO_ENABLED=0 GOOS=linux go build -o mcp-client ./MCPtest01.go

# 第二阶段：运行阶段
FROM mcr.microsoft.com/playwright:v1.48.0-noble

# 设置环境变量
ENV PLAYWRIGHT_BROWSERS_PATH=/home/pwuser/.cache/ms-playwright

# 设置npm国内源
RUN npm config set registry https://registry.npmmirror.com

# 安装Playwright MCP
RUN npm install -g @playwright/mcp@latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/mcp-client .

# 使用现有的pwuser用户（Playwright镜像默认用户）
# 创建符号链接，让/opt/google/chrome/chrome指向Chromium
RUN mkdir -p /opt/google/chrome && \
    ln -sf /ms-playwright/chromium-1140/chrome-linux/chrome /opt/google/chrome/chrome

# 创建Playwright缓存目录并设置权限
RUN mkdir -p /home/pwuser/.cache/ms-playwright && \
    chown -R pwuser:pwuser /home/pwuser/.cache

# 更改文件所有权
RUN chown -R pwuser:pwuser /app

# 切换到非root用户
USER pwuser

# 运行应用
CMD ["./mcp-client"]