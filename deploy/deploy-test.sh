#!/usr/bin/env bash
# SDPivot test environment deployment helper.
# Usage: bash deploy-test.sh [build|up|down|restart|status|logs|smoke]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.test.yml"
ENV_FILE="$SCRIPT_DIR/.env.test"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { printf '%b[INFO]%b %s\n' "$GREEN" "$NC" "$1"; }
log_warn()  { printf '%b[WARN]%b %s\n' "$YELLOW" "$NC" "$1"; }
log_error() { printf '%b[ERROR]%b %s\n' "$RED" "$NC" "$1"; }

require_env_file() {
    if [ ! -f "$ENV_FILE" ]; then
        log_error ".env.test not found; copy deploy/.env.example and provide test secrets."
        exit 1
    fi
}

compose() {
    require_env_file
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

preflight() {
    log_info "Running pre-flight checks..."
    command -v docker &>/dev/null || { log_error "Docker is not installed"; exit 1; }
    docker compose version &>/dev/null || { log_error "Docker Compose plugin is not installed"; exit 1; }
    require_env_file
    docker network inspect weknora_WeKnora-network &>/dev/null || {
        log_error "WeKnora network (weknora_WeKnora-network) not found."
        log_error "Start WeKnora services before deploying SDPivot."
        exit 1
    }
    compose config --quiet
    log_info "Pre-flight checks passed."
}

cmd_build() {
    preflight
    log_info "Building SDPivot images..."
    compose build
    log_info "Build complete."
}

cmd_up() {
    preflight
    log_info "Starting SDPivot services..."
    compose up -d
    compose ps
    log_info "Services started."
}

cmd_down() {
    log_info "Stopping SDPivot services..."
    compose down
}

cmd_restart() {
    preflight
    log_info "Restarting SDPivot services..."
    compose restart
}

cmd_status() {
    log_info "SDPivot services status:"
    compose ps
}

cmd_logs() {
    compose logs -f --tail=100 "$@"
}

cmd_smoke() {
    "$SCRIPT_DIR/smoke-api-routes.sh"
}

cmd_help() {
    cat <<'HELP'
SDPivot Test Environment Deployment

Usage: bash deploy-test.sh <command>

Commands:
  build     Build SDPivot Docker images
  up        Start all SDPivot services
  down      Stop all SDPivot services
  restart   Restart all SDPivot services
  status    Show service status
  logs      Follow service logs
  smoke     Verify canonical and legacy API routes (requires SDP_BASE_URL)
HELP
}

case "${1:-help}" in
    build)   cmd_build "${@:2}" ;;
    up)      cmd_up "${@:2}" ;;
    down)    cmd_down "${@:2}" ;;
    restart) cmd_restart "${@:2}" ;;
    status)  cmd_status "${@:2}" ;;
    logs)    cmd_logs "${@:2}" ;;
    smoke)   cmd_smoke "${@:2}" ;;
    help|-h|--help) cmd_help ;;
    *) log_error "Unknown command: $1"; cmd_help; exit 2 ;;
esac
