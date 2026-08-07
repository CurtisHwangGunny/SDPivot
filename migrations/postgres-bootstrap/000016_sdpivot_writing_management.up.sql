CREATE TABLE IF NOT EXISTS writing_category (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id   BIGINT NOT NULL,
    name        VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort        INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_writing_category_tenant_sort
    ON writing_category (tenant_id, sort, created_at);

CREATE TABLE IF NOT EXISTS writing_template (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id   BIGINT NOT NULL,
    category_id VARCHAR(36) NOT NULL REFERENCES writing_category(id) ON DELETE RESTRICT,
    name        VARCHAR(100) NOT NULL,
    content     TEXT NOT NULL,
    is_builtin  BOOLEAN NOT NULL DEFAULT FALSE,
    sort        INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, category_id, name)
);

CREATE INDEX IF NOT EXISTS idx_writing_template_tenant_category_sort
    ON writing_template (tenant_id, category_id, sort, created_at);

ALTER TABLE writing_category ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_writing_category_tenant ON writing_category;
CREATE POLICY sdpivot_writing_category_tenant ON writing_category
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE writing_template ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_writing_template_tenant ON writing_template;
CREATE POLICY sdpivot_writing_template_tenant ON writing_template
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
