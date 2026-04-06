@echo off
setlocal

set SCRIPT_DIR=%~dp0
set REPO_ROOT=%SCRIPT_DIR%..\..
set GIT_HOOKS_DIR=%REPO_ROOT%\.git\hooks

echo Installing Git hooks...

if not exist "%GIT_HOOKS_DIR%" (
    mkdir "%GIT_HOOKS_DIR%"
)

copy "%SCRIPT_DIR%pre-commit" "%GIT_HOOKS_DIR%pre-commit" /Y

echo Git hooks installed successfully!
echo.
echo The following hooks have been installed:
echo   - pre-commit: Sensitive information detection
echo.
echo To uninstall, remove the hooks from %GIT_HOOKS_DIR%

pause
