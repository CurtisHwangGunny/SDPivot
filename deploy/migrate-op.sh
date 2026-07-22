#!/usr/bin/env bash
set -Eeuo pipefail

readonly CORE_LATEST_VERSION=63
readonly CORE_AMBIGUOUS_VERSION=12
readonly SDPIVOT_BASELINE_VERSION=12
readonly SDPIVOT_LATEST_VERSION=14
readonly CORE_MIGRATIONS_DIR=/migrations/versioned
readonly BOOTSTRAP_MIGRATIONS_DIR=/migrations/postgres-bootstrap
readonly SDPIVOT_MIGRATIONS_DIR=/migrations/postgres
readonly PSQL_BIN="${PSQL_BIN:-psql}"
readonly MIGRATE_BIN="${MIGRATE_BIN:-migrate}"

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
    local output

    if ! output="$(PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-10}" \
        "$PSQL_BIN" "$OP_DATABASE_URL" -X -v ON_ERROR_STOP=1 -Atqc "$1" 2>/dev/null)"; then
        printf '[migration] ERROR: database inspection failed; connection details were not logged\n' >&2
        return 1
    fi
    printf '%s\n' "$output"
}

migration_state() {
    local table_name="$1"
    local exists
    local state

    if ! exists="$(sql_scalar "SELECT to_regclass('public.${table_name}') IS NOT NULL")"; then
        return 1
    fi
    if [[ "$exists" != "t" ]]; then
        printf 'absent\n'
        return 0
    fi

    if ! state="$(sql_scalar "SELECT version::text || ':' || dirty::text FROM public.${table_name}")"; then
        return 1
    fi
    printf '%s\n' "$state"
}

assert_clean_version() {
    local table_name="$1"
    local expected_version="$2"
    local state version dirty

    if ! state="$(migration_state "$table_name")"; then
        fail "failed to inspect ${table_name}"
    fi
    [[ "$state" != "absent" ]] || fail "${table_name} was not created"
    [[ "$state" == *:* ]] || fail "${table_name} has an invalid or empty state"

    version="${state%%:*}"
    dirty="${state##*:}"
    [[ "$dirty" == "f" ]] || fail "${table_name} is dirty; automatic force is forbidden"
    [[ "$version" == "$expected_version" ]] || fail \
        "${table_name} version is ${version}; expected ${expected_version}"
}

assert_clean_min_version() {
    local table_name="$1"
    local minimum_version="$2"
    local state version dirty

    if ! state="$(migration_state "$table_name")"; then
        fail "failed to inspect ${table_name}"
    fi
    [[ "$state" != "absent" && "$state" == *:* ]] || fail "${table_name} has an invalid or empty state"
    version="${state%%:*}"
    dirty="${state##*:}"
    [[ "$dirty" == "f" ]] || fail "${table_name} is dirty; automatic force is forbidden"
    [[ "$version" =~ ^[0-9]+$ ]] || fail "${table_name} version is invalid"
    (( version >= minimum_version )) || fail \
        "${table_name} version is ${version}; expected at least ${minimum_version}"
}

run_migrate() {
    local database_url="$1"
    local source_dir="$2"

    if ! "$MIGRATE_BIN" -database "$database_url" -path "$source_dir" up >/dev/null 2>&1; then
        fail "migration command failed for ${source_dir}; connection details were not logged"
    fi
}

with_migrations_table() {
    local table_name="$1"
    if [[ "$OP_DATABASE_URL" == *\?* ]]; then
        printf '%s&x-migrations-table=%s' "$OP_DATABASE_URL" "$table_name"
    else
        printf '%s?x-migrations-table=%s' "$OP_DATABASE_URL" "$table_name"
    fi
}

sdpivot_marker_count() {
    sql_scalar "
        /* op-probe:sdpivot-flags */
        SELECT count(*)
        FROM (VALUES
            ('org_ext'), ('knowledge_spaces'), ('documents'), ('document_chunks'),
            ('write_category_config'), ('sensitive_words'), ('billing_plans')
        ) AS markers(object_name)
        WHERE to_regclass(format('public.%I', object_name)) IS NOT NULL
    "
}

core_v12_fingerprint() {
    sql_scalar "
        /* op-probe:core-v12-fp */
        WITH required_columns(table_name, column_name, expected_type) AS (
            VALUES
                ('kb_shares', 'knowledge_base_id', 'character varying'),
                ('kb_shares', 'organization_id', 'character varying'),
                ('kb_shares', 'source_tenant_id', 'integer'),
                ('organization_join_requests', 'request_type', 'character varying'),
                ('organization_join_requests', 'reviewed_by', 'character varying'),
                ('agent_shares', 'agent_id', 'character varying'),
                ('agent_shares', 'organization_id', 'character varying'),
                ('agent_shares', 'source_tenant_id', 'integer'),
                ('tenant_disabled_shared_agents', 'tenant_id', 'bigint'),
                ('tenant_disabled_shared_agents', 'agent_id', 'character varying'),
                ('tenant_disabled_shared_agents', 'source_tenant_id', 'bigint')
        ), invalid_core AS (
            SELECT 1
            FROM required_columns required
            LEFT JOIN information_schema.columns existing
              ON existing.table_schema = 'public'
             AND existing.table_name = required.table_name
             AND existing.column_name = required.column_name
            WHERE existing.column_name IS NULL
               OR existing.data_type <> required.expected_type
        ), sdpivot_markers(object_name) AS (
            VALUES
                ('org_ext'), ('knowledge_spaces'), ('documents'), ('document_chunks'),
                ('write_category_config'), ('sensitive_words'), ('billing_plans')
        ), core_v13_drift AS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'tenants'
              AND column_name IN ('parser_engine_config', 'storage_engine_config')
        )
        SELECT CASE
            WHEN EXISTS (SELECT 1 FROM invalid_core)
              OR EXISTS (
                    SELECT 1 FROM sdpivot_markers
                    WHERE to_regclass(format('public.%I', object_name)) IS NOT NULL
                 )
              OR EXISTS (SELECT 1 FROM core_v13_drift)
            THEN 'invalid' ELSE 'complete' END
    "
}

sdpivot_v12_fingerprint() {
    sql_scalar "
        /* op-probe:sdpivot-v12 */
        WITH marker_objects(object_name) AS (
            VALUES
                ('org_ext'), ('smartknora_user_profiles'), ('refresh_tokens'),
                ('token_usage'), ('knowledge_spaces'), ('space_members'),
                ('space_categories'), ('documents'), ('document_chunks'),
                ('document_versions'), ('chunk_strategies'), ('qa_sessions'),
                ('qa_messages'), ('writing_drafts'), ('write_category_config'),
                ('announcements'), ('audit_logs'), ('sensitive_words'),
                ('billing_plans'), ('enterprise_subscriptions'), ('invoices'),
                ('sdpivot_brand_migration_000011'), ('sdpivot_rls_migration_000012_state')
        ), required_tables(table_name) AS (
            VALUES
                ('org_ext'), ('smartknora_user_profiles'), ('refresh_tokens'),
                ('token_usage'), ('knowledge_spaces'), ('space_members'),
                ('space_categories'), ('documents'), ('document_chunks'),
                ('document_versions'), ('chunk_strategies'), ('qa_sessions'),
                ('qa_messages'), ('writing_drafts'), ('write_category_config'),
                ('announcements'), ('audit_logs'), ('sensitive_words'),
                ('billing_plans'), ('enterprise_subscriptions'), ('invoices')
        ), existing_markers AS (
            SELECT count(*) AS count
            FROM marker_objects
            WHERE to_regclass(format('public.%I', object_name)) IS NOT NULL
        ), required_columns(table_name, column_name, allowed_types) AS (
            VALUES
                ('users', 'is_ops_admin', ARRAY['boolean']),
                ('users', 'must_change_password', ARRAY['boolean']),
                ('users', 'trial_phase', ARRAY['character varying']),
                ('org_ext', 'org_id', ARRAY['uuid', 'character varying']),
                ('org_ext', 'tenant_id', ARRAY['bigint']),
                ('smartknora_user_profiles', 'user_id', ARRAY['character varying']),
                ('refresh_tokens', 'user_id', ARRAY['character varying']),
                ('refresh_tokens', 'device_id', ARRAY['character varying']),
                ('refresh_tokens', 'family', ARRAY['character varying']),
                ('refresh_tokens', 'expires_at', ARRAY['timestamp with time zone']),
                ('token_usage', 'tenant_id', ARRAY['bigint']),
                ('knowledge_spaces', 'id', ARRAY['uuid', 'character varying']),
                ('knowledge_spaces', 'tenant_id', ARRAY['bigint']),
                ('knowledge_spaces', 'icon', ARRAY['character varying']),
                ('knowledge_spaces', 'creator_id', ARRAY['character varying']),
                ('space_members', 'space_id', ARRAY['character varying']),
                ('space_categories', 'tenant_id', ARRAY['bigint']),
                ('documents', 'id', ARRAY['uuid', 'character varying']),
                ('documents', 'tenant_id', ARRAY['bigint']),
                ('documents', 'space_id', ARRAY['character varying']),
                ('documents', 'parse_status', ARRAY['character varying']),
                ('documents', 'content_hash', ARRAY['character varying']),
                ('document_chunks', 'id', ARRAY['uuid', 'character varying']),
                ('document_chunks', 'document_id', ARRAY['character varying']),
                ('document_chunks', 'tenant_id', ARRAY['bigint']),
                ('document_chunks', 'chunk_index', ARRAY['integer']),
                ('document_chunks', 'content', ARRAY['text']),
                ('document_versions', 'document_id', ARRAY['character varying']),
                ('chunk_strategies', 'tenant_id', ARRAY['bigint']),
                ('qa_sessions', 'user_id', ARRAY['character varying']),
                ('qa_sessions', 'tenant_id', ARRAY['bigint']),
                ('qa_messages', 'session_id', ARRAY['character varying']),
                ('qa_messages', 'tenant_id', ARRAY['bigint']),
                ('qa_messages', 'sources', ARRAY['jsonb']),
                ('writing_drafts', 'user_id', ARRAY['character varying']),
                ('writing_drafts', 'tenant_id', ARRAY['bigint']),
                ('writing_drafts', 'source_type', ARRAY['character varying']),
                ('writing_drafts', 'web_search_enabled', ARRAY['boolean']),
                ('write_category_config', 'tenant_id', ARRAY['bigint']),
                ('write_category_config', 'category', ARRAY['character varying']),
                ('announcements', 'tenant_id', ARRAY['bigint']),
                ('audit_logs', 'tenant_id', ARRAY['bigint']),
                ('audit_logs', 'action', ARRAY['character varying']),
                ('sensitive_words', 'word', ARRAY['character varying']),
                ('sensitive_words', 'status', ARRAY['character varying']),
                ('billing_plans', 'status', ARRAY['character varying']),
                ('enterprise_subscriptions', 'org_id', ARRAY['character varying']),
                ('enterprise_subscriptions', 'plan_id', ARRAY['character varying']),
                ('invoices', 'org_id', ARRAY['character varying']),
                ('invoices', 'status', ARRAY['character varying']),
                ('invoices', 'period_start', ARRAY['timestamp with time zone']),
                ('invoices', 'period_end', ARRAY['timestamp with time zone'])
        ), invalid_columns AS (
            SELECT 1
            FROM required_columns required
            LEFT JOIN information_schema.columns existing
              ON existing.table_schema = 'public'
             AND existing.table_name = required.table_name
             AND existing.column_name = required.column_name
            WHERE existing.column_name IS NULL
               OR NOT (existing.data_type = ANY(required.allowed_types))
        ), id_type_profile AS (
            SELECT
                max(data_type) FILTER (WHERE table_name = 'org_ext' AND column_name = 'org_id') AS org_ext_id,
                max(data_type) FILTER (WHERE table_name = 'knowledge_spaces' AND column_name = 'id') AS space_id,
                max(data_type) FILTER (WHERE table_name = 'documents' AND column_name = 'id') AS document_id,
                max(data_type) FILTER (WHERE table_name = 'document_chunks' AND column_name = 'id') AS chunk_id
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND (table_name, column_name) IN (
                    ('org_ext', 'org_id'),
                    ('knowledge_spaces', 'id'),
                    ('documents', 'id'),
                    ('document_chunks', 'id')
              )
        ), invalid_id_profile AS (
            SELECT 1
            FROM id_type_profile
            WHERE NOT (
                (org_ext_id = 'character varying' AND space_id = 'character varying'
                 AND document_id = 'character varying' AND chunk_id = 'uuid')
                OR
                (org_ext_id = 'character varying' AND space_id = 'character varying'
                 AND document_id = 'character varying' AND chunk_id = 'character varying')
            )
        ), expected_rls(table_name) AS (
            VALUES
                ('write_category_config'), ('knowledge_spaces'), ('documents'),
                ('qa_sessions'), ('qa_messages'), ('writing_drafts'),
                ('announcements'), ('token_usage'), ('document_chunks')
        ), invalid_rls AS (
            SELECT 1
            FROM expected_rls expected
            LEFT JOIN pg_catalog.pg_class target ON target.oid = to_regclass(format('public.%I', expected.table_name))
            WHERE target.oid IS NULL OR NOT target.relrowsecurity
        ), tenant_function AS (
            SELECT p.prorettype = 'void'::regtype AS valid_return
            FROM pg_catalog.pg_proc p
            JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
            WHERE n.nspname = 'public'
              AND p.oid = to_regprocedure('public.set_tenant_context(bigint,boolean)')
        ), current_tenant_function AS (
            SELECT p.prorettype = 'bigint'::regtype AS valid_return
            FROM pg_catalog.pg_proc p
            JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
            WHERE n.nspname = 'public'
              AND p.oid = to_regprocedure('public.get_current_tenant_id()')
        ), ops_function AS (
            SELECT p.prorettype = 'boolean'::regtype AS valid_return
            FROM pg_catalog.pg_proc p
            JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
            WHERE n.nspname = 'public'
              AND p.oid = to_regprocedure('public.is_ops_admin_context()')
        ), chunks_policy AS (
            SELECT policy.polcmd,
                   regexp_replace(
                       regexp_replace(lower(COALESCE(pg_get_expr(policy.polqual, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                       '::text', '', 'g'
                   ) AS using_expression,
                   regexp_replace(
                       regexp_replace(lower(COALESCE(pg_get_expr(policy.polwithcheck, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                       '::text', '', 'g'
                   ) AS check_expression
            FROM pg_catalog.pg_policy policy
            WHERE policy.polrelid = to_regclass('public.document_chunks')
        )
        SELECT CASE
            WHEN (SELECT count FROM existing_markers) = 0
              AND to_regprocedure('public.set_tenant_context(bigint,boolean)') IS NULL
              AND to_regprocedure('public.get_current_tenant_id()') IS NULL
              AND to_regprocedure('public.is_ops_admin_context()') IS NULL
            THEN 'empty'
            WHEN EXISTS (
                    SELECT 1 FROM required_tables
                    WHERE to_regclass(format('public.%I', table_name)) IS NULL
                 )
              OR EXISTS (SELECT 1 FROM invalid_columns)
              OR EXISTS (SELECT 1 FROM invalid_id_profile)
              OR EXISTS (SELECT 1 FROM invalid_rls)
              OR NOT EXISTS (SELECT 1 FROM tenant_function WHERE valid_return)
              OR NOT EXISTS (SELECT 1 FROM current_tenant_function WHERE valid_return)
              OR NOT EXISTS (SELECT 1 FROM ops_function WHERE valid_return)
              OR NOT EXISTS (
                    SELECT 1 FROM pg_catalog.pg_class
                    WHERE oid = to_regclass('public.document_chunks')
                      AND relrowsecurity AND relforcerowsecurity
                 )
              OR NOT EXISTS (
                    SELECT 1 FROM chunks_policy
                    WHERE polcmd = '*'
                      AND using_expression LIKE '%d.id=document_chunks.document_id%'
                      AND using_expression LIKE '%d.tenant_id=document_chunks.tenant_id%'
                      AND using_expression LIKE '%d.tenant_id=get_current_tenant_id%'
                      AND check_expression LIKE '%d.id=document_chunks.document_id%'
                      AND check_expression LIKE '%d.tenant_id=document_chunks.tenant_id%'
                      AND check_expression LIKE '%d.tenant_id=get_current_tenant_id%'
                 )
            THEN 'partial'
            ELSE 'complete' END
    "
}

sdpivot_v13_fingerprint() {
    local baseline
    local state_schema
    local account_state

    if ! baseline="$(sdpivot_v12_fingerprint)"; then
        return 1
    fi
    [[ "$baseline" == "complete" ]] || {
        printf '%s\n' "$baseline"
        return 0
    }
    if ! state_schema="$(sql_scalar "
        /* op-probe:sdpivot-v13-schema */
        WITH required_columns(column_name, data_type, is_nullable) AS (
            VALUES
                ('user_id', 'character varying', 'NO'),
                ('email', 'character varying', 'NO'),
                ('original_password_hash', 'character varying', 'NO'),
                ('original_is_active', 'boolean', 'YES'),
                ('original_must_change_password', 'boolean', 'YES'),
                ('original_is_ops_admin', 'boolean', 'YES'),
                ('original_is_system_admin', 'boolean', 'YES'),
                ('disabled_password_hash', 'character varying', 'NO'),
                ('disabled_at', 'timestamp with time zone', 'NO'),
                ('restored_at', 'timestamp with time zone', 'YES')
        ), invalid_columns AS (
            SELECT 1
            FROM required_columns required
            LEFT JOIN information_schema.columns existing
              ON existing.table_schema = 'public'
             AND existing.table_name = 'sdpivot_disable_legacy_ops_admin_000013_state'
             AND existing.column_name = required.column_name
            WHERE existing.column_name IS NULL
               OR existing.data_type <> required.data_type
               OR existing.is_nullable <> required.is_nullable
        ), invalid_primary_key AS (
            SELECT 1
            WHERE NOT EXISTS (
                SELECT 1
                FROM pg_catalog.pg_constraint key_constraint
                JOIN pg_catalog.pg_class target_table ON target_table.oid = key_constraint.conrelid
                JOIN pg_catalog.pg_namespace target_schema ON target_schema.oid = target_table.relnamespace
                JOIN pg_catalog.pg_attribute key_column
                  ON key_column.attrelid = target_table.oid
                 AND key_column.attname = 'user_id'
                 AND NOT key_column.attisdropped
                WHERE target_schema.nspname = 'public'
                  AND target_table.relname = 'sdpivot_disable_legacy_ops_admin_000013_state'
                  AND key_constraint.contype = 'p'
                  AND key_constraint.conkey = ARRAY[key_column.attnum]::smallint[]
            )
        )
        SELECT CASE
            WHEN to_regclass('public.sdpivot_disable_legacy_ops_admin_000013_state') IS NULL
              OR EXISTS (SELECT 1 FROM invalid_columns)
              OR EXISTS (SELECT 1 FROM invalid_primary_key)
            THEN 'partial' ELSE 'complete' END
    ")"; then
        return 1
    fi
    [[ "$state_schema" == "complete" ]] || {
        printf '%s\n' "$state_schema"
        return 0
    }
    if ! account_state="$(sql_scalar "
        /* op-probe:sdpivot-v13-account */
        WITH remaining_legacy_credential AS (
            SELECT 1
            FROM public.users target
            WHERE target.email = 'admin@smartknora.com'
              AND target.password_hash = '\$2a\$10\$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2'
        ), invalid_state_rows AS (
            SELECT 1
            FROM public.sdpivot_disable_legacy_ops_admin_000013_state state
            LEFT JOIN public.users target ON target.id = state.user_id
            WHERE state.email <> 'admin@smartknora.com'
               OR state.original_password_hash <> '\$2a\$10\$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2'
               OR state.disabled_password_hash <> '!sdpivot-disabled-legacy-ops-admin:' || state.user_id
               OR state.restored_at IS NOT NULL
               OR target.id IS NULL
               OR target.email <> state.email
               OR target.password_hash <> state.disabled_password_hash
               OR target.is_active IS DISTINCT FROM FALSE
               OR target.must_change_password IS DISTINCT FROM TRUE
        )
        SELECT CASE
            WHEN EXISTS (SELECT 1 FROM remaining_legacy_credential)
              OR EXISTS (SELECT 1 FROM invalid_state_rows)
            THEN 'partial' ELSE 'complete' END
    ")"; then
        return 1
    fi
    printf '%s\n' "$account_state"
}

sdpivot_latest_fingerprint() {
    local version13
    local latest

    if ! version13="$(sdpivot_v13_fingerprint)"; then
        return 1
    fi
    [[ "$version13" == "complete" ]] || {
        printf '%s\n' "$version13"
        return 0
    }
    if ! latest="$(sql_scalar "
        /* op-probe:sdpivot-latest */
        WITH expected_policies(table_name, policy_name, is_chunk_policy) AS (
            VALUES
                ('write_category_config', 'sdpivot_secure_000014_write_category_config', FALSE),
                ('knowledge_spaces', 'sdpivot_secure_000014_knowledge_spaces', FALSE),
                ('documents', 'sdpivot_secure_000014_documents', FALSE),
                ('qa_sessions', 'sdpivot_secure_000014_qa_sessions', FALSE),
                ('qa_messages', 'sdpivot_secure_000014_qa_messages', FALSE),
                ('writing_drafts', 'sdpivot_secure_000014_writing_drafts', FALSE),
                ('announcements', 'sdpivot_secure_000014_announcements', FALSE),
                ('token_usage', 'sdpivot_secure_000014_token_usage', FALSE),
                ('document_chunks', 'sdpivot_secure_000014_document_chunks', TRUE)
        ), target_relations AS (
            SELECT expected.*,
                   relation.oid AS relation_id,
                   relation.relrowsecurity,
                   relation.relforcerowsecurity
            FROM expected_policies expected
            LEFT JOIN pg_catalog.pg_class relation
              ON relation.oid = to_regclass(format('public.%I', expected.table_name))
        ), invalid_rls AS (
            SELECT 1
            FROM target_relations
            WHERE relation_id IS NULL
               OR NOT relrowsecurity
               OR (is_chunk_policy AND NOT relforcerowsecurity)
        ), normalized_policies AS (
            SELECT policy.polrelid,
                   policy.polname,
                   policy.polcmd,
                   policy.polpermissive,
                   policy.polroles,
                   regexp_replace(
                       regexp_replace(lower(COALESCE(pg_get_expr(policy.polqual, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                       '::text', '', 'g'
                   ) AS using_expression,
                   regexp_replace(
                       regexp_replace(lower(COALESCE(pg_get_expr(policy.polwithcheck, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                       '::text', '', 'g'
                   ) AS check_expression
            FROM pg_catalog.pg_policy policy
            WHERE policy.polrelid IN (SELECT relation_id FROM target_relations WHERE relation_id IS NOT NULL)
        ), invalid_expected_policies AS (
            SELECT 1
            FROM target_relations target
            LEFT JOIN normalized_policies policy
              ON policy.polrelid = target.relation_id
             AND policy.polname = target.policy_name
            WHERE policy.polname IS NULL
               OR policy.polcmd <> '*'
               OR NOT policy.polpermissive
               OR policy.polroles <> ARRAY[0::oid]
               OR (
                   NOT target.is_chunk_policy
                   AND (policy.using_expression <> 'tenant_id=get_current_tenant_id'
                        OR policy.check_expression <> 'tenant_id=get_current_tenant_id')
               )
               OR (
                   target.is_chunk_policy
                   AND (policy.using_expression NOT LIKE '%d.id=document_chunks.document_id%'
                        OR policy.using_expression NOT LIKE '%d.tenant_id=document_chunks.tenant_id%'
                        OR policy.using_expression NOT LIKE '%d.tenant_id=get_current_tenant_id%'
                        OR policy.using_expression LIKE '%is_ops_admin_context%'
                        OR policy.using_expression LIKE '%or%'
                        OR policy.using_expression LIKE '%true%'
                        OR policy.using_expression LIKE '%case%'
                        OR policy.using_expression LIKE '%coalesce%'
                        OR policy.check_expression NOT LIKE '%d.id=document_chunks.document_id%'
                        OR policy.check_expression NOT LIKE '%d.tenant_id=document_chunks.tenant_id%'
                        OR policy.check_expression NOT LIKE '%d.tenant_id=get_current_tenant_id%'
                        OR policy.check_expression LIKE '%is_ops_admin_context%'
                        OR policy.check_expression LIKE '%or%'
                        OR policy.check_expression LIKE '%true%'
                        OR policy.check_expression LIKE '%case%'
                        OR policy.check_expression LIKE '%coalesce%')
               )
        ), extra_permissive_policies AS (
            SELECT 1
            FROM normalized_policies policy
            JOIN target_relations target ON target.relation_id = policy.polrelid
            WHERE policy.polpermissive
              AND policy.polname <> target.policy_name
        ), tenant_function AS (
            SELECT p.prorettype = 'void'::regtype
                   AND NOT p.prosecdef
                   AND pg_get_functiondef(p.oid) ILIKE '%set_config(''app.is_ops_admin'', ''false'', true)%'
                   AS valid
            FROM pg_catalog.pg_proc p
            WHERE p.oid = to_regprocedure('public.set_tenant_context(bigint,boolean)')
        ), ops_function AS (
            SELECT p.prorettype = 'boolean'::regtype
                   AND NOT p.prosecdef
                   AND regexp_replace(lower(pg_get_functiondef(p.oid)), '[[:space:];]', '', 'g') LIKE '%selectfalse%'
                   AS valid
            FROM pg_catalog.pg_proc p
            WHERE p.oid = to_regprocedure('public.is_ops_admin_context()')
        )
        SELECT CASE
            WHEN EXISTS (SELECT 1 FROM invalid_rls)
              OR EXISTS (SELECT 1 FROM invalid_expected_policies)
              OR EXISTS (SELECT 1 FROM extra_permissive_policies)
              OR NOT EXISTS (SELECT 1 FROM tenant_function WHERE valid)
              OR NOT EXISTS (SELECT 1 FROM ops_function WHERE valid)
            THEN 'partial' ELSE 'complete' END
    ")"; then
        return 1
    fi
    printf '%s\n' "$latest"
}

sdpivot_fingerprint_for_version() {
    local version="$1"
    case "$version" in
        12) sdpivot_v12_fingerprint ;;
        13) sdpivot_v13_fingerprint ;;
        *) sdpivot_latest_fingerprint ;;
    esac
}

require_command "$MIGRATE_BIN"
require_command "$PSQL_BIN"
[[ -n "${OP_DATABASE_URL:-}" ]] || fail "OP_DATABASE_URL is required"
[[ "$OP_DATABASE_URL" == postgres://* || "$OP_DATABASE_URL" == postgresql://* ]] || fail \
    "OP_DATABASE_URL must use the postgres or postgresql scheme"

readonly SDPIVOT_DATABASE_URL="$(with_migrations_table sdpivot_schema_migrations)"

log "checking core migration state"
if ! core_state="$(migration_state schema_migrations)"; then
    fail "failed to inspect schema_migrations"
fi
if [[ "$core_state" == "absent" ]]; then
    if ! existing_public_tables="$(sql_scalar "
        SELECT count(*)
        FROM pg_catalog.pg_tables
        WHERE schemaname = 'public'
          AND tablename NOT IN ('schema_migrations', 'sdpivot_schema_migrations')
    ")"; then
        fail "failed to inspect public application objects"
    fi
    [[ "$existing_public_tables" == "0" ]] || fail \
        "public application objects exist without schema_migrations; environment is ambiguous"

    log "recognized an empty greenfield database; applying core migrations"
    run_migrate "$OP_DATABASE_URL" "$CORE_MIGRATIONS_DIR"
else
    [[ "$core_state" == *:* ]] || fail "schema_migrations has an invalid or empty state"
    core_version="${core_state%%:*}"
    core_dirty="${core_state##*:}"
    [[ "$core_dirty" == "f" ]] || fail "schema_migrations is dirty; automatic force is forbidden"
    [[ "$core_version" =~ ^[0-9]+$ ]] || fail "schema_migrations version is invalid"
    (( core_version <= CORE_LATEST_VERSION )) || fail \
        "schema_migrations version ${core_version} is newer than supported ${CORE_LATEST_VERSION}"

    if (( core_version == CORE_AMBIGUOUS_VERSION )); then
        if ! core_fingerprint="$(core_v12_fingerprint)"; then
            fail "failed to inspect the core version 12 fingerprint"
        fi
        [[ "$core_fingerprint" == "complete" ]] || fail \
            "schema_migrations version 12 cannot be proven to belong exclusively to the core migration chain"
    fi

    if (( core_version < CORE_LATEST_VERSION )); then
        if ! pre_sdpivot_state="$(migration_state sdpivot_schema_migrations)"; then
            fail "failed to inspect sdpivot_schema_migrations before core migration"
        fi
        if ! pre_sdpivot_markers="$(sdpivot_marker_count)"; then
            fail "failed to inspect SDPivot markers before core migration"
        fi
        [[ "$pre_sdpivot_state" == "absent" && "$pre_sdpivot_markers" == "0" ]] || fail \
            "SDPivot objects or ledger exist while core migrations are incomplete; environment is ambiguous"
        log "applying pending core migrations"
        run_migrate "$OP_DATABASE_URL" "$CORE_MIGRATIONS_DIR"
    else
        log "core migrations are already current"
    fi
fi
assert_clean_version schema_migrations "$CORE_LATEST_VERSION"

log "identifying the SDPivot migration path"
if ! sdpivot_state="$(migration_state sdpivot_schema_migrations)"; then
    fail "failed to inspect sdpivot_schema_migrations"
fi
if [[ "$sdpivot_state" != "absent" ]]; then
    [[ "$sdpivot_state" == *:* ]] || fail "sdpivot_schema_migrations has an invalid or empty state"
    sdpivot_version="${sdpivot_state%%:*}"
    sdpivot_dirty="${sdpivot_state##*:}"
    [[ "$sdpivot_dirty" == "f" ]] || fail "sdpivot_schema_migrations is dirty; automatic force is forbidden"
    [[ "$sdpivot_version" =~ ^[0-9]+$ ]] || fail "sdpivot_schema_migrations version is invalid"
    (( sdpivot_version >= SDPIVOT_BASELINE_VERSION )) || fail \
        "sdpivot_schema_migrations version ${sdpivot_version} is older than supported ${SDPIVOT_BASELINE_VERSION}"
    (( sdpivot_version <= SDPIVOT_LATEST_VERSION )) || fail \
        "sdpivot_schema_migrations version ${sdpivot_version} is newer than supported ${SDPIVOT_LATEST_VERSION}"
fi

if [[ "$sdpivot_state" == "absent" ]]; then
    if ! sdpivot_fingerprint="$(sdpivot_v12_fingerprint)"; then
        fail "failed to inspect the SDPivot version 12 structure fingerprint"
    fi
    case "$sdpivot_fingerprint" in
        empty)
            log "recognized a greenfield SDPivot database; applying the secure baseline"
            ;;
        complete)
            log "recognized complete SDPivot version 12 objects without a dedicated ledger; adopting through the idempotent baseline"
            ;;
        *)
            fail "SDPivot objects without sdpivot_schema_migrations are partial or drifted"
            ;;
    esac
    run_migrate "$SDPIVOT_DATABASE_URL" "$BOOTSTRAP_MIGRATIONS_DIR"
    assert_clean_version sdpivot_schema_migrations "$SDPIVOT_BASELINE_VERSION"
    sdpivot_version="$SDPIVOT_BASELINE_VERSION"
else
    if ! sdpivot_fingerprint="$(sdpivot_fingerprint_for_version "$sdpivot_version")"; then
        fail "failed to inspect the SDPivot version ${sdpivot_version} structure fingerprint"
    fi
    [[ "$sdpivot_fingerprint" == "complete" ]] || fail \
        "sdpivot_schema_migrations version ${sdpivot_version} objects are partial or drifted"
    log "recognized an existing SDPivot database at version ${sdpivot_version}"
fi

if (( sdpivot_version < SDPIVOT_LATEST_VERSION )); then
    log "applying pending SDPivot security migrations"
    run_migrate "$SDPIVOT_DATABASE_URL" "$SDPIVOT_MIGRATIONS_DIR"
    assert_clean_version sdpivot_schema_migrations "$SDPIVOT_LATEST_VERSION"
else
    log "SDPivot migrations are already current"
    assert_clean_min_version sdpivot_schema_migrations "$SDPIVOT_LATEST_VERSION"
fi
if ! sdpivot_fingerprint="$(sdpivot_latest_fingerprint)"; then
    fail "failed to inspect the latest SDPivot security fingerprint"
fi
[[ "$sdpivot_fingerprint" == "complete" ]] || fail "latest SDPivot security objects are partial or drifted"

log "migration orchestration completed successfully"
