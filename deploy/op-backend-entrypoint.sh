#!/bin/sh
set -eu

export DB_DRIVER="${DB_DRIVER:-postgres}"
export DB_HOST="${DB_HOST:-${SDP_DB_HOST:-postgres}}"
export DB_PORT="${DB_PORT:-${SDP_DB_PORT:-5432}}"
export DB_USER="${DB_USER:-${SDP_DB_USER:-}}"
export DB_PASSWORD="${DB_PASSWORD:-${SDP_DB_PASSWORD:-}}"
export DB_NAME="${DB_NAME:-${SDP_DB_NAME:-}}"
export REDIS_ADDR="${REDIS_ADDR:-${SDP_REDIS_ADDR:-redis:6379}}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-${SDP_REDIS_PASSWORD:-}}"
export JWT_SECRET="${JWT_SECRET:-${SDP_JWT_SECRET:-}}"
export SERVER_PORT="${SERVER_PORT:-${PORT:-8081}}"

exec "$@"
