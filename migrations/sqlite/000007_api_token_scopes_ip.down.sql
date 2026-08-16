DROP INDEX IF EXISTS idx_api_tokens_prefix;
DROP INDEX IF EXISTS idx_api_tokens_tenant_active;
ALTER TABLE api_tokens DROP COLUMN updated_at;
ALTER TABLE api_tokens DROP COLUMN created_by;
ALTER TABLE api_tokens DROP COLUMN scope_enforced;
ALTER TABLE api_tokens DROP COLUMN allowed_ips;
ALTER TABLE api_tokens DROP COLUMN scopes;
ALTER TABLE api_tokens DROP COLUMN prefix;
