@echo off
chcp 65001 >nul
echo Starting development environment...

:: Check if docker-compose is available
where docker-compose >nul 2>nul
if %errorlevel% neq 0 (
    echo Error: docker-compose not found. Please install Docker Desktop.
    exit /b 1
)

:: Parse arguments
if "%1"=="up" goto :up
if "%1"=="down" goto :down
if "%1"=="restart" goto :restart
if "%1"=="logs" goto :logs
if "%1"=="build" goto :build
if "%1"=="clean" goto :clean
if "%1"=="shell-backend" goto :shell_backend
if "%1"=="shell-frontend" goto :shell_frontend
if "%1"=="db" goto :db
goto :help

:up
echo Starting development services...
docker-compose -f docker-compose.dev.yaml up -d
if %errorlevel% neq 0 (
    echo Failed to start services. Trying to build first...
    docker-compose -f docker-compose.dev.yaml up -d --build
)
echo.
echo Services started:
echo - Frontend: http://localhost:5173
echo - Backend API: http://localhost:8080
echo - Database: localhost:5432
echo - Redis: localhost:6379
echo.
echo Run 'dev.bat logs' to view logs
goto :eof

:down
echo Stopping development services...
docker-compose -f docker-compose.dev.yaml down
goto :eof

:restart
echo Restarting development services...
docker-compose -f docker-compose.dev.yaml restart
goto :eof

:logs
echo Showing logs (Ctrl+C to exit)...
docker-compose -f docker-compose.dev.yaml logs -f %2
goto :eof

:build
echo Rebuilding development services...
docker-compose -f docker-compose.dev.yaml down
docker-compose -f docker-compose.dev.yaml build --no-cache
docker-compose -f docker-compose.dev.yaml up -d
goto :eof

:clean
echo Cleaning up development environment...
docker-compose -f docker-compose.dev.yaml down -v
docker volume prune -f
echo Cleanup complete!
goto :eof

:shell_backend
echo Opening backend shell...
docker-compose -f docker-compose.dev.yaml exec backend sh
goto :eof

:shell_frontend
echo Opening frontend shell...
docker-compose -f docker-compose.dev.yaml exec frontend sh
goto :eof

:db
echo Connecting to database...
docker-compose -f docker-compose.dev.yaml exec db psql -U postgres -d yuanbao
goto :eof

:help
echo Usage: dev.bat [command]
echo.
echo Commands:
echo   up              Start development services
echo   down            Stop development services
echo   restart         Restart all services
echo   logs [service]  View logs (backend/frontend/db/redis)
echo   build           Rebuild and restart services
echo   clean           Clean up volumes and containers
echo   shell-backend   Open shell in backend container
echo   shell-frontend  Open shell in frontend container
echo   db              Connect to PostgreSQL database
echo.
echo Examples:
echo   dev.bat up
echo   dev.bat logs backend
echo   dev.bat restart
goto :eof
