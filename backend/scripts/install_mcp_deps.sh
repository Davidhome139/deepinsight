#!/bin/bash

# MCP Dependency Installation Script for Linux/Mac
# This script installs required dependencies for MCP servers

set -e

echo "=========================================="
echo "MCP Dependency Installation Script"
echo "=========================================="

# Configuration
CONFIG_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../config" && pwd)"
REGISTRY_FILE="$CONFIG_DIR/mcp_registry.json"
LOG_FILE="/tmp/mcp_install_$(date +%Y%m%d_%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
    echo "[INFO] $(date): $1" >> "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    echo "[SUCCESS] $(date): $1" >> "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
    echo "[WARNING] $(date): $1" >> "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    echo "[ERROR] $(date): $1" >> "$LOG_FILE"
}

# Check if jq is installed
check_jq() {
    if ! command -v jq &> /dev/null; then
        log_error "jq is not installed. Please install jq first."
        log_info "On Ubuntu/Debian: sudo apt-get install jq"
        log_info "On Mac: brew install jq"
        log_info "On CentOS/RHEL: sudo yum install jq"
        exit 1
    fi
    log_success "jq is installed"
}

# Check if registry file exists
check_registry() {
    if [ ! -f "$REGISTRY_FILE" ]; then
        log_error "Registry file not found: $REGISTRY_FILE"
        exit 1
    fi
    log_success "Registry file found: $REGISTRY_FILE"
}

# Install NPM package
install_npm_package() {
    local package_name="$1"
    local install_cmd="$2"
    local test_cmd="$3"
    
    log_info "Installing NPM package: $package_name"
    
    # Check if package is already installed
    if command -v "$(echo "$test_cmd" | awk '{print $1}')" &> /dev/null; then
        log_success "Package already installed: $package_name"
        return 0
    fi
    
    # Install package
    if eval "$install_cmd"; then
        log_success "Successfully installed: $package_name"
        
        # Test installation
        if eval "$test_cmd" &> /dev/null; then
            log_success "Package test passed: $package_name"
        else
            log_warning "Package installed but test failed: $package_name"
        fi
    else
        log_error "Failed to install: $package_name"
        return 1
    fi
}

# Install Go package
install_go_package() {
    local package_name="$1"
    local install_cmd="$2"
    local test_cmd="$3"
    
    log_info "Installing Go package: $package_name"
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Skipping Go packages."
        return 1
    fi
    
    # Install package
    if eval "$install_cmd"; then
        log_success "Successfully installed: $package_name"
        
        # Test installation
        if eval "$test_cmd" &> /dev/null; then
            log_success "Package test passed: $package_name"
        else
            log_warning "Package installed but test failed: $package_name"
        fi
    else
        log_error "Failed to install: $package_name"
        return 1
    fi
}

# Install Pip package
install_pip_package() {
    local package_name="$1"
    local install_cmd="$2"
    local test_cmd="$3"
    
    log_info "Installing Pip package: $package_name"
    
    # Check if pip is installed
    if ! command -v pip3 &> /dev/null && ! command -v pip &> /dev/null; then
        log_error "pip/pip3 is not installed. Skipping Python packages."
        return 1
    fi
    
    # Use pip3 if available, otherwise pip
    local pip_cmd="pip3"
    if ! command -v pip3 &> /dev/null; then
        pip_cmd="pip"
    fi
    
    # Install package
    if eval "$install_cmd"; then
        log_success "Successfully installed: $package_name"
        
        # Test installation
        if eval "$test_cmd" &> /dev/null; then
            log_success "Package test passed: $package_name"
        else
            log_warning "Package installed but test failed: $package_name"
        fi
    else
        log_error "Failed to install: $package_name"
        return 1
    fi
}

# Install Docker image
install_docker_image() {
    local package_name="$1"
    local install_cmd="$2"
    local test_cmd="$3"
    
    log_info "Installing Docker image: $package_name"
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Skipping Docker images."
        return 1
    fi
    
    # Install image
    if eval "$install_cmd"; then
        log_success "Successfully pulled: $package_name"
        
        # Test installation
        if eval "$test_cmd" &> /dev/null; then
            log_success "Image test passed: $package_name"
        else
            log_warning "Image pulled but test failed: $package_name"
        fi
    else
        log_error "Failed to pull: $package_name"
        return 1
    fi
}

# Main installation function
install_mcp_dependencies() {
    log_info "Starting MCP dependency installation..."
    
    # Read registry file
    local servers=$(jq -r '.mcp_registry.servers | keys[]' "$REGISTRY_FILE")
    
    for server in $servers; do
        log_info "Processing server: $server"
        
        # Get server info
        local package_name=$(jq -r ".mcp_registry.servers.\"$server\".package_name" "$REGISTRY_FILE")
        local package_type=$(jq -r ".mcp_registry.servers.\"$server\".package_type" "$REGISTRY_FILE")
        local install_cmd=$(jq -r ".mcp_registry.servers.\"$server\".install_command" "$REGISTRY_FILE")
        local test_cmd=$(jq -r ".mcp_registry.servers.\"$server\".test_command" "$REGISTRY_FILE")
        
        # Install based on package type
        case "$package_type" in
            "npm")
                install_npm_package "$package_name" "$install_cmd" "$test_cmd"
                ;;
            "go")
                install_go_package "$package_name" "$install_cmd" "$test_cmd"
                ;;
            "pip")
                install_pip_package "$package_name" "$install_cmd" "$test_cmd"
                ;;
            "docker")
                install_docker_image "$package_name" "$install_cmd" "$test_cmd"
                ;;
            *)
                log_warning "Unknown package type: $package_type for $server"
                ;;
        esac
        
        echo ""
    done
    
    log_success "MCP dependency installation completed!"
    log_info "Log file: $LOG_FILE"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  -h, --help     Show this help message"
                echo "  -l, --list     List available MCP servers"
                echo "  -s, --server   Install specific server (comma-separated)"
                echo "  -c, --category Install servers by category"
                echo ""
                echo "Categories:"
                echo "  documentation  Documentation servers (context7)"
                echo "  automation     Automation servers (playwright)"
                echo "  productivity   Productivity servers (todoist)"
                echo "  reasoning      Reasoning servers (sequential-thinking)"
                exit 0
                ;;
            --list|-l)
                list_servers
                exit 0
                ;;
            --server|-s)
                if [[ -n "$2" ]]; then
                    install_specific_servers "$2"
                    exit 0
                else
                    log_error "Server names required for --server option"
                    exit 1
                fi
                ;;
            --category|-c)
                if [[ -n "$2" ]]; then
                    install_by_category "$2"
                    exit 0
                else
                    log_error "Category required for --category option"
                    exit 1
                fi
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
        shift
    done
}

# List available servers
list_servers() {
    log_info "Available MCP servers:"
    echo ""
    
    jq -r '.mcp_registry.servers | to_entries[] | "\(.key):" + "\n  Description: \(.value.description)" + "\n  Package: \(.value.package_name) (\(.value.package_type))" + "\n  Install: \(.value.install_command)"' "$REGISTRY_FILE"
    
    echo ""
    log_info "Categories:"
    jq -r '.mcp_registry.categories | to_entries[] | "\(.key): \(.value | join(\", \"))"' "$REGISTRY_FILE"
}

# Install specific servers
install_specific_servers() {
    local servers="$1"
    log_info "Installing specific servers: $servers"
    
    # TODO: Implement specific server installation
    log_warning "Specific server installation not yet implemented"
}

# Install by category
install_by_category() {
    local category="$1"
    log_info "Installing servers by category: $category"
    
    # TODO: Implement category-based installation
    log_warning "Category-based installation not yet implemented"
}

# Main execution
main() {
    # Parse arguments
    parse_args "$@"
    
    # Check prerequisites
    check_jq
    check_registry
    
    # Install dependencies
    install_mcp_dependencies
    
    echo "=========================================="
    echo "Installation completed successfully!"
    echo "=========================================="
}

# Run main function
main "$@"