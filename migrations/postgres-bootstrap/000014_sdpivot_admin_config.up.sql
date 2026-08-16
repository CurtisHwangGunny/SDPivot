-- SDPivot tenant API tokens and administrator-managed settings.

CREATE TABLE IF NOT EXISTS api_tokens (
    id           VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id      VARCHAR(36) NOT NULL,
    tenant_id    BIGINT NOT NULL,
    name         VARCHAR(100) NOT NULL,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    prefix       VARCHAR(6) NOT NULL DEFAULT '',
    scopes       JSONB NOT NULL DEFAULT '[]'::JSONB,
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_by   VARCHAR(36),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- The official core chain owns public.users.  Keep the FK when that shared
-- table is present, but allow an isolated OP bootstrap database to build the
-- token table before the core user table has been provisioned.
DO $$
BEGIN
    IF to_regclass('public.users') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1
             FROM pg_constraint c
             JOIN pg_class parent ON parent.oid = c.confrelid
             JOIN pg_namespace parent_schema ON parent_schema.oid = parent.relnamespace
            WHERE c.conrelid = 'public.api_tokens'::regclass
              AND parent_schema.nspname = 'public'
              AND parent.relname = 'users'
              AND c.contype = 'f'
              AND c.conkey = ARRAY[
                  (SELECT attnum FROM pg_attribute
                    WHERE attrelid = 'public.api_tokens'::regclass
                      AND attname = 'user_id')
              ]::smallint[]
       ) THEN
        ALTER TABLE api_tokens
            ADD CONSTRAINT fk_api_tokens_user
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS user_id VARCHAR(36);
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS prefix VARCHAR(6) NOT NULL DEFAULT '';
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS scopes JSONB NOT NULL DEFAULT '[]'::JSONB;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS created_by VARCHAR(36);
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
UPDATE api_tokens SET created_by = user_id WHERE created_by IS NULL;
CREATE INDEX IF NOT EXISTS idx_api_tokens_tenant_active ON api_tokens (tenant_id, revoked_at, expires_at);
CREATE INDEX IF NOT EXISTS idx_api_tokens_prefix ON api_tokens (tenant_id, prefix);

CREATE TABLE IF NOT EXISTS sdpivot_system_settings (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id  BIGINT NOT NULL,
    section    VARCHAR(32) NOT NULL CHECK (section IN ('storage', 'sms', 'wechat', 'search', 'cli_mcp', 'global')),
    key        VARCHAR(100) NOT NULL,
    value      TEXT NOT NULL DEFAULT '',
    updated_by VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, section, key)
);

CREATE TABLE IF NOT EXISTS security_settings (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id  BIGINT NOT NULL,
    section    VARCHAR(32) NOT NULL CHECK (section IN ('ip_whitelist', 'password_policy', 'session', 'desensitize', 'audit')),
    key        VARCHAR(100) NOT NULL,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, section, key)
);

ALTER TABLE sdpivot_system_settings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000014_system_settings ON sdpivot_system_settings;
CREATE POLICY sdpivot_op_bootstrap_000014_system_settings ON sdpivot_system_settings
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE security_settings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000014_security_settings ON security_settings;
CREATE POLICY sdpivot_op_bootstrap_000014_security_settings ON security_settings
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
