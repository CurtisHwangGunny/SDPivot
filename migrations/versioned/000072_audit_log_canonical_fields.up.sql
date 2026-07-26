-- Add the canonical audit-log field names used by external consumers while
-- retaining the existing actor/target columns for API compatibility.
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS user_id VARCHAR(36),
    ADD COLUMN IF NOT EXISTS resource_type VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS resource_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45) NOT NULL DEFAULT '';

UPDATE audit_logs
SET user_id = actor_user_id
WHERE (user_id IS NULL OR user_id = '') AND actor_user_id <> '';

UPDATE audit_logs SET user_id = '' WHERE user_id IS NULL;

UPDATE audit_logs
SET resource_type = target_type
WHERE resource_type = '' AND target_type <> '';

UPDATE audit_logs
SET resource_id = target_id
WHERE (resource_id IS NULL OR resource_id = '') AND target_id <> '';

UPDATE audit_logs SET resource_id = '' WHERE resource_id IS NULL;

ALTER TABLE audit_logs
    ALTER COLUMN user_id SET DEFAULT '',
    ALTER COLUMN user_id SET NOT NULL,
    ALTER COLUMN resource_id SET DEFAULT '',
    ALTER COLUMN resource_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id
    ON audit_logs (user_id);
