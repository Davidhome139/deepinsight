# Backend Dockerfile - Cross-platform compatible (Windows/Linux, amd64/arm64)
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

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
# Install ca-certificates, curl, jq, python, and timezone data
RUN apk add --no-cache ca-certificates curl jq python3 py3-pip tzdata
# Set default timezone
ENV TZ=Asia/Shanghai
EXPOSE 8080
CMD ["./main"]
