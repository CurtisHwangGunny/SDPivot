#!/usr/bin/env bash
# Live Phase 4 platform smoke test. Requires a disposable tenant and working model providers.

set -Eeuo pipefail
umask 077

BASE_URL="${WEKNORA_E2E_BASE_URL:-http://127.0.0.1:8080}"
BASE_URL="${BASE_URL%/}"
ADMIN_EMAIL="${WEKNORA_E2E_ADMIN_EMAIL:-}"
ADMIN_PASSWORD="${WEKNORA_E2E_ADMIN_PASSWORD:-}"
MODEL_BASE_URL="${WEKNORA_E2E_MODEL_BASE_URL:-https://api.openai.com/v1}"
MODEL_API_KEY="${WEKNORA_E2E_MODEL_API_KEY:-}"
CHAT_MODEL_NAME="${WEKNORA_E2E_CHAT_MODEL:-}"
EMBEDDING_MODEL_NAME="${WEKNORA_E2E_EMBEDDING_MODEL:-}"
EMBEDDING_DIMENSION="${WEKNORA_E2E_EMBEDDING_DIMENSION:-1536}"
TIMEOUT="${WEKNORA_E2E_TIMEOUT:-600}"
POLL_INTERVAL="${WEKNORA_E2E_POLL_INTERVAL:-2}"
KEEP_RESOURCES="${WEKNORA_E2E_KEEP_RESOURCES:-0}"

TMP_ROOT=""
TOKEN=""
TENANT_ID=""
RUN_ID="phase4-$(date -u +%Y%m%d%H%M%S)-$$"
TEST_USER_ID=""
DEPARTMENT_ID=""
CHAT_MODEL_ID=""
EMBEDDING_MODEL_ID=""
TAG_ID=""
KB_ID=""
KNOWLEDGE_ID=""
SESSION_ID=""
BACKUP_ID=""
LAST_RESPONSE=""
LAST_STATUS=""

usage() {
    cat <<'EOF'
Usage: deploy/tests/test-phase4-platform-smoke.sh

Required environment:
  WEKNORA_E2E_ADMIN_EMAIL
  WEKNORA_E2E_ADMIN_PASSWORD
  WEKNORA_E2E_MODEL_API_KEY
  WEKNORA_E2E_CHAT_MODEL
  WEKNORA_E2E_EMBEDDING_MODEL

Optional environment:
  WEKNORA_E2E_BASE_URL              default: http://127.0.0.1:8080
  WEKNORA_E2E_MODEL_BASE_URL        default: https://api.openai.com/v1
  WEKNORA_E2E_EMBEDDING_DIMENSION   default: 1536
  WEKNORA_E2E_TIMEOUT               default: 600 seconds
  WEKNORA_E2E_POLL_INTERVAL         default: 2 seconds
  WEKNORA_E2E_KEEP_RESOURCES        default: 0; set to 1 for debugging

The admin must be both the active tenant Owner and a System Admin. The target
deployment must allow self-registration and have working document processing,
Redis workers, vector search, pg_dump, and the configured model provider.
EOF
}

pass() {
    printf 'PASS %s\n' "$*"
}

fail() {
    printf 'FAIL %s\n' "$*" >&2
    if [[ -n "$LAST_RESPONSE" && -s "$LAST_RESPONSE" ]]; then
        printf 'Last response (HTTP %s):\n' "${LAST_STATUS:-unknown}" >&2
        jq . "$LAST_RESPONSE" >&2 2>/dev/null || tr -cd '\11\12\15\40-\176' < "$LAST_RESPONSE" >&2
    fi
    exit 1
}

require_command() {
    command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_value() {
    local name="$1"
    local value="$2"
    [[ -n "$value" ]] || fail "set $name"
}

request() {
    local method="$1"
    local path="$2"
    shift 2
    LAST_RESPONSE="$TMP_ROOT/response.json"
    : > "$LAST_RESPONSE"
    LAST_STATUS="$(curl --silent --show-error --connect-timeout 5 --max-time 180 \
        --output "$LAST_RESPONSE" --write-out '%{http_code}' --request "$method" \
        "$@" "${BASE_URL}${path}")" || return 1
}

api_json() {
    local method="$1"
    local path="$2"
    local expected="$3"
    local payload="${4:-}"
    local args=(--header "Authorization: Bearer $TOKEN" --header 'Accept: application/json')
    if [[ -n "$payload" ]]; then
        args+=(--header 'Content-Type: application/json' --data "$payload")
    fi
    request "$method" "$path" "${args[@]}" || fail "$method $path request failed"
    [[ "$LAST_STATUS" == "$expected" ]] || fail "$method $path returned HTTP $LAST_STATUS, expected $expected"
}

json_value() {
    jq -er "$1" "$LAST_RESPONSE" || fail "response did not contain $1"
}

cleanup_request() {
    local method="$1"
    local path="$2"
    [[ -n "$TOKEN" ]] || return 0
    curl --silent --show-error --connect-timeout 3 --max-time 30 --output /dev/null \
        --request "$method" --header "Authorization: Bearer $TOKEN" "${BASE_URL}${path}" || true
}

cleanup() {
    local status=$?
    trap - EXIT HUP INT TERM
    set +e
    if [[ "$KEEP_RESOURCES" != "1" ]]; then
        [[ -z "$BACKUP_ID" ]] || cleanup_request DELETE "/api/v1/system/admin/backups/$BACKUP_ID"
        [[ -z "$SESSION_ID" ]] || cleanup_request DELETE "/api/v1/sessions/$SESSION_ID"
        [[ -z "$KNOWLEDGE_ID" ]] || cleanup_request DELETE "/api/v1/knowledge/$KNOWLEDGE_ID"
        [[ -z "$KB_ID" ]] || cleanup_request DELETE "/api/v1/knowledge-bases/$KB_ID"
        [[ -z "$CHAT_MODEL_ID" ]] || cleanup_request DELETE "/api/v1/models/$CHAT_MODEL_ID"
        [[ -z "$EMBEDDING_MODEL_ID" ]] || cleanup_request DELETE "/api/v1/models/$EMBEDDING_MODEL_ID"
        [[ -z "$TAG_ID" ]] || cleanup_request DELETE "/api/v1/system/admin/tag-dictionary/$TAG_ID"
        [[ -z "$DEPARTMENT_ID" ]] || cleanup_request DELETE "/api/v1/tenants/$TENANT_ID/departments/$DEPARTMENT_ID"
        [[ -z "$TEST_USER_ID" ]] || cleanup_request DELETE "/api/v1/tenants/$TENANT_ID/members/$TEST_USER_ID"
    fi
    [[ -z "$TMP_ROOT" ]] || rm -rf -- "$TMP_ROOT"
    exit "$status"
}
trap cleanup EXIT HUP INT TERM

poll_document() {
    local deadline=$((SECONDS + TIMEOUT))
    local status=""
    while ((SECONDS < deadline)); do
        api_json GET "/api/v1/knowledge/$KNOWLEDGE_ID" 200
        status="$(jq -r '.data.parse_status // ""' "$LAST_RESPONSE")"
        case "$status" in
            completed)
                if jq -e '(.data.classification_tags // []) | length > 0' "$LAST_RESPONSE" >/dev/null; then
                    pass "document processing completed with automatic classification tags"
                    return 0
                fi
                ;;
            failed|cancelled)
                fail "document processing entered terminal state: $status"
                ;;
        esac
        sleep "$POLL_INTERVAL"
    done
    fail "timed out waiting for document processing and automatic tags"
}

poll_backup() {
    local deadline=$((SECONDS + TIMEOUT))
    local status=""
    while ((SECONDS < deadline)); do
        api_json GET "/api/v1/system/admin/backups/$BACKUP_ID" 200
        status="$(jq -r '.status // ""' "$LAST_RESPONSE")"
        case "$status" in
            succeeded)
                jq -e '.size_bytes > 0 and (.checksum_sha256 | test("^[0-9a-f]{64}$"))' "$LAST_RESPONSE" >/dev/null \
                    || fail "successful backup has invalid size or checksum"
                pass "database backup completed"
                return 0
                ;;
            failed)
                fail "database backup failed: $(jq -r '.error_message // "unknown error"' "$LAST_RESPONSE")"
                ;;
        esac
        sleep "$POLL_INTERVAL"
    done
    fail "timed out waiting for database backup"
}

poll_audit_log() {
    local deadline=$((SECONDS + TIMEOUT))
    while ((SECONDS < deadline)); do
        api_json GET "/api/v1/tenants/$TENANT_ID/audit-log?action=rbac.member_role_changed&limit=100" 200
        if jq -e --arg user "$TEST_USER_ID" \
            '.success == true and any(.data[]?; .target_user_id == $user and .outcome == "success")' \
            "$LAST_RESPONSE" >/dev/null; then
            pass "tenant audit log verification"
            return 0
        fi
        sleep "$POLL_INTERVAL"
    done
    fail "tenant audit log does not contain the role change"
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    exit 0
fi
[[ $# -eq 0 ]] || { usage >&2; fail "unexpected arguments"; }

for command in bash curl jq mktemp sha256sum stat date grep cut basename; do
    require_command "$command"
done
require_value WEKNORA_E2E_ADMIN_EMAIL "$ADMIN_EMAIL"
require_value WEKNORA_E2E_ADMIN_PASSWORD "$ADMIN_PASSWORD"
require_value WEKNORA_E2E_MODEL_API_KEY "$MODEL_API_KEY"
require_value WEKNORA_E2E_CHAT_MODEL "$CHAT_MODEL_NAME"
require_value WEKNORA_E2E_EMBEDDING_MODEL "$EMBEDDING_MODEL_NAME"
[[ "$TIMEOUT" =~ ^[1-9][0-9]*$ ]] || fail "WEKNORA_E2E_TIMEOUT must be a positive integer"
[[ "$POLL_INTERVAL" =~ ^[1-9][0-9]*$ ]] || fail "WEKNORA_E2E_POLL_INTERVAL must be a positive integer"
[[ "$EMBEDDING_DIMENSION" =~ ^[1-9][0-9]*$ ]] || fail "WEKNORA_E2E_EMBEDDING_DIMENSION must be a positive integer"

TMP_ROOT="$(mktemp -d /tmp/weknora-phase4-e2e.XXXXXX)"
TEST_EMAIL="${RUN_ID}@example.invalid"
TEST_PASSWORD="Phase4-${RUN_ID}-Aa1!"
FIXTURE="$TMP_ROOT/${RUN_ID}.txt"
printf '%s\n' \
    "Phase 4 verification document ${RUN_ID}." \
    'The heliotrope release protocol requires a signed checklist before deployment.' \
    'The audit owner reviews the checklist, and the backup operator verifies its checksum.' \
    > "$FIXTURE"

request GET /health || fail "health check request failed"
[[ "$LAST_STATUS" == 200 ]] || fail "health check returned HTTP $LAST_STATUS"
jq -e '.status == "ok"' "$LAST_RESPONSE" >/dev/null || fail "health check body is invalid"
pass "server health check"

LOGIN_PAYLOAD="$(jq -nc --arg email "$ADMIN_EMAIL" --arg password "$ADMIN_PASSWORD" \
    '{email:$email,password:$password}')"
request POST /api/v1/auth/login --header 'Content-Type: application/json' --data "$LOGIN_PAYLOAD" \
    || fail "admin login request failed"
[[ "$LAST_STATUS" == 200 ]] || fail "admin login returned HTTP $LAST_STATUS"
TOKEN="$(json_value '.token')"
TENANT_ID="$(json_value '.active_tenant.id')"
jq -e '.user.is_system_admin == true' "$LAST_RESPONSE" >/dev/null \
    || fail "admin account is not a System Admin"
pass "admin login as System Admin"

REGISTER_PAYLOAD="$(jq -nc --arg username "$RUN_ID" --arg email "$TEST_EMAIL" --arg password "$TEST_PASSWORD" \
    '{username:$username,email:$email,password:$password}')"
request POST /api/v1/auth/register --header 'Content-Type: application/json' --data "$REGISTER_PAYLOAD" \
    || fail "test user registration request failed"
[[ "$LAST_STATUS" == 201 || "$LAST_STATUS" == 200 ]] \
    || fail "test user registration returned HTTP $LAST_STATUS"
TEST_USER_ID="$(json_value '.user.id')"
pass "test user creation"

ADD_MEMBER_PAYLOAD="$(jq -nc --arg email "$TEST_EMAIL" '{email:$email,role:"viewer"}')"
api_json POST "/api/v1/tenants/$TENANT_ID/members" 201 "$ADD_MEMBER_PAYLOAD"
[[ "$(json_value '.data.user_id')" == "$TEST_USER_ID" ]] || fail "added tenant member ID does not match created user"
api_json PUT "/api/v1/tenants/$TENANT_ID/members/$TEST_USER_ID" 200 '{"role":"contributor"}'
jq -e '.success == true' "$LAST_RESPONSE" >/dev/null || fail "role assignment response is invalid"
pass "tenant role assignment"

DEPARTMENT_PAYLOAD="$(jq -nc --arg name "$RUN_ID" \
    '{parent_id:"",name:$name,description:"Phase 4 E2E department",sort_order:999}')"
api_json POST "/api/v1/tenants/$TENANT_ID/departments" 201 "$DEPARTMENT_PAYLOAD"
DEPARTMENT_ID="$(json_value '.data.id')"
pass "department creation"

CHAT_PAYLOAD="$(jq -nc --arg name "$CHAT_MODEL_NAME" --arg display "$RUN_ID chat" --arg base "$MODEL_BASE_URL" \
    '{name:$name,display_name:$display,type:"KnowledgeQA",source:"remote",description:"Phase 4 E2E chat model",parameters:{base_url:$base,provider:"openai",interface_type:"openai",embedding_parameters:{dimension:0,truncate_prompt_tokens:0,supports_dimension_override:false},supports_vision:false}}')"
api_json POST /api/v1/models 201 "$CHAT_PAYLOAD"
CHAT_MODEL_ID="$(json_value '.data.id')"
api_json PUT "/api/v1/models/$CHAT_MODEL_ID/credentials" 200 "$(jq -nc --arg key "$MODEL_API_KEY" '{api_key:$key}')"

EMBEDDING_PAYLOAD="$(jq -nc --arg name "$EMBEDDING_MODEL_NAME" --arg display "$RUN_ID embedding" \
    --arg base "$MODEL_BASE_URL" --argjson dimension "$EMBEDDING_DIMENSION" \
    '{name:$name,display_name:$display,type:"Embedding",source:"remote",description:"Phase 4 E2E embedding model",parameters:{base_url:$base,provider:"openai",interface_type:"openai",embedding_parameters:{dimension:$dimension,truncate_prompt_tokens:8191,supports_dimension_override:false},supports_vision:false}}')"
api_json POST /api/v1/models 201 "$EMBEDDING_PAYLOAD"
EMBEDDING_MODEL_ID="$(json_value '.data.id')"
api_json PUT "/api/v1/models/$EMBEDDING_MODEL_ID/credentials" 200 "$(jq -nc --arg key "$MODEL_API_KEY" '{api_key:$key}')"
pass "chat and embedding model configuration"

api_json GET /api/v1/system/admin/tag-dimensions 200
DIMENSION_ID="$(jq -er '.[0].id' "$LAST_RESPONSE")" || fail "no automatic-tag dimensions are configured"
api_json GET /api/v1/system/admin/tag-dictionary 200
if ! jq -e '.tags | length > 0' "$LAST_RESPONSE" >/dev/null; then
    TAG_PAYLOAD="$(jq -nc --arg dimension "$DIMENSION_ID" --arg name "$RUN_ID" \
        '{dimension_id:$dimension,name:$name,color:"#5B5BD6",sort_order:999}')"
    api_json POST /api/v1/system/admin/tag-dictionary 201 "$TAG_PAYLOAD"
    TAG_ID="$(json_value '.id')"
    pass "automatic-tag dictionary fixture creation"
else
    pass "automatic-tag dictionary prerequisite"
fi

KB_PAYLOAD="$(jq -nc --arg name "$RUN_ID" --arg embedding "$EMBEDDING_MODEL_ID" --arg chat "$CHAT_MODEL_ID" \
    '{name:$name,description:"Phase 4 E2E knowledge base",type:"document",chunking_config:{chunk_size:500,chunk_overlap:50,separators:["\n\n","\n","."],strategy:"auto"},image_processing_config:{model_id:""},embedding_model_id:$embedding,summary_model_id:$chat,storage_provider_config:{provider:"local"},question_generation_config:{enabled:false,question_count:3},indexing_strategy:{vector_enabled:true,keyword_enabled:true,wiki_enabled:false,graph_enabled:false}}')"
api_json POST /api/v1/knowledge-bases 201 "$KB_PAYLOAD"
KB_ID="$(json_value '.data.id')"
pass "knowledge base creation"

request POST "/api/v1/knowledge-bases/$KB_ID/knowledge/file" \
    --header "Authorization: Bearer $TOKEN" --header 'Accept: application/json' \
    --form "file=@$FIXTURE;type=text/plain" --form "fileName=$(basename "$FIXTURE")" \
    --form 'metadata={"source":"phase4-e2e"}' --form 'channel=api' \
    || fail "document upload request failed"
[[ "$LAST_STATUS" == 200 ]] || fail "document upload returned HTTP $LAST_STATUS"
KNOWLEDGE_ID="$(json_value '.data.id')"
pass "document upload"
poll_document

SEARCH_PAYLOAD="$(jq -nc --arg query heliotrope --arg kb "$KB_ID" --arg knowledge "$KNOWLEDGE_ID" \
    '{query:$query,knowledge_base_ids:[$kb],knowledge_ids:[$knowledge]}')"
api_json POST /api/v1/knowledge-search 200 "$SEARCH_PAYLOAD"
jq -e --arg knowledge "$KNOWLEDGE_ID" '.success == true and any(.data[]?; .knowledge_id == $knowledge)' "$LAST_RESPONSE" >/dev/null \
    || fail "search did not return the uploaded document"
pass "knowledge search"

api_json POST /api/v1/sessions 201 "$(jq -nc --arg title "$RUN_ID" '{title:$title,description:"Phase 4 E2E QA"}')"
SESSION_ID="$(json_value '.data.id')"
QA_PAYLOAD="$(jq -nc --arg query 'What protocol is required before deployment?' --arg kb "$KB_ID" \
    --arg knowledge "$KNOWLEDGE_ID" --arg model "$CHAT_MODEL_ID" \
    '{query:$query,knowledge_base_ids:[$kb],knowledge_ids:[$knowledge],summary_model_id:$model,disable_title:true,channel:"api"}')"
QA_RESPONSE="$TMP_ROOT/qa.sse"
QA_STATUS="$(curl --silent --show-error --connect-timeout 5 --max-time "$TIMEOUT" --output "$QA_RESPONSE" \
    --write-out '%{http_code}' --request POST --header "Authorization: Bearer $TOKEN" \
    --header 'Content-Type: application/json' --header 'Accept: text/event-stream' \
    --data "$QA_PAYLOAD" "${BASE_URL}/api/v1/knowledge-chat/$SESSION_ID")" || fail "QA request failed"
[[ "$QA_STATUS" == 200 ]] || { LAST_RESPONSE="$QA_RESPONSE"; LAST_STATUS="$QA_STATUS"; fail "QA returned HTTP $QA_STATUS"; }
grep -q '"response_type":"answer"' "$QA_RESPONSE" || fail "QA stream did not contain an answer event"
grep -q '"response_type":"complete"' "$QA_RESPONSE" || fail "QA stream did not complete"
pass "retrieval-augmented QA"

poll_audit_log

api_json POST /api/v1/system/admin/backups 202
BACKUP_ID="$(json_value '.id')"
poll_backup
EXPECTED_SIZE="$(json_value '.size_bytes')"
EXPECTED_SHA="$(json_value '.checksum_sha256')"
BACKUP_FILE="$TMP_ROOT/backup.dump"
BACKUP_STATUS="$(curl --silent --show-error --connect-timeout 5 --max-time "$TIMEOUT" --output "$BACKUP_FILE" \
    --write-out '%{http_code}' --header "Authorization: Bearer $TOKEN" \
    "${BASE_URL}/api/v1/system/admin/backups/$BACKUP_ID/download")" || fail "backup download failed"
[[ "$BACKUP_STATUS" == 200 ]] || fail "backup download returned HTTP $BACKUP_STATUS"
[[ "$(stat -c '%s' "$BACKUP_FILE")" == "$EXPECTED_SIZE" ]] || fail "downloaded backup size does not match record"
[[ "$(sha256sum "$BACKUP_FILE" | cut -d ' ' -f 1)" == "$EXPECTED_SHA" ]] || fail "downloaded backup checksum does not match record"
pass "backup download size and SHA-256 verification"

printf 'PASS Phase 4 end-to-end workflow completed at %s\n' "$BASE_URL"
