-- Reverse Sprint 2-4 tables for smartKnora

-- Drop RLS policies first
DROP POLICY IF EXISTS tenant_isolation ON writing_drafts;
ALTER TABLE writing_drafts DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON qa_sessions;
ALTER TABLE qa_sessions DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON chunk_strategies;
ALTER TABLE chunk_strategies DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON document_chunks;
ALTER TABLE document_chunks DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON documents;
ALTER TABLE documents DISABLE ROW LEVEL SECURITY;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS writing_drafts;
DROP TABLE IF EXISTS qa_messages;
DROP TABLE IF EXISTS qa_sessions;
DROP TABLE IF EXISTS chunk_strategies;
DROP TABLE IF EXISTS document_versions;
DROP TABLE IF EXISTS document_chunks;
DROP TABLE IF EXISTS documents;
