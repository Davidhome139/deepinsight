# Backend Dockerfile - Cross-platform compatible (Windows/Linux, amd64/arm64)
# Updated: 2026-04-04 - Added playwright keyword patterns
FROM --platform=$BUILDPLATFORM golang:latest AS builder

# Docker auto-injects these for multi-platform builds
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod tidy
# Build static binary for target platform (auto-detected by Docker)
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o main cmd/main.go

# 使用微软官方 Playwright Docker 镜像
FROM mcr.microsoft.com/playwright:v1.61.1-noble

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
COPY --from=builder /app/certs ./certs
COPY --from=builder /app/internal ./internal
COPY --from=builder /app/scripts ./scripts
COPY --from=builder /app/config/.env ./config/.env
# 复制搜索脚本
COPY scripts/search_meiyi.js ./scripts/search_meiyi.js

# 设置 npm 国内源
RUN npm config set registry https://registry.npmmirror.com

# 安装 Playwright MCP 服务器
RUN npm install -g @playwright/mcp@latest

# 创建符号链接，让 /opt/google/chrome/chrome 指向 Chromium
RUN mkdir -p /opt/google/chrome && \
    ln -sf /ms-playwright/chromium-*/chrome-linux/chrome /opt/google/chrome/chrome

# 创建 Playwright 缓存目录并确保权限正确
RUN mkdir -p /home/pwuser/.cache/ms-playwright && \
    chown -R pwuser:pwuser /home/pwuser/.cache

# 创建日志目录（在切换用户之前）
RUN mkdir -p /var/log/mcp && chown -R pwuser:pwuser /var/log/mcp

# 安装 Xvfb（虚拟帧缓冲）用于无头环境中的浏览器
RUN apt-get update && apt-get install -y xvfb && rm -rf /var/lib/apt/lists/*

# 更改文件所有权给 pwuser 用户
RUN chown -R pwuser:pwuser /app

# 设置环境变量（在容器级别设置，playwright-mcp 需要这些）
ENV PLAYWRIGHT_BROWSERS_PATH=/home/pwuser/.cache/ms-playwright
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=0
ENV PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/opt/google/chrome/chrome
ENV PLAYWRIGHT_BROWSER=chrome
ENV DISPLAY=:99
ENV TZ=Asia/Shanghai

# 切换到非 root 用户（Playwright 镜像默认用户）
USER pwuser

# 运行应用 - 启动 Xvfb 然后运行主程序
CMD xvfb-run --server-args="-screen 0 1280x1024x24 -ac" /app/main