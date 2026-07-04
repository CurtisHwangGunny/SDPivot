-- smartKnora Phase 1 fix migration - ROLLBACK
-- Reverts all changes made by 000006_smartknora_fixes.up.sql

-- 1. Drop RLS policies (created in this migration)
DROP POLICY IF EXISTS al_tenant_isolation ON audit_logs;
DROP POLICY IF EXISTS sc_tenant_isolation ON space_categories;
DROP POLICY IF EXISTS cs_tenant_isolation ON chunk_strategies;
DROP POLICY IF EXISTS dv_tenant_isolation ON document_versions;

-- 2. Drop added columns (keep data safety in mind)
-- Note: These columns are NOT dropped in normal rollback to avoid data loss
-- Uncomment only if you truly need to roll back the schema:
-- ALTER TABLE qa_messages DROP COLUMN IF EXISTS sources;
-- ALTER TABLE writing_drafts DROP COLUMN IF EXISTS space_id;
-- ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS device_id;
-- ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS family;
-- ALTER TABLE knowledge_spaces DROP COLUMN IF EXISTS icon;
-- ALTER TABLE knowledge_spaces DROP COLUMN IF EXISTS creator_id;

-- 3. Drop tables created in this migration
-- Note: Tables are NOT dropped in normal rollback to avoid data loss
-- Uncomment only if you truly need to roll back:
-- DROP TABLE IF EXISTS audit_logs CASCADE;
-- DROP TABLE IF EXISTS document_versions CASCADE;
-- DROP TABLE IF EXISTS chunk_strategies CASCADE;
-- DROP TABLE IF EXISTS space_categories CASCADE;
