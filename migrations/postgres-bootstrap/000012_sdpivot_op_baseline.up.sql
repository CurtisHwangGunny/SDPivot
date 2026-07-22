-- SDPivot OP bootstrap baseline for fresh OP environments.
-- PRECONDITION: migrations/versioned (the WeKnora core chain) has completed.
-- This migration fails closed when required shared tables or key columns are absent;
-- it never creates replacement core tables, application roles, users, credentials, or seed rows.

DO $$
DECLARE
    incompatible_columns TEXT;
    missing_unique_keys TEXT;
BEGIN
    IF to_regclass('public.users') IS NULL THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap requires core table public.users';
    END IF;
    IF to_regclass('public.organizations') IS NULL THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap requires core table public.organizations';
    END IF;

    SELECT string_agg(
               format('%I.%I expected %s, found %s',
                   required.table_name,
                   required.column_name,
                   array_to_string(required.allowed_types, '/'),
                   COALESCE(existing.data_type, 'missing')),
               '; ' ORDER BY required.table_name, required.column_name)
      INTO incompatible_columns
      FROM (VALUES
          ('users', 'id', ARRAY['character varying', 'text']),
          ('users', 'email', ARRAY['character varying', 'text']),
          ('users', 'password_hash', ARRAY['character varying', 'text']),
          ('users', 'tenant_id', ARRAY['bigint']),
          ('users', 'is_active', ARRAY['boolean']),
          ('users', 'is_system_admin', ARRAY['boolean']),
          ('organizations', 'id', ARRAY['character varying', 'text']),
          ('organizations', 'owner_id', ARRAY['character varying', 'text']),
          ('organizations', 'owner_tenant_id', ARRAY['bigint'])
      ) AS required(table_name, column_name, allowed_types)
      LEFT JOIN information_schema.columns existing
        ON existing.table_schema = 'public'
       AND existing.table_name = required.table_name
       AND existing.column_name = required.column_name
     WHERE existing.column_name IS NULL
        OR NOT (existing.data_type = ANY(required.allowed_types));

    IF incompatible_columns IS NOT NULL THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap requires compatible completed core migrations: %', incompatible_columns;
    END IF;

    SELECT string_agg(format('%I.%I', required.table_name, required.column_name), ', ' ORDER BY required.table_name)
      INTO missing_unique_keys
      FROM (VALUES ('users', 'id'), ('organizations', 'id')) AS required(table_name, column_name)
     WHERE NOT EXISTS (
         SELECT 1
           FROM pg_catalog.pg_constraint key_constraint
           JOIN pg_catalog.pg_class target_table ON target_table.oid = key_constraint.conrelid
           JOIN pg_catalog.pg_namespace target_schema ON target_schema.oid = target_table.relnamespace
           JOIN pg_catalog.pg_attribute key_column
             ON key_column.attrelid = target_table.oid
            AND key_column.attname = required.column_name
            AND NOT key_column.attisdropped
          WHERE target_schema.nspname = 'public'
            AND target_table.relname = required.table_name
            AND key_constraint.contype IN ('p', 'u')
            AND array_length(key_constraint.conkey, 1) = 1
            AND key_constraint.conkey[1] = key_column.attnum
     );

    IF missing_unique_keys IS NOT NULL THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap requires PRIMARY KEY or UNIQUE constraints on: %', missing_unique_keys;
    END IF;
END $$;

CREATE OR REPLACE FUNCTION sdpivot_op_bootstrap_000012_assert_schema(p_require_all BOOLEAN)
RETURNS void AS $$
DECLARE
    incompatible_columns TEXT;
    id_type_profile RECORD;
BEGIN
    SELECT string_agg(
               format('%I.%I expected %s, found %s',
                   required.table_name,
                   required.column_name,
                   array_to_string(required.allowed_types, '/'),
                   COALESCE(existing.data_type, 'missing')),
               '; ' ORDER BY required.table_name, required.column_name)
      INTO incompatible_columns
      FROM (VALUES
          ('users', 'trial_phase', ARRAY['character varying']),
          ('org_ext', 'org_id', ARRAY['character varying']),
          ('org_ext', 'auth_status', ARRAY['character varying']),
          ('org_ext', 'tenant_id', ARRAY['bigint']),
          ('org_members', 'id', ARRAY['character varying']),
          ('org_members', 'org_id', ARRAY['character varying']),
          ('org_members', 'user_id', ARRAY['character varying']),
          ('smartknora_user_profiles', 'id', ARRAY['character varying']),
          ('smartknora_user_profiles', 'user_id', ARRAY['character varying']),
          ('smartknora_user_profiles', 'phone', ARRAY['character varying']),
          ('smartknora_user_profiles', 'status', ARRAY['character varying']),
          ('refresh_tokens', 'id', ARRAY['character varying']),
          ('refresh_tokens', 'user_id', ARRAY['character varying']),
          ('refresh_tokens', 'expires_at', ARRAY['timestamp with time zone']),
          ('token_usage', 'id', ARRAY['character varying']),
          ('token_usage', 'user_id', ARRAY['character varying']),
          ('token_usage', 'tenant_id', ARRAY['bigint']),
          ('token_usage', 'created_at', ARRAY['timestamp with time zone']),
          ('knowledge_spaces', 'id', ARRAY['character varying']),
          ('knowledge_spaces', 'tenant_id', ARRAY['bigint']),
          ('knowledge_spaces', 'org_id', ARRAY['character varying']),
          ('space_members', 'id', ARRAY['character varying']),
          ('space_members', 'space_id', ARRAY['character varying']),
          ('space_members', 'user_id', ARRAY['character varying']),
          ('space_categories', 'id', ARRAY['character varying']),
          ('space_categories', 'tenant_id', ARRAY['bigint']),
          ('documents', 'id', ARRAY['character varying']),
          ('documents', 'tenant_id', ARRAY['bigint']),
          ('documents', 'space_id', ARRAY['character varying']),
          ('documents', 'parse_status', ARRAY['character varying']),
          ('documents', 'content_hash', ARRAY['character varying']),
          ('document_chunks', 'id', ARRAY['character varying', 'uuid']),
          ('document_chunks', 'document_id', ARRAY['character varying']),
          ('document_chunks', 'tenant_id', ARRAY['bigint']),
          ('document_versions', 'id', ARRAY['character varying']),
          ('document_versions', 'document_id', ARRAY['character varying']),
          ('document_versions', 'tenant_id', ARRAY['bigint']),
          ('chunk_strategies', 'id', ARRAY['character varying']),
          ('chunk_strategies', 'tenant_id', ARRAY['bigint']),
          ('qa_sessions', 'id', ARRAY['character varying']),
          ('qa_sessions', 'user_id', ARRAY['character varying']),
          ('qa_sessions', 'tenant_id', ARRAY['bigint']),
          ('qa_messages', 'id', ARRAY['character varying']),
          ('qa_messages', 'session_id', ARRAY['character varying']),
          ('qa_messages', 'tenant_id', ARRAY['bigint']),
          ('writing_drafts', 'id', ARRAY['character varying']),
          ('writing_drafts', 'user_id', ARRAY['character varying']),
          ('writing_drafts', 'tenant_id', ARRAY['bigint']),
          ('write_category_config', 'id', ARRAY['character varying']),
          ('write_category_config', 'tenant_id', ARRAY['bigint']),
          ('write_category_config', 'category', ARRAY['character varying']),
          ('announcements', 'id', ARRAY['character varying']),
          ('announcements', 'tenant_id', ARRAY['bigint']),
          ('audit_logs', 'id', ARRAY['bigint']),
          ('audit_logs', 'tenant_id', ARRAY['bigint']),
          ('audit_logs', 'user_id', ARRAY['character varying']),
          ('audit_logs', 'action', ARRAY['character varying']),
          ('audit_logs', 'created_at', ARRAY['timestamp with time zone']),
          ('sensitive_words', 'id', ARRAY['character varying']),
          ('sensitive_words', 'word', ARRAY['character varying']),
          ('sensitive_words', 'category', ARRAY['character varying']),
          ('sensitive_words', 'status', ARRAY['character varying']),
          ('billing_plans', 'id', ARRAY['character varying']),
          ('billing_plans', 'status', ARRAY['character varying']),
          ('enterprise_subscriptions', 'id', ARRAY['character varying']),
          ('enterprise_subscriptions', 'org_id', ARRAY['character varying']),
          ('enterprise_subscriptions', 'plan_id', ARRAY['character varying']),
          ('invoices', 'id', ARRAY['character varying']),
          ('invoices', 'org_id', ARRAY['character varying']),
          ('invoices', 'status', ARRAY['character varying']),
          ('invoices', 'period_start', ARRAY['timestamp with time zone']),
          ('invoices', 'period_end', ARRAY['timestamp with time zone'])
      ) AS required(table_name, column_name, allowed_types)
      LEFT JOIN information_schema.columns existing
        ON existing.table_schema = 'public'
       AND existing.table_name = required.table_name
       AND existing.column_name = required.column_name
     WHERE (existing.column_name IS NULL
            AND (p_require_all
                 OR (required.table_name <> 'users'
                     AND to_regclass(format('public.%I', required.table_name)) IS NOT NULL)))
        OR (existing.column_name IS NOT NULL
            AND NOT (existing.data_type = ANY(required.allowed_types)));

    IF incompatible_columns IS NOT NULL THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap schema compatibility check failed: %', incompatible_columns;
    END IF;

    SELECT
        max(data_type) FILTER (WHERE table_name = 'org_ext' AND column_name = 'org_id') AS org_ext_id,
        max(data_type) FILTER (WHERE table_name = 'knowledge_spaces' AND column_name = 'id') AS space_id,
        max(data_type) FILTER (WHERE table_name = 'documents' AND column_name = 'id') AS document_id,
        max(data_type) FILTER (WHERE table_name = 'document_chunks' AND column_name = 'id') AS chunk_id
      INTO id_type_profile
      FROM information_schema.columns
     WHERE table_schema = 'public'
       AND (table_name, column_name) IN (
           ('org_ext', 'org_id'),
           ('knowledge_spaces', 'id'),
           ('documents', 'id'),
           ('document_chunks', 'id')
       );

    IF id_type_profile.org_ext_id IS NOT NULL
       AND id_type_profile.space_id IS NOT NULL
       AND id_type_profile.document_id IS NOT NULL
       AND id_type_profile.chunk_id IS NOT NULL
       AND NOT (
           (id_type_profile.org_ext_id = 'character varying'
            AND id_type_profile.space_id = 'character varying'
            AND id_type_profile.document_id = 'character varying'
            AND id_type_profile.chunk_id = 'uuid')
           OR
           (id_type_profile.org_ext_id = 'character varying'
            AND id_type_profile.space_id = 'character varying'
            AND id_type_profile.document_id = 'character varying'
            AND id_type_profile.chunk_id = 'character varying')
       ) THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap schema compatibility check failed: incompatible ID type profile';
    END IF;

    IF to_regclass('public.knowledge_spaces') IS NOT NULL AND NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint key_constraint
          JOIN pg_catalog.pg_class target_table ON target_table.oid = key_constraint.conrelid
          JOIN pg_catalog.pg_namespace target_schema ON target_schema.oid = target_table.relnamespace
         WHERE target_schema.nspname = 'public'
           AND target_table.relname = 'knowledge_spaces'
           AND key_constraint.contype IN ('p', 'u')
           AND array_length(key_constraint.conkey, 1) = 1
           AND key_constraint.conkey[1] = (
               SELECT key_column.attnum
                 FROM pg_catalog.pg_attribute key_column
                WHERE key_column.attrelid = target_table.oid
                  AND key_column.attname = 'id'
                  AND NOT key_column.attisdropped
           )
    ) THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap requires PRIMARY KEY or UNIQUE constraint on knowledge_spaces.id';
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY INVOKER;

SELECT sdpivot_op_bootstrap_000012_assert_schema(FALSE);

-- Shared core table extensions from the final historical SDPivot schema.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_ops_admin BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS trial_phase VARCHAR(20) DEFAULT '30day';
ALTER TABLE users ADD COLUMN IF NOT EXISTS authenticated_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_extended_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;

DO $$
DECLARE
    incompatible_columns TEXT;
BEGIN
    SELECT string_agg(
               format('%I.%I expected %s, found %s',
                   required.table_name,
                   required.column_name,
                   array_to_string(required.allowed_types, '/'),
                   COALESCE(existing.data_type, 'missing')),
               '; ' ORDER BY required.table_name, required.column_name)
      INTO incompatible_columns
      FROM (VALUES
          ('users', 'is_ops_admin', ARRAY['boolean']),
          ('users', 'trial_phase', ARRAY['character varying'])
      ) AS required(table_name, column_name, allowed_types)
      LEFT JOIN information_schema.columns existing
        ON existing.table_schema = 'public'
       AND existing.table_name = required.table_name
       AND existing.column_name = required.column_name
     WHERE existing.column_name IS NULL
        OR NOT (existing.data_type = ANY(required.allowed_types));

    IF incompatible_columns IS NOT NULL THEN
        RAISE EXCEPTION 'SDPivot OP bootstrap core extension compatibility check failed: %', incompatible_columns;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_users_ops_admin ON users (is_ops_admin) WHERE is_ops_admin = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_trial_phase ON users (trial_phase);

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS auth_status VARCHAR(20) DEFAULT 'trial';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS auth_type VARCHAR(20);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS auth_expires_at TIMESTAMPTZ;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS logo_url TEXT;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS config JSONB DEFAULT '{}'::JSONB;

-- Extension and authentication support tables.
CREATE TABLE IF NOT EXISTS org_ext (
    org_id              VARCHAR(36) PRIMARY KEY REFERENCES organizations(id),
    auth_status         VARCHAR(20) NOT NULL DEFAULT 'trial',
    auth_type           VARCHAR(20),
    auth_expires_at     TIMESTAMPTZ,
    tenant_id           BIGINT NOT NULL,
    subscription_status VARCHAR(20) DEFAULT 'free',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_org_ext_auth_status ON org_ext (auth_status);
CREATE INDEX IF NOT EXISTS idx_org_ext_tenant_id ON org_ext (tenant_id);

CREATE TABLE IF NOT EXISTS org_members (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    org_id     VARCHAR(36) NOT NULL REFERENCES organizations(id),
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id),
    role       VARCHAR(20) NOT NULL DEFAULT 'member',
    status     VARCHAR(20) NOT NULL DEFAULT 'active',
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_org_members_org ON org_members (org_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON org_members (user_id);

CREATE TABLE IF NOT EXISTS smartknora_user_profiles (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id    VARCHAR(36) NOT NULL UNIQUE REFERENCES users(id),
    phone      VARCHAR(20) UNIQUE,
    nickname   VARCHAR(100),
    status     VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sknp_phone ON smartknora_user_profiles (phone);
CREATE INDEX IF NOT EXISTS idx_sknp_user_id ON smartknora_user_profiles (user_id);
CREATE INDEX IF NOT EXISTS idx_sknp_status ON smartknora_user_profiles (status);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN DEFAULT FALSE,
    device_id  VARCHAR(255) DEFAULT '',
    family     VARCHAR(36) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rt_user_id ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_rt_expires ON refresh_tokens (expires_at);

CREATE TABLE IF NOT EXISTS token_usage (
    id            VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id       VARCHAR(36) NOT NULL,
    tenant_id     BIGINT NOT NULL DEFAULT 0,
    model         VARCHAR(100),
    input_tokens  INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    action        VARCHAR(50),
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tu_user_id ON token_usage (user_id);
CREATE INDEX IF NOT EXISTS idx_tu_tenant_id ON token_usage (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tu_created ON token_usage (created_at);

-- Knowledge-space objects are created before every dependent object and policy.
CREATE TABLE IF NOT EXISTS knowledge_spaces (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id   BIGINT NOT NULL,
    org_id      VARCHAR(36) REFERENCES organizations(id),
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    visibility  VARCHAR(20) DEFAULT 'private',
    owner_id    VARCHAR(36),
    icon        VARCHAR(255),
    creator_id  VARCHAR(36),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ks_tenant ON knowledge_spaces (tenant_id);
CREATE INDEX IF NOT EXISTS idx_ks_org ON knowledge_spaces (org_id);

CREATE TABLE IF NOT EXISTS space_members (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    space_id   VARCHAR(36) NOT NULL REFERENCES knowledge_spaces(id),
    user_id    VARCHAR(36) NOT NULL,
    role       VARCHAR(20) DEFAULT 'viewer',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (space_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_sm_space ON space_members (space_id);
CREATE INDEX IF NOT EXISTS idx_sm_user ON space_members (user_id);

CREATE TABLE IF NOT EXISTS space_categories (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id   BIGINT NOT NULL DEFAULT 1,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_space_categories_tenant ON space_categories (tenant_id);

-- Document objects.
CREATE TABLE IF NOT EXISTS documents (
    id               VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id        BIGINT NOT NULL,
    space_id         VARCHAR(36) NOT NULL,
    uploader_id      VARCHAR(36),
    title            VARCHAR(500) NOT NULL,
    file_name        VARCHAR(255),
    file_type        VARCHAR(50),
    file_size        BIGINT DEFAULT 0,
    file_path        TEXT,
    content_hash     VARCHAR(64),
    parse_status     VARCHAR(20) NOT NULL DEFAULT 'pending',
    chunk_count      INTEGER DEFAULT 0,
    embedding_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    version          INTEGER DEFAULT 1,
    tags             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_doc_space ON documents (space_id);
CREATE INDEX IF NOT EXISTS idx_doc_tenant ON documents (tenant_id);
CREATE INDEX IF NOT EXISTS idx_documents_parse_status ON documents (parse_status);
CREATE INDEX IF NOT EXISTS idx_documents_content_hash ON documents (content_hash);

CREATE TABLE IF NOT EXISTS document_chunks (
    id           VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    document_id  VARCHAR(36) NOT NULL,
    tenant_id    BIGINT NOT NULL,
    chunk_index  INTEGER NOT NULL,
    content      TEXT NOT NULL,
    token_count  INTEGER,
    embedding_id VARCHAR(64),
    metadata     JSONB DEFAULT '{}'::JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_document_id ON document_chunks (document_id);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_tenant_id ON document_chunks (tenant_id);

CREATE TABLE IF NOT EXISTS document_versions (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    document_id VARCHAR(36) NOT NULL,
    tenant_id   BIGINT NOT NULL DEFAULT 0,
    version     INTEGER NOT NULL,
    file_path   TEXT,
    file_size   BIGINT,
    chunk_count INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(36)
);
CREATE INDEX IF NOT EXISTS idx_doc_versions_document_id ON document_versions (document_id);
CREATE INDEX IF NOT EXISTS idx_doc_versions_tenant_id ON document_versions (tenant_id);

CREATE TABLE IF NOT EXISTS chunk_strategies (
    id            VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id     BIGINT NOT NULL DEFAULT 1,
    space_id      VARCHAR(36),
    name          VARCHAR(100) NOT NULL,
    strategy_type VARCHAR(50) NOT NULL DEFAULT 'fixed_size',
    chunk_size    INTEGER DEFAULT 512,
    chunk_overlap INTEGER DEFAULT 50,
    split_markers TEXT,
    is_active     BOOLEAN DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_chunk_strategies_tenant ON chunk_strategies (tenant_id);

-- Q&A and writing objects.
CREATE TABLE IF NOT EXISTS qa_sessions (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id  BIGINT NOT NULL,
    user_id    VARCHAR(36) NOT NULL,
    space_id   VARCHAR(36),
    title      VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_qa_sessions_user_id ON qa_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_qa_sessions_tenant_id ON qa_sessions (tenant_id);

CREATE TABLE IF NOT EXISTS qa_messages (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    session_id VARCHAR(36) NOT NULL,
    tenant_id  BIGINT NOT NULL DEFAULT 0,
    role       VARCHAR(20) NOT NULL,
    content    TEXT NOT NULL,
    sources    JSONB DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_qa_messages_session_id ON qa_messages (session_id);

CREATE TABLE IF NOT EXISTS writing_drafts (
    id                 VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id          BIGINT NOT NULL,
    user_id            VARCHAR(36) NOT NULL,
    space_id           VARCHAR(36),
    title              VARCHAR(500),
    category           VARCHAR(50),
    content            TEXT,
    status             VARCHAR(20) DEFAULT 'draft',
    source_type        VARCHAR(30) NOT NULL DEFAULT 'knowledge_base',
    web_search_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_writing_drafts_user_id ON writing_drafts (user_id);
CREATE INDEX IF NOT EXISTS idx_writing_drafts_tenant_id ON writing_drafts (tenant_id);

CREATE TABLE IF NOT EXISTS write_category_config (
    id                 VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id          BIGINT NOT NULL,
    category           VARCHAR(50) NOT NULL,
    default_space_id   VARCHAR(36),
    source_type        VARCHAR(30) NOT NULL DEFAULT 'knowledge_base',
    web_search_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, category)
);
CREATE INDEX IF NOT EXISTS idx_wcc_tenant_category ON write_category_config (tenant_id, category);

-- Historical operations objects are schema-only: no accounts, credentials, or SaaS seed rows.
CREATE TABLE IF NOT EXISTS announcements (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id  BIGINT NOT NULL DEFAULT 0,
    title      VARCHAR(500) NOT NULL,
    content    TEXT,
    status     VARCHAR(20) DEFAULT 'draft',
    created_by VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ann_tenant ON announcements (tenant_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    tenant_id   BIGINT NOT NULL DEFAULT 1,
    user_id     VARCHAR(36),
    username    VARCHAR(100),
    action      VARCHAR(100) NOT NULL,
    resource    VARCHAR(100),
    resource_id VARCHAR(64),
    detail      TEXT,
    ip          VARCHAR(50),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_smart_audit_tenant ON audit_logs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_smart_audit_user ON audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_smart_audit_action ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_smart_audit_created ON audit_logs (created_at);

CREATE TABLE IF NOT EXISTS sensitive_words (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    word       VARCHAR(200) NOT NULL,
    category   VARCHAR(50) NOT NULL DEFAULT 'general',
    status     VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sw_word ON sensitive_words (word);
CREATE INDEX IF NOT EXISTS idx_sw_category ON sensitive_words (category);
CREATE INDEX IF NOT EXISTS idx_sw_status ON sensitive_words (status);

CREATE TABLE IF NOT EXISTS billing_plans (
    id            VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    name          VARCHAR(100) NOT NULL,
    price         DECIMAL(10,2) NOT NULL DEFAULT 0,
    token_quota   BIGINT NOT NULL DEFAULT 0,
    storage_quota BIGINT NOT NULL DEFAULT 0,
    features      JSONB NOT NULL DEFAULT '{}'::JSONB,
    status        VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_bp_status ON billing_plans (status);

CREATE TABLE IF NOT EXISTS enterprise_subscriptions (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    org_id     VARCHAR(36) NOT NULL,
    plan_id    VARCHAR(36) NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_es_org ON enterprise_subscriptions (org_id);
CREATE INDEX IF NOT EXISTS idx_es_plan ON enterprise_subscriptions (plan_id);

CREATE TABLE IF NOT EXISTS invoices (
    id           VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    org_id       VARCHAR(36) NOT NULL,
    plan_id      VARCHAR(36),
    amount       DECIMAL(10,2) NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_inv_org ON invoices (org_id);
CREATE INDEX IF NOT EXISTS idx_inv_status ON invoices (status);
CREATE INDEX IF NOT EXISTS idx_inv_period ON invoices (period_start, period_end);

SELECT sdpivot_op_bootstrap_000012_assert_schema(TRUE);
DROP FUNCTION sdpivot_op_bootstrap_000012_assert_schema(BOOLEAN);

-- Historical tenant helpers, made safe when application GUCs are unset or empty.
CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, _p_is_ops_admin BOOLEAN DEFAULT FALSE)
RETURNS void AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, false);
    PERFORM set_config('app.is_ops_admin', 'false', false);
END;
$$ LANGUAGE plpgsql SECURITY INVOKER;

CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS BIGINT AS $$
    SELECT CASE
        WHEN COALESCE(current_setting('app.current_tenant_id', true), '') ~ '^[0-9]+$'
        THEN current_setting('app.current_tenant_id', true)::BIGINT
        ELSE 0
    END;
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION is_ops_admin_context()
RETURNS BOOLEAN AS $$
    SELECT FALSE;
$$ LANGUAGE sql STABLE SECURITY INVOKER;

-- Bootstrap policies are uniquely named and created only after their tables exist.
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_organizations ON organizations;
CREATE POLICY sdpivot_op_bootstrap_000012_organizations ON organizations
    FOR ALL
    USING (owner_tenant_id = get_current_tenant_id())
    WITH CHECK (owner_tenant_id = get_current_tenant_id());

ALTER TABLE org_ext ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_org_ext ON org_ext;
CREATE POLICY sdpivot_op_bootstrap_000012_org_ext ON org_ext
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE org_members ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_org_members ON org_members;
CREATE POLICY sdpivot_op_bootstrap_000012_org_members ON org_members
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM org_ext oe WHERE oe.org_id = org_members.org_id AND oe.tenant_id = get_current_tenant_id()
    ))
    WITH CHECK (EXISTS (
        SELECT 1 FROM org_ext oe WHERE oe.org_id = org_members.org_id AND oe.tenant_id = get_current_tenant_id()
    ));

ALTER TABLE smartknora_user_profiles ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_user_profiles ON smartknora_user_profiles;
CREATE POLICY sdpivot_op_bootstrap_000012_user_profiles ON smartknora_user_profiles
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM users u WHERE u.id = smartknora_user_profiles.user_id AND u.tenant_id = get_current_tenant_id()
    ))
    WITH CHECK (EXISTS (
        SELECT 1 FROM users u WHERE u.id = smartknora_user_profiles.user_id AND u.tenant_id = get_current_tenant_id()
    ));

ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_refresh_tokens ON refresh_tokens;
CREATE POLICY sdpivot_op_bootstrap_000012_refresh_tokens ON refresh_tokens
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM users u WHERE u.id = refresh_tokens.user_id AND u.tenant_id = get_current_tenant_id()
    ))
    WITH CHECK (EXISTS (
        SELECT 1 FROM users u WHERE u.id = refresh_tokens.user_id AND u.tenant_id = get_current_tenant_id()
    ));

ALTER TABLE token_usage ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_token_usage ON token_usage;
CREATE POLICY sdpivot_op_bootstrap_000012_token_usage ON token_usage
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE knowledge_spaces ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_knowledge_spaces ON knowledge_spaces;
CREATE POLICY sdpivot_op_bootstrap_000012_knowledge_spaces ON knowledge_spaces
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE space_members ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_space_members ON space_members;
CREATE POLICY sdpivot_op_bootstrap_000012_space_members ON space_members
    FOR ALL
    USING (EXISTS (
        SELECT 1 FROM knowledge_spaces ks WHERE ks.id = space_members.space_id AND ks.tenant_id = get_current_tenant_id()
    ))
    WITH CHECK (EXISTS (
        SELECT 1 FROM knowledge_spaces ks WHERE ks.id = space_members.space_id AND ks.tenant_id = get_current_tenant_id()
    ));

ALTER TABLE space_categories ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_space_categories ON space_categories;
CREATE POLICY sdpivot_op_bootstrap_000012_space_categories ON space_categories
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_documents ON documents;
CREATE POLICY sdpivot_op_bootstrap_000012_documents ON documents
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_document_chunks ON document_chunks;
CREATE POLICY sdpivot_op_bootstrap_000012_document_chunks ON document_chunks
    FOR ALL
    USING (
        EXISTS (
            SELECT 1
              FROM documents d
             WHERE d.id = document_chunks.document_id
               AND d.tenant_id = document_chunks.tenant_id
               AND d.tenant_id = get_current_tenant_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1
              FROM documents d
             WHERE d.id = document_chunks.document_id
               AND d.tenant_id = document_chunks.tenant_id
               AND d.tenant_id = get_current_tenant_id()
        )
    );

ALTER TABLE document_versions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_document_versions ON document_versions;
CREATE POLICY sdpivot_op_bootstrap_000012_document_versions ON document_versions
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE chunk_strategies ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_chunk_strategies ON chunk_strategies;
CREATE POLICY sdpivot_op_bootstrap_000012_chunk_strategies ON chunk_strategies
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE qa_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_qa_sessions ON qa_sessions;
CREATE POLICY sdpivot_op_bootstrap_000012_qa_sessions ON qa_sessions
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE qa_messages ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_qa_messages ON qa_messages;
CREATE POLICY sdpivot_op_bootstrap_000012_qa_messages ON qa_messages
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE writing_drafts ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_writing_drafts ON writing_drafts;
CREATE POLICY sdpivot_op_bootstrap_000012_writing_drafts ON writing_drafts
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE write_category_config ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_write_category_config ON write_category_config;
CREATE POLICY sdpivot_op_bootstrap_000012_write_category_config ON write_category_config
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE announcements ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_announcements ON announcements;
CREATE POLICY sdpivot_op_bootstrap_000012_announcements ON announcements
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_audit_logs ON audit_logs;
CREATE POLICY sdpivot_op_bootstrap_000012_audit_logs ON audit_logs
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
