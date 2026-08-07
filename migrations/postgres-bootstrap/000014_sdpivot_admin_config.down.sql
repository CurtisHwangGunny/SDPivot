DROP POLICY IF EXISTS sdpivot_op_bootstrap_000014_security_settings ON security_settings;
DROP TABLE IF EXISTS security_settings;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000014_system_settings ON system_settings;
DROP TABLE IF EXISTS system_settings;

DROP INDEX IF EXISTS idx_api_tokens_prefix;
DROP INDEX IF EXISTS idx_api_tokens_tenant_active;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS updated_at;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS created_by;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS scopes;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS prefix;
