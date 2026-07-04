-- smartKnora 运营管理端登录 (PRD 1.1.4)
-- 添加运营人员账户字段 + 密码策略 + 预置账号

-- ============================================================
-- 1. users 表扩展：运营人员字段
-- ============================================================
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_ops_admin BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_ops_admin ON users(is_ops_admin) WHERE is_ops_admin = TRUE;

-- ============================================================
-- 2. 预置运营管理员账号
--    账号: admin@smartknora.com
--    密码: SmartKnora@2026! (首次登录强制修改)
--    bcrypt hash: $2a$10$... (generated at deploy time)
-- ============================================================
-- Use a DO block to create the ops admin only if not exists
DO $$
DECLARE
    ops_user_id VARCHAR(36);
BEGIN
    -- Check if ops admin already exists
    SELECT id INTO ops_user_id FROM users WHERE email = 'admin@smartknora.com';
    
    IF ops_user_id IS NULL THEN
        -- Create the ops admin user
        -- Default password hash for 'SmartKnora@2026!' 
        -- (This hash is pre-computed; actual deployment should use env var)
        INSERT INTO users (id, username, email, password_hash, is_active, is_system_admin, is_ops_admin, 
                          must_change_password, tenant_id, created_at, updated_at)
        VALUES (
            gen_random_uuid()::text,
            'ops_admin',
            'admin@smartknora.com',
            '$2a$10$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2',
            TRUE,    -- is_active
            TRUE,    -- is_system_admin
            TRUE,    -- is_ops_admin
            TRUE,    -- must_change_password (首次登录改密)
            1,       -- tenant_id
            NOW(),
            NOW()
        );
        
        RAISE NOTICE 'Ops admin account created: admin@smartknora.com';
    ELSE
        RAISE NOTICE 'Ops admin account already exists';
    END IF;
END $$;
