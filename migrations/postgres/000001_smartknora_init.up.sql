-- smartKnora (随越·智枢) multi-tenant PostgreSQL schema
-- Built on top of WeKnora; uses VARCHAR(36) for IDs (consistent with WeKnora)

-- users 表已由 WeKnora 创建，仅补充索引（避免冲突）
CREATE INDEX IF NOT EXISTS idx_users_phone ON users (phone);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- organizations 表已由 WeKnora 创建，仅补充 smartknora 扩展字段
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS auth_status VARCHAR(20) DEFAULT 'trial';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS auth_type VARCHAR(20);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS auth_expires_at TIMESTAMPTZ;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS logo_url TEXT;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS config JSONB DEFAULT '{}';

-- org_members 表已由 WeKnora 创建（organization_tenant_members），补充 smartknora 扩展
CREATE TABLE IF NOT EXISTS org_ext (
    org_id          VARCHAR(36)  PRIMARY KEY REFERENCES organizations(id),
    auth_status     VARCHAR(20)  NOT NULL DEFAULT 'trial',
    auth_type       VARCHAR(20),
    auth_expires_at TIMESTAMPTZ,
    tenant_id       BIGINT       NOT NULL,
    created_at      TIMESTAMPTZ  DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_org_ext_auth_status ON org_ext(auth_status);

-- smartknora_user_profiles（扩展 WeKnora 用户）
CREATE TABLE IF NOT EXISTS smartknora_user_profiles (
    id           VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      VARCHAR(36)  UNIQUE NOT NULL,
    phone        VARCHAR(20)  UNIQUE,
    nickname     VARCHAR(100),
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_skup_status ON smartknora_user_profiles(status);

-- refresh_tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id     VARCHAR(36)  NOT NULL,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL,
    revoked     BOOLEAN      DEFAULT FALSE,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rt_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_rt_expires ON refresh_tokens(expires_at);

-- token_usage
CREATE TABLE IF NOT EXISTS token_usage (
    id           VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      VARCHAR(36)  NOT NULL,
    tenant_id    BIGINT       NOT NULL DEFAULT 0,
    model        VARCHAR(100),
    input_tokens INTEGER      DEFAULT 0,
    output_tokens INTEGER     DEFAULT 0,
    action       VARCHAR(50),
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tu_user_id ON token_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_tu_tenant_id ON token_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tu_created ON token_usage(created_at);

-- knowledge_spaces
CREATE TABLE IF NOT EXISTS knowledge_spaces (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id   BIGINT       NOT NULL,
    org_id      VARCHAR(36)  REFERENCES organizations(id),
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    visibility  VARCHAR(20)  DEFAULT 'private',
    owner_id    VARCHAR(36),
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ks_tenant ON knowledge_spaces(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ks_org ON knowledge_spaces(org_id);

-- space_members
CREATE TABLE IF NOT EXISTS space_members (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    space_id    VARCHAR(36)  NOT NULL REFERENCES knowledge_spaces(id),
    user_id     VARCHAR(36)  NOT NULL,
    role        VARCHAR(20)  DEFAULT 'viewer',
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    UNIQUE(space_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_sm_space ON space_members(space_id);
CREATE INDEX IF NOT EXISTS idx_sm_user ON space_members(user_id);

-- documents
CREATE TABLE IF NOT EXISTS documents (
    id               VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id        BIGINT       NOT NULL,
    space_id         VARCHAR(36)  NOT NULL,
    uploader_id      VARCHAR(36),
    title            VARCHAR(500) NOT NULL,
    file_name        VARCHAR(255),
    file_type        VARCHAR(50),
    file_size        BIGINT       DEFAULT 0,
    file_path        TEXT,
    content_hash     VARCHAR(64),
    parse_status     VARCHAR(20)  DEFAULT 'pending',
    chunk_count      INTEGER      DEFAULT 0,
    embedding_status VARCHAR(20)  DEFAULT 'pending',
    version          INTEGER      DEFAULT 1,
    tags             TEXT,
    created_at       TIMESTAMPTZ  DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_doc_space ON documents(space_id);
CREATE INDEX IF NOT EXISTS idx_doc_tenant ON documents(tenant_id);

-- qa_sessions
CREATE TABLE IF NOT EXISTS qa_sessions (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id   BIGINT       NOT NULL,
    user_id     VARCHAR(36)  NOT NULL,
    title       VARCHAR(200),
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_qs_user ON qa_sessions(user_id);

-- qa_messages
CREATE TABLE IF NOT EXISTS qa_messages (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    session_id  VARCHAR(36)  NOT NULL,
    tenant_id   BIGINT       NOT NULL,
    role        VARCHAR(20)  NOT NULL,
    content     TEXT,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_qm_session ON qa_messages(session_id);

-- writing_drafts
CREATE TABLE IF NOT EXISTS writing_drafts (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id   BIGINT       NOT NULL,
    user_id     VARCHAR(36)  NOT NULL,
    category    VARCHAR(50),
    title       VARCHAR(200),
    content     TEXT,
    status      VARCHAR(20)  DEFAULT 'draft',
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wd_user ON writing_drafts(user_id);

-- announcements
CREATE TABLE IF NOT EXISTS announcements (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id   BIGINT       NOT NULL,
    title       VARCHAR(200),
    content     TEXT,
    status      VARCHAR(20)  DEFAULT 'draft',
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ann_tenant ON announcements(tenant_id);

-- Enable RLS on all smartknora tables
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge_spaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE space_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE qa_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE qa_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE writing_drafts ENABLE ROW LEVEL SECURITY;
ALTER TABLE announcements ENABLE ROW LEVEL SECURITY;
ALTER TABLE token_usage ENABLE ROW LEVEL SECURITY;

-- RLS Policies (tenant_id based isolation)
CREATE POLICY org_tenant_isolation ON organizations USING (
    owner_tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
CREATE POLICY ks_tenant_isolation ON knowledge_spaces USING (
    tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
CREATE POLICY doc_tenant_isolation ON documents USING (
    tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
CREATE POLICY qs_tenant_isolation ON qa_sessions USING (
    tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
CREATE POLICY qm_tenant_isolation ON qa_messages USING (
    tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
CREATE POLICY wd_tenant_isolation ON writing_drafts USING (
    tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
CREATE POLICY ann_tenant_isolation ON announcements USING (
    tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
CREATE POLICY tu_tenant_isolation ON token_usage USING (
    tenant_id = COALESCE(current_setting('app.current_tenant_id')::bigint, 0)
);
