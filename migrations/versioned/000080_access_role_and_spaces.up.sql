-- OP RBAC v2 compatibility layer on the official v0.7.2 schema.
ALTER TABLE users ADD COLUMN IF NOT EXISTS access_role VARCHAR(32) NOT NULL DEFAULT 'knowledge_viewer';
ALTER TABLE users ADD COLUMN IF NOT EXISTS department_id VARCHAR(36);
CREATE INDEX IF NOT EXISTS idx_users_access_role ON users(access_role);
CREATE INDEX IF NOT EXISTS idx_users_department_id ON users(department_id);

CREATE TABLE IF NOT EXISTS knowledge_spaces (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant_id BIGINT NOT NULL,
    org_id VARCHAR(36),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private',
    owner_id VARCHAR(36),
    icon VARCHAR(50),
    creator_id VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_knowledge_spaces_tenant ON knowledge_spaces(tenant_id);

CREATE TABLE IF NOT EXISTS space_members (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    space_id VARCHAR(36) NOT NULL REFERENCES knowledge_spaces(id),
    user_id VARCHAR(36) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(space_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_space_members_user ON space_members(user_id);

CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, p_is_ops_admin BOOLEAN DEFAULT FALSE)
RETURNS void LANGUAGE plpgsql SECURITY INVOKER AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::text, true);
    PERFORM set_config('app.is_ops_admin', CASE WHEN p_is_ops_admin THEN 'true' ELSE 'false' END, true);
    PERFORM set_config('app.access_role', CASE WHEN p_is_ops_admin THEN 'super_admin' ELSE 'knowledge_viewer' END, true);
END;
$$;

CREATE OR REPLACE FUNCTION get_current_access_role()
RETURNS text LANGUAGE sql STABLE SECURITY INVOKER AS $$
    SELECT COALESCE(NULLIF(current_setting('app.access_role', true), ''), 'knowledge_viewer');
$$;

CREATE OR REPLACE FUNCTION prevent_last_super_admin_change()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.access_role = 'super_admin' AND NEW.access_role <> 'super_admin'
       AND (SELECT COUNT(*) FROM users WHERE access_role = 'super_admin' AND deleted_at IS NULL) <= 1 THEN
        RAISE EXCEPTION 'cannot demote the last super_admin';
    END IF;
    IF OLD.access_role = 'super_admin' AND NEW.is_active = FALSE
       AND (SELECT COUNT(*) FROM users WHERE access_role = 'super_admin' AND deleted_at IS NULL AND is_active = TRUE) <= 1 THEN
        RAISE EXCEPTION 'cannot disable the last super_admin';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS users_last_super_admin_guard ON users;
CREATE TRIGGER users_last_super_admin_guard BEFORE UPDATE ON users FOR EACH ROW
EXECUTE FUNCTION prevent_last_super_admin_change();

ALTER TABLE knowledge_spaces ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_space_tenant_isolation ON knowledge_spaces;
CREATE POLICY sdpivot_space_tenant_isolation ON knowledge_spaces FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE space_members ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_space_member_tenant_isolation ON space_members;
CREATE POLICY sdpivot_space_member_tenant_isolation ON space_members FOR ALL
    USING (EXISTS (SELECT 1 FROM knowledge_spaces s WHERE s.id = space_members.space_id AND s.tenant_id = get_current_tenant_id()))
    WITH CHECK (EXISTS (SELECT 1 FROM knowledge_spaces s WHERE s.id = space_members.space_id AND s.tenant_id = get_current_tenant_id()));
