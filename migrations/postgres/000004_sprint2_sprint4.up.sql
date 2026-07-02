-- Sprint 2-4 tables for smartKnora

-- ============================================================
-- documents 文档表 (Sprint 2)
-- ============================================================
CREATE TABLE IF NOT EXISTS documents (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        BIGINT       NOT NULL,
    space_id         VARCHAR(36)  NOT NULL,
    uploader_id      VARCHAR(36),
    title            VARCHAR(500) NOT NULL,
    file_name        VARCHAR(255),
    file_type        VARCHAR(50),
    file_size        BIGINT,
    file_path        TEXT,
    content_hash     VARCHAR(64),
    parse_status     VARCHAR(20)  NOT NULL DEFAULT 'pending',
    chunk_count      INTEGER      DEFAULT 0,
    embedding_status VARCHAR(20)  NOT NULL DEFAULT 'pending',
    version          INTEGER      DEFAULT 1,
    tags             TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_documents_tenant_id ON documents (tenant_id);
CREATE INDEX IF NOT EXISTS idx_documents_space_id ON documents (space_id);
CREATE INDEX IF NOT EXISTS idx_documents_parse_status ON documents (parse_status);
CREATE INDEX IF NOT EXISTS idx_documents_content_hash ON documents (content_hash);

-- ============================================================
-- document_chunks 文档分块表 (Sprint 2)
-- ============================================================
CREATE TABLE IF NOT EXISTS document_chunks (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  VARCHAR(36)  NOT NULL,
    tenant_id    BIGINT       NOT NULL,
    chunk_index  INTEGER      NOT NULL,
    content      TEXT         NOT NULL,
    token_count  INTEGER,
    embedding_id VARCHAR(64),
    metadata     JSONB        DEFAULT '{}',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_chunks_document_id ON document_chunks (document_id);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_tenant_id ON document_chunks (tenant_id);

-- ============================================================
-- document_versions 文档版本表 (Sprint 2)
-- ============================================================
CREATE TABLE IF NOT EXISTS document_versions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id VARCHAR(36)  NOT NULL,
    version     INTEGER      NOT NULL,
    file_path   TEXT,
    file_size   BIGINT,
    chunk_count INTEGER,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(36)
);

CREATE INDEX IF NOT EXISTS idx_doc_versions_document_id ON document_versions (document_id);

-- ============================================================
-- chunk_strategies 分块策略表 (Sprint 2)
-- ============================================================
CREATE TABLE IF NOT EXISTS chunk_strategies (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     BIGINT       NOT NULL,
    space_id      VARCHAR(36),
    name          VARCHAR(100) NOT NULL,
    strategy_type VARCHAR(50)  NOT NULL,
    chunk_size    INTEGER      DEFAULT 512,
    chunk_overlap INTEGER      DEFAULT 50,
    split_markers TEXT,
    is_active     BOOLEAN      DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chunk_strategies_tenant_id ON chunk_strategies (tenant_id);

-- ============================================================
-- qa_sessions 问答会话表 (Sprint 3)
-- ============================================================
CREATE TABLE IF NOT EXISTS qa_sessions (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(36)  NOT NULL,
    tenant_id  BIGINT       NOT NULL,
    space_id   VARCHAR(36),
    title      VARCHAR(500),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_qa_sessions_user_id ON qa_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_qa_sessions_tenant_id ON qa_sessions (tenant_id);

-- ============================================================
-- qa_messages 问答消息表 (Sprint 3)
-- ============================================================
CREATE TABLE IF NOT EXISTS qa_messages (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(36)  NOT NULL,
    role       VARCHAR(20)  NOT NULL,
    content    TEXT         NOT NULL,
    sources    JSONB        DEFAULT '[]',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_qa_messages_session_id ON qa_messages (session_id);

-- ============================================================
-- writing_drafts 写作草稿表 (Sprint 4)
-- ============================================================
CREATE TABLE IF NOT EXISTS writing_drafts (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(36)  NOT NULL,
    tenant_id  BIGINT       NOT NULL,
    title      VARCHAR(500),
    category   VARCHAR(50),
    content    TEXT,
    space_id   VARCHAR(36),
    status     VARCHAR(20)  DEFAULT 'draft',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_writing_drafts_user_id ON writing_drafts (user_id);
CREATE INDEX IF NOT EXISTS idx_writing_drafts_tenant_id ON writing_drafts (tenant_id);

-- ============================================================
-- announcements 公告表 (Sprint 4)
-- ============================================================
CREATE TABLE IF NOT EXISTS announcements (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title      VARCHAR(500) NOT NULL,
    content    TEXT,
    status     VARCHAR(20)  DEFAULT 'draft',
    created_by VARCHAR(36),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- RLS for Sprint 2-4 tables
-- ============================================================
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON documents
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_chunks
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE chunk_strategies ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON chunk_strategies
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE qa_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON qa_sessions
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE writing_drafts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON writing_drafts
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));
