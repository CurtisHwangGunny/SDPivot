#!/usr/bin/env bash
set -Eeuo pipefail

fail() {
    printf 'OP deployment validation failed: %s\n' "$1" >&2
    exit 1
}

if (( $# > 1 )); then
    fail "usage: $0 [env-file]"
fi

readonly SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
readonly REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
readonly ENV_FILE="${1:-$SCRIPT_DIR/.env.op}"

[[ -f "$ENV_FILE" ]] || fail "environment file not found"
[[ -r "$ENV_FILE" ]] || fail "environment file is not readable"

declare -A allowed=()
declare -A values=()
readonly ALLOWED_KEYS=(
    COMPOSE_PROJECT_NAME
    OP_DB_USER OP_DB_PASSWORD OP_DB_NAME
    OP_REDIS_PASSWORD OP_REDIS_DB OP_REDIS_PREFIX
    OP_JWT_SECRET OP_SDP_JWT_SECRET OP_TENANT_AES_KEY OP_SYSTEM_AES_KEY
    OP_ALLOWED_ORIGINS OP_HTTP_BIND OP_HTTP_PORT
    OP_POSTGRES_IMAGE OP_REDIS_IMAGE OP_APP_IMAGE OP_DOCREADER_IMAGE
    OP_MIGRATION_IMAGE OP_SDP_BACKEND_IMAGE OP_SDP_FRONTEND_IMAGE
    SDP_ENABLE_LEGACY_ALIAS DISABLE_REGISTRATION MAX_FILE_SIZE_MB
    WEKNORA_ASYNQ_CONCURRENCY CONCURRENCY_POOL_SIZE
    SDP_DB_MAX_OPEN_CONNS SDP_DB_MAX_IDLE_CONNS SDP_DB_CONN_MAX_LIFETIME
    SDP_READY_TIMEOUT DOCREADER_PDF_FORCE_SCANNED DOCREADER_ODL_MAX_WORKERS
    TZ APT_MIRROR APK_MIRROR_ARG
)
for key in "${ALLOWED_KEYS[@]}"; do
    allowed["$key"]=1
done

line_number=0
while IFS= read -r line || [[ -n "$line" ]]; do
    ((line_number += 1))
    line="${line%$'\r'}"
    case "$line" in
        ''|'#'*) continue ;;
    esac
    [[ "$line" == *=* ]] || fail "line $line_number must use literal KEY=VALUE syntax"
    key="${line%%=*}"
    value="${line#*=}"
    [[ "$key" =~ ^[A-Z][A-Z0-9_]*$ ]] || fail "line $line_number has an invalid key"
    [[ -n "${allowed[$key]+set}" ]] || fail "line $line_number uses unsupported key $key"
    [[ -z "${values[$key]+set}" ]] || fail "duplicate key $key"
    [[ ! "$value" =~ [[:space:]] ]] || fail "$key must be a whitespace-free literal"
    case "$value" in
        *'$'*|*'`'*|*';'*|*'&&'*|*'||'*|*'<'*|*'>'*|*"'"*|*'"'*|*'\\'*)
            fail "$key contains unsupported shell syntax"
            ;;
    esac
    values["$key"]="$value"
done < "$ENV_FILE"

readonly REQUIRED_KEYS=(
    OP_DB_USER OP_DB_PASSWORD OP_DB_NAME OP_REDIS_PASSWORD
    OP_JWT_SECRET OP_SDP_JWT_SECRET OP_TENANT_AES_KEY OP_SYSTEM_AES_KEY
    OP_ALLOWED_ORIGINS
    OP_POSTGRES_IMAGE OP_REDIS_IMAGE OP_APP_IMAGE OP_DOCREADER_IMAGE
    OP_MIGRATION_IMAGE OP_SDP_BACKEND_IMAGE OP_SDP_FRONTEND_IMAGE
)
for key in "${REQUIRED_KEYS[@]}"; do
    [[ -n "${values[$key]:-}" ]] || fail "required key $key is missing or empty"
done

for key in "${REQUIRED_KEYS[@]}"; do
    value="${values[$key]}"
    upper="${value^^}"
    case "$upper" in
        *CHANGE_ME*|*CHANGEME*|*PLACEHOLDER*|*REPLACE_ME*|*DUMMY*|*SAMPLE*|*TODO*|*YOUR_*|*INSERT_*)
            fail "$key still contains a known placeholder"
            ;;
    esac
done

readonly SECRET_KEYS=(
    OP_DB_PASSWORD OP_REDIS_PASSWORD OP_JWT_SECRET OP_SDP_JWT_SECRET
    OP_TENANT_AES_KEY OP_SYSTEM_AES_KEY
)
for key in "${SECRET_KEYS[@]}"; do
    lower="${values[$key],,}"
    case "$lower" in
        password|password[0-9]*|passwordpassword*|admin|admin[0-9]*|adminadmin*|\
        postgres|postgres[0-9]*|postgrespostgres*|redis|redis[0-9]*|redisredis*|\
        secret|secret[0-9]*|secretsecret*|changeme|change_me|changeme[0-9]*|\
        test|test[0-9]*|testtest*|default|default[0-9]*|sdpivot|sdpivot[0-9]*|\
        qwerty|qwerty[0-9]*|letmein|letmein[0-9]*|welcome|welcome[0-9]*|root|root[0-9]*)
            fail "$key uses a known weak value or pattern"
            ;;
    esac
done

(( ${#values[OP_DB_PASSWORD]} >= 16 )) || fail "OP_DB_PASSWORD must be at least 16 characters"
(( ${#values[OP_REDIS_PASSWORD]} >= 16 )) || fail "OP_REDIS_PASSWORD must be at least 16 characters"
(( ${#values[OP_JWT_SECRET]} >= 32 )) || fail "OP_JWT_SECRET must be at least 32 characters"
(( ${#values[OP_SDP_JWT_SECRET]} >= 32 )) || fail "OP_SDP_JWT_SECRET must be at least 32 characters"
(( ${#values[OP_TENANT_AES_KEY]} == 32 )) || fail "OP_TENANT_AES_KEY must be exactly 32 characters"
(( ${#values[OP_SYSTEM_AES_KEY]} == 32 )) || fail "OP_SYSTEM_AES_KEY must be exactly 32 characters"
[[ "${values[OP_JWT_SECRET]}" != "${values[OP_SDP_JWT_SECRET]}" ]] || fail "JWT secrets must be independent"
[[ "${values[OP_TENANT_AES_KEY]}" != "${values[OP_SYSTEM_AES_KEY]}" ]] || fail "AES keys must be independent"

case "${values[OP_ALLOWED_ORIGINS]}" in
    http://*|https://*) ;;
    *) fail "OP_ALLOWED_ORIGINS must contain explicit HTTP(S) origins" ;;
esac
[[ "${values[OP_ALLOWED_ORIGINS]}" != *'*'* ]] || fail "OP_ALLOWED_ORIGINS must not contain a wildcard"
case "${values[OP_ALLOWED_ORIGINS],,}" in
    *example.com*|*example.invalid*) fail "OP_ALLOWED_ORIGINS still uses an example origin" ;;
esac

validate_image() {
    local key="$1"
    local image="${values[$key]}"
    local last tag

    if [[ "$image" == *@sha256:* ]]; then
        [[ "$image" =~ @sha256:[0-9a-fA-F]{64}$ ]] || fail "$key has an invalid sha256 digest"
        return
    fi
    last="${image##*/}"
    [[ "$last" == *:* ]] || fail "$key must use an explicit non-latest tag or sha256 digest"
    tag="${last##*:}"
    [[ -n "$tag" && "${tag,,}" != latest ]] || fail "$key must not use latest"
}
readonly IMAGE_KEYS=(
    OP_POSTGRES_IMAGE OP_REDIS_IMAGE OP_APP_IMAGE OP_DOCREADER_IMAGE
    OP_MIGRATION_IMAGE OP_SDP_BACKEND_IMAGE OP_SDP_FRONTEND_IMAGE
)
for key in "${IMAGE_KEYS[@]}"; do
    validate_image "$key"
done

readonly REQUIRED_FILES=(
    deploy/docker-compose.op.yml
    deploy/migration/Dockerfile
    deploy/migrate-op.sh
    docker/Dockerfile.app
    docker/Dockerfile.docreader
    frontend/sdpivot/Dockerfile.backend
    frontend/sdpivot/Dockerfile.frontend
    frontend/sdpivot/nginx.conf
)
for relative_path in "${REQUIRED_FILES[@]}"; do
    [[ -f "$REPO_ROOT/$relative_path" ]] || fail "required deployment file is missing: $relative_path"
done
[[ -f "$REPO_ROOT/frontend/sdpivot/sdp-server" ]] || fail "frontend/sdpivot/sdp-server is missing"
[[ -x "$REPO_ROOT/frontend/sdpivot/sdp-server" ]] || fail "frontend/sdpivot/sdp-server is not executable"
[[ -f "$REPO_ROOT/frontend/sdpivot/dist/index.html" ]] || fail "frontend/sdpivot/dist/index.html is missing"
[[ ! -e "$REPO_ROOT/frontend/sdpivot/dist/ops.html" ]] || fail "frontend/sdpivot/dist/ops.html must not exist"

printf '%s\n' 'OP deployment prerequisites validated.'
