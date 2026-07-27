#!/usr/bin/env bash
set -Eeuo pipefail

readonly CORE_LATEST_VERSION=72
readonly CORE_AMBIGUOUS_VERSION=12
readonly SDPIVOT_BASELINE_VERSION=12
readonly SDPIVOT_LATEST_VERSION=17
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

    if ! state="$(sql_scalar "SELECT version::text || ':' || CASE WHEN dirty THEN 't' ELSE 'f' END FROM public.${table_name}")"; then
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
    local diagnostics authority userinfo username password

    if ! diagnostics="$("$MIGRATE_BIN" -verbose -database "$database_url" -path "$source_dir" up 2>&1)"; then
        diagnostics="${diagnostics//"$database_url"/[REDACTED_DATABASE_URL]}"
        authority="${database_url#*://}"
        authority="${authority%%/*}"
        if [[ "$authority" == *@* ]]; then
            userinfo="${authority%@*}"
            username="${userinfo%%:*}"
            password="${userinfo#*:}"
            [[ -z "$username" ]] || diagnostics="${diagnostics//"$username"/[REDACTED_DATABASE_USER]}"
            [[ "$password" == "$userinfo" || -z "$password" ]] || \
                diagnostics="${diagnostics//"$password"/[REDACTED_DATABASE_PASSWORD]}"
        fi
        if [[ -n "$diagnostics" ]]; then
            printf '[migration] migrate diagnostics for %s:\n%s\n' "$source_dir" "$diagnostics" >&2
        fi
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

core_audit_m44_fingerprint() {
    sql_scalar "
        /* op-probe:core-audit-m44 */
        WITH target AS (
            SELECT c.oid AS table_oid, c.relkind, c.relpersistence,
                   pg_catalog.pg_get_serial_sequence('public.audit_logs', 'id')::regclass AS sequence_oid
            FROM pg_catalog.pg_class c
            JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname = 'public' AND c.relname = 'audit_logs'
        ), expected_columns(attname, attnum, atttypid, atttypmod, attnotnull, default_kind) AS (
            VALUES
                ('id', 1, 'bigint'::regtype, -1, TRUE, 'sequence'),
                ('tenant_id', 2, 'bigint'::regtype, -1, TRUE, 'none'),
                ('actor_user_id', 3, 'character varying'::regtype, 40, TRUE, 'empty_varchar'),
                ('actor_role', 4, 'character varying'::regtype, 36, TRUE, 'empty_varchar'),
                ('action', 5, 'character varying'::regtype, 68, TRUE, 'none'),
                ('target_type', 6, 'character varying'::regtype, 36, TRUE, 'empty_varchar'),
                ('target_id', 7, 'character varying'::regtype, 68, TRUE, 'empty_varchar'),
                ('target_user_id', 8, 'character varying'::regtype, 40, TRUE, 'empty_varchar'),
                ('request_path', 9, 'character varying'::regtype, 516, TRUE, 'empty_varchar'),
                ('request_method', 10, 'character varying'::regtype, 20, TRUE, 'empty_varchar'),
                ('outcome', 11, 'character varying'::regtype, 20, TRUE, 'success_varchar'),
                ('details', 12, 'jsonb'::regtype, -1, TRUE, 'empty_jsonb'),
                ('created_at', 13, 'timestamp with time zone'::regtype, -1, TRUE, 'current_timestamp')
        ), expected_projection(attname, attnum, atttypid, atttypmod) AS (
            VALUES
                ('user_id', 14, 'character varying'::regtype, 40),
                ('username', 15, 'character varying'::regtype, 104),
                ('resource', 16, 'character varying'::regtype, 104),
                ('resource_id', 17, 'character varying'::regtype, 68),
                ('detail', 18, 'text'::regtype, -1),
                ('ip', 19, 'character varying'::regtype, 54)
        ), expected_current(attname, attnum, atttypid, atttypmod) AS (
            VALUES
                ('user_id', 14, 'character varying'::regtype, 40),
                ('resource_type', 15, 'character varying'::regtype, 36),
                ('resource_id', 16, 'character varying'::regtype, 68),
                ('ip_address', 17, 'character varying'::regtype, 49)
        ), expected_current_projection(attname, attnum, atttypid, atttypmod) AS (
            VALUES
                ('username', 18, 'character varying'::regtype, 104),
                ('resource', 19, 'character varying'::regtype, 104),
                ('detail', 20, 'text'::regtype, -1),
                ('ip', 21, 'character varying'::regtype, 54)
        ), actual_columns AS (
            SELECT a.attname, a.attnum, a.atttypid, a.atttypmod, a.attnotnull, a.attidentity,
                   d.oid AS default_oid, pg_catalog.pg_get_expr(d.adbin, d.adrelid) AS default_expr
            FROM target t
            JOIN pg_catalog.pg_attribute a ON a.attrelid = t.table_oid
            LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
            WHERE a.attnum > 0 AND NOT a.attisdropped
        ), invalid_core_columns AS (
            SELECT 1
            FROM expected_columns e
            LEFT JOIN actual_columns a ON a.attname = e.attname
            LEFT JOIN target t ON TRUE
            WHERE a.attname IS NULL
               OR a.attnum <> e.attnum OR a.atttypid <> e.atttypid OR a.atttypmod <> e.atttypmod
               OR a.attnotnull <> e.attnotnull
               OR CASE e.default_kind
                    WHEN 'none' THEN a.default_oid IS NOT NULL
                    WHEN 'sequence' THEN a.attidentity <> '' OR t.sequence_oid IS NULL OR a.default_oid IS NULL
                        OR a.default_expr IS DISTINCT FROM pg_catalog.format(
                            'nextval(%L::regclass)', t.sequence_oid::regclass::text)
                        OR NOT EXISTS (
                            SELECT 1 FROM pg_catalog.pg_depend dep
                            WHERE dep.classid = 'pg_catalog.pg_attrdef'::regclass
                              AND dep.objid = a.default_oid AND dep.objsubid = 0
                              AND dep.refclassid = 'pg_catalog.pg_class'::regclass
                              AND dep.refobjid = t.sequence_oid AND dep.refobjsubid = 0
                              AND dep.deptype = 'n')
                    WHEN 'empty_varchar' THEN a.default_expr IS DISTINCT FROM chr(39) || chr(39) || '::character varying'
                    WHEN 'success_varchar' THEN a.default_expr IS DISTINCT FROM chr(39) || 'success' || chr(39) || '::character varying'
                    WHEN 'empty_jsonb' THEN a.default_expr IS DISTINCT FROM '''{}''::jsonb'
                    WHEN 'current_timestamp' THEN regexp_replace(lower(a.default_expr), '[[:space:]()]', '', 'g')
                        NOT IN ('current_timestamp', 'now')
                    ELSE TRUE
                  END
        ), invalid_projection_columns AS (
            SELECT 1
            FROM expected_projection e
            LEFT JOIN actual_columns a ON a.attname = e.attname
            WHERE a.attname IS NULL OR a.attnum <> e.attnum OR a.atttypid <> e.atttypid
               OR a.atttypmod <> e.atttypmod OR a.attnotnull OR a.default_oid IS NOT NULL
        ), invalid_current_columns AS (
            SELECT 1
            FROM expected_current e
            LEFT JOIN actual_columns a ON a.attname = e.attname
            WHERE a.attname IS NULL OR a.attnum <> e.attnum OR a.atttypid <> e.atttypid
               OR a.atttypmod <> e.atttypmod OR NOT a.attnotnull
               OR a.default_expr IS DISTINCT FROM chr(39) || chr(39) || '::character varying'
        ), invalid_current_projection_columns AS (
            SELECT 1
            FROM expected_current_projection e
            LEFT JOIN actual_columns a ON a.attname = e.attname
            WHERE a.attname IS NULL OR a.attnum <> e.attnum OR a.atttypid <> e.atttypid
               OR a.atttypmod <> e.atttypmod OR a.attnotnull OR a.default_oid IS NOT NULL
        ), invalid_sequence AS (
            SELECT 1
            FROM target t
            LEFT JOIN actual_columns id ON id.attname = 'id'
            WHERE t.sequence_oid IS NULL OR id.attidentity <> ''
               OR NOT EXISTS (
                    SELECT 1 FROM pg_catalog.pg_class s
                    JOIN pg_catalog.pg_sequence p ON p.seqrelid = s.oid
                    WHERE s.oid = t.sequence_oid AND s.relkind = 'S' AND s.relpersistence = 'p'
                      AND p.seqtypid = 'bigint'::regtype
                      AND p.seqstart = 1 AND p.seqincrement = 1
                      AND p.seqmax = 9223372036854775807 AND p.seqmin = 1
                      AND p.seqcache = 1 AND NOT p.seqcycle)
               OR NOT EXISTS (
                    SELECT 1 FROM pg_catalog.pg_depend dep
                    WHERE dep.classid = 'pg_catalog.pg_class'::regclass
                      AND dep.objid = t.sequence_oid AND dep.objsubid = 0
                      AND dep.refclassid = 'pg_catalog.pg_class'::regclass
                      AND dep.refobjid = t.table_oid AND dep.refobjsubid = id.attnum
                      AND dep.deptype = 'a')
        ), invalid_primary_key AS (
            SELECT 1 FROM target t
            WHERE 1 <> (SELECT count(*) FROM pg_catalog.pg_constraint c WHERE c.conrelid = t.table_oid AND c.contype = 'p')
               OR NOT EXISTS (
                    SELECT 1 FROM pg_catalog.pg_constraint c
                    JOIN actual_columns id ON id.attname = 'id'
                    WHERE c.conrelid = t.table_oid AND c.contype = 'p'
                      AND c.convalidated AND NOT c.condeferrable AND NOT c.condeferred
                      AND c.conkey = ARRAY[id.attnum]::smallint[])
        ), expected_indexes(kind, key_count) AS (
            VALUES ('tenant_id_id_desc', 2), ('actor_user_id', 1), ('tenant_id_action', 2), ('created_at', 1)
        ), matching_indexes AS (
            SELECT e.kind, i.indexrelid
            FROM expected_indexes e
            JOIN target t ON TRUE
            JOIN pg_catalog.pg_index i ON i.indrelid = t.table_oid
            JOIN pg_catalog.pg_class idx ON idx.oid = i.indexrelid
            JOIN pg_catalog.pg_am am ON am.oid = idx.relam AND am.amname = 'btree'
            JOIN actual_columns c1 ON c1.attnum = i.indkey[0]
            LEFT JOIN actual_columns c2 ON c2.attnum = i.indkey[1]
            WHERE idx.relkind = 'i' AND i.indisvalid AND i.indisready AND i.indislive
              AND NOT i.indisunique AND NOT i.indisprimary
              AND i.indpred IS NULL AND i.indexprs IS NULL
              AND i.indnkeyatts = e.key_count AND i.indnatts = e.key_count
              AND pg_catalog.pg_index_column_has_property(i.indexrelid, 1, 'orderable')
              AND pg_catalog.pg_index_column_has_property(i.indexrelid, 1, 'asc')
              AND pg_catalog.pg_index_column_has_property(i.indexrelid, 1, 'nulls_last')
              AND (e.key_count = 1 OR (
                  pg_catalog.pg_index_column_has_property(i.indexrelid, 2, 'orderable')
                  AND CASE e.kind
                      WHEN 'tenant_id_id_desc' THEN c2.attname = 'id'
                          AND pg_catalog.pg_index_column_has_property(i.indexrelid, 2, 'desc')
                          AND pg_catalog.pg_index_column_has_property(i.indexrelid, 2, 'nulls_first')
                      WHEN 'tenant_id_action' THEN c2.attname = 'action'
                          AND pg_catalog.pg_index_column_has_property(i.indexrelid, 2, 'asc')
                          AND pg_catalog.pg_index_column_has_property(i.indexrelid, 2, 'nulls_last')
                      ELSE FALSE END))
              AND CASE e.kind
                  WHEN 'tenant_id_id_desc' THEN c1.attname = 'tenant_id'
                  WHEN 'actor_user_id' THEN c1.attname = 'actor_user_id'
                  WHEN 'tenant_id_action' THEN c1.attname = 'tenant_id'
                  WHEN 'created_at' THEN c1.attname = 'created_at'
                  ELSE FALSE END
        ), invalid_indexes AS (
            SELECT 1 FROM expected_indexes e
            WHERE NOT EXISTS (SELECT 1 FROM matching_indexes m WHERE m.kind = e.kind)
        )
        SELECT CASE
            WHEN NOT EXISTS (SELECT 1 FROM target) THEN 'missing'
            WHEN NOT EXISTS (SELECT 1 FROM target WHERE relkind = 'r' AND relpersistence = 'p')
              OR EXISTS (SELECT 1 FROM invalid_core_columns)
              OR EXISTS (SELECT 1 FROM invalid_sequence)
              OR EXISTS (SELECT 1 FROM invalid_primary_key)
              OR EXISTS (SELECT 1 FROM invalid_indexes)
              OR (SELECT count(*) FROM actual_columns) NOT IN (13, 17, 19, 21)
            THEN 'invalid'
            WHEN (SELECT count(*) FROM actual_columns) = 13 THEN 'migration44_exact'
            WHEN (SELECT count(*) FROM actual_columns) = 17
              AND NOT EXISTS (SELECT 1 FROM invalid_current_columns) THEN 'core_current_exact'
            WHEN (SELECT count(*) FROM actual_columns) = 19
              AND NOT EXISTS (SELECT 1 FROM invalid_projection_columns) THEN 'baseline_exact'
            WHEN (SELECT count(*) FROM actual_columns) = 21
              AND NOT EXISTS (SELECT 1 FROM invalid_current_columns)
              AND NOT EXISTS (SELECT 1 FROM invalid_current_projection_columns) THEN 'baseline_current_exact'
            ELSE 'invalid'
        END
    "
}

sdpivot_marker_count() {
    sql_scalar "
        /* op-probe:sdpivot-flags */
        SELECT count(*)
        FROM (VALUES
            ('org_ext'), ('knowledge_spaces'), ('documents'), ('document_chunks'),
            ('write_category_config'), ('sensitive_words')
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
                ('write_category_config'), ('sensitive_words')
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
                ('announcements'), ('sensitive_words'),
                ('sdpivot_brand_migration_000011'), ('sdpivot_rls_migration_000012_state')
        ), required_tables(table_name) AS (
            VALUES
                ('org_ext'), ('smartknora_user_profiles'), ('refresh_tokens'),
                ('token_usage'), ('knowledge_spaces'), ('space_members'),
                ('space_categories'), ('documents'), ('document_chunks'),
                ('document_versions'), ('chunk_strategies'), ('qa_sessions'),
                ('qa_messages'), ('writing_drafts'), ('write_category_config'),
                ('announcements'), ('audit_logs'), ('sensitive_words')
        ), existing_markers AS (
            SELECT count(*) AS count
            FROM marker_objects
            WHERE object_name <> 'audit_logs'
              AND to_regclass(format('public.%I', object_name)) IS NOT NULL
        ), core_audit_required(column_name, allowed_types, is_nullable) AS (
            VALUES
                ('id', ARRAY['bigint'], 'NO'),
                ('tenant_id', ARRAY['bigint'], 'NO'),
                ('actor_user_id', ARRAY['character varying'], 'NO'),
                ('actor_role', ARRAY['character varying'], 'NO'),
                ('action', ARRAY['character varying'], 'NO'),
                ('target_type', ARRAY['character varying'], 'NO'),
                ('target_id', ARRAY['character varying'], 'NO'),
                ('target_user_id', ARRAY['character varying'], 'NO'),
                ('request_path', ARRAY['character varying'], 'NO'),
                ('request_method', ARRAY['character varying'], 'NO'),
                ('outcome', ARRAY['character varying'], 'NO'),
                ('details', ARRAY['jsonb'], 'NO'),
                ('created_at', ARRAY['timestamp with time zone'], 'NO')
        ), sdpivot_audit_required(column_name, allowed_types, is_nullable) AS (
            VALUES
                ('user_id', ARRAY['character varying'], 'YES'),
                ('username', ARRAY['character varying'], 'YES'),
                ('resource', ARRAY['character varying'], 'YES'),
                ('resource_id', ARRAY['character varying'], 'YES'),
                ('detail', ARRAY['text'], 'YES'),
                ('ip', ARRAY['character varying'], 'YES')
        ), core_audit_invalid AS (
            SELECT 1
            FROM core_audit_required required
            LEFT JOIN information_schema.columns existing
              ON existing.table_schema = 'public'
             AND existing.table_name = 'audit_logs'
             AND existing.column_name = required.column_name
            WHERE existing.column_name IS NULL
               OR NOT (existing.data_type = ANY(required.allowed_types))
               OR existing.is_nullable <> required.is_nullable
        ), sdpivot_audit_invalid AS (
            SELECT 1
            FROM sdpivot_audit_required required
            LEFT JOIN information_schema.columns existing
              ON existing.table_schema = 'public'
             AND existing.table_name = 'audit_logs'
             AND existing.column_name = required.column_name
            WHERE existing.column_name IS NULL
               OR NOT (existing.data_type = ANY(required.allowed_types))
               OR existing.is_nullable <> required.is_nullable
        ), audit_column_profile AS (
            SELECT
                count(*) FILTER (WHERE existing.column_name IN (SELECT column_name FROM core_audit_required)) AS core_count,
                count(*) FILTER (WHERE existing.column_name IN (SELECT column_name FROM sdpivot_audit_required)) AS sdpivot_count,
                count(*) AS total_count
            FROM information_schema.columns existing
            WHERE existing.table_schema = 'public'
              AND existing.table_name = 'audit_logs'
        ), core_audit_shape AS (
            SELECT CASE
                WHEN to_regclass('public.audit_logs') IS NULL THEN 'missing'
                WHEN NOT EXISTS (
                        SELECT 1 FROM pg_catalog.pg_class target
                        WHERE target.oid = to_regclass('public.audit_logs') AND target.relkind IN ('r', 'p')
                     )
                  OR EXISTS (SELECT 1 FROM core_audit_invalid)
                  OR (SELECT core_count FROM audit_column_profile) <> 13
                  OR (SELECT total_count FROM audit_column_profile) NOT IN (13, 17, 19)
                THEN 'invalid'
                WHEN (SELECT sdpivot_count FROM audit_column_profile) = 0
                  AND (SELECT total_count FROM audit_column_profile) IN (13, 17)
                THEN 'core_exact'
                WHEN (SELECT sdpivot_count FROM audit_column_profile) = 6
                  AND (SELECT total_count FROM audit_column_profile) = 19
                  AND NOT EXISTS (SELECT 1 FROM sdpivot_audit_invalid)
                THEN 'baseline_exact'
                ELSE 'invalid'
            END AS state
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
                   replace(
                       regexp_replace(
                           regexp_replace(lower(COALESCE(pg_get_expr(policy.polqual, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                           '::text', '', 'g'
                       ),
                       'public.', ''
                   ) AS using_expression,
                   replace(
                       regexp_replace(
                           regexp_replace(lower(COALESCE(pg_get_expr(policy.polwithcheck, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                           '::text', '', 'g'
                       ),
                       'public.', ''
                   ) AS check_expression
            FROM pg_catalog.pg_policy policy
            WHERE policy.polrelid = to_regclass('public.document_chunks')
        )
        SELECT CASE
            WHEN (SELECT count FROM existing_markers) = 0
              AND (SELECT state FROM core_audit_shape) = 'core_exact'
            THEN 'empty'
            WHEN EXISTS (
                    SELECT 1 FROM required_tables
                    WHERE to_regclass(format('public.%I', table_name)) IS NULL
                 )
              OR EXISTS (SELECT 1 FROM invalid_columns)
              OR (SELECT state FROM core_audit_shape) <> 'baseline_exact'
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
    local require_op_admin="${1:-true}"
    local version13
    local latest

    [[ "$require_op_admin" == "true" || "$require_op_admin" == "false" ]] || return 1

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
                   replace(
                       regexp_replace(
                           regexp_replace(lower(COALESCE(pg_get_expr(policy.polqual, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                           '::text', '', 'g'
                       ),
                       'public.', ''
                   ) AS using_expression,
                   replace(
                       regexp_replace(
                           regexp_replace(lower(COALESCE(pg_get_expr(policy.polwithcheck, policy.polrelid), '')), '[[:space:]()]', '', 'g'),
                           '::text', '', 'g'
                       ),
                       'public.', ''
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
                   AND (policy.using_expression <> 'existsselect1fromdocumentsdwhered.id=document_chunks.document_idandd.tenant_id=document_chunks.tenant_idandd.tenant_id=get_current_tenant_id'
                        OR policy.check_expression <> 'existsselect1fromdocumentsdwhered.id=document_chunks.document_idandd.tenant_id=document_chunks.tenant_idandd.tenant_id=get_current_tenant_id')
               )
        ), extra_permissive_policies AS (
            SELECT 1
            FROM normalized_policies policy
            JOIN target_relations target ON target.relation_id = policy.polrelid
            WHERE policy.polpermissive
              AND policy.polname <> target.policy_name
        ), op_admin_schema AS (
            SELECT COUNT(*) = 2 AS valid
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'users'
              AND column_name IN ('access_role', 'department_id')
        ), invalid_op_admins AS (
            SELECT 1
            FROM users u
            WHERE (u.is_ops_admin OR u.is_system_admin)
              AND (
                  u.tenant_id <> 1
                  OR u.can_access_all_tenants
                  OR COALESCE(to_jsonb(u)->>'access_role', '') <> 'super_admin'
                  OR (u.deleted_at IS NULL AND u.is_active AND NOT EXISTS (
                      SELECT 1
                      FROM tenant_members tm
                      WHERE tm.user_id = u.id
                        AND tm.tenant_id = 1
                        AND tm.role = 'owner'
                        AND tm.status = 'active'
                        AND tm.deleted_at IS NULL
                  ))
              )
        ), tenant_function AS (
            SELECT p.prorettype = 'void'::regtype
                   AND NOT p.prosecdef
                   AND p.proconfig = ARRAY['search_path=pg_catalog, public']
                   AND regexp_replace(lower(p.prosrc), '[[:space:]]', '', 'g') =
                       'beginperformpg_catalog.set_config(''app.current_tenant_id'',p_tenant_id::text,true);performpg_catalog.set_config(''app.is_ops_admin'',''false'',true);end;'
                   AS valid
            FROM pg_catalog.pg_proc p
            WHERE p.oid = to_regprocedure('public.set_tenant_context(bigint,boolean)')
        ), ops_function AS (
            SELECT p.prorettype = 'boolean'::regtype
                   AND NOT p.prosecdef
                   AND p.proconfig = ARRAY['search_path=pg_catalog, public']
                   AND regexp_replace(lower(p.prosrc), '[[:space:]]', '', 'g') = 'selectfalse;'
                   AS valid
            FROM pg_catalog.pg_proc p
            WHERE p.oid = to_regprocedure('public.is_ops_admin_context()')
        )
        SELECT CASE
            WHEN EXISTS (SELECT 1 FROM invalid_rls)
              OR EXISTS (SELECT 1 FROM invalid_expected_policies)
              OR EXISTS (SELECT 1 FROM extra_permissive_policies)
              OR (${require_op_admin} AND NOT EXISTS (SELECT 1 FROM op_admin_schema WHERE valid))
              OR (${require_op_admin} AND EXISTS (SELECT 1 FROM invalid_op_admins))
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
        14|15|16) sdpivot_latest_fingerprint false ;;
        *) sdpivot_latest_fingerprint true ;;
    esac
}

require_command "$MIGRATE_BIN"
require_command "$PSQL_BIN"
[[ -n "${OP_DATABASE_URL:-}" ]] || fail "OP_DATABASE_URL is required"
[[ "$OP_DATABASE_URL" == postgres://* || "$OP_DATABASE_URL" == postgresql://* ]] || fail \
    "OP_DATABASE_URL must use the postgres or postgresql scheme"

readonly SDPIVOT_DATABASE_URL="$(with_migrations_table sdpivot_schema_migrations)"

log "prechecking SDPivot migration state"
if ! initial_sdpivot_state="$(migration_state sdpivot_schema_migrations)"; then
    fail "failed to inspect sdpivot_schema_migrations before migration"
fi
if [[ "$initial_sdpivot_state" != "absent" ]]; then
    [[ "$initial_sdpivot_state" == *:* ]] || fail "sdpivot_schema_migrations has an invalid or empty state"
    initial_sdpivot_version="${initial_sdpivot_state%%:*}"
    initial_sdpivot_dirty="${initial_sdpivot_state##*:}"
    [[ "$initial_sdpivot_dirty" == "f" ]] || fail "sdpivot_schema_migrations is dirty; automatic force is forbidden"
    [[ "$initial_sdpivot_version" =~ ^[0-9]+$ ]] || fail "sdpivot_schema_migrations version is invalid"
    (( initial_sdpivot_version >= SDPIVOT_BASELINE_VERSION )) || fail \
        "sdpivot_schema_migrations version ${initial_sdpivot_version} is older than supported ${SDPIVOT_BASELINE_VERSION}"
    (( initial_sdpivot_version <= SDPIVOT_LATEST_VERSION )) || fail \
        "sdpivot_schema_migrations version ${initial_sdpivot_version} is newer than supported ${SDPIVOT_LATEST_VERSION}"
fi

log "checking core migration state"
if ! core_state="$(migration_state schema_migrations)"; then
    fail "failed to inspect schema_migrations"
fi
if ! core_audit_fingerprint="$(core_audit_m44_fingerprint)"; then
    fail "failed to inspect the Core migration 44 audit_logs fingerprint"
fi
case "$core_audit_fingerprint" in
    missing|migration44_exact|core_current_exact|baseline_exact|baseline_current_exact|invalid) ;;
    *) fail "Core migration 44 audit_logs fingerprint returned an unknown state" ;;
esac
if [[ "$core_state" == "absent" ]]; then
    [[ "$core_audit_fingerprint" == "missing" ]] || fail \
        "audit_logs must be missing before Core migration 44"
    if ! existing_public_tables="$(sql_scalar "
        SELECT count(*)
        FROM pg_catalog.pg_class application_object
        JOIN pg_catalog.pg_namespace application_schema
          ON application_schema.oid = application_object.relnamespace
        WHERE application_schema.nspname = 'public'
          AND application_object.relkind IN ('r', 'p')
          AND application_object.relname NOT IN ('schema_migrations', 'sdpivot_schema_migrations')
          AND NOT EXISTS (
                SELECT 1
                FROM pg_catalog.pg_depend extension_dependency
                JOIN pg_catalog.pg_extension owner_extension
                  ON owner_extension.oid = extension_dependency.refobjid
                WHERE extension_dependency.classid = 'pg_catalog.pg_class'::regclass
                  AND extension_dependency.objid = application_object.oid
                  AND extension_dependency.deptype = 'e'
          )
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

    if (( core_version < 44 )); then
        [[ "$core_audit_fingerprint" == "missing" ]] || fail \
            "audit_logs must be missing before Core migration 44"
    elif (( core_version < CORE_LATEST_VERSION )); then
        [[ "$core_audit_fingerprint" == "migration44_exact" ]] || fail \
            "Core versions 44-71 require the exact migration 44 audit_logs contract"
    fi

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
if ! current_audit_fingerprint="$(core_audit_m44_fingerprint)"; then
    fail "failed to inspect the current Core migration 44 audit_logs fingerprint"
fi
case "$current_audit_fingerprint" in
    core_current_exact|baseline_current_exact) ;;
    *) fail "current Core migrations require an exact audit_logs contract" ;;
esac

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
            [[ "$current_audit_fingerprint" == "core_current_exact" ]] || fail \
                "greenfield SDPivot baseline requires the exact current Core audit_logs contract"
            log "recognized a greenfield SDPivot database; applying the secure baseline"
            ;;
        complete)
            [[ "$current_audit_fingerprint" == "baseline_current_exact" ]] || fail \
                "complete SDPivot version 12 objects require the exact baseline audit_logs contract"
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
    [[ "$current_audit_fingerprint" == "baseline_current_exact" ]] || fail \
        "existing SDPivot migrations require the exact baseline audit_logs contract"
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
