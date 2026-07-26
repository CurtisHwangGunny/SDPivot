-- Align OP administrator identities with the single-tenant authorization model.
INSERT INTO tenants (id, name, description, api_key, status, business, created_at, updated_at)
VALUES (1, 'SDPivot', 'SDPivot OP tenant', gen_random_uuid()::TEXT, 'active', 'sdpivot', NOW(), NOW())
ON CONFLICT (id) DO UPDATE
SET status = 'active', deleted_at = NULL, updated_at = NOW();

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS access_role VARCHAR(32) NOT NULL DEFAULT 'knowledge_viewer',
    ADD COLUMN IF NOT EXISTS department_id VARCHAR(36);

CREATE INDEX IF NOT EXISTS idx_users_access_role ON users (access_role);
CREATE INDEX IF NOT EXISTS idx_users_department_id ON users (department_id);

UPDATE users
   SET tenant_id = 1,
       can_access_all_tenants = FALSE,
       access_role = 'super_admin',
       updated_at = NOW()
 WHERE is_ops_admin = TRUE OR is_system_admin = TRUE;

INSERT INTO tenant_members (user_id, tenant_id, role, status, joined_at, created_at, updated_at)
SELECT id, 1, 'owner', 'active', NOW(), NOW(), NOW()
  FROM users
 WHERE deleted_at IS NULL
   AND is_active = TRUE
   AND (is_ops_admin = TRUE OR is_system_admin = TRUE)
ON CONFLICT (user_id, tenant_id) WHERE deleted_at IS NULL
DO UPDATE SET role = 'owner', status = 'active', updated_at = NOW();
