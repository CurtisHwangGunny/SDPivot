ALTER TABLE audit_logs ADD COLUMN user_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN resource_type VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN resource_id VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN ip_address VARCHAR(45) NOT NULL DEFAULT '';

UPDATE audit_logs SET user_id = actor_user_id WHERE user_id = '' AND actor_user_id <> '';
UPDATE audit_logs SET resource_type = target_type WHERE resource_type = '' AND target_type <> '';
UPDATE audit_logs SET resource_id = target_id WHERE resource_id = '' AND target_id <> '';

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
