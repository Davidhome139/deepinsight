@echo off
REM MCP Dependency Installation Script for Windows
REM This script installs required dependencies for MCP servers

echo ==========================================
echo MCP Dependency Installation Script
echo ==========================================

REM Configuration
set CONFIG_DIR=%~dp0..\config
set REGISTRY_FILE=%CONFIG_DIR%\mcp_registry.json
set LOG_FILE=%TEMP%\mcp_install_%DATE:~-4%%DATE:~4,2%%DATE:~7,2%_%TIME:~0,2%%TIME:~3,2%%TIME:~6,2%.log

REM Logging functions
:log_info
echo [INFO] %DATE% %TIME%: %~1 >> "%LOG_FILE%"
echo [INFO] %~1
goto :eof

:log_success
echo [SUCCESS] %DATE% %TIME%: %~1 >> "%LOG_FILE%"
echo [SUCCESS] %~1
goto :eof

:log_warning
echo [WARNING] %DATE% %TIME%: %~1 >> "%LOG_FILE%"
echo [WARNING] %~1
goto :eof

:log_error
echo [ERROR] %DATE% %TIME%: %~1 >> "%LOG_FILE%"
echo [ERROR] %~1
goto :eof

REM Check if jq is installed
:check_jq
where jq >nul 2>nul
if %ERRORLEVEL% neq 0 (
    call :log_error "jq is not installed. Please install jq first."
    call :log_info "Download from: https://stedolan.github.io/jq/download/"
    exit /b 1
)
call :log_success "jq is installed"
goto :eof

REM Check if registry file exists
:check_registry
if not exist "%REGISTRY_FILE%" (
    call :log_error "Registry file not found: %REGISTRY_FILE%"
    exit /b 1
)
call :log_success "Registry file found: %REGISTRY_FILE%"
goto :eof

REM Install NPM package
:install_npm_package
setlocal
set package_name=%~1
set install_cmd=%~2
set test_cmd=%~3

call :log_info "Installing NPM package: %package_name%"

REM Check if package is already installed
for /f "tokens=1" %%a in ("%test_cmd%") do set test_cmd_first=%%a
where %test_cmd_first% >nul 2>nul
if %ERRORLEVEL% equ 0 (
    call :log_success "Package already installed: %package_name%"
    endlocal
    exit /b 0
)

REM Install package
call %install_cmd%
if %ERRORLEVEL% equ 0 (
    call :log_success "Successfully installed: %package_name%"
    
    REM Test installation
    call %test_cmd% >nul 2>nul
    if %ERRORLEVEL% equ 0 (
        call :log_success "Package test passed: %package_name%"
    ) else (
        call :log_warning "Package installed but test failed: %package_name%"
    )
) else (
    call :log_error "Failed to install: %package_name%"
    endlocal
    exit /b 1
)

endlocal
goto :eof

REM Install Go package
:install_go_package
setlocal
set package_name=%~1
set install_cmd=%~2
set test_cmd=%~3

call :log_info "Installing Go package: %package_name%"

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    call :log_error "Go is not installed. Skipping Go packages."
    endlocal
    exit /b 1
)

REM Install package
call %install_cmd%
if %ERRORLEVEL% equ 0 (
    call :log_success "Successfully installed: %package_name%"
    
    REM Test installation
    call %test_cmd% >nul 2>nul
    if %ERRORLEVEL% equ 0 (
        call :log_success "Package test passed: %package_name%"
    ) else (
        call :log_warning "Package installed but test failed: %package_name%"
    )
) else (
    call :log_error "Failed to install: %package_name%"
    endlocal
    exit /b 1
)

endlocal
goto :eof

REM Install Pip package
:install_pip_package
setlocal
set package_name=%~1
set install_cmd=%~2
set test_cmd=%~3

call :log_info "Installing Pip package: %package_name%"

REM Check if pip is installed
where pip3 >nul 2>nul
if %ERRORLEVEL% equ 0 (
    set pip_cmd=pip3
) else (
    where pip >nul 2>nul
    if %ERRORLEVEL% equ 0 (
        set pip_cmd=pip
    ) else (
        call :log_error "pip/pip3 is not installed. Skipping Python packages."
        endlocal
        exit /b 1
    )
)

REM Install package
call %install_cmd%
if %ERRORLEVEL% equ 0 (
    call :log_success "Successfully installed: %package_name%"
    
    REM Test installation
    call %test_cmd% >nul 2>nul
    if %ERRORLEVEL% equ 0 (
        call :log_success "Package test passed: %package_name%"
    ) else (
        call :log_warning "Package installed but test failed: %package_name%"
    )
) else (
    call :log_error "Failed to install: %package_name%"
    endlocal
    exit /b 1
)

endlocal
goto :eof

REM Install Docker image
:install_docker_image
setlocal
set package_name=%~1
set install_cmd=%~2
set test_cmd=%~3

call :log_info "Installing Docker image: %package_name%"

REM Check if Docker is installed
where docker >nul 2>nul
if %ERRORLEVEL% neq 0 (
    call :log_error "Docker is not installed. Skipping Docker images."
    endlocal
    exit /b 1
)

REM Install image
call %install_cmd%
if %ERRORLEVEL% equ 0 (
    call :log_success "Successfully pulled: %package_name%"
    
    REM Test installation
    call %test_cmd% >nul 2>nul
    if %ERRORLEVEL% equ 0 (
        call :log_success "Image test passed: %package_name%"
    ) else (
        call :log_warning "Image pulled but test failed: %package_name%"
    )
) else (
    call :log_error "Failed to pull: %package_name%"
    endlocal
    exit /b 1
)

endlocal
goto :eof

REM Main installation function
:install_mcp_dependencies
call :log_info "Starting MCP dependency installation..."

REM Read server names from registry
for /f "usebackq tokens=*" %%s in (`jq -r ".mcp_registry.servers | keys[]" "%REGISTRY_FILE%"`) do (
    call :log_info "Processing server: %%s"
    
    REM Get server info
    for /f "usebackq tokens=*" %%p in (`jq -r ".mcp_registry.servers.\"%%s\".package_name" "%REGISTRY_FILE%"`) do set package_name=%%p
    for /f "usebackq tokens=*" %%t in (`jq -r ".mcp_registry.servers.\"%%s\".package_type" "%REGISTRY_FILE%"`) do set package_type=%%t
    for /f "usebackq tokens=*" %%i in (`jq -r ".mcp_registry.servers.\"%%s\".install_command" "%REGISTRY_FILE%"`) do set install_cmd=%%i
    for /f "usebackq tokens=*" %%c in (`jq -r ".mcp_registry.servers.\"%%s\".test_command" "%REGISTRY_FILE%"`) do set test_cmd=%%c
    
    REM Install based on package type
    if "%package_type%"=="npm" (
        call :install_npm_package "%package_name%" "%install_cmd%" "%test_cmd%"
    ) else if "%package_type%"=="go" (
        call :install_go_package "%package_name%" "%install_cmd%" "%test_cmd%"
    ) else if "%package_type%"=="pip" (
        call :install_pip_package "%package_name%" "%install_cmd%" "%test_cmd%"
    ) else if "%package_type%"=="docker" (
        call :install_docker_image "%package_name%" "%install_cmd%" "%test_cmd%"
    ) else (
        call :log_warning "Unknown package type: %package_type% for %%s"
    )
    
    echo.
)

call :log_success "MCP dependency installation completed!"
call :log_info "Log file: %LOG_FILE%"
goto :eof

REM List available servers
:list_servers
call :log_info "Available MCP servers:"
echo.

REM TODO: Implement server listing
call :log_warning "Server listing not yet implemented"
goto :eof

REM Main execution
:main
REM Check prerequisites
call :check_jq
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%

call :check_registry
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%

REM Install dependencies
call :install_mcp_dependencies

echo ==========================================
echo Installation completed successfully!
echo ==========================================
goto :eof

REM Parse command line arguments
if "%1"=="" (
    call :main
) else if "%1"=="--help" (
    echo Usage: %0 [OPTIONS]
    echo.
    echo Options:
    echo   --help     Show this help message
    echo   --list     List available MCP servers
    echo.
    echo Note: Advanced options not yet implemented for Windows
) else if "%1"=="--list" (
    call :list_servers
) else (
    echo Unknown option: %1
    echo Use --help for usage information
    exit /b 1
)