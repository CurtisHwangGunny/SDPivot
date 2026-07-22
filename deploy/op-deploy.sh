#!/bin/bash
exec /usr/bin/python3 - "$(/usr/bin/realpath -- "$0")" "$@" <<'PY'
import fcntl
import os
import stat
import subprocess
import sys

script, *args = sys.argv[1:]
try:
    tmp = os.lstat("/tmp")
except OSError:
    raise SystemExit("OP deployment failed: /tmp is unavailable")
if stat.S_ISLNK(tmp.st_mode) or not stat.S_ISDIR(tmp.st_mode) or tmp.st_uid != 0 or not (tmp.st_mode & stat.S_ISVTX):
    raise SystemExit("OP deployment failed: /tmp must be a root-owned sticky directory")
lock_path = f"/tmp/weknora-op-deploy-{os.getuid()}.lock"
flags = os.O_RDWR | os.O_CREAT | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
try:
    fd = os.open(lock_path, flags, 0o600)
except OSError:
    raise SystemExit("OP deployment failed: repository lock could not be opened safely")
try:
    opened = os.fstat(fd)
    named = os.lstat(lock_path)
    if not stat.S_ISREG(opened.st_mode) or opened.st_uid != os.getuid() or opened.st_nlink != 1:
        raise SystemExit("OP deployment failed: repository lock must be a current-user single-link regular file")
    if (opened.st_dev, opened.st_ino) != (named.st_dev, named.st_ino):
        raise SystemExit("OP deployment failed: repository lock changed while it was opened")
    os.fchmod(fd, 0o600)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError:
        raise SystemExit("OP deployment failed: another OP deployment action holds the repository lock")
    source = open(script, encoding="utf-8").read()
    marker = "\n# __OP_LOCKED_BODY__\n"
    if source.count(marker) != 1:
        raise SystemExit("OP deployment failed: wrapper body marker is invalid")
    body = source.split(marker, 1)[1]
    result = subprocess.run(["/bin/bash", "-s", "--", script, *args], input=body, text=True, close_fds=True)
    raise SystemExit(result.returncode)
finally:
    os.close(fd)
PY
# __OP_LOCKED_BODY__
PATH=/usr/bin:/bin
LC_ALL=C
export PATH LC_ALL
unset CDPATH ENV BASH_ENV
set -Eeuo pipefail
umask 077

fail() {
    printf 'OP deployment failed: %s\n' "$1" >&2
    exit 1
}

readonly REALPATH=/usr/bin/realpath
readonly MKTEMP=/usr/bin/mktemp
readonly ENV_BIN=/usr/bin/env
readonly DOCKER=/usr/bin/docker
readonly NPM=/usr/bin/npm
readonly RM=/bin/rm
for tool in "$REALPATH" "$MKTEMP" "$ENV_BIN" "$DOCKER" "$NPM" "$RM"; do
    [[ -x "$tool" ]] || fail "required system tool is unavailable"
done

readonly SCRIPT_PATH="$($REALPATH -- "$1")"
shift
readonly DEPLOY_DIR="${SCRIPT_PATH%/*}"
readonly REPO_ROOT="${DEPLOY_DIR%/*}"
readonly VALIDATOR="$DEPLOY_DIR/validate-op-deployment.sh"
readonly COMPOSE_FILE="$DEPLOY_DIR/docker-compose.op.yml"
[[ -x "$VALIDATOR" ]] || fail "validator is unavailable"
[[ -f "$COMPOSE_FILE" ]] || fail "Compose file is unavailable"

usage() {
    printf '%s\n' "usage: $SCRIPT_PATH [--env-file PATH] {validate|config|build-assets|compose-build|up} [up: -d|--detach]" >&2
    exit 2
}

env_file="$DEPLOY_DIR/.env.op"
if [[ "${1:-}" == "--env-file" ]]; then
    (( $# >= 3 )) || usage
    env_file="$2"
    shift 2
fi
(( $# >= 1 )) || usage
action="$1"
shift
case "$action" in
    validate|config|build-assets)
        (( $# == 0 )) || usage
        ;;
    compose-build)
        (( $# == 0 )) || fail "compose-build accepts no additional arguments"
        ;;
    up)
        for arg in "$@"; do
            case "$arg" in
                -d|--detach) ;;
                *) fail "up accepts only -d or --detach; build and no-build overrides are forbidden" ;;
            esac
        done
        ;;
    *) usage ;;
esac

tmp_dir="$($MKTEMP -d /tmp/weknora-op-deploy.XXXXXX)"
cleanup() {
    "$RM" -rf -- "$tmp_dir"
}
trap cleanup EXIT HUP INT TERM
snapshot="$tmp_dir/env.snapshot"
manifest="$tmp_dir/artifacts.manifest"
config_json="$tmp_dir/config.json"
stage_override="$tmp_dir/stage-override.json"
stage_root="$tmp_dir/stage"
private_home="$tmp_dir/home"
docker_config="$private_home/.docker"
receipt="$DEPLOY_DIR/.op-build-receipt.json"
: > "$snapshot"
: > "$manifest"
: > "$config_json"
: > "$stage_override"
/bin/mkdir -m 700 -p -- "$docker_config" "$stage_root"
/bin/chmod 600 "$snapshot" "$manifest" "$config_json" "$stage_override"

compose_clean() {
    "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" DOCKER_CONFIG="$docker_config" LC_ALL=C \
        "$DOCKER" compose --env-file "$snapshot" -f "$COMPOSE_FILE" "$@"
}

compose_staged() {
    "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" DOCKER_CONFIG="$docker_config" LC_ALL=C \
        "$DOCKER" compose --env-file "$snapshot" -f "$COMPOSE_FILE" -f "$stage_override" "$@"
}

"$VALIDATOR" env "$env_file" "$snapshot"

case "$action" in
    validate)
        "$VALIDATOR" artifacts "$manifest"
        printf '%s\n' 'OP deployment prerequisites validated.'
        ;;
    config)
        compose_clean config --format json > "$config_json"
        "$VALIDATOR" config "$snapshot" "$config_json"
        /bin/cat -- "$config_json"
        ;;
    build-assets)
        "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" LC_ALL=C \
            "$NPM" --prefix "$REPO_ROOT/frontend/sdpivot" run build:op
        "$VALIDATOR" artifacts "$manifest"
        printf '%s\n' 'OP frontend assets built and recorded.'
        ;;
    compose-build)
        "$VALIDATOR" artifacts "$manifest"
        compose_clean config --format json > "$config_json"
        "$VALIDATOR" config "$snapshot" "$config_json"
        "$VALIDATOR" stage "$manifest" "$stage_root" "$stage_override"
        "$VALIDATOR" compare-stage "$manifest" "$stage_root"
        compose_staged build
        "$VALIDATOR" compare-stage "$manifest" "$stage_root"
        "$VALIDATOR" receipt-write "$snapshot" "$manifest" "$config_json" "$receipt"
        ;;
    up)
        [[ -e "$receipt" ]] || fail "no successful compose-build receipt exists for this deployment"
        "$VALIDATOR" artifacts "$manifest"
        compose_clean config --format json > "$config_json"
        "$VALIDATOR" config "$snapshot" "$config_json"
        "$VALIDATOR" receipt-verify "$snapshot" "$manifest" "$config_json" "$receipt"
        compose_clean up "$@" --no-build
        ;;
esac
