-- SDPivot intelligent tags, manual adjustments, and review feedback.
-- The core tag dictionary is shared; tenant-created dimensions are isolated by tenant_id.

ALTER TABLE tag_dimensions ADD COLUMN IF NOT EXISTS tenant_id BIGINT;
ALTER TABLE tag_dimensions ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE tag_dimensions ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE tag_dimensions DROP CONSTRAINT IF EXISTS tag_dimensions_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_dimensions_global_code
    ON tag_dimensions (code) WHERE tenant_id IS NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_dimensions_tenant_code
    ON tag_dimensions (tenant_id, code) WHERE tenant_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tag_dimensions_tenant_enabled
    ON tag_dimensions (tenant_id, enabled) WHERE deleted_at IS NULL;

ALTER TABLE document_tags DROP CONSTRAINT IF EXISTS document_tags_document_id_fkey;
DROP INDEX IF EXISTS idx_document_tags_dimension;
ALTER TABLE document_tags ADD COLUMN IF NOT EXISTS id VARCHAR(36) DEFAULT uuid_generate_v4()::TEXT;
ALTER TABLE document_tags ADD COLUMN IF NOT EXISTS source VARCHAR(16) NOT NULL DEFAULT 'manual';
ALTER TABLE document_tags ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE document_tags ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE document_tags ADD CONSTRAINT document_tags_source_check CHECK (source IN ('auto', 'manual'));
ALTER TABLE document_tags ALTER COLUMN id SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_tags_id ON document_tags (id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_tags_document_tag
    ON document_tags (document_id, tag_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tag_adjustment_log (
    id              VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::TEXT,
    tenant_id       BIGINT NOT NULL,
    document_id     VARCHAR(36) NOT NULL,
    old_tag_id      VARCHAR(36),
    new_tag_id      VARCHAR(36),
    operator        VARCHAR(36) NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'applied',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tag_adjustment_log_document
    ON tag_adjustment_log (tenant_id, document_id, created_at DESC);

CREATE TABLE IF NOT EXISTS tag_feedback_queue (
    id              VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::TEXT,
    tenant_id       BIGINT NOT NULL,
    document_id     VARCHAR(36) NOT NULL,
    tag_id          VARCHAR(36) NOT NULL,
    original_tag    VARCHAR(128) NOT NULL DEFAULT '',
    feedback        VARCHAR(16) NOT NULL CHECK (feedback IN ('correct', 'incorrect')),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'reviewed', 'rejected')),
    created_by      VARCHAR(36) NOT NULL,
    reviewed_by     VARCHAR(36),
    reviewed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tag_feedback_queue_status
    ON tag_feedback_queue (tenant_id, status, created_at DESC);

ALTER TABLE tag_dimensions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000013_tag_dimensions ON tag_dimensions;
CREATE POLICY sdpivot_op_bootstrap_000013_tag_dimensions ON tag_dimensions
    FOR ALL
    USING (tenant_id IS NULL OR tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id IS NULL OR tenant_id = get_current_tenant_id());

ALTER TABLE document_tags ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000013_document_tags ON document_tags;
CREATE POLICY sdpivot_op_bootstrap_000013_document_tags ON document_tags
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE tag_adjustment_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_op_bootstrap_000013_tag_adjustment_log ON tag_adjustment_log
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE tag_feedback_queue ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_op_bootstrap_000013_tag_feedback_queue ON tag_feedback_queue
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
