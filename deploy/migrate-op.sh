#!/usr/bin/env bash
set -Eeuo pipefail

readonly CORE_LATEST_VERSION=63
readonly SDPIVOT_BASELINE_VERSION=12
readonly SDPIVOT_LATEST_VERSION=14
readonly CORE_MIGRATIONS_DIR=/migrations/versioned
readonly BOOTSTRAP_MIGRATIONS_DIR=/migrations/postgres-bootstrap
readonly SDPIVOT_MIGRATIONS_DIR=/migrations/postgres

log() {
    printf '[migration] %s\n' "$*"
}

fail() {
    printf '[migration] ERROR: %s\n' "$*" >&2
    exit 1
}

on_error() {
    printf '[migration] ERROR: migration orchestration stopped; database credentials were not logged.\n' >&2
}
trap on_error ERR

require_command() {
    command -v "$1" >/dev/null 2>&1 || fail "required command is unavailable: $1"
}

sql_scalar() {
    PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-10}" \
        psql "$OP_DATABASE_URL" -X -v ON_ERROR_STOP=1 -Atqc "$1"
}

migration_state() {
    local table_name="$1"
    local exists

    exists="$(sql_scalar "SELECT to_regclass('public.${table_name}') IS NOT NULL")"
    if [[ "$exists" != "t" ]]; then
        printf 'absent\n'
        return
    fi

    sql_scalar "SELECT version::text || ':' || dirty::text FROM public.${table_name}"
}

assert_clean_version() {
    local table_name="$1"
    local expected_version="$2"
    local state version dirty

    state="$(migration_state "$table_name")"
    [[ "$state" != "absent" ]] || fail "${table_name} was not created"
    [[ "$state" == *:* ]] || fail "${table_name} has an invalid or empty state"

    version="${state%%:*}"
    dirty="${state##*:}"
    [[ "$dirty" == "f" ]] || fail "${table_name} is dirty; automatic force is forbidden"
    [[ "$version" == "$expected_version" ]] || fail \
        "${table_name} version is ${version}; expected ${expected_version}"
}

run_migrate() {
    local database_url="$1"
    local source_dir="$2"

    migrate -database "$database_url" -path "$source_dir" up
}

with_migrations_table() {
    local table_name="$1"
    if [[ "$OP_DATABASE_URL" == *\?* ]]; then
        printf '%s&x-migrations-table=%s' "$OP_DATABASE_URL" "$table_name"
    else
        printf '%s?x-migrations-table=%s' "$OP_DATABASE_URL" "$table_name"
    fi
}

require_command migrate
require_command psql
[[ -n "${OP_DATABASE_URL:-}" ]] || fail "OP_DATABASE_URL is required"
[[ "$OP_DATABASE_URL" == postgres://* || "$OP_DATABASE_URL" == postgresql://* ]] || fail \
    "OP_DATABASE_URL must use the postgres or postgresql scheme"

readonly SDPIVOT_DATABASE_URL="$(with_migrations_table sdpivot_schema_migrations)"

log "checking core migration state"
core_state="$(migration_state schema_migrations)"
if [[ "$core_state" == "absent" ]]; then
    existing_public_tables="$(sql_scalar "
        SELECT count(*)
        FROM pg_catalog.pg_tables
        WHERE schemaname = 'public'
          AND tablename NOT IN ('schema_migrations', 'sdpivot_schema_migrations')
    ")"
    [[ "$existing_public_tables" == "0" ]] || fail \
        "public application objects exist without schema_migrations; environment is ambiguous"

    log "recognized an empty greenfield database; applying core migrations"
    run_migrate "$OP_DATABASE_URL" "$CORE_MIGRATIONS_DIR"
else
    [[ "$core_state" == *:* ]] || fail "schema_migrations has an invalid or empty state"
    core_version="${core_state%%:*}"
    core_dirty="${core_state##*:}"
    [[ "$core_dirty" == "f" ]] || fail \
        "schema_migrations is dirty; automatic force is forbidden"
    [[ "$core_version" =~ ^[0-9]+$ ]] || fail "schema_migrations version is invalid"
    (( core_version <= CORE_LATEST_VERSION )) || fail \
        "schema_migrations version ${core_version} is newer than supported ${CORE_LATEST_VERSION}"

    log "applying pending core migrations"
    run_migrate "$OP_DATABASE_URL" "$CORE_MIGRATIONS_DIR"
fi
assert_clean_version schema_migrations "$CORE_LATEST_VERSION"

log "identifying the SDPivot migration path"
sdpivot_state="$(migration_state sdpivot_schema_migrations)"
sdpivot_object_count="$(sql_scalar "
    SELECT count(*)
    FROM pg_catalog.pg_class c
    JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public'
      AND c.relkind IN ('r', 'p')
      AND c.relname IN (
          'org_ext',
          'knowledge_spaces',
          'documents',
          'document_chunks',
          'qa_sessions',
          'writing_drafts'
      )
")"

if [[ "$sdpivot_state" == "absent" ]]; then
    [[ "$sdpivot_object_count" == "0" ]] || fail \
        "SDPivot objects exist without sdpivot_schema_migrations; environment is ambiguous"

    log "recognized a greenfield SDPivot database; applying the secure baseline"
    run_migrate "$SDPIVOT_DATABASE_URL" "$BOOTSTRAP_MIGRATIONS_DIR"
    assert_clean_version sdpivot_schema_migrations "$SDPIVOT_BASELINE_VERSION"
else
    [[ "$sdpivot_state" == *:* ]] || fail \
        "sdpivot_schema_migrations has an invalid or empty state"
    sdpivot_version="${sdpivot_state%%:*}"
    sdpivot_dirty="${sdpivot_state##*:}"
    [[ "$sdpivot_dirty" == "f" ]] || fail \
        "sdpivot_schema_migrations is dirty; automatic force is forbidden"
    [[ "$sdpivot_object_count" -gt 0 ]] || fail \
        "sdpivot_schema_migrations exists without recognizable SDPivot objects"
    [[ "$sdpivot_version" =~ ^(12|13|14)$ ]] || fail \
        "sdpivot_schema_migrations version ${sdpivot_version} is unsupported; expected 12, 13, or 14"

    log "recognized an existing SDPivot database at version ${sdpivot_version}"
fi

log "applying SDPivot security migrations 13 and 14"
run_migrate "$SDPIVOT_DATABASE_URL" "$SDPIVOT_MIGRATIONS_DIR"
assert_clean_version sdpivot_schema_migrations "$SDPIVOT_LATEST_VERSION"

log "migration orchestration completed successfully"
