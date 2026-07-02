-- Reverse: drop RLS policies, helper function, and roles

-- Drop policies
DROP POLICY IF EXISTS tenant_isolation ON knowledge_spaces;
DROP POLICY IF EXISTS tenant_isolation ON token_usage;
DROP POLICY IF EXISTS tenant_isolation ON space_categories;

-- Disable RLS
ALTER TABLE knowledge_spaces DISABLE ROW LEVEL SECURITY;
ALTER TABLE token_usage      DISABLE ROW LEVEL SECURITY;
ALTER TABLE space_categories DISABLE ROW LEVEL SECURITY;

-- Drop helper function
DROP FUNCTION IF EXISTS set_tenant_context(UUID);

-- Revoke and drop roles (ignore errors if roles don't exist)
DO $$
BEGIN
    REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM app_user;
    REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ops_admin;
    DROP ROLE IF EXISTS app_user;
    DROP ROLE IF EXISTS ops_admin;
EXCEPTION WHEN OTHERS THEN
    NULL;
END
$$;
