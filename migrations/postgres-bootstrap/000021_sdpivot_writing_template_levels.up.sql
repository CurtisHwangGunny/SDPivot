CREATE TABLE IF NOT EXISTS template_configs (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id   BIGINT NOT NULL,
    category_id VARCHAR(36) NOT NULL REFERENCES writing_category(id) ON DELETE RESTRICT,
    name        VARCHAR(100) NOT NULL,
    content     TEXT NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    is_builtin  BOOLEAN NOT NULL DEFAULT FALSE,
    sort        INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, category_id, name)
);

CREATE INDEX IF NOT EXISTS idx_template_configs_tenant_category_sort
    ON template_configs (tenant_id, category_id, sort, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_template_configs_one_default
    ON template_configs (tenant_id, category_id)
    WHERE is_default = TRUE;

CREATE TABLE IF NOT EXISTS user_template_prefs (
    id                  VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id             VARCHAR(36) NOT NULL,
    tenant_id           BIGINT NOT NULL,
    default_template_id VARCHAR(36),
    category_order      JSONB NOT NULL DEFAULT '[]'::JSONB,
    templates           JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_user_template_prefs_tenant_user
    ON user_template_prefs (tenant_id, user_id);

ALTER TABLE template_configs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_template_configs_tenant ON template_configs;
CREATE POLICY sdpivot_template_configs_tenant ON template_configs
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE user_template_prefs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_user_template_prefs_tenant ON user_template_prefs;
CREATE POLICY sdpivot_user_template_prefs_tenant ON user_template_prefs
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
