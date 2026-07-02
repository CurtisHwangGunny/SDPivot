-- smartKnora Row-Level Security (RLS) policies
-- Requires PostgreSQL 15+

-- ============================================================
-- Tenant context helper
-- ============================================================
CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id UUID)
RETURNS void AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, false);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================
-- Application roles
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user WITH LOGIN;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'ops_admin') THEN
        CREATE ROLE ops_admin WITH LOGIN BYPASSRLS;
    END IF;
END
$$;

-- ============================================================
-- knowledge_spaces
-- ============================================================
ALTER TABLE knowledge_spaces ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON knowledge_spaces
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- token_usage
-- ============================================================
ALTER TABLE token_usage ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON token_usage
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- space_categories
-- ============================================================
ALTER TABLE space_categories ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON space_categories
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));
