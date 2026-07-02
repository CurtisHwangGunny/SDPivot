-- smartKnora (随越·智枢) multi-tenant PostgreSQL schema
-- Built on top of WeKnora; requires PostgreSQL 15+

-- ============================================================
-- users 用户表
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    phone           VARCHAR(20) UNIQUE,
    email           VARCHAR(255) UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    nickname        VARCHAR(100),
    avatar_url      TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_phone   ON users (phone);
CREATE INDEX IF NOT EXISTS idx_users_email   ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_status  ON users (status);

-- ============================================================
-- organizations 企业表
-- ============================================================
CREATE TABLE IF NOT EXISTS organizations (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255)  NOT NULL UNIQUE,
    logo_url        TEXT,
    description     TEXT,
    auth_status     VARCHAR(20)   NOT NULL DEFAULT 'trial',
    auth_type       VARCHAR(20),
    auth_expires_at TIMESTAMPTZ,
    config          JSONB         DEFAULT '{}',
    invite_code     VARCHAR(32)   UNIQUE,
    owner_id        UUID          REFERENCES users(id),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- ============================================================
-- org_members 企业成员表
-- ============================================================
CREATE TABLE IF NOT EXISTS org_members (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id),
    user_id     UUID        NOT NULL REFERENCES users(id),
    role        VARCHAR(20) NOT NULL DEFAULT 'member',
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, user_id)
);

-- ============================================================
-- refresh_tokens 刷新令牌表
-- ============================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID          NOT NULL REFERENCES users(id),
    token_hash  VARCHAR(64)   NOT NULL UNIQUE,
    device_id   VARCHAR(255),
    family      UUID          NOT NULL,
    expires_at  TIMESTAMPTZ   NOT NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id    ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);

-- ============================================================
-- token_usage 用量计量表
-- ============================================================
CREATE TABLE IF NOT EXISTS token_usage (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID          REFERENCES users(id),
    org_id            UUID          REFERENCES organizations(id),
    tenant_id         INTEGER       NOT NULL,
    model_id          VARCHAR(64),
    prompt_tokens     INTEGER       NOT NULL DEFAULT 0,
    completion_tokens INTEGER       NOT NULL DEFAULT 0,
    total_tokens      INTEGER       NOT NULL DEFAULT 0,
    api_path          VARCHAR(255),
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_token_usage_org_created  ON token_usage (org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_token_usage_user_created ON token_usage (user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_token_usage_tenant_created ON token_usage (tenant_id, created_at);

-- ============================================================
-- knowledge_spaces 知识空间表
-- ============================================================
CREATE TABLE IF NOT EXISTS knowledge_spaces (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   INTEGER       NOT NULL,
    org_id      UUID          REFERENCES organizations(id),
    name        VARCHAR(255)  NOT NULL,
    description TEXT,
    visibility  VARCHAR(20)   NOT NULL DEFAULT 'team',
    icon        VARCHAR(50),
    creator_id  UUID          REFERENCES users(id),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_knowledge_spaces_tenant_id ON knowledge_spaces (tenant_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_spaces_org_id    ON knowledge_spaces (org_id);

-- ============================================================
-- space_members 空间成员表
-- ============================================================
CREATE TABLE IF NOT EXISTS space_members (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id    UUID        NOT NULL REFERENCES knowledge_spaces(id),
    user_id     UUID        NOT NULL REFERENCES users(id),
    role        VARCHAR(20) NOT NULL DEFAULT 'viewer',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(space_id, user_id)
);

-- ============================================================
-- space_categories 空间分类表
-- ============================================================
CREATE TABLE IF NOT EXISTS space_categories (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   INTEGER       NOT NULL,
    name        VARCHAR(100)  NOT NULL,
    color       VARCHAR(20),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_space_categories_tenant_id ON space_categories (tenant_id);
