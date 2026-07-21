#!/usr/bin/env bash
# SDPivot test environment deployment helper.
# Usage: bash deploy-test.sh [build|up|down|restart|status|logs|smoke]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FRONTEND_DIR="$REPO_ROOT/frontend/sdpivot"
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

initialize_upload_dir() {
    local upload_dir
    upload_dir="$(compose config --format json | python3 -c 'import json, sys
config = json.load(sys.stdin)
for mount in config["services"]["sdp-backend"].get("volumes", []):
    if mount.get("type") == "bind" and mount.get("target") == "/tmp/sdpivot-uploads":
        print(mount["source"])
        break')"
    if [ -z "$upload_dir" ]; then
        log_error "Unable to resolve the SDPivot upload bind mount source."
        exit 1
    fi

    log_info "Initializing SDPivot upload directory at $upload_dir..."
    if [ "$(id -u)" -eq 0 ]; then
        install -d -m 0750 -o 999 -g 999 "$upload_dir"
    elif command -v sudo &>/dev/null && sudo -n true 2>/dev/null; then
        sudo -n install -d -m 0750 -o 999 -g 999 "$upload_dir"
    else
        log_error "Root privileges are required to assign the upload directory to container uid/gid 999."
        exit 1
    fi
}

preflight() {
    log_info "Running pre-flight checks..."
    command -v docker &>/dev/null || { log_error "Docker is not installed"; exit 1; }
    docker compose version &>/dev/null || { log_error "Docker Compose plugin is not installed"; exit 1; }
    command -v npm &>/dev/null || { log_error "npm is not installed"; exit 1; }
    require_env_file
    docker network inspect weknora_WeKnora-network &>/dev/null || {
        log_error "WeKnora network (weknora_WeKnora-network) not found."
        log_error "Start WeKnora services before deploying SDPivot."
        exit 1
    }
    compose config --quiet
    log_info "Pre-flight checks passed."
}

build_artifacts() {
    local go_bin=""
    local go_image="${SDP_GO_BUILD_IMAGE:-golang:1.26-bookworm}"
    local frontend_build_dir
    if [ -x /usr/local/go/bin/go ]; then
        go_bin=/usr/local/go/bin/go
    elif command -v go &>/dev/null; then
        go_bin="$(command -v go)"
    fi

    if [ -n "$go_bin" ]; then
        log_info "Building SDPivot backend artifact with $go_bin..."
        (cd "$REPO_ROOT" && "$go_bin" build -trimpath -buildvcs=false -o frontend/sdpivot/sdp-server ./cmd/sdp-server)
    else
        log_info "Building SDPivot backend artifact with $go_image..."
        docker run --rm \
            --user "$(id -u):$(id -g)" \
            -e CGO_ENABLED=1 \
            -e GOCACHE=/tmp/go-build-cache \
            -e GOMODCACHE=/tmp/go-mod-cache \
            -v "$REPO_ROOT:/src" \
            -w /src \
            "$go_image" \
            go build -trimpath -buildvcs=false -o frontend/sdpivot/sdp-server ./cmd/sdp-server
    fi

    log_info "Building SDPivot frontend artifact in an isolated workspace..."
    frontend_build_dir="$(mktemp -d)"
    trap 'rm -rf "$frontend_build_dir"' RETURN
    (
        cd "$FRONTEND_DIR"
        tar --exclude='./node_modules' --exclude='./dist' -cf - .
    ) | (cd "$frontend_build_dir" && tar -xf -)
    (cd "$frontend_build_dir" && npm ci && npm run build:op)
    test -f "$frontend_build_dir/dist/index.html" || { log_error "Frontend index artifact was not produced"; exit 1; }
    test ! -e "$frontend_build_dir/dist/ops.html" || { log_error "OP frontend artifact unexpectedly contains ops.html"; exit 1; }
    rm -rf "$FRONTEND_DIR/dist"
    cp -R "$frontend_build_dir/dist" "$FRONTEND_DIR/dist"
    rm -rf "$frontend_build_dir"
    trap - RETURN

    test -x "$FRONTEND_DIR/sdp-server" || { log_error "Backend artifact was not produced"; exit 1; }
    test -f "$FRONTEND_DIR/dist/index.html" || { log_error "Frontend artifact was not produced"; exit 1; }
    log_info "Artifacts are up to date."
}

cmd_build() {
    preflight
    build_artifacts
    log_info "Building SDPivot images..."
    compose build
    log_info "Build complete."
}

cmd_up() {
    preflight
    initialize_upload_dir
    build_artifacts
    log_info "Building and starting SDPivot services..."
    compose up -d --build
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
  build     Build current Go/Vue artifacts and Docker images
  up        Rebuild current artifacts/images and start services
  down      Stop all SDPivot services
  restart   Restart existing SDPivot containers without rebuilding
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
