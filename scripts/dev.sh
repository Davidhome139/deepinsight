#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: docker-compose not found. Please install Docker.${NC}"
    exit 1
fi

# Help function
show_help() {
    echo "Usage: ./scripts/dev.sh [command]"
    echo ""
    echo "Commands:"
    echo "  up              Start development services"
    echo "  down            Stop development services"
    echo "  restart         Restart all services"
    echo "  logs [service]  View logs (backend/frontend/db/redis)"
    echo "  build           Rebuild and restart services"
    echo "  clean           Clean up volumes and containers"
    echo "  shell-backend   Open shell in backend container"
    echo "  shell-frontend  Open shell in frontend container"
    echo "  db              Connect to PostgreSQL database"
    echo "  status          Show services status"
    echo ""
    echo "Examples:"
    echo "  ./scripts/dev.sh up"
    echo "  ./scripts/dev.sh logs backend"
    echo "  ./scripts/dev.sh restart"
}

# Start services
cmd_up() {
    echo -e "${GREEN}Starting development services...${NC}"
    
    if ! docker-compose -f docker-compose.dev.yaml up -d; then
        echo -e "${YELLOW}Failed to start services. Trying to build first...${NC}"
        docker-compose -f docker-compose.dev.yaml up -d --build
    fi
    
    echo ""
    echo -e "${GREEN}Services started:${NC}"
    echo -e "  ${BLUE}Frontend:${NC}     http://localhost:5173"
    echo -e "  ${BLUE}Backend API:${NC}  http://localhost:8080"
    echo -e "  ${BLUE}Database:${NC}     localhost:5432"
    echo -e "  ${BLUE}Redis:${NC}        localhost:6379"
    echo ""
    echo -e "Run '${YELLOW}./scripts/dev.sh logs${NC}' to view logs"
}

# Stop services
cmd_down() {
    echo -e "${GREEN}Stopping development services...${NC}"
    docker-compose -f docker-compose.dev.yaml down
}

# Restart services
cmd_restart() {
    echo -e "${GREEN}Restarting development services...${NC}"
    docker-compose -f docker-compose.dev.yaml restart
}

# View logs
cmd_logs() {
    local service=$1
    echo -e "${GREEN}Showing logs (Ctrl+C to exit)...${NC}"
    if [ -n "$service" ]; then
        docker-compose -f docker-compose.dev.yaml logs -f "$service"
    else
        docker-compose -f docker-compose.dev.yaml logs -f
    fi
}

# Build services
cmd_build() {
    echo -e "${GREEN}Rebuilding development services...${NC}"
    docker-compose -f docker-compose.dev.yaml down
    docker-compose -f docker-compose.dev.yaml build --no-cache
    docker-compose -f docker-compose.dev.yaml up -d
}

# Clean up
cmd_clean() {
    echo -e "${YELLOW}Cleaning up development environment...${NC}"
    docker-compose -f docker-compose.dev.yaml down -v
    docker volume prune -f
    echo -e "${GREEN}Cleanup complete!${NC}"
}

# Open backend shell
cmd_shell_backend() {
    echo -e "${GREEN}Opening backend shell...${NC}"
    docker-compose -f docker-compose.dev.yaml exec backend sh
}

# Open frontend shell
cmd_shell_frontend() {
    echo -e "${GREEN}Opening frontend shell...${NC}"
    docker-compose -f docker-compose.dev.yaml exec frontend sh
}

# Connect to database
cmd_db() {
    echo -e "${GREEN}Connecting to database...${NC}"
    docker-compose -f docker-compose.dev.yaml exec db psql -U postgres -d yuanbao
}

# Show status
cmd_status() {
    echo -e "${GREEN}Services status:${NC}"
    docker-compose -f docker-compose.dev.yaml ps
}

# Main command dispatcher
case "${1:-}" in
    up)
        cmd_up
        ;;
    down)
        cmd_down
        ;;
    restart)
        cmd_restart
        ;;
    logs)
        cmd_logs "$2"
        ;;
    build)
        cmd_build
        ;;
    clean)
        cmd_clean
        ;;
    shell-backend)
        cmd_shell_backend
        ;;
    shell-frontend)
        cmd_shell_frontend
        ;;
    db)
        cmd_db
        ;;
    status)
        cmd_status
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        show_help
        exit 1
        ;;
esac
