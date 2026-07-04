#!/usr/bin/env bash
# smartKnora (随越·智枢) Test Environment Deployment Script
# Usage: bash deploy-test.sh [build|up|down|restart|status|logs]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.test.yml"
ENV_FILE="$SCRIPT_DIR/.env.test"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Pre-flight checks
preflight() {
    log_info "Running pre-flight checks..."

    # Check docker
    if ! command -v docker &>/dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi

    # Check docker compose
    if ! docker compose version &>/dev/null; then
        log_error "Docker Compose plugin is not installed"
        exit 1
    fi

    # Check WeKnora network
    if ! docker network ls --format '{{.Name}}' | grep -q 'weknora_WeKnora-network'; then
        log_error "WeKnora network (weknora_WeKnora-network) not found."
        log_error "Please start WeKnora services first: cd ~/projects/weknora && docker compose up -d"
        exit 1
    fi

    # Check .env file
    if [ ! -f "$ENV_FILE" ]; then
        log_warn ".env.test not found, copying from template..."
        cp "$SCRIPT_DIR/.env.test.template" "$ENV_FILE" 2>/dev/null || {
            log_error "No .env.test or .env.test.template found. Please create .env.test first."
            exit 1
        }
        log_warn "Please edit .env.test with your actual configuration before deploying."
        exit 1
    fi

    log_info "Pre-flight checks passed!"
}

# Build images
cmd_build() {
    log_info "Building smartKnora images..."
    cd "$SCRIPT_DIR"

    # Build backend
    log_info "Building backend..."
    docker build \
        -t smartknora-backend:test \
        -f ../../frontend/smartknora/Dockerfile.backend \
        "$(cd "$SCRIPT_DIR/../.." && pwd)"

    # Build frontend (if using production mode)
    log_info "Building frontend..."
    cd "$SCRIPT_DIR/../../frontend/smartknora"
    docker build \
        -t smartknora-frontend:test \
        -f Dockerfile.frontend \
        .

    cd "$SCRIPT_DIR"
    log_info "Build complete!"
}

# Start services
cmd_up() {
    preflight
    log_info "Starting smartKnora services..."
    docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d
    log_info "Waiting for health checks..."
    sleep 5
    docker compose -f "$COMPOSE_FILE" ps
    log_info "Services started. Frontend: http://localhost:$(grep SMARTKNORA_FRONTEND_PORT "$ENV_FILE" | cut -d= -f2 || echo 3099)"
}

# Stop services
cmd_down() {
    log_info "Stopping smartKnora services..."
    docker compose -f "$COMPOSE_FILE" down
    log_info "Services stopped."
}

# Restart services
cmd_restart() {
    log_info "Restarting smartKnora services..."
    docker compose -f "$COMPOSE_FILE" restart
    log_info "Services restarted."
}

# Show status
cmd_status() {
    log_info "smartKnora services status:"
    docker compose -f "$COMPOSE_FILE" ps 2>/dev/null || log_warn "No services running"
    echo ""
    log_info "WeKnora services status:"
    cd "$SCRIPT_DIR/../.." && docker compose ps 2>/dev/null || true
}

# Show logs
cmd_logs() {
    docker compose -f "$COMPOSE_FILE" logs -f --tail=100 "${@}"
}

# Help
cmd_help() {
    echo "smartKnora Test Environment Deployment"
    echo ""
    echo "Usage: bash deploy-test.sh <command>"
    echo ""
    echo "Commands:"
    echo "  build     Build smartKnora Docker images"
    echo "  up        Start all smartKnora services"
    echo "  down      Stop all smartKnora services"
    echo "  restart   Restart all smartKnora services"
    echo "  status    Show service status"
    echo "  logs      Follow service logs"
    echo ""
}

# Main
case "${1:-help}" in
    build)    cmd_build "${@:2}" ;;
    up)       cmd_up "${@:2}" ;;
    down)     cmd_down "${@:2}" ;;
    restart)  cmd_restart "${@:2}" ;;
    status)   cmd_status "${@:2}" ;;
    logs)     cmd_logs "${@:2}" ;;
    help|*)   cmd_help ;;
esac
