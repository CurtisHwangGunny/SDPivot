DROP INDEX IF EXISTS idx_audit_logs_user_id;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS ip_address,
    DROP COLUMN IF EXISTS resource_id,
    DROP COLUMN IF EXISTS resource_type,
    DROP COLUMN IF EXISTS user_id;
