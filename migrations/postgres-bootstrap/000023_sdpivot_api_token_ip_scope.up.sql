-- 000014 normally owns this table.  Recreate its base shape here as an
-- idempotent dependency repair for databases where that migration was
-- interrupted before the table was committed.
-- uuid handled by PG-native gen_random_uuid (PG13+); no extension needed

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

-- public.users is created by the official versioned 000001 migration.  The
-- OP token migration also runs in isolated smoke/upgrade databases where the
-- shared table may not have been provisioned yet.  Preserve the production
-- FK whenever users exists without making CREATE TABLE fail otherwise.
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

ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS prefix VARCHAR(6) NOT NULL DEFAULT '';
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS scopes JSONB NOT NULL DEFAULT '[]'::JSONB;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS allowed_ips JSONB NOT NULL DEFAULT '[]'::JSONB;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS scope_enforced BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS created_by VARCHAR(36);
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE api_tokens SET created_by = user_id WHERE created_by IS NULL;

CREATE INDEX IF NOT EXISTS idx_api_tokens_tenant_active ON api_tokens (tenant_id, revoked_at, expires_at);
CREATE INDEX IF NOT EXISTS idx_api_tokens_prefix ON api_tokens (tenant_id, prefix);
