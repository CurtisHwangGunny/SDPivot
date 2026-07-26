DROP INDEX IF EXISTS idx_audit_logs_user_id;
ALTER TABLE audit_logs DROP COLUMN ip_address;
ALTER TABLE audit_logs DROP COLUMN resource_id;
ALTER TABLE audit_logs DROP COLUMN resource_type;
ALTER TABLE audit_logs DROP COLUMN user_id;
