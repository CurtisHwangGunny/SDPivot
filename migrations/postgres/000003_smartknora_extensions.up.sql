-- smartKnora extension tables migration
-- These tables extend WeKnora core tables with smartKnora-specific fields

-- ============================================================
-- smartknora_user_profiles 用户扩展信息表
-- Extends WeKnora users table with phone and nickname
-- ============================================================
CREATE TABLE IF NOT EXISTS smartknora_user_profiles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(36)  NOT NULL UNIQUE,
    phone       VARCHAR(20)  UNIQUE,
    nickname    VARCHAR(100),
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sknp_phone   ON smartknora_user_profiles (phone);
CREATE INDEX IF NOT EXISTS idx_sknp_user_id ON smartknora_user_profiles (user_id);
CREATE INDEX IF NOT EXISTS idx_sknp_status  ON smartknora_user_profiles (status);

-- ============================================================
-- org_ext 企业扩展信息表
-- Extends WeKnora organizations with auth/certification fields
-- ============================================================
CREATE TABLE IF NOT EXISTS org_ext (
    org_id          UUID         PRIMARY KEY,
    auth_status     VARCHAR(20)  NOT NULL DEFAULT 'trial',
    auth_type       VARCHAR(20),
    auth_expires_at TIMESTAMPTZ,
    tenant_id       BIGINT       NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_ext_auth_status ON org_ext (auth_status);
CREATE INDEX IF NOT EXISTS idx_org_ext_tenant_id   ON org_ext (tenant_id);

-- ============================================================
-- RLS for extension tables
-- ============================================================
ALTER TABLE org_ext ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON org_ext
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));
