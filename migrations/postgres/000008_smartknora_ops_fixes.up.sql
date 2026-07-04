-- 000008_smartknora_ops_fixes.up.sql
-- Ops admin management fixes (PRD §4.1)

-- 1. email length: align DB varchar(200) -> varchar(255) to match Go struct
ALTER TABLE users ALTER COLUMN email TYPE varchar(255);

-- 2. org_ext: add subscription_status if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='org_ext' AND column_name='subscription_status') THEN
        ALTER TABLE org_ext ADD COLUMN subscription_status VARCHAR(20) DEFAULT 'free';
    END IF;
END $$;

-- 3. audit_logs: ensure indexes for ops filtering
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

-- 4. announcements: ensure status column
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='announcements' AND column_name='status') THEN
        ALTER TABLE announcements ADD COLUMN status VARCHAR(20) DEFAULT 'draft';
    END IF;
END $$;
