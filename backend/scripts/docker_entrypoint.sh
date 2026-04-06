#!/bin/bash

# Docker Entrypoint Script for MCP Automation
# This script initializes MCP dependencies and starts the application

set -e

echo "=========================================="
echo "MCP Automation Docker Entrypoint"
echo "=========================================="

# Configuration
APP_DIR="/app"
CONFIG_DIR="$APP_DIR/config"
SCRIPTS_DIR="$APP_DIR/scripts"
LOG_DIR="/var/log/mcp"
REGISTRY_FILE="$CONFIG_DIR/mcp_registry.json"

# Create log directory
mkdir -p "$LOG_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
    echo "[INFO] $(date): $1" >> "$LOG_DIR/entrypoint.log"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    echo "[SUCCESS] $(date): $1" >> "$LOG_DIR/entrypoint.log"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
    echo "[WARNING] $(date): $1" >> "$LOG_DIR/entrypoint.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    echo "[ERROR] $(date): $1" >> "$LOG_DIR/entrypoint.log"
}

# Check if jq is installed
check_jq() {
    if ! command -v jq &> /dev/null; then
        log_error "jq is not installed. Please install jq first."
        exit 1
    fi
    log_success "jq is installed: $(jq --version)"
}

# Check if registry file exists
check_registry() {
    if [ ! -f "$REGISTRY_FILE" ]; then
        log_warning "Registry file not found: $REGISTRY_FILE"
        log_info "Creating default registry..."
        create_default_registry
    fi
    log_success "Registry file ready: $REGISTRY_FILE"
}

# Create default registry
create_default_registry() {
    cat > "$REGISTRY_FILE" << 'EOF'
{
  "servers": {},
  "categories": {
    "documentation": [],
    "automation": [],
    "productivity": [],
    "reasoning": []
  },
  "settings": {
    "auto_update": true,
    "check_interval_hours": 24,
    "notify_on_update": true
  }
}
EOF
    log_success "Default registry created"
}

# Start Xvfb for headless browser support (required for Playwright)
start_xvfb() {
    log_info "Starting Xvfb virtual display server..."
    
    # Clean up any existing Xvfb lock file
    if [ -f /tmp/.X99-lock ]; then
        log_info "Removing existing Xvfb lock file: /tmp/.X99-lock"
        rm -f /tmp/.X99-lock
    fi
    
    # Kill any existing Xvfb process on display 99
    if command -v fuser &> /dev/null; then
        fuser -k /tmp/.X11-unix/X99 2>/dev/null || true
    fi
    
    # Start Xvfb
    Xvfb :99 -screen 0 1280x1024x24 -ac +extension GLX +render -noreset &
    XVFB_PID=$!

    # Wait for Xvfb to start
    sleep 2
    if kill -0 $XVFB_PID 2>/dev/null; then
        log_success "Xvfb started successfully (PID: $XVFB_PID)"
        # Set DISPLAY environment variable for all processes
        export DISPLAY=:99
        echo "export DISPLAY=:99" >> /etc/profile
    else
        log_warning "Xvfb failed to start, but continuing anyway..."
    fi
}

# Verify Playwright installation
verify_playwright() {
    log_info "Verifying Playwright installation..."
    
    # Check if playwright-mcp command exists
    if command -v playwright-mcp &> /dev/null; then
        log_success "Playwright MCP is installed"
    else
        log_warning "Playwright MCP command not found, attempting to install..."
        npm install -g @playwright/mcp@latest
        if command -v playwright-mcp &> /dev/null; then
            log_success "Playwright MCP installed successfully"
        else
            log_error "Failed to install Playwright MCP"
        fi
    fi
    
    # Check Chromium installation
    if [ -f "/opt/google/chrome/chrome" ]; then
        log_success "Chromium found at /opt/google/chrome/chrome"
    else
        log_warning "Chromium not found at /opt/google/chrome/chrome"
    fi
}

# Set environment variables
set_environment() {
    log_info "Setting environment variables..."
    
    # Set global environment variables for browser automation
    export DISPLAY=:99
    export PLAYWRIGHT_BROWSERS_PATH=/home/pwuser/.cache/ms-playwright
    export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=0
    export PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/opt/google/chrome/chrome
    export PLAYWRIGHT_BROWSER=chrome
    
    log_info "Environment variables set:"
    log_info "  DISPLAY=$DISPLAY"
    log_info "  PLAYWRIGHT_BROWSERS_PATH=$PLAYWRIGHT_BROWSERS_PATH"
    log_info "  PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=$PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD"
    log_info "  PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=$PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH"
    log_info "  PLAYWRIGHT_BROWSER=$PLAYWRIGHT_BROWSER"
}

# Main execution
main() {
    log_info "Starting entrypoint script..."
    
    # Check dependencies
    check_jq
    
    # Check registry
    check_registry
    
    # Start Xvfb for headless browser support
    start_xvfb
    
    # Verify Playwright installation
    verify_playwright
    
    # Set environment variables
    set_environment
    
    # Start application
    log_info "Starting application..."
    if [ -f "$APP_DIR/main" ]; then
        log_success "Application binary found: $APP_DIR/main"
        log_info "Starting server..."
        exec "$APP_DIR/main"
    else
        log_error "Application binary not found: $APP_DIR/main"
        exit 1
    fi
}

# Run main function
main "$@"