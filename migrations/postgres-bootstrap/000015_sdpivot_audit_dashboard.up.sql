-- Complete the canonical SDPivot audit-log fields used by admin audit queries.
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS user_id VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS resource_type VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS resource_id VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45) NOT NULL DEFAULT '';

UPDATE audit_logs
SET user_id = actor_user_id
WHERE user_id = '' AND actor_user_id <> '';

UPDATE audit_logs
SET resource_type = target_type
WHERE resource_type = '' AND target_type <> '';

UPDATE audit_logs
SET resource_id = target_id
WHERE resource_id = '' AND target_id <> '';

UPDATE audit_logs
SET ip_address = ip
WHERE ip_address = '' AND COALESCE(ip, '') <> '';

CREATE INDEX IF NOT EXISTS idx_sdpivot_audit_tenant_created
    ON audit_logs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sdpivot_audit_tenant_resource
    ON audit_logs (tenant_id, resource_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sdpivot_qa_messages_tenant_created
    ON qa_messages (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sdpivot_documents_tenant_created
    ON documents (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sdpivot_token_usage_tenant_created
    ON token_usage (tenant_id, created_at DESC);
