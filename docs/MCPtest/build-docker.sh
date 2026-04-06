#!/bin/bash

# MCP Go客户端Docker构建脚本
# 用法: ./build-docker.sh [选项]
# 选项:
#   -b, --build    构建Docker镜像
#   -r, --run      运行Docker容器
#   -t, --test     测试Docker容器
#   -c, --clean    清理Docker资源
#   -h, --help     显示帮助信息

set -e

IMAGE_NAME="mcp-client"
CONTAINER_NAME="mcp-client"
VERSION="1.0.0"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装，请先安装Docker"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker守护进程未运行，请启动Docker"
        exit 1
    fi
    
    print_info "Docker已安装并运行"
}

# 构建Docker镜像
build_image() {
    print_info "开始构建Docker镜像: ${IMAGE_NAME}:${VERSION}"
    
    # 清理旧的构建缓存
    print_info "清理构建缓存..."
    docker builder prune -f
    
    # 构建镜像
    docker build \
        -t ${IMAGE_NAME}:${VERSION} \
        -t ${IMAGE_NAME}:latest \
        .
    
    if [ $? -eq 0 ]; then
        print_success "Docker镜像构建成功"
        docker images | grep ${IMAGE_NAME}
    else
        print_error "Docker镜像构建失败"
        exit 1
    fi
}

# 运行Docker容器
run_container() {
    print_info "启动Docker容器: ${CONTAINER_NAME}"
    
    # 停止并删除已存在的容器
    if docker ps -a | grep -q ${CONTAINER_NAME}; then
        print_info "停止并删除已存在的容器..."
        docker stop ${CONTAINER_NAME} 2>/dev/null || true
        docker rm ${CONTAINER_NAME} 2>/dev/null || true
    fi
    
    # 运行容器
    docker run \
        --name ${CONTAINER_NAME} \
        --rm \
        -it \
        ${IMAGE_NAME}:latest
    
    if [ $? -eq 0 ]; then
        print_success "Docker容器运行完成"
    else
        print_error "Docker容器运行失败"
        exit 1
    fi
}

# 测试Docker容器
test_container() {
    print_info "测试Docker容器..."
    
    # 构建测试镜像
    print_info "构建测试镜像..."
    docker build -t ${IMAGE_NAME}-test:latest .
    
    # 运行测试容器
    print_info "运行测试容器..."
    docker run \
        --name ${CONTAINER_NAME}-test \
        --rm \
        -d \
        ${IMAGE_NAME}-test:latest
    
    # 等待容器启动
    sleep 5
    
    # 检查容器状态
    if docker ps | grep -q ${CONTAINER_NAME}-test; then
        print_success "测试容器运行正常"
        
        # 查看容器日志
        print_info "容器日志:"
        docker logs ${CONTAINER_NAME}-test --tail=20
        
        # 停止测试容器
        docker stop ${CONTAINER_NAME}-test 2>/dev/null || true
    else
        print_error "测试容器启动失败"
        docker logs ${CONTAINER_NAME}-test --tail=50
        exit 1
    fi
}

# 清理Docker资源
clean_resources() {
    print_info "清理Docker资源..."
    
    # 停止并删除容器
    if docker ps -a | grep -q ${CONTAINER_NAME}; then
        print_info "停止容器: ${CONTAINER_NAME}"
        docker stop ${CONTAINER_NAME} 2>/dev/null || true
        docker rm ${CONTAINER_NAME} 2>/dev/null || true
    fi
    
    if docker ps -a | grep -q "${CONTAINER_NAME}-test"; then
        print_info "停止容器: ${CONTAINER_NAME}-test"
        docker stop ${CONTAINER_NAME}-test 2>/dev/null || true
        docker rm ${CONTAINER_NAME}-test 2>/dev/null || true
    fi
    
    # 删除镜像
    if docker images | grep -q "${IMAGE_NAME}"; then
        print_info "删除镜像: ${IMAGE_NAME}"
        docker rmi ${IMAGE_NAME}:latest ${IMAGE_NAME}:${VERSION} 2>/dev/null || true
    fi
    
    if docker images | grep -q "${IMAGE_NAME}-test"; then
        print_info "删除镜像: ${IMAGE_NAME}-test"
        docker rmi ${IMAGE_NAME}-test:latest 2>/dev/null || true
    fi
    
    # 清理构建缓存
    print_info "清理构建缓存..."
    docker builder prune -f
    
    # 清理未使用的镜像
    print_info "清理未使用的镜像..."
    docker image prune -f
    
    print_success "Docker资源清理完成"
}

# 显示帮助信息
show_help() {
    echo "MCP Go客户端Docker构建脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -b, --build    构建Docker镜像"
    echo "  -r, --run      运行Docker容器"
    echo "  -t, --test     测试Docker容器"
    echo "  -c, --clean    清理Docker资源"
    echo "  -h, --help     显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 --build     构建Docker镜像"
    echo "  $0 --run       运行Docker容器"
    echo "  $0 --test      测试Docker容器"
    echo "  $0 --clean     清理所有Docker资源"
    echo ""
}

# 主函数
main() {
    # 检查参数
    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi
    
    # 检查Docker
    check_docker
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--build)
                build_image
                shift
                ;;
            -r|--run)
                run_container
                shift
                ;;
            -t|--test)
                test_container
                shift
                ;;
            -c|--clean)
                clean_resources
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 执行主函数
main "$@"