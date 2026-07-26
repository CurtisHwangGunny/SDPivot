#!/usr/bin/env bash
# OP end-to-end workflow:
# admin login -> create user -> assign role -> create department -> configure
# models -> upload document -> verify automatic tags -> search -> QA -> verify
# audit log -> create, download, and checksum-verify a database backup.

set -Eeuo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
readonly WORKFLOW="$SCRIPT_DIR/test-phase4-platform-smoke.sh"

if [[ ! -x "$WORKFLOW" ]]; then
    printf 'BLOCKED E2E workflow is missing or not executable: %s\n' "$WORKFLOW" >&2
    exit 1
fi

exec "$WORKFLOW" "$@"
