DROP INDEX IF EXISTS idx_api_tokens_prefix;
DROP INDEX IF EXISTS idx_api_tokens_tenant_active;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS updated_at;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS created_by;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS scope_enforced;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS allowed_ips;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS scopes;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS prefix;
