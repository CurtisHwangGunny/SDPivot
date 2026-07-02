DROP POLICY IF EXISTS document_versions_tenant_isolation ON document_versions;
ALTER TABLE document_versions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS qa_messages_tenant_isolation ON qa_messages;
ALTER TABLE qa_messages DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS announcements_tenant_isolation ON announcements;
ALTER TABLE announcements DISABLE ROW LEVEL SECURITY;
