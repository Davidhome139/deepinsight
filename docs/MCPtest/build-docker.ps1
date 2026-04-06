# MCP Go客户端Docker构建脚本 (PowerShell版本)
# 用法: .\build-docker.ps1 [选项]
# 选项:
#   -Build    构建Docker镜像
#   -Run      运行Docker容器
#   -Test     测试Docker容器
#   -Clean    清理Docker资源
#   -Help     显示帮助信息

param(
    [switch]$Build,
    [switch]$Run,
    [switch]$Test,
    [switch]$Clean,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

$ImageName = "mcp-client"
$ContainerName = "mcp-client"
$Version = "1.0.0"

# 颜色输出函数
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# 检查Docker是否安装
function Check-Docker {
    Write-Info "检查Docker安装状态..."
    
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Error "Docker未安装，请先安装Docker"
        exit 1
    }
    
    try {
        docker info | Out-Null
        Write-Info "Docker已安装并运行"
    } catch {
        Write-Error "Docker守护进程未运行，请启动Docker"
        exit 1
    }
}

# 构建Docker镜像
function Build-Image {
    Write-Info "开始构建Docker镜像: ${ImageName}:${Version}"
    
    # 清理旧的构建缓存
    Write-Info "清理构建缓存..."
    docker builder prune -f
    
    # 构建镜像
    docker build `
        -t "${ImageName}:${Version}" `
        -t "${ImageName}:latest" `
        .
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Docker镜像构建成功"
        docker images | Select-String $ImageName
    } else {
        Write-Error "Docker镜像构建失败"
        exit 1
    }
}

# 运行Docker容器
function Run-Container {
    Write-Info "启动Docker容器: ${ContainerName}"
    
    # 停止并删除已存在的容器
    if (docker ps -a | Select-String -Quiet $ContainerName) {
        Write-Info "停止并删除已存在的容器..."
        docker stop $ContainerName 2>$null
        docker rm $ContainerName 2>$null
    }
    
    # 运行容器
    docker run `
        --name $ContainerName `
        --rm `
        -it `
        "${ImageName}:latest"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Docker容器运行完成"
    } else {
        Write-Error "Docker容器运行失败"
        exit 1
    }
}

# 测试Docker容器
function Test-Container {
    Write-Info "测试Docker容器..."
    
    # 构建测试镜像
    Write-Info "构建测试镜像..."
    docker build -t "${ImageName}-test:latest" .
    
    # 运行测试容器
    Write-Info "运行测试容器..."
    docker run `
        --name "${ContainerName}-test" `
        --rm `
        -d `
        "${ImageName}-test:latest"
    
    # 等待容器启动
    Start-Sleep -Seconds 5
    
    # 检查容器状态
    if (docker ps | Select-String -Quiet "${ContainerName}-test") {
        Write-Success "测试容器运行正常"
        
        # 查看容器日志
        Write-Info "容器日志:"
        docker logs "${ContainerName}-test" --tail=20
        
        # 停止测试容器
        docker stop "${ContainerName}-test" 2>$null
    } else {
        Write-Error "测试容器启动失败"
        docker logs "${ContainerName}-test" --tail=50
        exit 1
    }
}

# 清理Docker资源
function Clean-Resources {
    Write-Info "清理Docker资源..."
    
    # 停止并删除容器
    if (docker ps -a | Select-String -Quiet $ContainerName) {
        Write-Info "停止容器: ${ContainerName}"
        docker stop $ContainerName 2>$null
        docker rm $ContainerName 2>$null
    }
    
    if (docker ps -a | Select-String -Quiet "${ContainerName}-test") {
        Write-Info "停止容器: ${ContainerName}-test"
        docker stop "${ContainerName}-test" 2>$null
        docker rm "${ContainerName}-test" 2>$null
    }
    
    # 删除镜像
    if (docker images | Select-String -Quiet $ImageName) {
        Write-Info "删除镜像: ${ImageName}"
        docker rmi "${ImageName}:latest" "${ImageName}:${Version}" 2>$null
    }
    
    if (docker images | Select-String -Quiet "${ImageName}-test") {
        Write-Info "删除镜像: ${ImageName}-test"
        docker rmi "${ImageName}-test:latest" 2>$null
    }
    
    # 清理构建缓存
    Write-Info "清理构建缓存..."
    docker builder prune -f
    
    # 清理未使用的镜像
    Write-Info "清理未使用的镜像..."
    docker image prune -f
    
    Write-Success "Docker资源清理完成"
}

# 显示帮助信息
function Show-Help {
    Write-Host "MCP Go客户端Docker构建脚本 (PowerShell版本)" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "用法: .\build-docker.ps1 [选项]"
    Write-Host ""
    Write-Host "选项:"
    Write-Host "  -Build    构建Docker镜像"
    Write-Host "  -Run      运行Docker容器"
    Write-Host "  -Test     测试Docker容器"
    Write-Host "  -Clean    清理Docker资源"
    Write-Host "  -Help     显示帮助信息"
    Write-Host ""
    Write-Host "示例:"
    Write-Host "  .\build-docker.ps1 -Build     构建Docker镜像"
    Write-Host "  .\build-docker.ps1 -Run       运行Docker容器"
    Write-Host "  .\build-docker.ps1 -Test      测试Docker容器"
    Write-Host "  .\build-docker.ps1 -Clean     清理所有Docker资源"
    Write-Host ""
}

# 主函数
function Main {
    # 显示帮助信息
    if ($Help -or (-not $Build -and -not $Run -and -not $Test -and -not $Clean)) {
        Show-Help
        return
    }
    
    # 检查Docker
    Check-Docker
    
    # 执行相应操作
    if ($Build) {
        Build-Image
    }
    
    if ($Run) {
        Run-Container
    }
    
    if ($Test) {
        Test-Container
    }
    
    if ($Clean) {
        Clean-Resources
    }
}

# 执行主函数
Main