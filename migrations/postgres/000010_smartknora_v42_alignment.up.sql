-- SmartKnora PRD v4.2 / architecture review v1.1 alignment
-- Phase 1 uses the shared WeKnora database and tenant_id-based isolation.

-- User lifecycle fields: 30-day full trial, 90-day certification extension, paid conversion.
ALTER TABLE users ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS trial_phase VARCHAR(20) DEFAULT '30day';
ALTER TABLE users ADD COLUMN IF NOT EXISTS authenticated_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_extended_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_trial_phase ON users(trial_phase);

-- Tenant context helpers use BIGINT because WeKnora/SMK tenant_id is uint64/BIGINT, not UUID.
CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, p_is_ops_admin BOOLEAN DEFAULT FALSE)
RETURNS void AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, false);
    PERFORM set_config('app.is_ops_admin', CASE WHEN p_is_ops_admin THEN 'true' ELSE 'false' END, false);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS BIGINT AS $$
BEGIN
    RETURN COALESCE(NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT, 0);
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION is_ops_admin_context()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(NULLIF(current_setting('app.is_ops_admin', true), '')::BOOLEAN, false);
END;
$$ LANGUAGE plpgsql STABLE;

-- AI writing source fields and category-to-knowledge-space mapping.
ALTER TABLE writing_drafts ADD COLUMN IF NOT EXISTS source_type VARCHAR(30) NOT NULL DEFAULT 'knowledge_base';
ALTER TABLE writing_drafts ADD COLUMN IF NOT EXISTS web_search_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS write_category_config (
    id                 VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id           BIGINT      NOT NULL,
    category            VARCHAR(50) NOT NULL,
    default_space_id    VARCHAR(36),
    source_type         VARCHAR(30) NOT NULL DEFAULT 'knowledge_base',
    web_search_enabled  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, category)
);
CREATE INDEX IF NOT EXISTS idx_wcc_tenant_category ON write_category_config(tenant_id, category);

ALTER TABLE write_category_config ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS wcc_tenant_isolation ON write_category_config;
CREATE POLICY wcc_tenant_isolation ON write_category_config
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());

-- Harden frequently used RLS policies with missing-setting-safe helper functions.
DROP POLICY IF EXISTS ks_tenant_isolation ON knowledge_spaces;
CREATE POLICY ks_tenant_isolation ON knowledge_spaces
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());

DROP POLICY IF EXISTS doc_tenant_isolation ON documents;
CREATE POLICY doc_tenant_isolation ON documents
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());

DROP POLICY IF EXISTS qs_tenant_isolation ON qa_sessions;
CREATE POLICY qs_tenant_isolation ON qa_sessions
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());

DROP POLICY IF EXISTS qm_tenant_isolation ON qa_messages;
CREATE POLICY qm_tenant_isolation ON qa_messages
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());

DROP POLICY IF EXISTS wd_tenant_isolation ON writing_drafts;
CREATE POLICY wd_tenant_isolation ON writing_drafts
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());

DROP POLICY IF EXISTS ann_tenant_isolation ON announcements;
CREATE POLICY ann_tenant_isolation ON announcements
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());

DROP POLICY IF EXISTS tu_tenant_isolation ON token_usage;
CREATE POLICY tu_tenant_isolation ON token_usage
    USING (is_ops_admin_context() OR tenant_id = get_current_tenant_id())
    WITH CHECK (is_ops_admin_context() OR tenant_id = get_current_tenant_id());
