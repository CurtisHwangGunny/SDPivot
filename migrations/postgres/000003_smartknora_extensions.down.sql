-- Rollback smartKnora extension tables

DROP POLICY IF EXISTS tenant_isolation ON org_ext;
ALTER TABLE org_ext DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS org_ext;
DROP TABLE IF EXISTS smartknora_user_profiles;
