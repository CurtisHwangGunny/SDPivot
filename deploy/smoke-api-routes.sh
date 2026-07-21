#!/usr/bin/env bash
# Verify the canonical SDPivot API and the legacy smartknora compatibility API.

set -euo pipefail

: "${SDP_BASE_URL:?Set SDP_BASE_URL to the deployed backend origin}"
BASE_URL="${SDP_BASE_URL%/}"
CANONICAL_PATH="/api/v1/sdp/health"
LEGACY_PATH="/api/v1/smartknora/health"

request() {
    local path="$1"
    local response_file="$2"
    curl --silent --show-error --output "$response_file" --write-out '%{http_code}' \
        --connect-timeout 5 --max-time 15 "${BASE_URL}${path}"
}

assert_route() {
    local label="$1"
    local path="$2"
    local response_file
    local status
    response_file="$(mktemp)"
    if ! status="$(request "$path" "$response_file")"; then
        rm -f "$response_file"
        printf 'FAIL %-9s %s -> request error
' "$label" "$path" >&2
        return 1
    fi
    if [ "$status" != "200" ]; then
        rm -f "$response_file"
        printf 'FAIL %-9s %s -> %s
' "$label" "$path" "$status" >&2
        return 1
    fi
    if ! python3 - "$response_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as response:
    payload = json.load(response)
if payload.get("status") != "ok" or payload.get("service") != "sdp":
    raise SystemExit(1)
PY
    then
        rm -f "$response_file"
        printf 'FAIL %-9s %s -> unexpected response body
' "$label" "$path" >&2
        return 1
    fi
    rm -f "$response_file"
    printf 'PASS %-9s %s -> 200 status=ok service=sdp
' "$label" "$path"
}

result=0
assert_route canonical "$CANONICAL_PATH" || result=1
assert_route legacy "$LEGACY_PATH" || result=1
if [ "$result" -ne 0 ]; then
    exit "$result"
fi
printf 'SDPivot API route compatibility verified at %s
' "$BASE_URL"
