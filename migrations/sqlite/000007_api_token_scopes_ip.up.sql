ALTER TABLE api_tokens ADD COLUMN prefix VARCHAR(6) NOT NULL DEFAULT '';
ALTER TABLE api_tokens ADD COLUMN scopes TEXT NOT NULL DEFAULT '[]';
ALTER TABLE api_tokens ADD COLUMN allowed_ips TEXT NOT NULL DEFAULT '[]';
ALTER TABLE api_tokens ADD COLUMN scope_enforced BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_tokens ADD COLUMN created_by VARCHAR(36);
ALTER TABLE api_tokens ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE api_tokens SET created_by = user_id WHERE created_by IS NULL;

CREATE INDEX IF NOT EXISTS idx_api_tokens_tenant_active ON api_tokens (tenant_id, revoked_at, expires_at);
CREATE INDEX IF NOT EXISTS idx_api_tokens_prefix ON api_tokens (tenant_id, prefix);
