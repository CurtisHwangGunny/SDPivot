#!/usr/bin/env bash
# Verify the canonical SDPivot API and the legacy smartknora compatibility API.

set -euo pipefail

: "${SDP_BASE_URL:?Set SDP_BASE_URL to the deployed backend origin}"
BASE_URL="${SDP_BASE_URL%/}"
INVALID_LOGIN_PAYLOAD='{"email":"__sdpivot_smoke_invalid__@invalid.example","password":"invalid"}'

request() {
    local method="$1"
    local path="$2"
    local response_file="$3"
    shift 3
    curl --silent --show-error --output "$response_file" --write-out '%{http_code}' \
        --request "$method" --connect-timeout 5 --max-time 15 "$@" "${BASE_URL}${path}"
}

assert_health() {
    local label="$1"
    local path="$2"
    local response_file
    local status
    response_file="$(mktemp)"
    if ! status="$(request GET "$path" "$response_file")"; then
        rm -f "$response_file"
        printf 'FAIL %-9s %s -> request error\n' "$label" "$path" >&2
        return 1
    fi
    if [ "$status" != "200" ]; then
        rm -f "$response_file"
        printf 'FAIL %-9s %s -> %s\n' "$label" "$path" "$status" >&2
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
        printf 'FAIL %-9s %s -> unexpected response body\n' "$label" "$path" >&2
        return 1
    fi
    rm -f "$response_file"
    printf 'PASS %-9s %s -> 200 status=ok service=sdp\n' "$label" "$path"
}

assert_login_route() {
    local label="$1"
    local path="$2"
    local response_file
    local status
    response_file="$(mktemp)"
    if ! status="$(request POST "$path" "$response_file" \
        --header 'Content-Type: application/json' --data "$INVALID_LOGIN_PAYLOAD")"; then
        rm -f "$response_file"
        printf 'FAIL %-9s %s -> request error\n' "$label" "$path" >&2
        return 1
    fi
    case "$status" in
        400|401|422)
            rm -f "$response_file"
            printf 'PASS %-9s %s -> %s route accepts POST\n' "$label" "$path" "$status"
            ;;
        *)
            rm -f "$response_file"
            printf 'FAIL %-9s %s -> unexpected HTTP %s (expected 400/401/422)\n' \
                "$label" "$path" "$status" >&2
            return 1
            ;;
    esac
}

result=0
assert_health canonical "/api/v1/sdp/health" || result=1
assert_health legacy "/api/v1/smartknora/health" || result=1
assert_login_route canonical "/api/v1/sdp/auth/login" || result=1
assert_login_route legacy "/api/v1/smartknora/auth/login" || result=1
if [ "$result" -ne 0 ]; then
    exit "$result"
fi
printf 'SDPivot API route compatibility verified at %s\n' "$BASE_URL"
